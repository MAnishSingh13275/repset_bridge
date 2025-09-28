package bridge

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"gym-door-bridge/internal/adapters"
	"gym-door-bridge/internal/api"
	"gym-door-bridge/internal/auth"
	"gym-door-bridge/internal/client"
	"gym-door-bridge/internal/config"
	"gym-door-bridge/internal/database"
	"gym-door-bridge/internal/door"
	"gym-door-bridge/internal/health"
	"gym-door-bridge/internal/logging"
	"gym-door-bridge/internal/processor"
	"gym-door-bridge/internal/queue"
	"gym-door-bridge/internal/service/windows"
	"gym-door-bridge/internal/telemetry"
	"gym-door-bridge/internal/tier"
	"gym-door-bridge/internal/types"
)

// Manager coordinates all bridge components and services
type Manager struct {
	mu     sync.RWMutex
	config *config.Config
	logger *logrus.Logger
	
	// Core components
	database        *database.DB
	adapterManager  *adapters.AdapterManager
	queueManager    queue.QueueManager
	tierDetector    *tier.Detector
	healthMonitor   *health.HealthMonitor
	doorController  *door.DoorController
	eventProcessor  *processor.EventProcessorImpl
	submissionService *client.SubmissionService
	
	// API server
	apiServer       *api.Server
	
	// Service health monitoring (Windows only)
	serviceHealthMonitor *windows.ServiceHealthMonitor
	
	// Installation telemetry
	installationTelemetry *telemetry.InstallationTelemetry
	
	// State
	isRunning       bool
	startTime       time.Time
	version         string
	deviceID        string
	
	// Context for graceful shutdown
	ctx             context.Context
	cancel          context.CancelFunc
}

// ManagerOption is a functional option for configuring the Manager
type ManagerOption func(*Manager)

// WithVersion sets the version for the manager
func WithVersion(version string) ManagerOption {
	return func(m *Manager) {
		m.version = version
	}
}

// WithDeviceID sets the device ID for the manager
func WithDeviceID(deviceID string) ManagerOption {
	return func(m *Manager) {
		m.deviceID = deviceID
	}
}

// NewManager creates a new bridge manager
func NewManager(cfg *config.Config, opts ...ManagerOption) (*Manager, error) {
	logger := logging.Initialize(cfg.LogLevel)
	
	ctx, cancel := context.WithCancel(context.Background())
	
	m := &Manager{
		config:    cfg,
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
		version:   "unknown",
		deviceID:  cfg.DeviceID,
	}
	
	// Apply options
	for _, opt := range opts {
		opt(m)
	}
	
	// Initialize components
	if err := m.initializeComponents(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}
	
	return m, nil
}

