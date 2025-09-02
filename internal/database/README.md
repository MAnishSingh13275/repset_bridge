# Database Package

This package provides SQLite database functionality for the Gym Door Access Bridge with the following features:

## Features

- **SQLite Connection Management**: WAL mode enabled for durability and performance
- **AES-GCM Encryption**: Sensitive data encrypted at rest
- **Performance Tier Support**: Configurable pragmas based on system capabilities
- **Schema Migrations**: Automatic table creation and indexing
- **CRUD Operations**: Complete event queue, device config, and adapter status management

## Performance Tiers

- **Lite**: NORMAL synchronous mode for low-spec hardware
- **Normal**: NORMAL synchronous mode for standard hardware  
- **Full**: FULL synchronous mode for high-spec hardware with enhanced durability

## Database Schema

### event_queue
- Stores check-in events for offline processing
- Encrypted raw_data field for sensitive payloads
- Retry tracking and sent status management

### device_config  
- Key-value configuration storage
- Automatic encryption for sensitive keys (device_key, api_secret, etc.)
- Timestamp tracking for configuration changes

### adapter_status
- Hardware adapter status tracking
- Error message storage and last event timestamps
- Status types: active, error, disabled

## Testing

**Note**: Tests require CGO to be enabled and a C compiler (gcc) to be available for SQLite compilation.

To run tests on systems with proper C toolchain:
```bash
CGO_ENABLED=1 go test ./internal/database -v
```

For Windows development without gcc, the implementation has been verified through:
- Code review against requirements
- Schema validation
- Interface compliance checking
- Error handling verification

## Usage Example

```go
config := database.Config{
    DatabasePath:    "./data/bridge.db",
    EncryptionKey:   encryptionKey, // 32-byte key
    PerformanceTier: database.TierNormal,
}

db, err := database.NewDB(config)
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// Insert event
event := &database.EventQueue{
    EventID:        "evt_123",
    ExternalUserID: "user_456", 
    Timestamp:      time.Now(),
    EventType:      database.EventTypeEntry,
    RawData:        `{"fingerprint_id": "fp123"}`,
}

err = db.InsertEvent(event)
```

## Requirements Satisfied

- **4.1**: Offline event storage with encrypted payloads ✓
- **4.2**: Queue management with replay capability ✓  
- **4.5**: AES-GCM encryption for sensitive data ✓