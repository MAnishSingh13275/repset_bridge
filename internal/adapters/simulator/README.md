# Simulator Hardware Adapter

The Simulator Hardware Adapter provides a mock implementation of the `HardwareAdapter` interface for testing and development purposes. It generates simulated door access events and logs door unlock operations without requiring actual hardware.

## Features

- **Automatic Event Generation**: Generates random check-in events at configurable intervals
- **Manual Event Triggering**: Allows manual triggering of specific events for testing
- **Door Unlock Simulation**: Simulates door unlock operations with logging
- **Configurable Users**: Supports custom list of simulated user IDs
- **Thread-Safe**: Safe for concurrent operations
- **Comprehensive Logging**: Detailed logging for all operations

## Configuration

The simulator accepts the following configuration settings:

```json
{
  "name": "simulator",
  "enabled": true,
  "settings": {
    "eventInterval": 30.0,
    "simulatedUsers": [
      "sim_user_001",
      "sim_user_002",
      "sim_user_003"
    ]
  }
}
```

### Configuration Options

- `eventInterval` (float): Interval in seconds between auto-generated events (minimum: 0.1 seconds)
- `simulatedUsers` (array): List of external user IDs to use for generated events

## Usage Example

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "time"

    "gym-door-bridge/internal/adapters"
    "gym-door-bridge/internal/adapters/simulator"
    "gym-door-bridge/internal/types"
)

func main() {
    logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
    
    // Create simulator adapter
    adapter := simulator.NewSimulatorAdapter(logger)
    
    // Configure adapter
    config := adapters.AdapterConfig{
        Name:    "simulator",
        Enabled: true,
        Settings: map[string]interface{}{
            "eventInterval": 5.0, // Generate event every 5 seconds
            "simulatedUsers": []interface{}{
                "test_user_001",
                "test_user_002",
            },
        },
    }
    
    ctx := context.Background()
    
    // Initialize adapter
    err := adapter.Initialize(ctx, config)
    if err != nil {
        logger.Error("Failed to initialize adapter", "error", err)
        return
    }
    
    // Register event callback
    adapter.OnEvent(func(event types.RawHardwareEvent) {
        logger.Info("Received event",
            "externalUserId", event.ExternalUserID,
            "eventType", event.EventType,
            "timestamp", event.Timestamp)
    })
    
    // Start listening for events
    err = adapter.StartListening(ctx)
    if err != nil {
        logger.Error("Failed to start listening", "error", err)
        return
    }
    
    // Manually trigger an event
    err = adapter.TriggerEvent("manual_user", types.EventTypeEntry)
    if err != nil {
        logger.Error("Failed to trigger event", "error", err)
    }
    
    // Simulate door unlock
    err = adapter.UnlockDoor(ctx, 3000) // Unlock for 3 seconds
    if err != nil {
        logger.Error("Failed to unlock door", "error", err)
    }
    
    // Let it run for a while
    time.Sleep(15 * time.Second)
    
    // Stop listening
    err = adapter.StopListening(ctx)
    if err != nil {
        logger.Error("Failed to stop listening", "error", err)
    }
    
    logger.Info("Simulator demo completed")
}
```

## Event Format

The simulator generates events with the following structure:

```json
{
  "externalUserId": "sim_user_001",
  "timestamp": "2024-01-01T10:00:00Z",
  "eventType": "entry",
  "rawData": {
    "simulator": true,
    "method": "auto_generated",
    "confidence": 1.0,
    "deviceInfo": "Simulator Hardware Adapter v1.0"
  }
}
```

### Event Types

- `entry`: Member entering the facility
- `exit`: Member leaving the facility  
- `denied`: Access denied (invalid credentials, etc.)

### Raw Data Fields

- `simulator`: Always `true` to identify simulated events
- `method`: Either `"auto_generated"` or `"manual_trigger"`
- `confidence`: Always `1.0` for simulated events
- `deviceInfo`: Static device information string

## Testing

Run the test suite:

```bash
go test -v ./internal/adapters/simulator
```

The test suite includes:

- Basic adapter lifecycle tests
- Event generation and callback tests
- Door unlock simulation tests
- Manual event triggering tests
- Concurrent operation safety tests
- Configuration validation tests

## Requirements Satisfied

This implementation satisfies the following requirements:

- **Requirement 1.3**: Operates in simulator mode for manual testing when no hardware is available
- **Requirement 9.5**: Implements door unlock simulation with logging for testing purposes

## Thread Safety

The simulator adapter is fully thread-safe and supports:

- Concurrent status checks
- Concurrent door unlock operations
- Concurrent manual event triggering
- Safe start/stop operations
- Thread-safe event callback execution

All internal state is protected by read-write mutexes to ensure data consistency during concurrent operations.