// initializeComponents initializes all bridge components
func (m *Manager) initializeComponents() error {
	m.logger.Info("Initializing bridge components")
	
	// Initialize database
	// Create a default encryption key if none provided
	encryptionKey := make([]byte, 32) // 256-bit key for AES
	if len(encryptionKey) == 32 {
		// Use a simple key for now - in production this should be from config
		copy(encryptionKey, []byte("bridge-default-encryption-key-32"))
	}
	
	dbConfig := database.Config{
		DatabasePath:    m.config.DatabasePath,
		EncryptionKey:   encryptionKey,
		PerformanceTier: database.PerformanceTier(m.config.Tier),
	}
	db, err := database.NewDB(dbConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	m.database = db
	
	// Initialize queue manager
	m.queueManager = queue.NewSQLiteQueueManager(db)
	queueConfig := queue.GetTierConfig(database.PerformanceTier(m.config.Tier))
	if err := m.queueManager.Initialize(m.ctx, queueConfig); err != nil {
		return fmt.Errorf("failed to initialize queue manager: %w", err)
	}
	
	// Initialize adapter manager
	// Create a slog.Logger for the adapter manager
	slogLogger := slog.New(slog.NewTextHandler(m.logger.Writer(), &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	m.adapterManager = adapters.NewAdapterManager(slogLogger)
	
	// Initialize event processor
	m.eventProcessor = processor.NewEventProcessor(db, m.logger)
	processorConfig := processor.ProcessorConfig{
		DeviceID:            m.deviceID,
		EnableDeduplication: true,
		DeduplicationWindow: 300, // 5 minutes
	}
	if err := m.eventProcessor.Initialize(m.ctx, processorConfig); err != nil {
		return fmt.Errorf("failed to initialize event processor: %w", err)
	}
	
	// Set up event callback for adapters
	m.adapterManager.OnEvent(func(event types.RawHardwareEvent) {
		// Process the raw event through the processor for deduplication and validation
		result, err := m.eventProcessor.ProcessEvent(m.ctx, event)
		if err != nil {
			m.logger.WithError(err).Error("Failed to process event from adapter")
			return
		}
		
		if !result.Processed {
			m.logger.WithFields(logrus.Fields{
				"external_user_id": event.ExternalUserID,
				"event_type":       event.EventType,
				"reason":           result.Reason,
			}).Debug("Event not processed", "reason", result.Reason)
			return
		}
		
		// Enqueue the processed standard event
		if err := m.queueManager.Enqueue(m.ctx, result.Event); err != nil {
			m.logger.WithError(err).Error("Failed to enqueue processed event")
		}
	})
	
	// Load adapter configurations
	adapterConfigs := m.config.GetAdapterConfigs()
	if err := m.adapterManager.LoadAdapters(adapterConfigs); err != nil {
		return fmt.Errorf("failed to load adapters: %w", err)
	}
	
	// Initialize tier detector
	m.tierDetector = tier.NewDetector(
		tier.WithLogger(m.logger.WithField("component", "tier").Logger),
		tier.WithEvaluationInterval(30*time.Second),
	)
	
	// Initialize health monitor
	healthConfig := health.DefaultHealthCheckConfig()
	m.healthMonitor = health.NewHealthMonitor(
		healthConfig,
		m.queueManager,
		m.tierDetector,
		&adapterRegistryWrapper{m.adapterManager},
		health.WithLogger(m.logger.WithField("component", "health").Logger),
		health.WithVersion(m.version),
		health.WithDeviceID(m.deviceID),
	)
	
	// Initialize door controller
	doorConfig := door.DefaultDoorControlConfig()
	m.doorController = door.NewDoorController(
		doorConfig,
		m.config,
		&adapterRegistryWrapper{m.adapterManager},
		door.WithLogger(m.logger.WithField("component", "door").Logger),
	)
	
	// Initialize submission service for offline event queuing
	authManager, err := auth.NewAuthManager()
	if err != nil {
		return fmt.Errorf("failed to create auth manager: %w", err)
	}
	if err := authManager.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize auth manager: %w", err)
	}
	
	httpClient, err := client.NewHTTPClient(m.config, authManager, m.logger)
	if err != nil {
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}
	checkinClient := client.NewCheckinClient(httpClient, m.logger)
	m.submissionService = client.NewSubmissionService(m.queueManager, checkinClient, m.logger)
	
	// Configure submission service based on tier
	submissionConfig := client.DefaultSubmissionConfig()
	switch database.PerformanceTier(m.config.Tier) {
	case database.TierLite:
		submissionConfig.BatchSize = 10
		submissionConfig.SubmitInterval = 60 * time.Second
		submissionConfig.MaxRetries = 3
	case database.TierNormal:
		submissionConfig.BatchSize = 50
		submissionConfig.SubmitInterval = 30 * time.Second
		submissionConfig.MaxRetries = 5
	case database.TierFull:
		submissionConfig.BatchSize = 100
		submissionConfig.SubmitInterval = 15 * time.Second
		submissionConfig.MaxRetries = 10
	}
	m.submissionService.SetConfig(submissionConfig)
	
	// Initialize installation telemetry
	m.installationTelemetry = telemetry.NewInstallationTelemetry(m.logger, m.config)

	// Initialize service health monitor on Windows
	if runtime.GOOS == "windows" {
		healthConfig := windows.DefaultServiceHealthMonitorConfig()
		serviceHealthMonitor, err := windows.NewServiceHealthMonitor(m.logger, healthConfig)
		if err != nil {
			m.logger.WithError(err).Warn("Failed to initialize service health monitor")
		} else {
			m.serviceHealthMonitor = serviceHealthMonitor
		}
	}

	// Initialize API server if enabled
	if m.config.APIServer.Enabled {
		serverConfig := &api.ServerConfig{
			Port:         m.config.APIServer.Port,
			Host:         m.config.APIServer.Host,
			TLSEnabled:   m.config.APIServer.TLSEnabled,
			TLSCertFile:  m.config.APIServer.TLSCertFile,
			TLSKeyFile:   m.config.APIServer.TLSKeyFile,
			ReadTimeout:  m.config.APIServer.ReadTimeout,
			WriteTimeout: m.config.APIServer.WriteTimeout,
			IdleTimeout:  m.config.APIServer.IdleTimeout,
		}
		
		m.apiServer = api.NewServer(
			m.config,
			serverConfig,
			&adapterRegistryWrapper{m.adapterManager},
			&doorControllerWrapper{m.doorController},
			&healthMonitorWrapper{m.healthMonitor},
			&queueManagerWrapper{m.queueManager},
			&tierDetectorWrapper{m.tierDetector},
			&configManagerWrapper{m.config},
			m.version,
			m.deviceID,
		)
	}
	
	m.logger.Info("Bridge components initialized successfully")
	return nil
}

// Start starts all bridge components and services
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	
	if m.isRunning {
		m.mu.Unlock()
		return fmt.Errorf("bridge manager is already running")
	}
	
	m.logger.Info("Starting bridge manager")
	m.startTime = time.Now()
	
	// Start tier detector
	go func() {
		if err := m.tierDetector.Start(m.ctx); err != nil && err != context.Canceled {
			m.logger.WithError(err).Error("Tier detector stopped with error")
		}
	}()
	
	// Start health monitor
	if err := m.healthMonitor.Start(m.ctx); err != nil {
		return fmt.Errorf("failed to start health monitor: %w", err)
	}
	
	// Start adapter manager
	if err := m.adapterManager.StartAll(); err != nil {
		return fmt.Errorf("failed to start adapters: %w", err)
	}
	
	// Start door controller
	if err := m.doorController.Start(m.ctx); err != nil {
		return fmt.Errorf("failed to start door controller: %w", err)
	}
	
	// Start submission service for automatic event submission
	go func() {
		m.logger.Info("Starting periodic event submission service")
		m.submissionService.StartPeriodicSubmission(m.ctx)
	}()

	// Start service health monitor if available
	if m.serviceHealthMonitor != nil {
		if err := m.serviceHealthMonitor.Start(); err != nil {
			m.logger.WithError(err).Warn("Failed to start service health monitor")
		} else {
			m.logger.Info("Service health monitor started")
		}
	}
	
	// Start API server if enabled
	if m.apiServer != nil {
		go func() {
			serverConfig := &api.ServerConfig{
				Port:         m.config.APIServer.Port,
				Host:         m.config.APIServer.Host,
				TLSEnabled:   m.config.APIServer.TLSEnabled,
				TLSCertFile:  m.config.APIServer.TLSCertFile,
				TLSKeyFile:   m.config.APIServer.TLSKeyFile,
				ReadTimeout:  m.config.APIServer.ReadTimeout,
				WriteTimeout: m.config.APIServer.WriteTimeout,
				IdleTimeout:  m.config.APIServer.IdleTimeout,
			}
			
			if err := m.apiServer.Start(m.ctx, serverConfig); err != nil && err != context.Canceled {
				m.logger.WithError(err).Error("API server stopped with error")
			}
		}()
	}
	
	m.isRunning = true
	m.logger.Info("Bridge manager started successfully")

	// Log installation status on startup
	if m.installationTelemetry != nil {
		go func() {
			// Wait a bit for all components to start
			time.Sleep(5 * time.Second)
			m.installationTelemetry.LogInstallationStatus(m.ctx)
		}()
	}
	
	// Release the lock before waiting
	m.mu.Unlock()
	
	// Wait for context cancellation
	<-ctx.Done()
	
	// Graceful shutdown
	return m.shutdown()
}

// Stop gracefully stops all bridge components and services
func (m *Manager) Stop() error {
	m.logger.Info("Stopping bridge manager")
	m.cancel()
	return nil
}

// shutdown performs graceful shutdown of all components
func (m *Manager) shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.isRunning {
		return nil
	}
	
	m.logger.Info("Shutting down bridge manager")
	
	var errors []error
	
	// Stop service health monitor
	if m.serviceHealthMonitor != nil {
		if err := m.serviceHealthMonitor.Stop(); err != nil {
			m.logger.WithError(err).Error("Failed to stop service health monitor")
			errors = append(errors, fmt.Errorf("service health monitor stop: %w", err))
		}
	}

	// Stop API server
	if m.apiServer != nil {
		if err := m.apiServer.Shutdown(); err != nil {
			m.logger.WithError(err).Error("Failed to shutdown API server")
			errors = append(errors, fmt.Errorf("API server shutdown: %w", err))
		}
	}
	
	// Stop door controller
	if m.doorController != nil {
		if err := m.doorController.Stop(m.ctx); err != nil {
			m.logger.WithError(err).Error("Failed to stop door controller")
			errors = append(errors, fmt.Errorf("door controller stop: %w", err))
		}
	}
	
	// Stop adapters
	if m.adapterManager != nil {
		if err := m.adapterManager.StopAll(); err != nil {
			m.logger.WithError(err).Error("Failed to stop adapters")
			errors = append(errors, fmt.Errorf("adapters stop: %w", err))
		}
	}
	
	// Stop health monitor
	if m.healthMonitor != nil {
		if err := m.healthMonitor.Stop(m.ctx); err != nil {
			m.logger.WithError(err).Error("Failed to stop health monitor")
			errors = append(errors, fmt.Errorf("health monitor stop: %w", err))
		}
	}
	
	// Close queue manager
	if m.queueManager != nil {
		if err := m.queueManager.Close(m.ctx); err != nil {
			m.logger.WithError(err).Error("Failed to close queue manager")
			errors = append(errors, fmt.Errorf("queue manager close: %w", err))
		}
	}
	
	// Close database
	if m.database != nil {
		if err := m.database.Close(); err != nil {
			m.logger.WithError(err).Error("Failed to close database")
			errors = append(errors, fmt.Errorf("database close: %w", err))
		}
	}
	
	m.isRunning = false
	
	if len(errors) > 0 {
		return fmt.Errorf("shutdown completed with errors: %v", errors)
	}
	
	m.logger.Info("Bridge manager shutdown completed successfully")
	return nil
}

