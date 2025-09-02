package monitoring

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"gym-door-bridge/internal/queue"
	"gym-door-bridge/internal/tier"
)

// AlertType represents different types of alerts
type AlertType string

const (
	AlertTypeDeviceOffline     AlertType = "device_offline"
	AlertTypeQueueThreshold    AlertType = "queue_threshold"
	AlertTypeSecurityEvent     AlertType = "security_event"
	AlertTypePerformanceDegradation AlertType = "performance_degradation"
	AlertTypeCriticalError     AlertType = "critical_error"
)

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	AlertSeverityLow      AlertSeverity = "low"
	AlertSeverityMedium   AlertSeverity = "medium"
	AlertSeverityHigh     AlertSeverity = "high"
	AlertSeverityCritical AlertSeverity = "critical"
)

// Alert represents a monitoring alert
type Alert struct {
	ID          string                 `json:"id"`
	Type        AlertType              `json:"type"`
	Severity    AlertSeverity          `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	DeviceID    string                 `json:"deviceId,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Resolved    bool                   `json:"resolved"`
	ResolvedAt  *time.Time             `json:"resolvedAt,omitempty"`
}

// SecurityEvent represents a security-related event
type SecurityEvent struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    AlertSeverity          `json:"severity"`
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	DeviceID    string                 `json:"deviceId,omitempty"`
	SourceIP    string                 `json:"sourceIp,omitempty"`
	UserAgent   string                 `json:"userAgent,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// PerformanceMetrics represents system performance metrics
type PerformanceMetrics struct {
	Timestamp       time.Time              `json:"timestamp"`
	DeviceID        string                 `json:"deviceId"`
	QueueDepth      int                    `json:"queueDepth"`
	QueueCapacity   int                    `json:"queueCapacity"`
	CPUUsage        float64                `json:"cpuUsage"`
	MemoryUsage     float64                `json:"memoryUsage"`
	DiskUsage       float64                `json:"diskUsage"`
	NetworkLatency  time.Duration          `json:"networkLatency"`
	AdapterStatuses map[string]string      `json:"adapterStatuses"`
	ErrorRate       float64                `json:"errorRate"`
	Tier            tier.Tier              `json:"tier"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// MonitoringConfig holds configuration for the monitoring system
type MonitoringConfig struct {
	// Alert thresholds
	QueueThresholdPercent     float64       `json:"queueThresholdPercent"`     // Alert when queue is X% full
	OfflineThreshold          time.Duration `json:"offlineThreshold"`          // Time before device considered offline
	PerformanceDegradedThreshold float64    `json:"performanceDegradedThreshold"` // CPU/Memory threshold for degraded performance
	
	// Collection intervals
	MetricsCollectionInterval time.Duration `json:"metricsCollectionInterval"` // How often to collect metrics
	AlertCheckInterval        time.Duration `json:"alertCheckInterval"`        // How often to check for alert conditions
	
	// Retention
	MetricsRetentionDays      int           `json:"metricsRetentionDays"`      // How long to keep metrics
	AlertRetentionDays        int           `json:"alertRetentionDays"`        // How long to keep alerts
	
	// Reporting
	EnableCloudReporting      bool          `json:"enableCloudReporting"`      // Send metrics to admin portal
	ReportingInterval         time.Duration `json:"reportingInterval"`         // How often to send metrics to cloud
}

// DefaultMonitoringConfig returns default monitoring configuration
func DefaultMonitoringConfig() MonitoringConfig {
	return MonitoringConfig{
		QueueThresholdPercent:        75.0,
		OfflineThreshold:             5 * time.Minute,
		PerformanceDegradedThreshold: 80.0,
		MetricsCollectionInterval:    30 * time.Second,
		AlertCheckInterval:           1 * time.Minute,
		MetricsRetentionDays:         7,
		AlertRetentionDays:           30,
		EnableCloudReporting:         true,
		ReportingInterval:            5 * time.Minute,
	}
}

// AlertHandler defines the interface for handling alerts
type AlertHandler interface {
	HandleAlert(ctx context.Context, alert Alert) error
}

// MetricsReporter defines the interface for reporting metrics to the cloud
type MetricsReporter interface {
	ReportMetrics(ctx context.Context, metrics PerformanceMetrics) error
	ReportAlert(ctx context.Context, alert Alert) error
	ReportSecurityEvent(ctx context.Context, event SecurityEvent) error
}

