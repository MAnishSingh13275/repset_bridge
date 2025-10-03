# Security Guide

## Overview

This document outlines security best practices for deploying and operating the Gym Door Access Bridge.

## Configuration Security

### Credential Management

**❌ Never do this:**
```yaml
# DON'T: Use default credentials
username: "admin"
password: "admin"
```

**✅ Do this instead:**
```yaml
# DO: Use strong, unique credentials
username: "bridge_user_2024"
password: "StrongP@ssw0rd!2024"
```

### Device Passwords

- **ZKTeco devices**: Change default communication password from "0"
- **ESSL devices**: Replace default admin/admin credentials
- **Realtime devices**: Verify device-specific authentication

### API Security

For production deployments, enable API security:

```yaml
api_server:
  auth:
    enabled: true
    hmac_secret: "your-32-character-secret-key-here"
    jwt_secret: "your-jwt-signing-secret-here"
    api_keys: 
      - "api-key-for-client-1"
      - "api-key-for-client-2"
  
  tls_enabled: true
  tls_cert_file: "/path/to/cert.pem"
  tls_key_file: "/path/to/key.pem"
```

## File Security

### Database Protection

- Database files are automatically excluded from version control
- Ensure proper file permissions on database files:
  ```bash
  chmod 600 data/bridge.db
  ```

### Configuration Files

- Never commit `config.yaml` with real credentials
- Use environment variables for sensitive values in production
- Regularly rotate API keys and passwords

### Log Security

- Log files may contain sensitive information
- Implement log rotation and secure deletion
- Monitor log access and implement appropriate permissions

## Network Security

### Firewall Configuration

Open only necessary ports:
- Bridge API: 8081 (configurable)
- Device connections: Device-specific ports
- Repset API: HTTPS outbound (443)

### TLS Configuration

Always use TLS in production:
```yaml
api_server:
  tls_enabled: true
  tls_cert_file: "/etc/ssl/certs/bridge.crt"
  tls_key_file: "/etc/ssl/private/bridge.key"
```

## Monitoring and Auditing

### Security Monitoring

- Monitor failed authentication attempts
- Log all configuration changes
- Track unusual device access patterns
- Monitor API usage and rate limiting

### Regular Security Checks

Run the security check script regularly:
```bash
./scripts/security-check.ps1 -ConfigPath config.yaml
```

## Incident Response

### Suspected Compromise

1. **Immediate Actions:**
   - Disable affected API keys
   - Rotate all credentials
   - Review access logs
   - Isolate affected systems

2. **Investigation:**
   - Check log files for unauthorized access
   - Verify device configurations
   - Review network traffic
   - Document findings

3. **Recovery:**
   - Update all credentials
   - Patch any vulnerabilities
   - Restore from clean backups if needed
   - Update security policies

## Security Checklist

### Pre-Deployment

- [ ] All default credentials changed
- [ ] TLS enabled for API server
- [ ] Authentication enabled
- [ ] Firewall rules configured
- [ ] Log rotation configured
- [ ] Security monitoring enabled

### Regular Maintenance

- [ ] Credentials rotated (quarterly)
- [ ] Security logs reviewed (weekly)
- [ ] Software updates applied
- [ ] Configuration audited
- [ ] Backup integrity verified

### Post-Incident

- [ ] All credentials rotated
- [ ] Logs analyzed
- [ ] Vulnerabilities patched
- [ ] Monitoring enhanced
- [ ] Documentation updated

## Compliance Considerations

### Data Protection

- Personal biometric data is processed locally
- No biometric templates stored in cloud
- Event logs contain minimal personal information
- Implement data retention policies

### Access Control

- Principle of least privilege
- Regular access reviews
- Strong authentication requirements
- Audit trail maintenance

## Contact

For security issues or questions:
- Review this documentation
- Check troubleshooting guide
- Contact system administrator
- Report security vulnerabilities responsibly