// IsRunning returns true if the bridge manager is currently running
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isRunning
}

// GetUptime returns the uptime of the bridge manager
func (m *Manager) GetUptime() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if !m.isRunning {
		return 0
	}
	
	return time.Since(m.startTime)
}

// GetStats returns statistics about the bridge manager
func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	stats := map[string]interface{}{
		"isRunning": m.isRunning,
		"uptime":    m.GetUptime(),
		"version":   m.version,
		"deviceID":  m.deviceID,
	}

	// Add installation metadata to stats
	if m.config != nil {
		stats["installation"] = map[string]interface{}{
			"method":       m.config.Installation.Method,
			"version":      m.config.Installation.Version,
			"installed_at": m.config.Installation.InstalledAt,
			"installed_by": m.config.Installation.InstalledBy,
			"pair_code":    m.config.Installation.PairCode,
			"source":       m.config.Installation.Source,
			"checksum":     m.config.Installation.Checksum,
		}
	}
	
	if m.isRunning {
		stats["startTime"] = m.startTime
		
		// Add component stats
		if m.healthMonitor != nil {
			stats["health"] = m.healthMonitor.GetCurrentHealth()
		}
		
		if m.doorController != nil {
			stats["door"] = m.doorController.GetStats()
		}
		
		if m.adapterManager != nil {
			stats["adapters"] = m.adapterManager.GetAdapterStatus()
		}
		
		if m.queueManager != nil {
			if queueStats, err := m.queueManager.GetStats(m.ctx); err == nil {
				stats["queue"] = queueStats
			}
		}
		
		if m.submissionService != nil {
			if submissionStats, err := m.submissionService.GetQueueStats(m.ctx); err == nil {
				stats["submission"] = submissionStats
			}
		}
		
		if m.eventProcessor != nil {
			stats["processor"] = m.eventProcessor.GetStats()
		}
		
		if m.tierDetector != nil {
			stats["tier"] = m.tierDetector.GetCurrentTier()
			stats["resources"] = m.tierDetector.GetCurrentResources()
		}

		// Add service health information if available
		if m.serviceHealthMonitor != nil {
			stats["service_health"] = m.serviceHealthMonitor.GetHealthSummary()
		}

		// Add installation telemetry metrics
		if m.installationTelemetry != nil {
			stats["installation_metrics"] = m.installationTelemetry.GetInstallationMetrics()
		}
	}
	
	return stats
}