// HealthMonitor interface to avoid import cycles
type HealthMonitor interface {
	GetCurrentHealth() SystemHealth
	UpdateHealth(ctx context.Context) error
}

// TierDetector interface to avoid import cycles
type TierDetector interface {
	GetCurrentResources() tier.SystemResources
	GetCurrentTier() tier.Tier
}

// SystemHealth represents the complete health information of the system
type SystemHealth struct {
	Status        string                 `json:"status"`
	Timestamp     time.Time              `json:"timestamp"`
	QueueDepth    int                    `json:"queueDepth"`
	AdapterStatus []AdapterStatus        `json:"adapterStatus"`
	Resources     tier.SystemResources   `json:"resources"`
	Tier          tier.Tier              `json:"tier"`
	LastEventTime *time.Time             `json:"lastEventTime,omitempty"`
	Uptime        time.Duration          `json:"uptime"`
	Version       string                 `json:"version"`
	DeviceID      string                 `json:"deviceId,omitempty"`
}

// AdapterStatus represents the status of a hardware adapter
type AdapterStatus struct {
	Name         string    `json:"name"`
	Status       string    `json:"status"`
	LastEvent    time.Time `json:"lastEvent"`
	ErrorMessage string    `json:"errorMessage,omitempty"`
}

// MonitoringSystem manages monitoring, alerting, and metrics collection
type MonitoringSystem struct {
	mu                sync.RWMutex
	config            MonitoringConfig
	logger            *logrus.Logger
	deviceID          string
	
	// Dependencies
	healthMonitor     HealthMonitor
	queueManager      queue.QueueManager
	tierDetector      TierDetector
	
	// Handlers and reporters
	alertHandlers     []AlertHandler
	metricsReporter   MetricsReporter
	
	// State
	isRunning         bool
	stopChan          chan struct{}
	wg                sync.WaitGroup
	
	// Storage for metrics and alerts
	recentMetrics     []PerformanceMetrics
	activeAlerts      map[string]Alert
	securityEvents    []SecurityEvent
	
	// Alert state tracking
	lastHealthCheck   time.Time
	lastQueueCheck    time.Time
	consecutiveErrors int
}

// MonitoringSystemOption is a functional option for configuring the MonitoringSystem
type MonitoringSystemOption func(*MonitoringSystem)

// WithLogger sets the logger for the monitoring system
func WithLogger(logger *logrus.Logger) MonitoringSystemOption {
	return func(m *MonitoringSystem) {
		m.logger = logger
	}
}

// WithDeviceID sets the device ID for the monitoring system
func WithDeviceID(deviceID string) MonitoringSystemOption {
	return func(m *MonitoringSystem) {
		m.deviceID = deviceID
	}
}

// WithMetricsReporter sets the metrics reporter
func WithMetricsReporter(reporter MetricsReporter) MonitoringSystemOption {
	return func(m *MonitoringSystem) {
		m.metricsReporter = reporter
	}
}

// NewMonitoringSystem creates a new monitoring system
func NewMonitoringSystem(
	config MonitoringConfig,
	healthMonitor HealthMonitor,
	queueManager queue.QueueManager,
	tierDetector TierDetector,
	opts ...MonitoringSystemOption,
) *MonitoringSystem {
	m := &MonitoringSystem{
		config:        config,
		logger:        logrus.New(),
		healthMonitor: healthMonitor,
		queueManager:  queueManager,
		tierDetector:  tierDetector,
		stopChan:      make(chan struct{}),
		activeAlerts:  make(map[string]Alert),
	}
	
	// Apply options
	for _, opt := range opts {
		opt(m)
	}
	
	return m
}

// AddAlertHandler adds an alert handler to the monitoring system
func (m *MonitoringSystem) AddAlertHandler(handler AlertHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alertHandlers = append(m.alertHandlers, handler)
}

// Start begins the monitoring system
func (m *MonitoringSystem) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.isRunning {
		return fmt.Errorf("monitoring system is already running")
	}
	
	m.logger.Info("Starting monitoring system")
	m.isRunning = true
	
	// Start metrics collection goroutine
	m.wg.Add(1)
	go m.metricsCollectionLoop(ctx)
	
	// Start alert checking goroutine
	m.wg.Add(1)
	go m.alertCheckLoop(ctx)
	
	// Start metrics reporting goroutine if enabled
	if m.config.EnableCloudReporting && m.metricsReporter != nil {
		m.wg.Add(1)
		go m.metricsReportingLoop(ctx)
	}
	
	m.logger.Info("Monitoring system started")
	return nil
}

