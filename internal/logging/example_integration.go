package logging

import (
	"context"
	"errors"
	"time"

	"github.com/sirupsen/logrus"
)

// ExampleIntegration demonstrates how to use the enhanced error handling and logging system
func ExampleIntegration() {
	// Initialize base logger
	baseLogger := logrus.New()
	baseLogger.SetLevel(logrus.InfoLevel)
	
	// Create enhanced logger with error handling and recovery capabilities
	enhanced := NewEnhancedLogger(baseLogger)
	
	ctx := context.Background()
	
	// Example 1: Hardware error with automatic recovery
	hardwareErr := errors.New("fingerprint scanner not responding")
	enhanced.LogHardwareErrorWithRecovery(ctx, hardwareErr, "fingerprint-reader", "scan")
	
	// Example 2: Network error with retry logic
	networkErr := errors.New("connection timeout")
	enhanced.LogNetworkErrorWithRecovery(ctx, networkErr, "submit_events", 3)
	
	// Example 3: Resource constraint triggering degradation
	enhanced.LogResourceConstraint(ctx, "memory", 85.0, 100.0)
	
	// Example 4: Custom error with structured context
	customErr := errors.New("custom service error")
	errContext := ErrorContext{
		Category:    ErrorCategoryService,
		Severity:    ErrorSeverityHigh,
		Component:   "event-processor",
		Operation:   "process_batch",
		UserID:      "user123",
		DeviceID:    "device456",
		Recoverable: true,
		Metadata: map[string]interface{}{
			"batch_size": 100,
			"retry_count": 2,
		},
	}
	enhanced.LogErrorWithRecovery(ctx, customErr, errContext)
	
	// Example 5: Register custom recovery action
	customRecoveryAction := RecoveryAction{
		Strategy:    RecoveryStrategyRestart,
		MaxAttempts: 2,
		Delay:       5 * time.Second,
		Action: func(ctx context.Context) error {
			// Custom recovery logic here
			baseLogger.Info("Executing custom recovery action")
			return nil
		},
		Description: "Custom service restart recovery",
	}
	enhanced.GetRecoveryManager().RegisterRecoveryAction(ErrorCategoryService, customRecoveryAction)
	
	// Example 6: Register custom degradation action
	customDegradationAction := DegradationAction{
		Name:        "disable_background_tasks",
		Description: "Disable background tasks to reduce CPU usage",
		Level:       DegradationMinor,
		Priority:    3,
		Action: func(ctx context.Context) error {
			baseLogger.Info("Disabling background tasks for resource conservation")
			return nil
		},
		Rollback: func(ctx context.Context) error {
			baseLogger.Info("Re-enabling background tasks")
			return nil
		},
	}
	enhanced.GetDegradationManager().RegisterAction(customDegradationAction)
	
	// Example 7: Get error statistics
	stats := enhanced.GetErrorStatistics()
	baseLogger.WithFields(logrus.Fields{
		"total_errors":       stats.TotalErrors,
		"recovery_attempts":  stats.RecoveryAttempts,
		"recovery_successes": stats.RecoverySuccesses,
		"current_degradation": enhanced.GetDegradationManager().GetCurrentLevel().String(),
	}).Info("Error handling statistics")
	
	// Example 8: Manual degradation control
	enhanced.GetDegradationManager().DegradeToLevel(ctx, DegradationModerate, "Manual degradation for maintenance")
	
	// Later restore
	time.Sleep(100 * time.Millisecond)
	enhanced.GetDegradationManager().RestoreToLevel(ctx, DegradationNone, "Maintenance complete")
}

// ExampleHardwareAdapterWithErrorHandling shows how to integrate error handling in a hardware adapter
func ExampleHardwareAdapterWithErrorHandling(enhanced *EnhancedLogger) {
	ctx := context.Background()
	
	// Simulate hardware operations with error handling
	simulateHardwareOperation := func(operation string) error {
		// Simulate various types of errors
		switch operation {
		case "initialize":
			err := errors.New("device not found")
			return enhanced.LogHardwareErrorWithRecovery(ctx, err, "rfid-reader", operation)
			
		case "scan":
			err := errors.New("scan timeout")
			return enhanced.LogHardwareErrorWithRecovery(ctx, err, "rfid-reader", operation)
			
		case "unlock_door":
			err := errors.New("door mechanism jammed")
			return enhanced.LogHardwareErrorWithRecovery(ctx, err, "door-controller", operation)
			
		default:
			return nil
		}
	}
	
	// Try operations with automatic error handling and recovery
	operations := []string{"initialize", "scan", "unlock_door"}
	for _, op := range operations {
		if err := simulateHardwareOperation(op); err != nil {
			enhanced.Logger.WithField("operation", op).Warn("Hardware operation failed after recovery attempts")
		}
	}
}

// ExampleNetworkClientWithErrorHandling shows how to integrate error handling in a network client
func ExampleNetworkClientWithErrorHandling(enhanced *EnhancedLogger) {
	ctx := context.Background()
	
	// Simulate network operations with error handling
	simulateNetworkOperation := func(operation string, retryCount int) error {
		// Simulate various network errors
		switch operation {
		case "submit_events":
			err := errors.New("connection refused")
			return enhanced.LogNetworkErrorWithRecovery(ctx, err, operation, retryCount)
			
		case "heartbeat":
			err := errors.New("request timeout")
			return enhanced.LogNetworkErrorWithRecovery(ctx, err, operation, retryCount)
			
		case "fetch_config":
			err := errors.New("server unavailable")
			return enhanced.LogNetworkErrorWithRecovery(ctx, err, operation, retryCount)
			
		default:
			return nil
		}
	}
	
	// Try operations with automatic retry and recovery
	operations := []string{"submit_events", "heartbeat", "fetch_config"}
	for _, op := range operations {
		for retry := 1; retry <= 3; retry++ {
			if err := simulateNetworkOperation(op, retry); err != nil {
				if retry == 3 {
					enhanced.Logger.WithField("operation", op).Error("Network operation failed after all retries")
				}
			} else {
				break // Success
			}
		}
	}
}

// ExampleResourceMonitoringWithDegradation shows how to monitor resources and trigger degradation
func ExampleResourceMonitoringWithDegradation(enhanced *EnhancedLogger) {
	ctx := context.Background()
	
	// Simulate resource monitoring
	resources := []struct {
		name      string
		usage     float64
		threshold float64
	}{
		{"cpu", 45.0, 100.0},     // Normal
		{"memory", 75.0, 100.0},  // Elevated - triggers minor degradation
		{"disk", 88.0, 100.0},    // High - triggers moderate degradation
		{"network", 96.0, 100.0}, // Critical - triggers severe degradation
	}
	
	for _, resource := range resources {
		enhanced.LogResourceConstraint(ctx, resource.name, resource.usage, resource.threshold)
		
		// Check current degradation level
		currentLevel := enhanced.GetDegradationManager().GetCurrentLevel()
		enhanced.Logger.WithFields(logrus.Fields{
			"resource":          resource.name,
			"usage_percent":     (resource.usage / resource.threshold) * 100,
			"degradation_level": currentLevel.String(),
		}).Info("Resource monitoring update")
	}
	
	// Get degradation status
	status := enhanced.GetDegradationManager().GetDegradationStatus()
	enhanced.Logger.WithField("degradation_status", status).Info("Current degradation status")
}