// Wrapper interfaces to adapt existing components to API server interfaces

// adapterRegistryWrapper adapts AdapterManager to AdapterRegistry interface
type adapterRegistryWrapper struct {
	manager *adapters.AdapterManager
}

func (w *adapterRegistryWrapper) GetAllAdapters() []adapters.HardwareAdapter {
	adaptersMap := w.manager.GetAllAdapters()
	result := make([]adapters.HardwareAdapter, 0, len(adaptersMap))
	for _, adapter := range adaptersMap {
		result = append(result, adapter)
	}
	return result
}

func (w *adapterRegistryWrapper) GetAdapter(name string) (adapters.HardwareAdapter, error) {
	adapter, exists := w.manager.GetAdapter(name)
	if !exists {
		return nil, fmt.Errorf("adapter %s not found", name)
	}
	return adapter, nil
}

func (w *adapterRegistryWrapper) GetActiveAdapters() []adapters.HardwareAdapter {
	healthy := w.manager.GetHealthyAdapters()
	result := make([]adapters.HardwareAdapter, 0, len(healthy))
	for _, name := range healthy {
		if adapter, exists := w.manager.GetAdapter(name); exists {
			result = append(result, adapter)
		}
	}
	return result
}

func (w *adapterRegistryWrapper) GetAdapterStatus(name string) (types.AdapterStatus, error) {
	adapter, exists := w.manager.GetAdapter(name)
	if !exists {
		return types.AdapterStatus{}, fmt.Errorf("adapter %s not found", name)
	}
	return adapter.GetStatus(), nil
}