// Stop gracefully shuts down the monitoring system
func (m *MonitoringSystem) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.isRunning {
		return nil
	}
	
	m.logger.Info("Stopping monitoring system")
	m.isRunning = false
	
	// Signal all goroutines to stop
	close(m.stopChan)
	
	// Wait for all goroutines to finish
	m.wg.Wait()
	
	m.logger.Info("Monitoring system stopped")
	return nil
}

// LogSecurityEvent logs a security event and generates an alert if necessary
func (m *MonitoringSystem) LogSecurityEvent(ctx context.Context, eventType, description string, metadata map[string]interface{}) error {
	event := SecurityEvent{
		ID:          generateID(),
		Type:        eventType,
		Severity:    m.determineSecurityEventSeverity(eventType),
		Description: description,
		Timestamp:   time.Now(),
		DeviceID:    m.deviceID,
		Metadata:    metadata,
	}
	
	// Store security event
	m.mu.Lock()
	m.securityEvents = append(m.securityEvents, event)
	m.mu.Unlock()
	
	// Log the security event
	m.logger.WithFields(logrus.Fields{
		"security_event_id":   event.ID,
		"security_event_type": event.Type,
		"severity":            event.Severity,
		"device_id":           event.DeviceID,
		"metadata":            event.Metadata,
	}).Warn("Security event detected", "description", event.Description)
	
	// Report to cloud if enabled
	if m.config.EnableCloudReporting && m.metricsReporter != nil {
		if err := m.metricsReporter.ReportSecurityEvent(ctx, event); err != nil {
			m.logger.WithError(err).Error("Failed to report security event to cloud")
		}
	}
	
	// Generate alert for high/critical severity events
	if event.Severity == AlertSeverityHigh || event.Severity == AlertSeverityCritical {
		alert := Alert{
			ID:          generateID(),
			Type:        AlertTypeSecurityEvent,
			Severity:    event.Severity,
			Title:       fmt.Sprintf("Security Event: %s", event.Type),
			Description: event.Description,
			Timestamp:   event.Timestamp,
			DeviceID:    event.DeviceID,
			Metadata:    event.Metadata,
		}
		
		return m.generateAlert(ctx, alert)
	}
	
	return nil
}

// GetCurrentMetrics returns the most recent performance metrics
func (m *MonitoringSystem) GetCurrentMetrics() *PerformanceMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if len(m.recentMetrics) == 0 {
		return nil
	}
	
	// Return a copy of the most recent metrics
	latest := m.recentMetrics[len(m.recentMetrics)-1]
	return &latest
}

// GetActiveAlerts returns all currently active alerts
func (m *MonitoringSystem) GetActiveAlerts() []Alert {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	alerts := make([]Alert, 0, len(m.activeAlerts))
	for _, alert := range m.activeAlerts {
		alerts = append(alerts, alert)
	}
	
	return alerts
}

// GetRecentSecurityEvents returns recent security events
func (m *MonitoringSystem) GetRecentSecurityEvents(limit int) []SecurityEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if limit <= 0 || limit > len(m.securityEvents) {
		limit = len(m.securityEvents)
	}
	
	// Return the most recent events
	start := len(m.securityEvents) - limit
	if start < 0 {
		start = 0
	}
	
	events := make([]SecurityEvent, limit)
	copy(events, m.securityEvents[start:])
	
	return events
}

// metricsCollectionLoop periodically collects performance metrics
func (m *MonitoringSystem) metricsCollectionLoop(ctx context.Context) {
	defer m.wg.Done()
	
	ticker := time.NewTicker(m.config.MetricsCollectionInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			if err := m.collectMetrics(ctx); err != nil {
				m.logger.WithError(err).Error("Failed to collect metrics")
			}
		}
	}
}

// alertCheckLoop periodically checks for alert conditions
func (m *MonitoringSystem) alertCheckLoop(ctx context.Context) {
	defer m.wg.Done()
	
	ticker := time.NewTicker(m.config.AlertCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			if err := m.checkAlertConditions(ctx); err != nil {
				m.logger.WithError(err).Error("Failed to check alert conditions")
			}
		}
	}
}

// metricsReportingLoop periodically reports metrics to the cloud
func (m *MonitoringSystem) metricsReportingLoop(ctx context.Context) {
	defer m.wg.Done()
	
	ticker := time.NewTicker(m.config.ReportingInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			if err := m.reportMetricsToCloud(ctx); err != nil {
				m.logger.WithError(err).Error("Failed to report metrics to cloud")
			}
		}
	}
}

