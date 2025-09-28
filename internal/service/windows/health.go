package windows

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ServiceHealthMonitor monitors Windows service health and provides automatic recovery
type ServiceHealthMonitor struct {
	mu              sync.RWMutex
	logger          *logrus.Logger
	serviceManager  *ServiceManager
	isMonitoring    bool
	monitorInterval time.Duration
	ctx             context.Context
	cancel          context.CancelFunc
	
	// Health metrics
	lastHealthCheck time.Time
	healthHistory   []ServiceHealthInfo
	maxHistorySize  int
	
	// Recovery settings
	recoveryEnabled    bool
	maxRecoveryAttempts int
	recoveryAttempts   int
	lastRecoveryTime   time.Time
	recoveryResetTime  time.Duration
}

// ServiceHealthMonitorConfig configures the health monitor
type ServiceHealthMonitorConfig struct {
	MonitorInterval     time.Duration
	MaxHistorySize      int
	RecoveryEnabled     bool
	MaxRecoveryAttempts int
	RecoveryResetTime   time.Duration
}

// DefaultServiceHealthMonitorConfig returns default configuration
func DefaultServiceHealthMonitorConfig() ServiceHealthMonitorConfig {
	return ServiceHealthMonitorConfig{
		MonitorInterval:     30 * time.Second,
		MaxHistorySize:      100,
		RecoveryEnabled:     true,
		MaxRecoveryAttempts: 3,
		RecoveryResetTime:   1 * time.Hour,
	}
}

// NewServiceHealthMonitor creates a new service health monitor
func NewServiceHealthMonitor(logger *logrus.Logger, config ServiceHealthMonitorConfig) (*ServiceHealthMonitor, error) {
	if runtime.GOOS != "windows" {
		return nil, fmt.Errorf("service health monitor is only supported on Windows")
	}

	serviceManager, err := NewServiceManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create service manager: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &ServiceHealthMonitor{
		logger:              logger,
		serviceManager:      serviceManager,
		monitorInterval:     config.MonitorInterval,
		maxHistorySize:      config.MaxHistorySize,
		recoveryEnabled:     config.RecoveryEnabled,
		maxRecoveryAttempts: config.MaxRecoveryAttempts,
		recoveryResetTime:   config.RecoveryResetTime,
		ctx:                 ctx,
		cancel:              cancel,
		healthHistory:       make([]ServiceHealthInfo, 0, config.MaxHistorySize),
	}, nil
}

// Start begins health monitoring
func (shm *ServiceHealthMonitor) Start() error {
	shm.mu.Lock()
	defer shm.mu.Unlock()

	if shm.isMonitoring {
		return fmt.Errorf("health monitor is already running")
	}

	shm.logger.Info("Starting service health monitor")
	shm.isMonitoring = true

	go shm.monitorLoop()

	return nil
}

// Stop stops health monitoring
func (shm *ServiceHealthMonitor) Stop() error {
	shm.mu.Lock()
	defer shm.mu.Unlock()

	if !shm.isMonitoring {
		return nil
	}

	shm.logger.Info("Stopping service health monitor")
	shm.cancel()
	shm.isMonitoring = false

	if shm.serviceManager != nil {
		shm.serviceManager.Close()
	}

	return nil
}

// monitorLoop is the main monitoring loop
func (shm *ServiceHealthMonitor) monitorLoop() {
	ticker := time.NewTicker(shm.monitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-shm.ctx.Done():
			return
		case <-ticker.C:
			shm.performHealthCheck()
		}
	}
}

// performHealthCheck performs a single health check
func (shm *ServiceHealthMonitor) performHealthCheck() {
	health, err := shm.serviceManager.GetServiceHealth()
	if err != nil {
		shm.logger.WithError(err).Error("Failed to get service health")
		return
	}

	shm.mu.Lock()
	defer shm.mu.Unlock()

	// Update health history
	shm.lastHealthCheck = time.Now()
	shm.addToHistory(*health)

	// Log health status
	shm.logger.WithFields(logrus.Fields{
		"status":           health.Status,
		"process_id":       health.ProcessID,
		"win32_exit_code":  health.Win32ExitCode,
		"service_exit_code": health.ServiceExitCode,
	}).Debug("Service health check completed")

	// Check if recovery is needed
	if shm.recoveryEnabled && shm.needsRecovery(*health) {
		shm.attemptRecovery(*health)
	}
}

// addToHistory adds a health record to the history
func (shm *ServiceHealthMonitor) addToHistory(health ServiceHealthInfo) {
	shm.healthHistory = append(shm.healthHistory, health)
	
	// Trim history if it exceeds max size
	if len(shm.healthHistory) > shm.maxHistorySize {
		shm.healthHistory = shm.healthHistory[1:]
	}
}

