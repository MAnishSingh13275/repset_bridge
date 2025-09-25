# Offline Event Queuing Implementation

This document describes the offline event queuing system implemented for the Gym Door Bridge, which ensures that biometric check-in events are never lost, even when the bridge device is offline.

## Overview

The offline event queuing system consists of four main components:

1. **Event Processing with Deduplication** - Processes raw hardware events and prevents duplicates
2. **Local SQLite Storage** - Persists events locally with encryption
3. **Batch Event Submission** - Automatically submits queued events when online
4. **Event Deduplication Logic** - Prevents duplicate events within a configurable time window

## Architecture

```
Hardware Adapter → Event Processor → Queue Manager → Submission Service → Cloud API
                      ↓                    ↓
                 Deduplication        SQLite Database
```

## Components

### 1. Event Processor (`internal/processor/event_processor.go`)

The event processor handles:
- **Validation**: Ensures events have required fields and valid timestamps
- **Deduplication**: Prevents duplicate events within a 5-minute window
- **ID Generation**: Creates unique, deterministic event IDs
- **User Mapping**: Resolves external user IDs to internal user IDs
- **Metadata Enrichment**: Adds device ID and other metadata

**Configuration:**
```go
ProcessorConfig{
    DeviceID:            "device-123",
    EnableDeduplication: true,
    DeduplicationWindow: 300, // 5 minutes in seconds
}
```

### 2. Queue Manager (`internal/queue/queue.go`)

The queue manager provides:
- **Persistent Storage**: SQLite database with encryption
- **Batch Operations**: Efficient batch retrieval and status updates
- **Capacity Management**: Automatic eviction when queue is full
- **Statistics**: Comprehensive queue and event statistics
- **Query Interface**: Flexible event querying with filters

**Key Features:**
- Configurable queue size based on performance tier
- FIFO eviction when capacity is exceeded
- Encrypted storage of sensitive event data
- Retry count tracking for failed submissions

### 3. Submission Service (`internal/client/submission_service.go`)

The submission service handles:
- **Periodic Submission**: Automatically submits pending events
- **Batch Processing**: Submits events in configurable batch sizes
- **Retry Logic**: Handles failed submissions with exponential backoff
- **Status Tracking**: Updates event status (sent/failed) in the queue

**Configuration by Tier:**
- **Lite**: 10 events/batch, 60s interval, 3 retries
- **Normal**: 50 events/batch, 30s interval, 5 retries  
- **Full**: 100 events/batch, 15s interval, 10 retries

### 4. Database Schema

The `event_queue` table stores events with the following structure:

```sql
CREATE TABLE event_queue (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id TEXT UNIQUE NOT NULL,
    external_user_id TEXT NOT NULL,
    timestamp DATETIME NOT NULL,
    event_type TEXT NOT NULL CHECK (event_type IN ('entry', 'exit', 'denied')),
    is_simulated BOOLEAN DEFAULT FALSE,
    device_id TEXT NOT NULL DEFAULT '',
    raw_data TEXT, -- Encrypted JSON
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    sent_at DATETIME NULL,
    retry_count INTEGER DEFAULT 0
);
```

## Event Flow

### 1. Event Generation
```go
// Hardware adapter generates raw event
rawEvent := types.RawHardwareEvent{
    ExternalUserID: "user123",
    Timestamp:      time.Now(),
    EventType:      "entry",
    RawData:        map[string]interface{}{"confidence": 0.95},
}
```

### 2. Event Processing
```go
// Process event with deduplication
result, err := processor.ProcessEvent(ctx, rawEvent)
if result.Processed {
    // Event is valid and not a duplicate
    standardEvent := result.Event
}
```

### 3. Event Queuing
```go
// Store event in local queue
err := queueManager.Enqueue(ctx, standardEvent)
```

### 4. Automatic Submission
```go
// Submission service runs periodically
submissionService.StartPeriodicSubmission(ctx)

// Submits pending events in batches
result, err := submissionService.SubmitPendingEvents(ctx)
```