// collectMetrics collects current system performance metrics
func (m *MonitoringSystem) collectMetrics(ctx context.Context) error {
	now := time.Now()
	
	// Get current system health
	systemHealth := m.healthMonitor.GetCurrentHealth()
	
	// Get queue statistics
	queueStats, err := m.queueManager.GetStats(ctx)
	if err != nil {
		m.logger.WithError(err).Error("Failed to get queue statistics")
		return err
	}
	
	// Calculate queue capacity based on current tier
	tierConfig := tier.GetTierConfig(systemHealth.Tier)
	queueCapacity := tierConfig.QueueMaxSize
	
	// Build adapter status map
	adapterStatuses := make(map[string]string)
	for _, status := range systemHealth.AdapterStatus {
		adapterStatuses[status.Name] = status.Status
	}
	
	// Calculate error rate (simplified - could be enhanced with more sophisticated calculation)
	errorRate := m.calculateErrorRate()
	
	// Create performance metrics
	metrics := PerformanceMetrics{
		Timestamp:       now,
		DeviceID:        m.deviceID,
		QueueDepth:      queueStats.QueueDepth,
		QueueCapacity:   queueCapacity,
		CPUUsage:        systemHealth.Resources.CPUUsage,
		MemoryUsage:     systemHealth.Resources.MemoryUsage,
		DiskUsage:       systemHealth.Resources.DiskUsage,
		AdapterStatuses: adapterStatuses,
		ErrorRate:       errorRate,
		Tier:            systemHealth.Tier,
	}
	
	// Store metrics
	m.mu.Lock()
	m.recentMetrics = append(m.recentMetrics, metrics)
	
	// Keep only recent metrics (last hour worth)
	maxMetrics := int(time.Hour / m.config.MetricsCollectionInterval)
	if len(m.recentMetrics) > maxMetrics {
		m.recentMetrics = m.recentMetrics[len(m.recentMetrics)-maxMetrics:]
	}
	m.mu.Unlock()
	
	return nil
}

// checkAlertConditions checks for conditions that should trigger alerts
func (m *MonitoringSystem) checkAlertConditions(ctx context.Context) error {
	now := time.Now()
	
	// Check device offline condition
	if err := m.checkDeviceOfflineCondition(ctx, now); err != nil {
		m.logger.WithError(err).Error("Failed to check device offline condition")
	}
	
	// Check queue threshold condition
	if err := m.checkQueueThresholdCondition(ctx, now); err != nil {
		m.logger.WithError(err).Error("Failed to check queue threshold condition")
	}
	
	// Check performance degradation condition
	if err := m.checkPerformanceDegradationCondition(ctx, now); err != nil {
		m.logger.WithError(err).Error("Failed to check performance degradation condition")
	}
	
	// Clean up resolved alerts
	m.cleanupResolvedAlerts(ctx, now)
	
	return nil
}

// checkDeviceOfflineCondition checks if the device should be considered offline
func (m *MonitoringSystem) checkDeviceOfflineCondition(ctx context.Context, now time.Time) error {
	// Check if we haven't had a successful health check recently
	timeSinceLastCheck := now.Sub(m.lastHealthCheck)
	if timeSinceLastCheck > m.config.OfflineThreshold {
		alertID := fmt.Sprintf("device_offline_%s", m.deviceID)
		
		// Check if we already have this alert active
		if _, exists := m.activeAlerts[alertID]; !exists {
			alert := Alert{
				ID:          alertID,
				Type:        AlertTypeDeviceOffline,
				Severity:    AlertSeverityHigh,
				Title:       "Device Offline",
				Description: fmt.Sprintf("Device has not reported health status for %v", timeSinceLastCheck),
				Timestamp:   now,
				DeviceID:    m.deviceID,
				Metadata: map[string]interface{}{
					"offline_duration": timeSinceLastCheck.String(),
					"threshold":        m.config.OfflineThreshold.String(),
				},
			}
			
			return m.generateAlert(ctx, alert)
		}
	} else {
		// Device is online, resolve offline alert if it exists
		alertID := fmt.Sprintf("device_offline_%s", m.deviceID)
		if alert, exists := m.activeAlerts[alertID]; exists {
			m.resolveAlert(ctx, alert.ID, now)
		}
		m.lastHealthCheck = now
	}
	
	return nil
}

