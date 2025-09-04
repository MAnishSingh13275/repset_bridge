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
- **Check-in**: `POST /api/v1/checkin`
- **Health Check**: `GET /api/v1/health`

### Fingerprint Management
- **Enroll Fingerprint**: `POST /api/v1/fingerprint/enroll`
- **List Users**: `GET /api/v1/fingerprint/users`
- **Delete Fingerprint**: `DELETE /api/v1/fingerprint/{id}`

### Device Management
- **Device Status**: `GET /api/v1/devices/{deviceId}/status`
- **Update Configuration**: `PUT /api/v1/devices/{deviceId}/config`

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

All API requests require HMAC authentication:

```bash
curl -X POST https://repset.onezy.in/api/v1/checkin \
  -H "X-Device-ID: your-device-id" \
  -H "X-Device-Key: your-device-key" \
  -H "Content-Type: application/json" \
  -d '{"eventType": "ENTRY", "userId": "user123"}'
```

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

## Integration Checklist

- [ ] Configure API base URL: `https://repset.onezy.in`
- [ ] Test network connectivity
- [ ] Pair device and obtain credentials
- [ ] Test authentication with health endpoint
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