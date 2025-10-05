# 🌉 Complete Bridge Flow Documentation

This document provides a comprehensive overview of the Bridge Flow in the gym management system, covering all endpoints and implementation details.

## 🔄 Bridge Architecture Overview

The bridge system connects physical gym hardware (fingerprint readers, RFID systems, door controls) to the cloud-based gym management platform through a secure, authenticated communication channel.

```
Physical Hardware → Bridge Software → Platform APIs → Database
↓                    ↓              ↓           ↓
Fingerprint Reader → Go Bridge Client → /api/v1/* → BridgeDeployment
RFID Scanner      → Windows Service   → HMAC Auth → Member Records  
Door Controller   → Device Adapters   → Retry Logic → Check-in Logs
```

## 🔧 Complete Bridge Flow

### 1. Bridge Setup & Pairing

#### Step 1: Generate Pair Code
- Admin navigates to Bridge Management in gym dashboard
- System generates secure pair code: `ABC1-DEF2-GHI3` format
- Code expires in 30 minutes for security
- Stored in PairCode table with expiration tracking

#### Step 2: Bridge Installation
```powershell
# Download and install bridge software
.\install-bridge.ps1 -PairCode "ABC1-DEF2-GHI3" -ServerUrl "https://repset.onezy.in"

# Install as Windows service
gym-door-bridge.exe install

# Service runs as background service with auto-start
```

#### Step 3: Device Pairing
```http
POST /api/v1/devices/pair
Content-Type: application/json

{
  "pairCode": "ABC1-DEF2-GHI3",
  "deviceInfo": {
    "hostname": "GYM-PC-01",
    "platform": "windows",
    "version": "1.3.0",
    "tier": "normal"
  }
}
```

**Response:**
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

### 2. Authentication & Security

#### HMAC-SHA256 Authentication
All authenticated requests use HMAC-SHA256 signatures:

**Headers:**
- `X-Device-ID`: Device identifier
- `X-Signature`: HMAC-SHA256 signature
- `X-Timestamp`: Unix timestamp

**Signature Calculation:**
```
signature = HMAC-SHA256(body + timestamp + deviceId, deviceKey)
```

#### Security Features
- ✅ Temporary pair codes (30-min expiry)
- ✅ One-time use pair codes
- ✅ Cryptographic request signing
- ✅ Device credential rotation capability
- ✅ 5-minute clock skew tolerance
- ✅ Audit trail of all pairing events

### 3. Hardware Discovery & Connection

#### Automatic Device Discovery
- Bridge scans network for compatible hardware
- Supports: ZKTeco, ESSL, Realtime, generic biometric devices
- Creates device records in BridgeDeployment.devices
- Configures connections automatically

#### Supported Hardware Types
- **Fingerprint Readers**: ZKTeco, Suprema BioMini, Digital Persona
- **RFID Readers**: HID ProxPoint, AWID MPR-6225, Wiegand readers
- **Door Controls**: Electric strikes, magnetic locks, smart handles
- **Simulator**: Virtual device for testing

### 4. Operational Flow

#### Heartbeat System
```http
POST /api/v1/devices/heartbeat
X-Device-ID: gym_12345_bridge_001
X-Signature: a1b2c3d4e5f6g7h8...
X-Timestamp: 1640995200
Content-Type: application/json

{
  "status": "active",
  "tier": "normal",
  "queueDepth": 5,
  "lastEventTime": "2024-01-15T10:30:00Z",
  "systemInfo": {
    "cpuUsage": 15.5,
    "memoryUsage": 45.2,
    "diskSpace": 85.0
  }
}
```

#### Member Check-in Flow
```http
POST /api/v1/checkin
X-Device-ID: gym_12345_bridge_001
X-Signature: a1b2c3d4e5f6g7h8...
X-Timestamp: 1640995200
Content-Type: application/json

{
  "events": [
    {
      "eventId": "evt_1640995200_a1b2c3d4",
      "externalUserId": "MEMBER_123",
      "timestamp": "2024-01-15T10:30:00Z",
      "eventType": "check_in",
      "isSimulated": false,
      "deviceId": "gym_12345_bridge_001"
    }
  ]
}
```

#### General Events Submission
```http
POST /api/v1/events
X-Device-ID: gym_12345_bridge_001
X-Signature: a1b2c3d4e5f6g7h8...
X-Timestamp: 1640995200
Content-Type: application/json

{
  "events": [
    {
      "eventId": "evt_1640995200_a1b2c3d4",
      "externalUserId": "MEMBER_123",
      "timestamp": "2024-01-15T10:30:00Z",
      "eventType": "door_open",
      "isSimulated": false,
      "deviceId": "gym_12345_bridge_001"
    }
  ]
}
```

