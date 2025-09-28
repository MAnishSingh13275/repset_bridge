package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	"gym-door-bridge/internal/config"
	"gym-door-bridge/internal/service/windows"

	"github.com/sirupsen/logrus"
)

// InstallationTelemetry handles telemetry data related to installation
type InstallationTelemetry struct {
	logger *logrus.Logger
	config *config.Config
}

// InstallationStatusReport contains installation-related status information
type InstallationStatusReport struct {
	// Installation metadata
	InstallationMethod   string    `json:"installation_method"`
	InstallationVersion  string    `json:"installation_version"`
	InstalledAt          string    `json:"installed_at"`
	InstalledBy          string    `json:"installed_by"`
	PairCode             string    `json:"pair_code,omitempty"`
	Source               string    `json:"source"`
	Checksum             string    `json:"checksum,omitempty"`
	InstallationAge      string    `json:"installation_age"`
	
	// Service information (Windows only)
	ServiceHealth        *ServiceHealthReport `json:"service_health,omitempty"`
	
	// System information
	Platform             string    `json:"platform"`
	Architecture         string    `json:"architecture"`
	
	// Timestamp
	ReportTimestamp      time.Time `json:"report_timestamp"`
}

// ServiceHealthReport contains Windows service health information
type ServiceHealthReport struct {
	ServiceName         string    `json:"service_name"`
	Status              string    `json:"status"`
	ProcessID           uint32    `json:"process_id"`
	Win32ExitCode       uint32    `json:"win32_exit_code"`
	ServiceExitCode     uint32    `json:"service_exit_code"`
	StartType           string    `json:"start_type"`
	ServiceType         string    `json:"service_type"`
	IsMonitored         bool      `json:"is_monitored"`
	RecoveryAttempts    int       `json:"recovery_attempts"`
	LastRecoveryTime    time.Time `json:"last_recovery_time,omitempty"`
	Uptime              string    `json:"uptime,omitempty"`
}

// NewInstallationTelemetry creates a new installation telemetry instance
func NewInstallationTelemetry(logger *logrus.Logger, config *config.Config) *InstallationTelemetry {
	return &InstallationTelemetry{
		logger: logger,
		config: config,
	}
}

// GenerateInstallationStatusReport generates a comprehensive installation status report
func (it *InstallationTelemetry) GenerateInstallationStatusReport(ctx context.Context) (*InstallationStatusReport, error) {
	report := &InstallationStatusReport{
		InstallationMethod:  it.config.Installation.Method,
		InstallationVersion: it.config.Installation.Version,
		InstalledAt:         it.config.Installation.InstalledAt,
		InstalledBy:         it.config.Installation.InstalledBy,
		PairCode:            it.config.Installation.PairCode,
		Source:              it.config.Installation.Source,
		Checksum:            it.config.Installation.Checksum,
		Platform:            runtime.GOOS,
		Architecture:        runtime.GOARCH,
		ReportTimestamp:     time.Now().UTC(),
	}

	// Calculate installation age
	if age, err := it.config.GetInstallationAge(); err == nil {
		report.InstallationAge = age.String()
	}

	// Add service health information on Windows
	if runtime.GOOS == "windows" {
		serviceHealth, err := it.getServiceHealthReport(ctx)
		if err != nil {
			it.logger.WithError(err).Warn("Failed to get service health report")
		} else {
			report.ServiceHealth = serviceHealth
		}
	}

	return report, nil
}

// getServiceHealthReport gets Windows service health information
func (it *InstallationTelemetry) getServiceHealthReport(ctx context.Context) (*ServiceHealthReport, error) {
	serviceManager, err := windows.NewServiceManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create service manager: %w", err)
	}
	defer serviceManager.Close()

	health, err := serviceManager.GetServiceHealth()
	if err != nil {
		return nil, fmt.Errorf("failed to get service health: %w", err)
	}

	report := &ServiceHealthReport{
		ServiceName:     health.ServiceName,
		Status:          health.Status,
		ProcessID:       health.ProcessID,
		Win32ExitCode:   health.Win32ExitCode,
		ServiceExitCode: health.ServiceExitCode,
		StartType:       health.StartType,
		ServiceType:     health.ServiceType,
	}

	// Try to get additional monitoring information
	healthMonitor, err := windows.NewServiceHealthMonitor(it.logger, windows.DefaultServiceHealthMonitorConfig())
	if err == nil {
		defer healthMonitor.Close()
		
		summary := healthMonitor.GetHealthSummary()
		if monitored, ok := summary["is_monitoring"].(bool); ok {
			report.IsMonitored = monitored
		}
		if attempts, ok := summary["recovery_attempts"].(int); ok {
			report.RecoveryAttempts = attempts
		}
		if lastRecovery, ok := summary["last_recovery_time"].(time.Time); ok && !lastRecovery.IsZero() {
			report.LastRecoveryTime = lastRecovery
		}

		// Get service uptime
		if uptime, err := healthMonitor.GetServiceUptime(); err == nil {
			report.Uptime = uptime.String()
		}
	}

	return report, nil
}

