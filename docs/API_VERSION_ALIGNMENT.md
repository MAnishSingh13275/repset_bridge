# API Version Alignment

This document ensures consistency between the bridge implementation and platform API documentation.

## ✅ Implemented Endpoints

### Authentication Endpoints
- ✅ `POST /api/v1/devices/pair` - Device pairing (implemented)

### Event Endpoints  
- ✅ `POST /api/v1/checkin` - Member check-in events (implemented)
- ✅ `POST /api/v1/events` - General event submission (implemented)
- ✅ `GET /api/v1/health` - Health check (implemented)

### Device Management Endpoints
- ✅ `POST /api/v1/devices/heartbeat` - Regular heartbeat (implemented)
- ✅ `POST /api/v1/devices/status` - Device status check (implemented)
- ✅ `POST /api/v1/devices/heartbeat/trigger` - Manual heartbeat trigger (implemented)
- ✅ `GET /api/v1/devices/config` - Get device configuration (implemented)

### Door Control Endpoints
- ✅ `POST /open-door` - Remote door control (implemented)

## 🔄 Bridge Commands Alignment

### Status & Monitoring
- ✅ `gym-door-bridge status` - Shows pairing status and connectivity
- ✅ `gym-door-bridge trigger-heartbeat` - Manually trigger heartbeat
- ✅ `gym-door-bridge device-status` - Check status with platform

### Pairing & Authentication
- ✅ `gym-door-bridge pair <code>` - Pair with platform
- ✅ `gym-door-bridge unpair` - Unpair from platform

### Service Management
- ✅ `gym-door-bridge install` - Install Windows service
- ✅ `gym-door-bridge uninstall` - Uninstall Windows service

## 🔒 Authentication Implementation

### HMAC-SHA256 Signature
```go
// Signature calculation in internal/auth/hmac.go
message := string(body) + strconv.FormatInt(timestamp, 10) + deviceID
signature := HMAC-SHA256(message, deviceKey)
```

### Required Headers
- `X-Device-ID`: Device identifier
- `X-Signature`: HMAC-SHA256 signature  
- `X-Timestamp`: Unix timestamp

### Clock Skew Tolerance
- ✅ 5-minute tolerance implemented in `internal/auth/hmac.go`

## 📊 Event Format Alignment

### Check-in Event Structure
```json
{
  "eventId": "evt_1640995200_a1b2c3d4",
  "externalUserId": "MEMBER_123", 
  "timestamp": "2024-01-15T10:30:00Z",
  "eventType": "check_in",
  "isSimulated": false,
  "deviceId": "gym_12345_bridge_001"
}
```

### Supported Event Types
- ✅ `check_in` - Member entry
- ✅ `check_out` - Member exit  
- ✅ `door_open` - Door opened
- ✅ `door_close` - Door closed
- ✅ `access_denied` - Access denied
- ✅ `system_event` - System events

## 🔧 Configuration Alignment

### Device Configuration
```yaml
# Bridge config.yaml matches platform expectations
device_id: "gym_12345_bridge_001"
device_key: "sk_live_a1b2c3d4e5f6g7h8"
server_url: "https://repset.onezy.in"
tier: "normal"
heartbeat_interval: 60
queue_max_size: 10000
unlock_duration: 3000
```

### Platform Response Format
```json
{
  "deviceId": "gym_12345_bridge_001",
  "deviceKey": "sk_live_a1b2c3d4e5f6g7h8", 
  "config": {
    "heartbeatInterval": 60,
    "queueMaxSize": 10000,
    "unlockDuration": 3000
  }
}
```

## 🎯 Version Consistency Checklist

- ✅ All documented endpoints are implemented
- ✅ Authentication format matches specification
- ✅ Event structure aligns with platform expectations
- ✅ Configuration format is consistent
- ✅ Error handling follows platform conventions
- ✅ Command-line interface provides all necessary operations
- ✅ Documentation reflects actual implementation

## 🔄 Continuous Alignment Process

1. **API Changes**: Update both bridge code and documentation
2. **New Endpoints**: Implement in `internal/client/api.go` and document
3. **Authentication Updates**: Modify `internal/auth/` and update docs
4. **Configuration Changes**: Update `internal/config/config.go` and examples
5. **Command Updates**: Modify `cmd/main.go` and update command documentation

## 📝 Documentation Sources

- **Bridge Implementation**: `internal/client/api.go`
- **Authentication**: `internal/auth/hmac.go`
- **Configuration**: `internal/config/config.go`
- **Commands**: `cmd/main.go`
- **Platform Integration**: `docs/PLATFORM_INTEGRATION.md`
- **Complete Flow**: `docs/BRIDGE_FLOW_COMPLETE.md`

This alignment ensures that the bridge implementation and platform documentation remain synchronized and provide a consistent developer experience.