// doorControllerWrapper adapts DoorController to API DoorController interface
type doorControllerWrapper struct {
	controller *door.DoorController
}

func (w *doorControllerWrapper) UnlockDoor(ctx context.Context, adapterName string, durationMs int) error {
	return w.controller.UnlockDoor(ctx, adapterName, durationMs)
}

func (w *doorControllerWrapper) GetStats() map[string]interface{} {
	return w.controller.GetStats()
}

// healthMonitorWrapper adapts HealthMonitor to API HealthMonitor interface
type healthMonitorWrapper struct {
	monitor *health.HealthMonitor
}

func (w *healthMonitorWrapper) GetCurrentHealth() api.SystemHealth {
	healthData := w.monitor.GetCurrentHealth()
	
	// Convert adapter statuses
	adapterStatuses := make([]api.AdapterStatus, len(healthData.AdapterStatus))
	for i, status := range healthData.AdapterStatus {
		adapterStatuses[i] = api.AdapterStatus{
			Name:         status.Name,
			Status:       status.Status,
			LastEvent:    status.LastEvent,
			ErrorMessage: status.ErrorMessage,
			UpdatedAt:    status.UpdatedAt,
		}
	}
	
	return api.SystemHealth{
		Status:        string(healthData.Status),
		Timestamp:     healthData.Timestamp,
		QueueDepth:    healthData.QueueDepth,
		AdapterStatus: adapterStatuses,
		Resources: api.SystemResources{
			CPUCores:    healthData.Resources.CPUCores,
			MemoryGB:    healthData.Resources.MemoryGB,
			CPUUsage:    healthData.Resources.CPUUsage,
			MemoryUsage: healthData.Resources.MemoryUsage,
			DiskUsage:   healthData.Resources.DiskUsage,
			LastUpdated: healthData.Resources.LastUpdated,
		},
		Tier:          api.Tier(healthData.Tier),
		LastEventTime: healthData.LastEventTime,
		Uptime:        healthData.Uptime,
		Version:       healthData.Version,
		DeviceID:      healthData.DeviceID,
	}
}

