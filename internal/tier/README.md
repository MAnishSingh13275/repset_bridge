# Tier Package

The tier package provides system resource monitoring and automatic performance tier detection for the Gym Door Access Bridge. It monitors CPU, memory, and disk usage to automatically determine the appropriate performance tier and adjust system behavior accordingly.

## Performance Tiers

The system operates in three performance tiers based on available system resources:

### Lite Mode
- **Criteria**: <2 CPU cores OR <2GB RAM
- **Queue Size**: 1,000 events
- **Heartbeat Interval**: 5 minutes
- **Features**: Basic functionality only, no web UI or metrics

### Normal Mode  
- **Criteria**: 2-4 CPU cores + ≥2GB RAM
- **Queue Size**: 10,000 events
- **Heartbeat Interval**: 1 minute
- **Features**: Standard functionality with metrics enabled

### Full Mode
- **Criteria**: >4 CPU cores + ≥8GB RAM
- **Queue Size**: 50,000 events
- **Heartbeat Interval**: 30 seconds
- **Features**: Enhanced functionality with web UI and detailed metrics

## Usage

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "time"
    
    "gym-door-bridge/internal/tier"
    "github.com/sirupsen/logrus"
)

func main() {
    logger := logrus.New()
    
    // Create a tier detector
    detector := tier.NewDetector(
        tier.WithLogger(logger),
        tier.WithEvaluationInterval(30*time.Second),
    )
    
    // Start monitoring in a goroutine
    ctx := context.Background()
    go func() {
        if err := detector.Start(ctx); err != nil {
            log.Printf("Tier detector error: %v", err)
        }
    }()
    
    // Get current tier
    currentTier := detector.GetCurrentTier()
    log.Printf("Current tier: %s", currentTier)
    
    // Get tier configuration
    config := tier.GetTierConfig(currentTier)
    log.Printf("Queue max size: %d", config.QueueMaxSize)
    log.Printf("Heartbeat interval: %v", config.HeartbeatInterval)
}
```

### With Tier Change Callback

```go
// Create detector with callback for tier changes
callback := func(oldTier, newTier tier.Tier) {
    log.Printf("Tier changed from %s to %s", oldTier, newTier)
    
    // Update application configuration based on new tier
    config := tier.GetTierConfig(newTier)
    updateQueueSize(config.QueueMaxSize)
    updateHeartbeatInterval(config.HeartbeatInterval)
}

detector := tier.NewDetector(
    tier.WithLogger(logger),
    tier.WithTierChangeCallback(callback),
)
```

### Using the Factory

```go
// Create factory
factory := tier.NewDetectorFactory(logger)

// Create detector with default settings
detector := factory.CreateDetector()

// Create detector with callback
detector = factory.CreateDetectorWithCallback(func(old, new tier.Tier) {
    // Handle tier change
})

// Create detector for testing
testResources := tier.CreateNormalSystemResources()
testDetector := factory.CreateDetectorForTesting(testResources)
```

## System Resource Monitoring

The package monitors the following system resources:

- **CPU Cores**: Number of available CPU cores
- **Memory**: Total system memory in GB
- **CPU Usage**: Current CPU usage percentage
- **Memory Usage**: Current memory usage percentage  
- **Disk Usage**: Current disk usage percentage

### Platform Support

The resource monitoring supports:
- **Windows**: Uses Windows API calls for accurate memory and disk detection
- **macOS**: Uses system commands (`sysctl`, `df`) for resource detection
- **Linux**: Uses `/proc/meminfo` and `syscall.Statfs` for resource detection
- **Other platforms**: Provides fallback implementations

## Testing

The package includes comprehensive test coverage:

```bash
# Run all tests
go test ./internal/tier -v

# Run specific test suites
go test ./internal/tier -run TestTier -v
go test ./internal/tier -run TestDetector -v
go test ./internal/tier -run TestSystemResourceMonitor -v
```

### Mock Testing

For testing purposes, use the mock resource monitor:

```go
// Create mock resources
resources := tier.SystemResources{
    CPUCores:    4,
    MemoryGB:    8.0,
    CPUUsage:    25.0,
    MemoryUsage: 40.0,
    DiskUsage:   30.0,
}

// Create mock monitor
mockMonitor := tier.NewMockResourceMonitor(resources)

// Create detector with mock
detector := tier.NewDetector(
    tier.WithResourceMonitor(mockMonitor),
)

// Change resources during test
mockMonitor.SetResources(tier.CreateLiteSystemResources())
```

## Integration with Configuration

The tier system integrates with the existing configuration system. When tier changes occur, the application should update its configuration accordingly:

```go
func updateConfigurationForTier(currentTier tier.Tier) {
    config := tier.GetTierConfig(currentTier)
    
    // Update queue configuration
    queueManager.SetMaxSize(config.QueueMaxSize)
    
    // Update heartbeat interval
    heartbeatManager.SetInterval(config.HeartbeatInterval)
    
    // Enable/disable features based on tier
    if config.EnableWebUI {
        webServer.Start()
    } else {
        webServer.Stop()
    }
    
    if config.EnableMetrics {
        metricsCollector.Start()
    } else {
        metricsCollector.Stop()
    }
}
```

## Error Handling

The tier detector handles various error conditions gracefully:

- **Resource monitoring failures**: Continues with last known values
- **Platform-specific errors**: Falls back to reasonable defaults
- **Context cancellation**: Cleanly shuts down monitoring

All errors are logged with appropriate context for debugging.