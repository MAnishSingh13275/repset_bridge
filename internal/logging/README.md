# Enhanced Error Handling and Logging System

This package provides a comprehensive error handling and logging system for the Gym Door Access Bridge. It includes structured error logging, automatic error recovery, graceful service degradation, and detailed error statistics.

## Features

### 1. Structured Error Logging
- **Error Classification**: Automatically categorizes errors into types (Hardware, Network, Security, Storage, Resource, Service, Config)
- **Error Severity**: Assigns severity levels (Critical, High, Medium, Low, Info)
- **Rich Context**: Captures detailed context including component, operation, user ID, device ID, and custom metadata
- **Stack Traces**: Automatically captures stack traces for critical and high-severity errors

### 2. Automatic Error Recovery
- **Recovery Strategies**: Supports multiple recovery strategies (Retry, Restart, Degrade, Failover, Skip)
- **Exponential Backoff**: Implements intelligent retry logic with exponential backoff
- **Custom Recovery Actions**: Allows registration of custom recovery functions
- **Recovery Statistics**: Tracks recovery attempts and success rates

### 3. Graceful Service Degradation
- **Performance Tiers**: Automatically adjusts service level based on resource constraints
- **Degradation Levels**: Supports multiple degradation levels (None, Minor, Moderate, Severe, Critical)
- **Custom Actions**: Allows registration of custom degradation and rollback actions
- **Resource Monitoring**: Monitors CPU, memory, and disk usage to trigger degradation

### 4. Comprehensive Statistics
- **Error Tracking**: Tracks total errors by category and severity
- **Recovery Metrics**: Monitors recovery attempts and success rates
- **Degradation Status**: Provides detailed degradation status and duration
- **Real-time Monitoring**: Offers real-time access to all statistics

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "errors"
    "gym-door-bridge/internal/logging"
)

func main() {
    // Initialize enhanced logger
    logger := logging.InitializeEnhanced("info")
    
    ctx := context.Background()
    
    // Log a hardware error with automatic recovery
    err := errors.New("fingerprint scanner not responding")
    logger.LogHardwareErrorWithRecovery(ctx, err, "fingerprint-reader", "scan")
    
    // Log a network error with retry logic
    networkErr := errors.New("connection timeout")
    logger.LogNetworkErrorWithRecovery(ctx, networkErr, "submit_events", 3)
    
    // Monitor resource constraints
    logger.LogResourceConstraint(ctx, "memory", 85.0, 100.0)
    
    // Get error statistics
    stats := logger.GetErrorStatistics()
    logger.WithField("stats", stats).Info("Current error statistics")
}
```

### Advanced Usage with Custom Recovery

```go
// Register custom recovery action
customRecovery := logging.RecoveryAction{
    Strategy:    logging.RecoveryStrategyRestart,
    MaxAttempts: 3,
    Delay:       5 * time.Second,
    Action: func(ctx context.Context) error {
        // Custom recovery logic
        return restartService()
    },
    Description: "Restart failed service",
}

logger.GetRecoveryManager().RegisterRecoveryAction(
    logging.ErrorCategoryService, 
    customRecovery,
)

// Register custom degradation action
customDegradation := logging.DegradationAction{
    Name:        "disable_background_tasks",
    Description: "Disable background tasks to reduce CPU usage",
    Level:       logging.DegradationMinor,
    Priority:    1,
    Action: func(ctx context.Context) error {
        return disableBackgroundTasks()
    },
    Rollback: func(ctx context.Context) error {
        return enableBackgroundTasks()
    },
}

logger.GetDegradationManager().RegisterAction(customDegradation)
```

## Error Categories

The system automatically classifies errors into the following categories:

- **Hardware**: Device communication failures, sensor errors
- **Network**: Connection timeouts, DNS failures, HTTP errors
- **Security**: Authentication failures, HMAC validation errors
- **Storage**: Database errors, file system issues
- **Resource**: Memory/CPU/disk constraints
- **Service**: Application logic errors, service failures
- **Config**: Configuration parsing errors, invalid settings
- **Unknown**: Unclassified errors

## Error Severity Levels

- **Critical**: Requires immediate attention, may cause system failure
- **High**: Important errors that should be addressed soon
- **Medium**: Moderate errors that need attention
- **Low**: Minor errors or warnings
- **Info**: Informational messages

## Recovery Strategies

### Retry Strategy
- Implements exponential backoff with jitter
- Configurable maximum attempts and delays
- Tracks retry state across attempts
- Automatically cleans up successful recoveries

### Restart Strategy
- Restarts failed components or services
- Configurable restart attempts
- Supports graceful shutdown and startup

### Degrade Strategy
- Reduces service functionality to conserve resources
- Multiple degradation levels with automatic rollback
- Custom degradation actions with priority ordering

### Failover Strategy
- Switches to backup systems or alternative methods
- Supports custom failover logic
- Automatic fallback detection

### Skip Strategy
- Ignores non-critical errors and continues operation
- Useful for optional features or best-effort operations

## Degradation Levels

### None (Normal Operation)
- All features enabled
- Full performance and functionality

### Minor Degradation
- Reduce non-essential features
- Lower heartbeat frequency
- Disable detailed metrics collection

### Moderate Degradation
- Reduce queue sizes
- Disable non-essential adapters
- Limit background processing

### Severe Degradation
- Disable file logging
- Emergency queue flush
- Minimal background activity

### Critical Degradation
- Essential functions only
- Maximum resource conservation
- Emergency operation mode

## Resource Monitoring

The system automatically monitors resource usage and triggers degradation:

- **70-79%**: Minor degradation
- **80-89%**: Moderate degradation  
- **90-94%**: Severe degradation
- **95%+**: Critical degradation

## Integration Examples

### Hardware Adapter Integration

```go
type HardwareAdapter struct {
    logger *logging.EnhancedLogger
}