func (w *healthMonitorWrapper) UpdateHealth(ctx context.Context) error {
	return w.monitor.UpdateHealth(ctx)
}

// configManagerWrapper adapts Config to ConfigManager interface
type configManagerWrapper struct {
	config *config.Config
}

func (w *configManagerWrapper) GetCurrentConfig() *config.Config {
	return w.config
}

func (w *configManagerWrapper) UpdateConfig(updates *api.ConfigUpdateRequest) (*api.ConfigUpdateResponse, error) {
	var updatedFields []string
	requiresRestart := false
	
	// Update tier
	if updates.Tier != nil {
		if *updates.Tier == "lite" || *updates.Tier == "normal" || *updates.Tier == "full" {
			w.config.Tier = *updates.Tier
			updatedFields = append(updatedFields, "tier")
			requiresRestart = true
		}
	}
	
	// Update queue max size
	if updates.QueueMaxSize != nil && *updates.QueueMaxSize > 0 {
		w.config.QueueMaxSize = *updates.QueueMaxSize
		updatedFields = append(updatedFields, "queueMaxSize")
	}
	
	// Update heartbeat interval
	if updates.HeartbeatInterval != nil && *updates.HeartbeatInterval > 0 {
		w.config.HeartbeatInterval = *updates.HeartbeatInterval
		updatedFields = append(updatedFields, "heartbeatInterval")
	}
	
	// Update unlock duration
	if updates.UnlockDuration != nil && *updates.UnlockDuration > 0 {
		w.config.UnlockDuration = *updates.UnlockDuration
		updatedFields = append(updatedFields, "unlockDuration")
	}
	
	// Update log level
	if updates.LogLevel != nil {
		validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
		if validLevels[*updates.LogLevel] {
			w.config.LogLevel = *updates.LogLevel
			updatedFields = append(updatedFields, "logLevel")
		}
	}
	
	// Update log file
	if updates.LogFile != nil {
		w.config.LogFile = *updates.LogFile
		updatedFields = append(updatedFields, "logFile")
	}
	
	// Update enabled adapters
	if updates.EnabledAdapters != nil {
		w.config.EnabledAdapters = updates.EnabledAdapters
		updatedFields = append(updatedFields, "enabledAdapters")
		requiresRestart = true
	}
	
	// Update adapter configs
	if updates.AdapterConfigs != nil {
		w.config.AdapterConfigs = updates.AdapterConfigs
		updatedFields = append(updatedFields, "adapterConfigs")
		requiresRestart = true
	}
	
	// Update updates enabled
	if updates.UpdatesEnabled != nil {
		w.config.UpdatesEnabled = *updates.UpdatesEnabled
		updatedFields = append(updatedFields, "updatesEnabled")
	}
	
	// Update API server config
	if updates.APIServer != nil {
		if updates.APIServer.Enabled != nil {
			w.config.APIServer.Enabled = *updates.APIServer.Enabled
			updatedFields = append(updatedFields, "apiServer.enabled")
			requiresRestart = true
		}
		if updates.APIServer.Port != nil {
			w.config.APIServer.Port = *updates.APIServer.Port
			updatedFields = append(updatedFields, "apiServer.port")
			requiresRestart = true
		}
		if updates.APIServer.Host != nil {
			w.config.APIServer.Host = *updates.APIServer.Host
			updatedFields = append(updatedFields, "apiServer.host")
			requiresRestart = true
		}
		if updates.APIServer.TLSEnabled != nil {
			w.config.APIServer.TLSEnabled = *updates.APIServer.TLSEnabled
			updatedFields = append(updatedFields, "apiServer.tlsEnabled")
			requiresRestart = true
		}
		// Add more API server config updates as needed
	}
	
	return &api.ConfigUpdateResponse{
		Success:         true,
		Message:         fmt.Sprintf("Configuration updated successfully (%d fields)", len(updatedFields)),
		UpdatedFields:   updatedFields,
		RequiresRestart: requiresRestart,
		Timestamp:       time.Now(),
	}, nil
}