// checkQueueThresholdCondition checks if queue depth exceeds threshold
func (m *MonitoringSystem) checkQueueThresholdCondition(ctx context.Context, now time.Time) error {
	queueStats, err := m.queueManager.GetStats(ctx)
	if err != nil {
		return err
	}
	
	// Get current tier configuration
	systemHealth := m.healthMonitor.GetCurrentHealth()
	tierConfig := tier.GetTierConfig(systemHealth.Tier)
	
	// Calculate queue usage percentage
	queueUsagePercent := float64(queueStats.QueueDepth) / float64(tierConfig.QueueMaxSize) * 100
	
	if queueUsagePercent >= m.config.QueueThresholdPercent {
		alertID := fmt.Sprintf("queue_threshold_%s", m.deviceID)
		
		// Check if we already have this alert active
		if existingAlert, exists := m.activeAlerts[alertID]; !exists || 
		   existingAlert.Metadata["queue_usage_percent"].(float64) < queueUsagePercent {
			
			severity := AlertSeverityMedium
			if queueUsagePercent >= 90 {
				severity = AlertSeverityHigh
			}
			if queueUsagePercent >= 95 {
				severity = AlertSeverityCritical
			}
			
			alert := Alert{
				ID:          alertID,
				Type:        AlertTypeQueueThreshold,
				Severity:    severity,
				Title:       "Queue Threshold Exceeded",
				Description: fmt.Sprintf("Queue is %.1f%% full (%d/%d events)", queueUsagePercent, queueStats.QueueDepth, tierConfig.QueueMaxSize),
				Timestamp:   now,
				DeviceID:    m.deviceID,
				Metadata: map[string]interface{}{
					"queue_depth":          queueStats.QueueDepth,
					"queue_capacity":       tierConfig.QueueMaxSize,
					"queue_usage_percent":  queueUsagePercent,
					"threshold_percent":    m.config.QueueThresholdPercent,
				},
			}
			
			return m.generateAlert(ctx, alert)
		}
	} else {
		// Queue usage is below threshold, resolve alert if it exists
		alertID := fmt.Sprintf("queue_threshold_%s", m.deviceID)
		if _, exists := m.activeAlerts[alertID]; exists {
			m.resolveAlert(ctx, alertID, now)
		}
	}
	
	return nil
}

// checkPerformanceDegradationCondition checks for performance degradation
func (m *MonitoringSystem) checkPerformanceDegradationCondition(ctx context.Context, now time.Time) error {
	systemHealth := m.healthMonitor.GetCurrentHealth()
	
	// Check CPU, memory, and disk usage
	degradationReasons := []string{}
	
	if systemHealth.Resources.CPUUsage >= m.config.PerformanceDegradedThreshold {
		degradationReasons = append(degradationReasons, fmt.Sprintf("CPU usage: %.1f%%", systemHealth.Resources.CPUUsage))
	}
	
	if systemHealth.Resources.MemoryUsage >= m.config.PerformanceDegradedThreshold {
		degradationReasons = append(degradationReasons, fmt.Sprintf("Memory usage: %.1f%%", systemHealth.Resources.MemoryUsage))
	}
	
	if systemHealth.Resources.DiskUsage >= m.config.PerformanceDegradedThreshold {
		degradationReasons = append(degradationReasons, fmt.Sprintf("Disk usage: %.1f%%", systemHealth.Resources.DiskUsage))
	}
	
	if len(degradationReasons) > 0 {
		alertID := fmt.Sprintf("performance_degradation_%s", m.deviceID)
		
		// Determine severity based on resource usage
		maxUsage := systemHealth.Resources.CPUUsage
		if systemHealth.Resources.MemoryUsage > maxUsage {
			maxUsage = systemHealth.Resources.MemoryUsage
		}
		if systemHealth.Resources.DiskUsage > maxUsage {
			maxUsage = systemHealth.Resources.DiskUsage
		}
		
		severity := AlertSeverityMedium
		if maxUsage >= 90 {
			severity = AlertSeverityHigh
		}
		if maxUsage >= 95 {
			severity = AlertSeverityCritical
		}
		
		alert := Alert{
			ID:          alertID,
			Type:        AlertTypePerformanceDegradation,
			Severity:    severity,
			Title:       "Performance Degradation Detected",
			Description: fmt.Sprintf("System performance degraded: %s", degradationReasons),
			Timestamp:   now,
			DeviceID:    m.deviceID,
			Metadata: map[string]interface{}{
				"cpu_usage":    systemHealth.Resources.CPUUsage,
				"memory_usage": systemHealth.Resources.MemoryUsage,
				"disk_usage":   systemHealth.Resources.DiskUsage,
				"threshold":    m.config.PerformanceDegradedThreshold,
				"reasons":      degradationReasons,
			},
		}
		
		return m.generateAlert(ctx, alert)
	} else {
		// Performance is normal, resolve alert if it exists
		alertID := fmt.Sprintf("performance_degradation_%s", m.deviceID)
		if _, exists := m.activeAlerts[alertID]; exists {
			m.resolveAlert(ctx, alertID, now)
		}
	}
	
	return nil
}