### 5. Device Management & Monitoring

#### Device Status Check
```http
POST /api/v1/devices/status
X-Device-ID: gym_12345_bridge_001
X-Signature: a1b2c3d4e5f6g7h8...
X-Timestamp: 1640995200
Content-Type: application/json

{
  "requestId": "status_check_001"
}
```

**Response:**
```json
{
  "status": "online",
  "lastSeen": "2024-01-15T10:30:00Z",
  "queueDepth": 5,
  "systemInfo": {
    "cpuUsage": 15.5,
    "memoryUsage": 45.2,
    "diskSpace": 85.0
  },
  "connectedDevices": ["fingerprint_reader_1", "door_controller_1"]
}
```

#### Manual Heartbeat Trigger
```http
POST /api/v1/devices/heartbeat/trigger
X-Device-ID: gym_12345_bridge_001
X-Signature: a1b2c3d4e5f6g7h8...
X-Timestamp: 1640995200
```

#### Get Device Configuration
```http
GET /api/v1/devices/config
X-Device-ID: gym_12345_bridge_001
X-Signature: a1b2c3d4e5f6g7h8...
X-Timestamp: 1640995200
```

**Response:**
```json
{
  "heartbeatInterval": 60,
  "queueMaxSize": 10000,
  "unlockDuration": 3000
}
```

### 6. Door Control

#### Remote Door Open
```http
POST /open-door
X-Device-ID: gym_12345_bridge_001
X-Signature: a1b2c3d4e5f6g7h8...
X-Timestamp: 1640995200
```

### 7. Health & Connectivity

#### Health Check (No Auth Required)
```http
GET /api/v1/health
```

**Response:**
```json
{
  "status": "ok",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## 🔧 Complete API Endpoint Reference

| Endpoint | Method | Auth Required | Purpose |
|----------|--------|---------------|---------|
| `/api/v1/devices/pair` | POST | No | Device pairing |
| `/api/v1/devices/heartbeat` | POST | Yes | Regular status updates |
| `/api/v1/devices/status` | POST | Yes | Manual status checks |
| `/api/v1/devices/heartbeat/trigger` | POST | Yes | Force heartbeat |
| `/api/v1/devices/config` | GET | Yes | Get device config |
| `/api/v1/checkin` | POST | Yes | Member check-in events |
| `/api/v1/events` | POST | Yes | General event submission |
| `/open-door` | POST | Yes | Remote door control |
| `/api/v1/health` | GET | No | Health check |

## 🎯 Bridge Management Features

### Real-time Monitoring
- Online/Offline device status
- Live event processing
- System resource monitoring
- Queue depth tracking

### Device Control
- Restart bridge service
- Update configuration
- Unpair/re-pair devices
- Manual heartbeat triggers

### Simulation & Testing
- Virtual device simulation
- Test check-in/check-out events
- Connectivity testing
- Authentication validation

### Health Monitoring
- Automatic restart on failures
- Queue management for offline periods
- Performance tier adjustment
- Event logging and audit trails

## 🔄 Data Flow Architecture

```
1. Hardware Event → 2. Bridge Processing → 3. Queue Management → 4. API Submission → 5. Platform Processing
     ↓                      ↓                     ↓                    ↓                    ↓
Fingerprint Scan → Event Normalization → Offline Queue → HMAC Auth → Database Storage
RFID Tap        → Batch Processing    → Retry Logic  → Rate Limiting → Member Update
Door Sensor     → Error Handling      → Persistence  → Validation   → Audit Log
```

## 🧹 Cleanup & Maintenance

### Automated Cleanup
- **Cron Job**: Daily cleanup at 2 AM UTC
- **Expired Codes**: Removes codes past expiration
- **Used Codes**: Cleans codes older than 24 hours
- **Gym Limits**: Keeps only 5 most recent codes per gym

### Manual Operations
- Force cleanup via admin interface
- Bridge software updates
- Device credential rotation
- Hardware reconfiguration

## 🎯 Business Benefits

- **Automated Access Control**: Members use fingerprints/RFID for entry
- **Real-time Monitoring**: Live status of all gym access points
- **Security**: Cryptographic authentication and audit trails
- **Scalability**: Support for multiple devices per gym
- **Reliability**: Offline queue management and auto-recovery
- **Integration**: Seamless connection to gym management system

## 🔒 Security Considerations

- All production traffic uses HTTPS
- Device credentials stored in OS credential manager
- HMAC signatures prevent request tampering
- Clock skew tolerance prevents replay attacks
- Automatic credential rotation capability
- Comprehensive audit logging

This bridge flow enables gyms to have sophisticated, automated access control while maintaining security and providing real-time visibility into member activity.