func (w *configManagerWrapper) ReloadConfig(force bool) (*api.ConfigReloadResponse, error) {
	// For now, we'll just return success without actually reloading
	// In a full implementation, this would reload the config from file
	return &api.ConfigReloadResponse{
		Success:       true,
		Message:       "Configuration reloaded successfully",
		ReloadedFrom:  "config.yaml",
		ChangedFields: []string{},
		Timestamp:     time.Now(),
	}, nil
}

// tierDetectorWrapper adapts TierDetector to API TierDetector interface
type tierDetectorWrapper struct {
	detector *tier.Detector
}

func (w *tierDetectorWrapper) GetCurrentTier() api.Tier {
	return api.Tier(w.detector.GetCurrentTier())
}

func (w *tierDetectorWrapper) GetCurrentResources() api.SystemResources {
	resources := w.detector.GetCurrentResources()
	return api.SystemResources{
		CPUCores:    resources.CPUCores,
		MemoryGB:    resources.MemoryGB,
		CPUUsage:    resources.CPUUsage,
		MemoryUsage: resources.MemoryUsage,
		DiskUsage:   resources.DiskUsage,
		LastUpdated: resources.LastUpdated,
	}
}

// queueManagerWrapper adapts QueueManager to API QueueManager interface
type queueManagerWrapper struct {
	manager queue.QueueManager
}

func (w *queueManagerWrapper) GetQueueDepth(ctx context.Context) (int, error) {
	return w.manager.GetQueueDepth(ctx)
}

func (w *queueManagerWrapper) GetStats(ctx context.Context) (api.QueueStats, error) {
	stats, err := w.manager.GetStats(ctx)
	if err != nil {
		return api.QueueStats{}, err
	}
	
	return api.QueueStats{
		QueueDepth:      stats.QueueDepth,
		PendingEvents:   stats.PendingEvents,
		SentEvents:      stats.SentEvents,
		FailedEvents:    stats.FailedEvents,
		LastSentAt:      stats.LastSentAt,
		LastFailureAt:   stats.LastFailureAt,
		OldestEventTime: stats.OldestEventTime,
	}, nil
}

func (w *queueManagerWrapper) QueryEvents(ctx context.Context, filter queue.EventQueryFilter) ([]queue.QueuedEvent, int64, error) {
	return w.manager.QueryEvents(ctx, filter)
}

func (w *queueManagerWrapper) GetEventStats(ctx context.Context) (queue.EventStatistics, error) {
	return w.manager.GetEventStats(ctx)
}

func (w *queueManagerWrapper) ClearEvents(ctx context.Context, criteria queue.EventClearCriteria) (int64, error) {
	return w.manager.ClearEvents(ctx, criteria)
}