// needsRecovery determines if the service needs recovery
func (shm *ServiceHealthMonitor) needsRecovery(health ServiceHealthInfo) bool {
	// Service is stopped with an error
	if health.Status == "Stopped" && health.Win32ExitCode != 0 {
		return true
	}

	// Service is in a pending state for too long
	if (health.Status == "Starting" || health.Status == "Stopping") && 
	   len(shm.healthHistory) > 0 {
		// Check if it's been in pending state for more than 5 minutes
		for i := len(shm.healthHistory) - 1; i >= 0 && i >= len(shm.healthHistory)-10; i-- {
			if shm.healthHistory[i].Status != health.Status {
				break
			}
			if time.Since(shm.healthHistory[i].Timestamp) > 5*time.Minute {
				return true
			}
		}
	}

	return false
}

// attemptRecovery attempts to recover the service
func (shm *ServiceHealthMonitor) attemptRecovery(health ServiceHealthInfo) {
	// Check if we've exceeded max recovery attempts
	if time.Since(shm.lastRecoveryTime) > shm.recoveryResetTime {
		shm.recoveryAttempts = 0
	}

	if shm.recoveryAttempts >= shm.maxRecoveryAttempts {
		shm.logger.WithField("attempts", shm.recoveryAttempts).
			Error("Maximum recovery attempts reached, manual intervention required")
		return
	}

	shm.recoveryAttempts++
	shm.lastRecoveryTime = time.Now()

	shm.logger.WithFields(logrus.Fields{
		"attempt":          shm.recoveryAttempts,
		"max_attempts":     shm.maxRecoveryAttempts,
		"service_status":   health.Status,
		"exit_code":        health.Win32ExitCode,
	}).Warn("Attempting service recovery")

	var err error
	switch health.Status {
	case "Stopped":
		err = shm.serviceManager.StartService()
	case "Starting", "Stopping":
		// Force stop and restart
		shm.serviceManager.StopService()
		time.Sleep(5 * time.Second)
		err = shm.serviceManager.StartService()
	default:
		shm.logger.WithField("status", health.Status).
			Warn("Unknown service status, attempting restart")
		err = shm.serviceManager.RestartService()
	}

	if err != nil {
		shm.logger.WithError(err).WithField("attempt", shm.recoveryAttempts).
			Error("Service recovery attempt failed")
	} else {
		shm.logger.WithField("attempt", shm.recoveryAttempts).
			Info("Service recovery attempt completed")
	}
}

// GetCurrentHealth returns the current service health
func (shm *ServiceHealthMonitor) GetCurrentHealth() (*ServiceHealthInfo, error) {
	return shm.serviceManager.GetServiceHealth()
}

// GetHealthHistory returns the health history
func (shm *ServiceHealthMonitor) GetHealthHistory() []ServiceHealthInfo {
	shm.mu.RLock()
	defer shm.mu.RUnlock()

	// Return a copy to prevent external modification
	history := make([]ServiceHealthInfo, len(shm.healthHistory))
	copy(history, shm.healthHistory)
	return history
}

// GetHealthSummary returns a summary of service health
func (shm *ServiceHealthMonitor) GetHealthSummary() map[string]interface{} {
	shm.mu.RLock()
	defer shm.mu.RUnlock()

	summary := map[string]interface{}{
		"is_monitoring":        shm.isMonitoring,
		"last_health_check":    shm.lastHealthCheck,
		"monitor_interval":     shm.monitorInterval,
		"recovery_enabled":     shm.recoveryEnabled,
		"recovery_attempts":    shm.recoveryAttempts,
		"max_recovery_attempts": shm.maxRecoveryAttempts,
		"last_recovery_time":   shm.lastRecoveryTime,
		"history_size":         len(shm.healthHistory),
	}

	// Add current health if available
	if len(shm.healthHistory) > 0 {
		latest := shm.healthHistory[len(shm.healthHistory)-1]
		summary["current_status"] = latest.Status
		summary["current_process_id"] = latest.ProcessID
		summary["current_exit_code"] = latest.Win32ExitCode
	}

	return summary
}

// IsServiceRunning checks if the service is currently running
func (shm *ServiceHealthMonitor) IsServiceRunning() (bool, error) {
	health, err := shm.GetCurrentHealth()
	if err != nil {
		return false, err
	}
	return health.Status == "Running", nil
}

// GetServiceUptime returns how long the service has been running
func (shm *ServiceHealthMonitor) GetServiceUptime() (time.Duration, error) {
	shm.mu.RLock()
	defer shm.mu.RUnlock()

	if len(shm.healthHistory) == 0 {
		return 0, fmt.Errorf("no health history available")
	}

	// Find the last time the service started
	var startTime time.Time
	for i := len(shm.healthHistory) - 1; i >= 0; i-- {
		if shm.healthHistory[i].Status == "Running" {
			if i == 0 || shm.healthHistory[i-1].Status != "Running" {
				startTime = shm.healthHistory[i].Timestamp
				break
			}
		}
	}

	if startTime.IsZero() {
		return 0, fmt.Errorf("service start time not found in history")
	}

	return time.Since(startTime), nil
}

// Close closes the health monitor and releases resources
func (shm *ServiceHealthMonitor) Close() error {
	return shm.Stop()
}