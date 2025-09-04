# ESSL & Biometric Device Integration Guide

This guide explains how to integrate ESSL fingerprint attendance devices and similar biometric access control systems with the Repset Bridge.

## Overview

The bridge connects to existing gym biometric devices (ESSL, ZKTeco, Realtime, etc.) and synchronizes attendance data with the Repset platform.

### Supported Biometric Devices

- **ESSL** - X990, K90, K40, K20, etc.
- **ZKTeco** - F18, F19, SpeedFace, etc.
- **Realtime** - T502, T501, etc.
- **Anviz** - FacePass, TC550, etc.
- **Any device with TCP/IP, RS485, or USB connectivity**

### Platform APIs (Repset SaaS)

- `POST /api/v1/fingerprint/enroll` - Enroll user fingerprints
- `POST /api/v1/fingerprint/authenticate` - Authenticate via fingerprint
- `GET /api/v1/fingerprint/users` - List users for enrollment
- `POST /api/v1/checkin` - Check-in/check-out (supports fingerprint auth)

### Bridge Configuration

Your bridge is configured with:

- **Device ID**: `bridge_1756948433034_2pkeoarbr`
- **Device Key**: `ef2dcc5f338c2363a715ac13f45b35cfd6992c8cc12d301bc789c7e4415186f4`
- **Server URL**: `https://repset.onezy.in`

## Integration Steps

### 1. Biometric Device Setup

Connect to existing gym biometric devices:

- **Network devices** - Connect via TCP/IP (most common)
- **Serial devices** - Connect via RS485/RS232
- **USB devices** - Direct USB connection to bridge computer

### 2. Bridge API Calls

#### Enroll a Fingerprint

```bash
curl -X POST https://repset.onezy.in/api/v1/fingerprint/enroll \
  -H "X-Device-ID: bridge_1756948433034_2pkeoarbr" \
  -H "X-Device-Key: ef2dcc5f338c2363a715ac13f45b35cfd6992c8cc12d301bc789c7e4415186f4" \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "user123",
    "externalUserId": "fp_device_001",
    "fingerIndex": 1,
    "quality": 95
  }'
```

#### Authenticate Fingerprint

```bash
curl -X POST https://repset.onezy.in/api/v1/checkin \
  -H "X-Device-ID: bridge_1756948433034_2pkeoarbr" \
  -H "X-Device-Key: ef2dcc5f338c2363a715ac13f45b35cfd6992c8cc12d301bc789c7e4415186f4" \
  -H "Content-Type: application/json" \
  -d '{
    "authMethod": "FINGERPRINT",
    "externalUserId": "fp_device_001",
    "eventType": "ENTRY",
    "confidence": 95
  }'
```

#### Get Users for Enrollment

```bash
curl -X GET "https://repset.onezy.in/api/v1/fingerprint/users?search=john" \
  -H "X-Device-ID: bridge_1756948433034_2pkeoarbr" \
  -H "X-Device-Key: ef2dcc5f338c2363a715ac13f45b35cfd6992c8cc12d301bc789c7e4415186f4"
```

### 3. Bridge Implementation

#### Enrollment Mode

1. Bridge enters enrollment mode (triggered by admin interface)
2. User places finger on scanner
3. Hardware generates `externalUserId` (unique fingerprint ID)
4. Bridge calls enrollment API with user mapping
5. Success/failure feedback to user

#### Authentication Mode

1. User places finger on scanner
2. Hardware recognizes fingerprint → returns `externalUserId`
3. Bridge calls check-in API with fingerprint data
4. Platform returns user info and check-in status
5. Bridge displays welcome message and opens door

### 4. Error Handling

#### Common Responses

- **404**: Fingerprint not enrolled → Show enrollment prompt
- **409**: Already checked in → Show current status
- **401/403**: Device authentication failed → Check credentials

#### Example Error Response

```json
{
  "error": "Fingerprint not recognized",
  "message": "No active enrollment found for this fingerprint",
  "shouldEnroll": true
}
```

### 5. Offline Operation

The bridge should cache fingerprint mappings locally for offline operation:

```sql
-- Local SQLite schema
CREATE TABLE fingerprint_cache (
    external_user_id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    user_name TEXT NOT NULL,
    user_email TEXT,
    finger_index INTEGER,
    quality INTEGER,
    enrolled_at DATETIME,
    last_synced DATETIME
);
```

### 6. Biometric Device Integration Examples

#### ESSL Device Integration