// generateAlert creates and handles a new alert
func (m *MonitoringSystem) generateAlert(ctx context.Context, alert Alert) error {
	m.mu.Lock()
	m.activeAlerts[alert.ID] = alert
	m.mu.Unlock()
	
	// Log the alert
	m.logger.WithFields(logrus.Fields{
		"alert_id":    alert.ID,
		"alert_type":  alert.Type,
		"severity":    alert.Severity,
		"device_id":   alert.DeviceID,
		"metadata":    alert.Metadata,
	}).Warn("Alert generated", "title", alert.Title, "description", alert.Description)
	
	// Send to alert handlers
	for _, handler := range m.alertHandlers {
		if err := handler.HandleAlert(ctx, alert); err != nil {
			m.logger.WithError(err).Error("Alert handler failed")
		}
	}
	
	// Report to cloud if enabled
	if m.config.EnableCloudReporting && m.metricsReporter != nil {
		if err := m.metricsReporter.ReportAlert(ctx, alert); err != nil {
			m.logger.WithError(err).Error("Failed to report alert to cloud")
		}
	}
	
	return nil
}

// resolveAlert marks an alert as resolved
func (m *MonitoringSystem) resolveAlert(ctx context.Context, alertID string, resolvedAt time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if alert, exists := m.activeAlerts[alertID]; exists {
		alert.Resolved = true
		alert.ResolvedAt = &resolvedAt
		m.activeAlerts[alertID] = alert
		
		m.logger.WithFields(logrus.Fields{
			"alert_id":   alert.ID,
			"alert_type": alert.Type,
			"device_id":  alert.DeviceID,
		}).Info("Alert resolved", "title", alert.Title)
		
		// Report resolution to cloud if enabled
		if m.config.EnableCloudReporting && m.metricsReporter != nil {
			if err := m.metricsReporter.ReportAlert(ctx, alert); err != nil {
				m.logger.WithError(err).Error("Failed to report alert resolution to cloud")
			}
		}
	}
}

// cleanupResolvedAlerts removes old resolved alerts
func (m *MonitoringSystem) cleanupResolvedAlerts(ctx context.Context, now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for alertID, alert := range m.activeAlerts {
		if alert.Resolved && alert.ResolvedAt != nil {
			// Remove alerts that have been resolved for more than 1 hour
			if now.Sub(*alert.ResolvedAt) > time.Hour {
				delete(m.activeAlerts, alertID)
			}
		}
	}
}

// reportMetricsToCloud sends current metrics to the cloud
func (m *MonitoringSystem) reportMetricsToCloud(ctx context.Context) error {
	if m.metricsReporter == nil {
		return nil
	}
	
	currentMetrics := m.GetCurrentMetrics()
	if currentMetrics == nil {
		return nil
	}
	
	return m.metricsReporter.ReportMetrics(ctx, *currentMetrics)
}

// calculateErrorRate calculates the current error rate
func (m *MonitoringSystem) calculateErrorRate() float64 {
	// This is a simplified implementation
	// In a real system, you might track errors over time windows
	if m.consecutiveErrors > 0 {
		return float64(m.consecutiveErrors) / 10.0 // Normalize to percentage
	}
	return 0.0
}

// determineSecurityEventSeverity determines the severity of a security event
func (m *MonitoringSystem) determineSecurityEventSeverity(eventType string) AlertSeverity {
	switch eventType {
	case "hmac_validation_failure", "invalid_signature", "authentication_failure":
		return AlertSeverityHigh
	case "suspicious_activity", "rate_limit_exceeded":
		return AlertSeverityMedium
	case "invalid_request", "malformed_data":
		return AlertSeverityLow
	default:
		return AlertSeverityMedium
	}
}

// generateID generates a unique ID for alerts and events
func generateID() string {
	return fmt.Sprintf("%d_%d", time.Now().UnixNano(), time.Now().Nanosecond())
}