func (h *HardwareAdapter) Scan(ctx context.Context) error {
    err := h.performScan()
    if err != nil {
        return h.logger.LogHardwareErrorWithRecovery(
            ctx, err, "fingerprint-reader", "scan")
    }
    return nil
}
```

### Network Client Integration

```go
type NetworkClient struct {
    logger *logging.EnhancedLogger
}

func (n *NetworkClient) SubmitEvents(ctx context.Context, events []Event) error {
    for attempt := 1; attempt <= 3; attempt++ {
        err := n.doSubmit(events)
        if err != nil {
            if recoveryErr := n.logger.LogNetworkErrorWithRecovery(
                ctx, err, "submit_events", attempt); recoveryErr != nil {
                continue // Retry
            }
            return nil // Recovered
        }
        return nil // Success
    }
    return errors.New("failed after all retries")
}
```

### Service Integration

```go
type EventProcessor struct {
    logger *logging.EnhancedLogger
}

func (e *EventProcessor) ProcessBatch(ctx context.Context, batch []Event) error {
    err := e.processBatch(batch)
    if err != nil {
        errContext := logging.ErrorContext{
            Category:    logging.ErrorCategoryService,
            Severity:    logging.ErrorSeverityHigh,
            Component:   "event-processor",
            Operation:   "process_batch",
            Recoverable: true,
            Metadata: map[string]interface{}{
                "batch_size": len(batch),
            },
        }
        return e.logger.LogErrorWithRecovery(ctx, err, errContext)
    }
    return nil
}
```

## Configuration

### Default Recovery Strategies

The system comes with pre-configured recovery strategies:

- **Hardware**: Retry with 5-second delays, max 3 attempts
- **Network**: Retry with 2-second delays, max 5 attempts
- **Security**: No automatic recovery (manual intervention required)
- **Storage**: Retry with 1-second delays, max 2 attempts
- **Resource**: Automatic degradation
- **Service**: Restart with 10-second delays, max 2 attempts
- **Config**: Skip invalid configuration and use defaults

### Default Degradation Actions

- **Minor**: Reduce heartbeat frequency, disable detailed metrics
- **Moderate**: Reduce queue size, disable non-essential adapters
- **Severe**: Disable file logging, emergency queue flush
- **Critical**: Minimal operation mode

## Testing

The package includes comprehensive unit tests covering:

- Error classification and structured logging
- Recovery strategy execution and retry logic
- Degradation level management and action execution
- Concurrent error handling and thread safety
- Statistics tracking and reporting

Run tests with:
```bash
go test ./internal/logging -v
```

## Performance Considerations

- **Minimal Overhead**: Structured logging adds minimal performance overhead
- **Efficient Recovery**: Recovery actions are executed asynchronously when possible
- **Memory Management**: Error statistics use bounded memory with automatic cleanup
- **Concurrent Safety**: All operations are thread-safe with minimal lock contention

## Best Practices

1. **Use Appropriate Severity**: Choose the correct severity level for each error
2. **Provide Rich Context**: Include relevant metadata for better debugging
3. **Register Custom Actions**: Implement custom recovery and degradation actions for your specific use cases
4. **Monitor Statistics**: Regularly check error statistics to identify patterns
5. **Test Recovery Logic**: Ensure your custom recovery actions work correctly
6. **Handle Degradation**: Design your application to work gracefully at all degradation levels

## Troubleshooting

### Common Issues

1. **Recovery Not Working**: Check if error is marked as recoverable
2. **Degradation Not Triggering**: Verify resource thresholds are configured correctly
3. **Statistics Not Updating**: Ensure you're using the enhanced logger methods
4. **Custom Actions Not Executing**: Verify action registration and error categories match

### Debug Logging

Enable debug logging to see detailed recovery and degradation operations:

```go
logger := logging.InitializeEnhanced("debug")
```

### Monitoring Integration

The system provides structured logs that can be easily integrated with monitoring systems like:

- **Prometheus**: Export error metrics and recovery statistics
- **Grafana**: Visualize error trends and degradation events
- **ELK Stack**: Centralized log analysis and alerting
- **DataDog**: Application performance monitoring

## Contributing

When adding new error handling capabilities:

1. Follow the existing error classification patterns
2. Add comprehensive unit tests
3. Update documentation and examples
4. Consider backward compatibility
5. Test with concurrent workloads