## Deduplication Logic

The system prevents duplicate events using:

1. **Time Window**: Events within 5 minutes of each other
2. **User + Event Type**: Same user and event type (entry/exit/denied)
3. **Database Check**: Queries existing events in the time window

```go
// Check for duplicates
isDuplicate, err := processor.IsEventDuplicate(ctx, rawEvent)
if isDuplicate {
    // Skip processing
    return ProcessingResult{Processed: false, Reason: "duplicate"}
}
```

## Error Handling

### Network Failures
- Events are queued locally when submission fails
- Automatic retry with exponential backoff
- Events remain in queue until successfully sent

### Storage Failures
- Database operations are transactional
- Encryption/decryption errors are logged
- Graceful degradation when storage is full

### Processing Failures
- Invalid events are logged but not queued
- Validation errors include specific field information
- Processing statistics track success/failure rates

## Monitoring and Statistics

### Queue Statistics
```go
stats, err := queueManager.GetStats(ctx)
// Returns: QueueDepth, PendingEvents, SentEvents, FailedEvents, etc.
```

### Event Statistics
```go
eventStats, err := queueManager.GetEventStats(ctx)
// Returns: TotalEvents, EventsByType, EventsByHour, UniqueUsers, etc.
```

### Processor Statistics
```go
processorStats := processor.GetStats()
// Returns: TotalProcessed, TotalDuplicates, TotalInvalid, LastProcessedAt
```

## Configuration

### Performance Tiers

The system adapts to different hardware capabilities:

```go
// Lite tier (low-end hardware)
QueueConfig{
    MaxSize:       1000,
    BatchSize:     10,
    SubmitInterval: 60 * time.Second,
    MaxRetries:    3,
}

// Normal tier (standard hardware)  
QueueConfig{
    MaxSize:       10000,
    BatchSize:     50,
    SubmitInterval: 30 * time.Second,
    MaxRetries:    5,
}

// Full tier (high-end hardware)
QueueConfig{
    MaxSize:       50000,
    BatchSize:     100,
    SubmitInterval: 15 * time.Second,
    MaxRetries:    10,
}
```

## Security

### Data Encryption
- Raw event data is encrypted using AES-256
- Encryption keys are managed by the credential system
- Sensitive fields are encrypted at rest

### Authentication
- All API requests are signed using HMAC-SHA256
- Device credentials are stored securely
- Request timestamps prevent replay attacks

## Testing

The implementation includes comprehensive tests:

- **Unit Tests**: Individual component testing
- **Integration Tests**: End-to-end event flow testing
- **Deduplication Tests**: Duplicate detection verification
- **Performance Tests**: Load testing with high event volumes

Run tests with:
```bash
go test ./internal/bridge -v
go test ./internal/queue -v
go test ./internal/processor -v
```

## Troubleshooting

### Common Issues

1. **Events Not Being Submitted**
   - Check network connectivity
   - Verify device authentication
   - Review submission service logs

2. **High Queue Depth**
   - Check submission service status
   - Verify API endpoint availability
   - Review retry counts and error messages

3. **Duplicate Events**
   - Verify deduplication is enabled
   - Check deduplication window configuration
   - Review event timestamps

### Diagnostic Commands

```bash
# Check queue status
curl http://localhost:8080/api/queue/stats

# View pending events
curl http://localhost:8080/api/queue/events?status=pending

# Check processor statistics
curl http://localhost:8080/api/processor/stats
```

## Future Enhancements

Potential improvements to the offline queuing system:

1. **Compression**: Compress event data to reduce storage usage
2. **Prioritization**: Priority queues for critical events
3. **Batching Optimization**: Dynamic batch sizing based on network conditions
4. **Event Archiving**: Long-term storage of historical events
5. **Real-time Sync**: WebSocket-based real-time event streaming