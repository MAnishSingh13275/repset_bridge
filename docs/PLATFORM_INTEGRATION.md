# Repset Platform Integration

This document provides quick reference information for integrating with the Repset SaaS platform at `https://repset.onezy.in`.

## Platform Details

- **Platform URL**: `https://repset.onezy.in`
- **API Base URL**: `https://repset.onezy.in/api/v1`
- **Documentation**: `https://docs.repset.onezy.in`
- **Support**: `support@repset.onezy.in`

## API Endpoints

### Authentication
- **Device Pairing**: `POST /api/v1/devices/pair`
- **Key Rotation**: `POST /api/v1/devices/rotate-key`

### Events
- **Submit Events**: `POST /api/v1/events`
- **Check-in Events**: `POST /api/v1/checkin`
- **Health Check**: `GET /api/v1/health`

### Device Management
- **Device Heartbeat**: `POST /api/v1/devices/heartbeat`
- **Device Status**: `POST /api/v1/devices/status`
- **Trigger Heartbeat**: `POST /api/v1/devices/heartbeat/trigger`
- **Get Device Config**: `GET /api/v1/devices/config`
- **Update Configuration**: `PUT /api/v1/devices/{deviceId}/config`

### Door Control
- **Remote Door Open**: `POST /open-door`

### Fingerprint Management
- **Enroll Fingerprint**: `POST /api/v1/fingerprint/enroll`
- **List Users**: `GET /api/v1/fingerprint/users`
- **Delete Fingerprint**: `DELETE /api/v1/fingerprint/{id}`

## Configuration Examples

### Basic Configuration
```yaml
api:
  baseUrl: "https://repset.onezy.in"
  timeout: "30s"
```

### Production Configuration
```yaml
api:
  baseUrl: "https://repset.onezy.in"
  timeout: "30s"
  retryAttempts: 3
  retryDelay: "5s"
```

## Authentication

All API requests require HMAC-SHA256 authentication with the following headers:

- `X-Device-ID`: Your device identifier
- `X-Signature`: HMAC-SHA256 signature of request
- `X-Timestamp`: Unix timestamp when request was signed

### HMAC Signature Generation

The signature is calculated as: `HMAC-SHA256(body + timestamp + deviceId, deviceKey)`

```bash
# Example authenticated request
curl -X POST https://repset.onezy.in/api/v1/checkin \
  -H "X-Device-ID: gym_12345_bridge_001" \
  -H "X-Signature: a1b2c3d4e5f6g7h8..." \
  -H "X-Timestamp: 1640995200" \
  -H "Content-Type: application/json" \
  -d '{"events": [{"eventId": "evt_123", "externalUserId": "user123", "timestamp": "2024-01-15T10:30:00Z", "eventType": "check_in", "deviceId": "gym_12345_bridge_001"}]}'
```

### Clock Skew Tolerance

Requests must be signed within 5 minutes of the current server time to prevent replay attacks.

## Testing

### Health Check
```bash
curl https://repset.onezy.in/api/v1/health
```

### Device Connectivity
```bash
# Test DNS resolution
nslookup repset.onezy.in

# Test HTTPS connectivity
curl -I https://repset.onezy.in

# Test API endpoint
curl https://repset.onezy.in/api/v1/health
```

## Troubleshooting

### Common Issues

1. **DNS Resolution**
   - Ensure `repset.onezy.in` resolves correctly
   - Check firewall/proxy settings

2. **SSL/TLS Issues**
   - Verify certificate chain
   - Check system time synchronization

3. **Authentication Failures**
   - Verify device ID and key
   - Check HMAC signature generation
   - Ensure system time is synchronized

### Support

For platform-specific issues:
- **Email**: support@repset.onezy.in
- **Documentation**: https://docs.repset.onezy.in
- **Status Page**: https://status.repset.onezy.in

## Bridge Commands

### Basic Commands
```bash
# Check bridge status and connectivity
gym-door-bridge status

# Pair with platform using pair code
gym-door-bridge pair ABC1-DEF2-GHI3

# Unpair from platform
gym-door-bridge unpair

# Manually trigger heartbeat
gym-door-bridge trigger-heartbeat

# Check device status with platform
gym-door-bridge device-status
```

### Service Management (Windows)
```bash
# Install bridge as Windows service
gym-door-bridge install

# Uninstall bridge service
gym-door-bridge uninstall

# Start/stop service
net start GymDoorBridge
net stop GymDoorBridge

# Check service status
sc query GymDoorBridge
```

## Integration Checklist

- [ ] Configure API base URL: `https://repset.onezy.in`
- [ ] Test network connectivity with `gym-door-bridge status`
- [ ] Pair device and obtain credentials with `gym-door-bridge pair`
- [ ] Test authentication with `gym-door-bridge trigger-heartbeat`
- [ ] Configure event submission
- [ ] Test fingerprint enrollment (if applicable)
- [ ] Set up monitoring and alerting
- [ ] Configure automatic updates

## Migration from Development

When moving from development to production:

1. Update `config.yaml`:
   ```yaml
   api:
     baseUrl: "https://repset.onezy.in"  # Change from localhost
   ```

2. Re-pair device with production platform
3. Update any hardcoded URLs in custom code
4. Test all functionality with production platform
5. Monitor logs for any connectivity issues

## Security Considerations

- Always use HTTPS for production
- Implement proper certificate validation
- Use secure storage for device credentials
- Enable automatic key rotation
- Monitor for authentication failures
- Implement rate limiting if needed

For detailed integration guides, see [docs/development/fingerprint-integration.md](development/fingerprint-integration.md).