// LogInstallationStatus logs installation status information
func (it *InstallationTelemetry) LogInstallationStatus(ctx context.Context) {
	report, err := it.GenerateInstallationStatusReport(ctx)
	if err != nil {
		it.logger.WithError(err).Error("Failed to generate installation status report")
		return
	}

	// Log basic installation information
	it.logger.WithFields(logrus.Fields{
		"installation_method":  report.InstallationMethod,
		"installation_version": report.InstallationVersion,
		"installed_at":         report.InstalledAt,
		"installed_by":         report.InstalledBy,
		"installation_age":     report.InstallationAge,
		"source":               report.Source,
		"platform":             report.Platform,
		"architecture":         report.Architecture,
	}).Info("Installation status report")

	// Log service health if available
	if report.ServiceHealth != nil {
		it.logger.WithFields(logrus.Fields{
			"service_status":       report.ServiceHealth.Status,
			"process_id":           report.ServiceHealth.ProcessID,
			"start_type":           report.ServiceHealth.StartType,
			"is_monitored":         report.ServiceHealth.IsMonitored,
			"recovery_attempts":    report.ServiceHealth.RecoveryAttempts,
			"uptime":               report.ServiceHealth.Uptime,
		}).Info("Service health status")
	}
}

// GetInstallationMetrics returns installation metrics for monitoring
func (it *InstallationTelemetry) GetInstallationMetrics() map[string]interface{} {
	metrics := map[string]interface{}{
		"installation_method":     it.config.Installation.Method,
		"installation_version":    it.config.Installation.Version,
		"installed_by":            it.config.Installation.InstalledBy,
		"source":                  it.config.Installation.Source,
		"is_automated":            it.config.IsAutomatedInstallation(),
		"platform":                runtime.GOOS,
		"architecture":            runtime.GOARCH,
	}

	// Add installation age
	if age, err := it.config.GetInstallationAge(); err == nil {
		metrics["installation_age_seconds"] = age.Seconds()
		metrics["installation_age_days"] = age.Hours() / 24
	}

	return metrics
}

// SendInstallationTelemetry sends installation telemetry to the platform
func (it *InstallationTelemetry) SendInstallationTelemetry(ctx context.Context, endpoint string) error {
	report, err := it.GenerateInstallationStatusReport(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate installation status report: %w", err)
	}

	// Convert report to JSON
	reportJSON, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("failed to marshal installation report: %w", err)
	}

	it.logger.WithFields(logrus.Fields{
		"endpoint":     endpoint,
		"report_size":  len(reportJSON),
		"method":       report.InstallationMethod,
	}).Debug("Sending installation telemetry")

	// TODO: Implement actual HTTP client to send telemetry
	// This would integrate with the existing client package
	it.logger.Info("Installation telemetry prepared for transmission")

	return nil
}

// ValidateInstallationIntegrity validates the installation integrity
func (it *InstallationTelemetry) ValidateInstallationIntegrity() (*InstallationIntegrityReport, error) {
	report := &InstallationIntegrityReport{
		Timestamp: time.Now().UTC(),
		Valid:     true,
		Issues:    []string{},
	}

	// Check if installation metadata is complete
	if it.config.Installation.Method == "" {
		report.Issues = append(report.Issues, "Installation method not specified")
		report.Valid = false
	}

	if it.config.Installation.InstalledAt == "" {
		report.Issues = append(report.Issues, "Installation timestamp missing")
		report.Valid = false
	}

	if it.config.Installation.Version == "" {
		report.Issues = append(report.Issues, "Installation version not recorded")
		report.Valid = false
	}

	// Validate installation timestamp
	if it.config.Installation.InstalledAt != "" {
		if _, err := time.Parse(time.RFC3339, it.config.Installation.InstalledAt); err != nil {
			report.Issues = append(report.Issues, fmt.Sprintf("Invalid installation timestamp format: %v", err))
			report.Valid = false
		}
	}

	// Check if automated installation has required fields
	if it.config.IsAutomatedInstallation() {
		if it.config.Installation.PairCode == "" {
			report.Issues = append(report.Issues, "Automated installation missing pair code")
			report.Valid = false
		}
		if it.config.Installation.Checksum == "" {
			report.Issues = append(report.Issues, "Automated installation missing checksum")
			report.Valid = false
		}
	}

	// On Windows, check service installation
	if runtime.GOOS == "windows" {
		serviceManager, err := windows.NewServiceManager()
		if err == nil {
			defer serviceManager.Close()
			
			if installed, err := serviceManager.IsServiceInstalled(); err != nil {
				report.Issues = append(report.Issues, fmt.Sprintf("Failed to check service installation: %v", err))
				report.Valid = false
			} else if !installed {
				report.Issues = append(report.Issues, "Windows service not installed")
				report.Valid = false
			}
		}
	}

	return report, nil
}

// InstallationIntegrityReport contains installation integrity validation results
type InstallationIntegrityReport struct {
	Timestamp time.Time `json:"timestamp"`
	Valid     bool      `json:"valid"`
	Issues    []string  `json:"issues"`
}