```go
// Example Go code for ESSL device
func connectToESSL(ip string, port int) (*ESSLDevice, error) {
    device := &ESSLDevice{
        IP:   ip,
        Port: port,
    }

    // Connect via TCP
    conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", ip, port))
    if err != nil {
        return nil, err
    }
    device.conn = conn

    // Authenticate with device
    if err := device.authenticate(); err != nil {
        return nil, err
    }

    return device, nil
}

// Poll for attendance records
func (d *ESSLDevice) pollAttendance() ([]AttendanceRecord, error) {
    // Send command to get new attendance records
    cmd := []byte{0x50, 0x50, 0x82, 0x7D, 0x13, 0x00, 0x00, 0x00}

    if _, err := d.conn.Write(cmd); err != nil {
        return nil, err
    }

    // Read response
    response := make([]byte, 1024)
    n, err := d.conn.Read(response)
    if err != nil {
        return nil, err
    }

    // Parse attendance records
    return d.parseAttendanceRecords(response[:n])
}

// Enroll user on ESSL device
func (d *ESSLDevice) enrollUser(userID string, name string) error {
    // Convert userID to device-specific format
    deviceUserID := d.convertUserID(userID)

    // Send enrollment command
    cmd := d.buildEnrollCommand(deviceUserID, name)

    if _, err := d.conn.Write(cmd); err != nil {
        return err
    }

    // Wait for enrollment completion
    return d.waitForEnrollmentComplete()
}
```

### 7. Testing

#### Test Enrollment

1. Use the admin interface to search for a user
2. Click "Enroll Fingerprint"
3. Bridge should receive enrollment request
4. Simulate finger scan and call enrollment API

#### Test Authentication

1. User places finger on scanner
2. Bridge recognizes fingerprint
3. Calls check-in API
4. Displays welcome message
5. Records attendance in platform

### 8. Configuration

Add to `config.yaml`:

```yaml
biometric_devices:
  - name: "main_entrance"
    type: "essl" # essl, zkteco, realtime, anviz
    connection: "tcp" # tcp, serial, usb
    config:
      ip_address: "192.168.1.100"
      port: 4370 # ESSL default port
      device_id: 1 # Device ID on the network
      password: "0" # Device password
    sync_interval: 30 # seconds - how often to poll for new records

  - name: "back_entrance"
    type: "zkteco"
    connection: "tcp"
    config:
      ip_address: "192.168.1.101"
      port: 4370
      device_id: 1
      password: "0"
    sync_interval: 30

  - name: "staff_entrance"
    type: "essl"
    connection: "serial"
    config:
      port: "/dev/ttyUSB0" # Serial port
      baud_rate: 115200
      device_id: 1
    sync_interval: 60
```

## API Reference

### Enrollment API

- **Endpoint**: `POST /api/v1/fingerprint/enroll`
- **Headers**: `X-Device-ID`, `X-Device-Key`
- **Body**: `userId`, `externalUserId`, `fingerIndex`, `quality`

### Authentication API

- **Endpoint**: `POST /api/v1/checkin`
- **Headers**: `X-Device-ID`, `X-Device-Key`
- **Body**: `authMethod: "FINGERPRINT"`, `externalUserId`, `eventType`

### User Lookup API

- **Endpoint**: `GET /api/v1/fingerprint/users`
- **Headers**: `X-Device-ID`, `X-Device-Key`
- **Query**: `search`, `includeEnrolled`

## Bridge Implementation Status

### ✅ Completed

- **Platform APIs** - All fingerprint/biometric APIs implemented and tested
- **Bridge Architecture** - Biometric adapter framework created
- **ESSL Integration** - Basic ESSL protocol implementation (partial)
- **Simulator** - Full simulator for testing without hardware
- **Configuration** - YAML configuration support for multiple devices

### 🚧 In Progress

- **ESSL Protocol** - Complete ESSL device communication protocol
- **ZKTeco Integration** - ZKTeco device protocol implementation
- **Realtime Integration** - Realtime device protocol implementation

### 📋 Next Steps

1. **Complete ESSL Implementation**

   - Finish user enrollment protocol
   - Implement attendance record parsing
   - Add error handling and reconnection logic

2. **Test with Real Hardware**

   - Connect to actual ESSL/ZKTeco devices
   - Verify attendance record synchronization
   - Test user enrollment from platform

3. **Add Device Management**

   - Device discovery and auto-configuration
   - Health monitoring and alerts
   - Firmware update support

4. **Production Deployment**
   - Install bridge on gym network
   - Configure device connections
   - Set up monitoring and logging

## Testing the System

### Using the Simulator

1. **Configure simulator** in `config.yaml`:

   ```yaml
   biometric_devices:
     - name: "test_device"
       type: "simulator"
       connection: "tcp"
       sync_interval: 30
   ```

2. **Start the bridge** with simulator enabled
3. **Use the platform interface** to enroll users
4. **Watch for simulated attendance** records in logs

### With Real Hardware

1. **Connect ESSL device** to network
2. **Configure device IP** in bridge config
3. **Test connection** using bridge logs
4. **Enroll users** via platform interface
5. **Verify attendance sync** when users scan fingerprints

The platform is ready - focus on completing the bridge-side device integration!