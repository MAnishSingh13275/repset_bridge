# Troubleshooting Guide

This guide provides solutions to common issues encountered when deploying and operating the Gym Door Access Bridge.

## Table of Contents

- [Quick Diagnostics](#quick-diagnostics)
- [Installation Issues](#installation-issues)
- [Configuration Problems](#configuration-problems)
- [Network Connectivity](#network-connectivity)
- [Hardware Adapter Issues](#hardware-adapter-issues)
- [Authentication Problems](#authentication-problems)
- [Performance Issues](#performance-issues)
- [Database Problems](#database-problems)
- [Service Management](#service-management)
- [Update Issues](#update-issues)
- [Log Analysis](#log-analysis)
- [Emergency Procedures](#emergency-procedures)

## Quick Diagnostics

### Health Check

```bash
# Basic health check
curl http://localhost:8080/health

# Detailed status
./gym-door-bridge status --verbose

# System diagnostics
./gym-door-bridge diagnostics --output diagnostics.json
```

### Common Status Codes

| Status | Meaning | Action |
|--------|---------|--------|
| `healthy` | All systems operational | None required |
| `degraded` | Some features unavailable | Check logs for warnings |
| `unhealthy` | Critical issues present | Immediate attention required |
| `offline` | No network connectivity | Check network configuration |

## Installation Issues

### Windows Installation Problems

#### Issue: "Access Denied" during installation

**Symptoms:**
- Installation script fails with permission errors
- Service registration fails

**Solution:**
```powershell
# Run PowerShell as Administrator
Start-Process powershell -Verb runAs

# Then run installation
iwr https://cdn.yourdomain.com/install.ps1 | iex; Install-Bridge -PairCode ABC123
```

#### Issue: "Windows Service failed to start"

**Symptoms:**
- Service installs but won't start
- Event log shows startup errors

**Diagnosis:**
```powershell
# Check service status
sc query "GymDoorBridge"

# View service configuration
sc qc "GymDoorBridge"

# Check event logs
Get-EventLog -LogName Application -Source "Gym Door Bridge" -Newest 10
```

**Solution:**
```powershell
# Verify service account permissions
# Check if service account has "Log on as a service" right

# Reinstall with correct parameters
.\gym-door-bridge.exe service uninstall
.\gym-door-bridge.exe service install --config "C:\Program Files\GymDoorBridge\config.yaml"
```

### macOS Installation Problems

#### Issue: "Operation not permitted" on macOS

**Symptoms:**
- Installation fails with permission errors
- Daemon won't start

**Solution:**
```bash
# Grant Full Disk Access to Terminal in System Preferences > Security & Privacy
# Or run with sudo
sudo curl -sSL https://cdn.yourdomain.com/install.sh | bash -s -- --pair-code ABC123
```

#### Issue: "Developer cannot be verified" warning

**Symptoms:**
- macOS blocks execution of unsigned binary
- Gatekeeper prevents running

**Solution:**
```bash
# Allow the application in System Preferences > Security & Privacy
# Or bypass Gatekeeper temporarily
sudo spctl --master-disable

# Run the application
./gym-door-bridge

# Re-enable Gatekeeper
sudo spctl --master-enable
```

### Linux/Docker Installation Problems

#### Issue: Docker container won't start

**Symptoms:**
- Container exits immediately
- Health check fails

**Diagnosis:**
```bash
# Check container logs
docker logs gym-door-bridge

# Inspect container
docker inspect gym-door-bridge

# Check resource usage
docker stats gym-door-bridge
```

**Solution:**
```bash
# Verify configuration file
docker run --rm -v $(pwd)/config.yaml:/app/config/config.yaml gym-door-bridge --validate

# Check file permissions
ls -la config.yaml

# Run with debug logging
docker run -e LOG_LEVEL=debug gym-door-bridge
```

## Configuration Problems

### Invalid Configuration File

#### Issue: YAML syntax errors

**Symptoms:**
- Application fails to start
- "Invalid configuration" error messages

**Diagnosis:**
```bash
# Validate YAML syntax
python -c "import yaml; yaml.safe_load(open('config.yaml'))"

# Or use online YAML validator
# Check indentation and special characters
```

**Solution:**
```bash
# Use configuration template
cp config.yaml.example config.yaml

# Validate configuration
./gym-door-bridge --config config.yaml --validate
```

### Missing Required Settings

#### Issue: Required configuration values missing

**Symptoms:**
- "Configuration validation failed" errors
- Specific missing field errors

**Common Missing Fields:**
```yaml
# Required fields that are often missing
device:
  id: ""          # Must be set after pairing
  key: ""         # Must be set after pairing

api:
  baseUrl: "https://api.yourdomain.com"  # Required
  timeout: "30s"                         # Recommended

database:
  path: "data/bridge.db"  # Required
```

**Solution:**
```bash
# Generate default configuration
./gym-door-bridge config generate --output config.yaml

# Pair device to get credentials
./gym-door-bridge pair --code ABC123
```

## Network Connectivity

### API Connection Issues

#### Issue: Cannot connect to SaaS platform

**Symptoms:**
- "Connection refused" errors
- Timeout errors
- SSL/TLS handshake failures

**Diagnosis:**
```bash
# Test basic connectivity
curl -v https://api.yourdomain.com/api/v1/health

# Check DNS resolution
nslookup api.yourdomain.com

# Test with specific timeout
curl --connect-timeout 10 --max-time 30 https://api.yourdomain.com/api/v1/health

# Check SSL certificate
openssl s_client -connect api.yourdomain.com:443 -servername api.yourdomain.com
```

**Solutions:**

**Firewall Issues:**
```bash
# Windows - Allow through firewall
netsh advfirewall firewall add rule name="Gym Door Bridge" dir=out action=allow protocol=TCP remoteport=443

# Linux - Check iptables
sudo iptables -L OUTPUT -v -n | grep 443
```

**Proxy Configuration:**
```yaml
# config.yaml
network:
  proxy:
    http: "http://proxy.company.com:8080"
    https: "http://proxy.company.com:8080"
    noProxy: ["localhost", "127.0.0.1"]
```

**DNS Issues:**
```bash
# Use alternative DNS servers
# Windows
netsh interface ip set dns "Local Area Connection" static 8.8.8.8
netsh interface ip add dns "Local Area Connection" 8.8.4.4 index=2

# Linux
echo "nameserver 8.8.8.8" | sudo tee /etc/resolv.conf
```

### Offline Mode Issues

#### Issue: Bridge not switching to offline mode

**Symptoms:**
- Events are lost during network outages
- No local queuing occurring

**Diagnosis:**
```bash
# Check offline queue status
./gym-door-bridge queue status

# Verify database connectivity
sqlite3 data/bridge.db ".tables"
```

**Solution:**
```yaml
# config.yaml - Ensure offline mode is enabled
queue:
  enabled: true
  maxSize: 10000
  retryInterval: "30s"
  offlineMode: true
```

## Hardware Adapter Issues

### Simulator Adapter Problems

#### Issue: Simulator not generating events

**Symptoms:**
- No test events appearing
- Simulator shows as inactive

**Solution:**
```bash
# Enable simulator in configuration
./gym-door-bridge adapters enable simulator

# Test simulator manually
./gym-door-bridge adapters test simulator --event entry --user test123

# Check simulator configuration
./gym-door-bridge adapters status simulator
```

### Real Hardware Adapter Issues

#### Issue: Fingerprint reader not responding

**Symptoms:**
- Adapter shows as "error" status
- No events from hardware

**Diagnosis:**
```bash
# Check adapter status
./gym-door-bridge adapters status fingerprint

# Test hardware connectivity
./gym-door-bridge adapters test fingerprint --diagnostic

# Check hardware logs
./gym-door-bridge logs --adapter fingerprint --level debug
```

**Solutions:**

**USB Connection Issues:**
```bash
# Windows - Check Device Manager
# Look for unknown devices or error indicators

# Linux - Check USB devices
lsusb
dmesg | grep -i usb

# macOS - System Information > USB
system_profiler SPUSBDataType
```

**Driver Issues:**
```bash
# Windows - Update drivers through Device Manager
# Linux - Install appropriate kernel modules
# macOS - Check for vendor-specific drivers
```

**Permission Issues:**
```bash
# Linux - Add user to appropriate groups
sudo usermod -a -G dialout,plugdev $USER

# Set udev rules for hardware access
sudo tee /etc/udev/rules.d/99-gym-hardware.rules << EOF
SUBSYSTEM=="usb", ATTR{idVendor}=="1234", ATTR{idProduct}=="5678", MODE="0666"
EOF
sudo udevadm control --reload-rules
```

## Authentication Problems

### Device Pairing Issues

#### Issue: Pairing fails with invalid code

**Symptoms:**
- "Invalid pair code" error
- Pairing process doesn't complete

**Solution:**
```bash
# Verify pair code is correct and not expired
# Pair codes typically expire after 24 hours

# Generate new pair code in admin portal
# Retry pairing with new code
./gym-door-bridge pair --code NEW123
```

#### Issue: Device already paired error

**Symptoms:**
- "Device already exists" error during pairing
- Cannot re-pair device

**Solution:**
```bash
# Unpair device first
./gym-door-bridge unpair

# Or reset device identity
./gym-door-bridge reset --confirm

# Then pair again
./gym-door-bridge pair --code ABC123
```

### HMAC Authentication Failures

#### Issue: "Authentication failed" errors

**Symptoms:**
- API requests rejected with 401 errors
- "Invalid signature" messages

**Diagnosis:**
```bash
# Check system time synchronization
# Windows
w32tm /query /status

# macOS/Linux
timedatectl status

# Test HMAC generation
./gym-door-bridge auth test --debug
```

**Solutions:**

**Time Synchronization:**
```bash
# Windows - Sync time
w32tm /resync

# Linux - Install and configure NTP
sudo apt-get install ntp
sudo systemctl enable ntp
sudo systemctl start ntp

# macOS - Enable automatic time sync
sudo sntp -sS time.apple.com
```

**Key Rotation:**
```bash
# Force key rotation
./gym-door-bridge auth rotate

# Verify new key is working
./gym-door-bridge auth test
```

## Performance Issues

### High Memory Usage

#### Issue: Bridge consuming excessive memory

**Symptoms:**
- Memory usage continuously growing
- System becomes unresponsive

**Diagnosis:**
```bash
# Monitor memory usage
# Windows
tasklist /fi "imagename eq gym-door-bridge.exe"

# Linux/macOS
ps aux | grep gym-door-bridge
top -p $(pgrep gym-door-bridge)

# Check queue size
./gym-door-bridge queue status
```

**Solutions:**

**Reduce Queue Size:**
```yaml
# config.yaml
queue:
  maxSize: 5000  # Reduce from default
  batchSize: 50  # Smaller batches
```

**Enable Memory Profiling:**
```bash
# Generate memory profile
./gym-door-bridge --profile-memory memory.prof

# Analyze profile
go tool pprof memory.prof
```

### High CPU Usage

#### Issue: Bridge using excessive CPU

**Symptoms:**
- High CPU usage even when idle
- System performance degraded

**Diagnosis:**
```bash
# Profile CPU usage
./gym-door-bridge --profile-cpu cpu.prof

# Check for busy loops in logs
./gym-door-bridge logs --level debug | grep -i "loop\|retry\|poll"
```

**Solutions:**

**Adjust Polling Intervals:**
```yaml
# config.yaml
adapters:
  fingerprint:
    pollInterval: "1s"  # Increase from default 100ms

health:
  checkInterval: "60s"  # Increase from default 30s
```

**Reduce Log Level:**
```yaml
# config.yaml
logging:
  level: "info"  # Change from "debug"
```

### Slow Event Processing

#### Issue: Events taking too long to process

**Symptoms:**
- Delays between hardware scan and cloud submission
- Queue backing up

**Diagnosis:**
```bash
# Check processing metrics
./gym-door-bridge metrics --format json | jq '.processing'

# Monitor queue depth over time
watch -n 5 './gym-door-bridge queue status'
```

**Solutions:**

**Increase Batch Size:**
```yaml
# config.yaml
queue:
  batchSize: 200  # Increase from default 100
  maxConcurrency: 5  # Allow more concurrent uploads
```

**Optimize Database:**
```bash
# Vacuum database
sqlite3 data/bridge.db "VACUUM;"

# Analyze query performance
sqlite3 data/bridge.db ".timer on" "SELECT COUNT(*) FROM event_queue;"
```

## Database Problems

### Database Corruption

#### Issue: SQLite database corrupted

**Symptoms:**
- "Database disk image is malformed" errors
- Application crashes on startup

**Diagnosis:**
```bash
# Check database integrity
sqlite3 data/bridge.db "PRAGMA integrity_check;"

# Check for locks
lsof data/bridge.db  # Linux/macOS
handle data/bridge.db  # Windows (with handle.exe)
```

**Solutions:**

**Repair Database:**
```bash
# Backup corrupted database
cp data/bridge.db data/bridge.db.corrupted

# Attempt repair
sqlite3 data/bridge.db ".recover" | sqlite3 data/bridge.db.recovered

# Replace with recovered database
mv data/bridge.db.recovered data/bridge.db
```

**Restore from Backup:**
```bash
# If automatic backups are enabled
cp data/backups/bridge.db.$(date +%Y%m%d) data/bridge.db

# Restart service
./gym-door-bridge service restart
```

### Database Locking Issues

#### Issue: "Database is locked" errors

**Symptoms:**
- Cannot write to database
- Application hangs on database operations

**Diagnosis:**
```bash
# Check for multiple processes
ps aux | grep gym-door-bridge

# Check file locks
lsof data/bridge.db
```

**Solution:**
```bash
# Stop all instances
./gym-door-bridge service stop
pkill gym-door-bridge

# Remove lock files
rm -f data/bridge.db-wal data/bridge.db-shm

# Restart service
./gym-door-bridge service start
```

## Service Management

### Windows Service Issues

#### Issue: Service won't start automatically

**Symptoms:**
- Service starts manually but not on boot
- Delayed start issues

**Solution:**
```powershell
# Set service to automatic start
sc config "GymDoorBridge" start= auto

# Set delayed start
sc config "GymDoorBridge" start= delayed-auto

# Check service dependencies
sc qc "GymDoorBridge"
```

#### Issue: Service crashes on startup

**Symptoms:**
- Service starts then immediately stops
- Event log shows crash information

**Diagnosis:**
```powershell
# Check event logs
Get-EventLog -LogName Application -Source "Gym Door Bridge" -Newest 10

# Run service in console mode for debugging
.\gym-door-bridge.exe --debug
```

### macOS Daemon Issues

#### Issue: Daemon not loading on boot

**Symptoms:**
- Daemon works when started manually
- Not running after system restart

**Diagnosis:**
```bash
# Check launchd status
sudo launchctl list | grep gym-door-bridge

# Check plist file
plutil -lint /Library/LaunchDaemons/com.yourdomain.gym-door-bridge.plist
```

**Solution:**
```bash
# Reload daemon
sudo launchctl unload /Library/LaunchDaemons/com.yourdomain.gym-door-bridge.plist
sudo launchctl load /Library/LaunchDaemons/com.yourdomain.gym-door-bridge.plist

# Enable daemon
sudo launchctl enable system/com.yourdomain.gym-door-bridge
```

## Update Issues

### Automatic Update Failures

#### Issue: Updates fail to download

**Symptoms:**
- "Update check failed" messages
- Running old version despite updates available

**Diagnosis:**
```bash
# Check update configuration
./gym-door-bridge config show | grep -i update

# Test manifest download
curl -v https://cdn.yourdomain.com/gym-door-bridge/manifest.json

# Check update logs
./gym-door-bridge logs --component updater
```

**Solutions:**

**Network Issues:**
```yaml
# config.yaml - Configure update proxy
updater:
  proxy: "http://proxy.company.com:8080"
  timeout: "60s"
```

**Certificate Issues:**
```bash
# Update CA certificates
# Windows - Windows Update
# macOS - Software Update
# Linux
sudo apt-get update && sudo apt-get install ca-certificates
```

#### Issue: Update downloads but fails to install

**Symptoms:**
- Update downloaded successfully
- Installation fails with permission errors

**Solution:**
```bash
# Windows - Ensure service has sufficient privileges
# Check service account permissions

# macOS/Linux - Check file permissions
ls -la /usr/local/bin/gym-door-bridge
sudo chown root:root /usr/local/bin/gym-door-bridge
sudo chmod 755 /usr/local/bin/gym-door-bridge
```

### Manual Update Process

#### When automatic updates fail

```bash
# Stop service
./gym-door-bridge service stop

# Backup current version
cp gym-door-bridge gym-door-bridge.backup

# Download new version
curl -L -o gym-door-bridge-new https://cdn.yourdomain.com/gym-door-bridge/gym-door-bridge-$(uname -s)-$(uname -m)

# Verify download
sha256sum gym-door-bridge-new
# Compare with manifest.json

# Replace binary
mv gym-door-bridge-new gym-door-bridge
chmod +x gym-door-bridge

# Start service
./gym-door-bridge service start

# Verify update
./gym-door-bridge version
```

## Log Analysis

### Understanding Log Levels

| Level | Purpose | When to Use |
|-------|---------|-------------|
| `DEBUG` | Detailed execution info | Development, troubleshooting |
| `INFO` | General operational info | Normal operation |
| `WARN` | Potential issues | Monitoring, alerts |
| `ERROR` | Error conditions | Immediate attention |
| `FATAL` | Critical failures | Emergency response |

### Common Log Patterns

#### Successful Operation
```
INFO[2024-01-01T10:00:00Z] Bridge starting up version=1.2.3
INFO[2024-01-01T10:00:01Z] Configuration loaded config_path=config.yaml
INFO[2024-01-01T10:00:02Z] Database initialized path=data/bridge.db
INFO[2024-01-01T10:00:03Z] Hardware adapter started adapter=simulator
INFO[2024-01-01T10:00:04Z] Event processor ready
INFO[2024-01-01T10:00:05Z] Health monitor active
```

#### Network Connectivity Issues
```
WARN[2024-01-01T10:00:00Z] API request failed error="connection timeout" retry_in=30s
INFO[2024-01-01T10:00:01Z] Switching to offline mode
INFO[2024-01-01T10:00:02Z] Event queued locally event_id=evt_123 queue_depth=1
WARN[2024-01-01T10:05:00Z] Retry attempt failed attempt=3 max_retries=5
INFO[2024-01-01T10:10:00Z] Network connectivity restored
INFO[2024-01-01T10:10:01Z] Replaying queued events count=5
```

#### Authentication Problems
```
ERROR[2024-01-01T10:00:00Z] HMAC validation failed request_id=req_123
WARN[2024-01-01T10:00:01Z] Clock skew detected local_time=... server_time=...
INFO[2024-01-01T10:00:02Z] Initiating key rotation
INFO[2024-01-01T10:00:03Z] Key rotation completed new_key_id=key_456
```

### Log Collection Commands

```bash
# Collect recent logs
./gym-door-bridge logs --since "1 hour ago" --output recent.log

# Filter by component
./gym-door-bridge logs --component adapter --level error

# Export structured logs
./gym-door-bridge logs --format json --since "24 hours ago" > logs.json

# Continuous monitoring
./gym-door-bridge logs --follow --level warn
```

## Emergency Procedures

### Complete System Failure

#### Immediate Actions

1. **Stop the service**
   ```bash
   ./gym-door-bridge service stop
   ```

2. **Backup current state**
   ```bash
   mkdir emergency-backup-$(date +%Y%m%d-%H%M%S)
   cp -r data/ config.yaml logs/ emergency-backup-*/
   ```

3. **Check system resources**
   ```bash
   df -h  # Disk space
   free -h  # Memory
   top  # CPU usage
   ```

4. **Review recent logs**
   ```bash
   tail -100 logs/bridge.log
   ```

#### Recovery Steps

1. **Reset to known good state**
   ```bash
   # Restore from backup
   cp backup/config.yaml ./
   cp backup/data/bridge.db data/
   ```

2. **Clear temporary files**
   ```bash
   rm -f data/bridge.db-wal data/bridge.db-shm
   rm -f logs/*.tmp
   ```

3. **Restart with minimal configuration**
   ```bash
   ./gym-door-bridge --config config.yaml --log-level debug
   ```

4. **Verify operation**
   ```bash
   curl http://localhost:8080/health
   ./gym-door-bridge status
   ```

### Data Recovery

#### Queue Recovery

```bash
# Export queued events
sqlite3 data/bridge.db "SELECT * FROM event_queue WHERE sent_at IS NULL;" > unsent_events.csv

# Manual event submission (if needed)
./gym-door-bridge queue replay --file unsent_events.csv
```

#### Configuration Recovery

```bash
# Generate default configuration
./gym-door-bridge config generate --output config-default.yaml

# Merge with backup settings
# Edit config-default.yaml to include custom settings

# Validate merged configuration
./gym-door-bridge --config config-default.yaml --validate
```

### Escalation Procedures

#### When to Escalate

- Database corruption cannot be repaired
- Security breach suspected (repeated HMAC failures)
- Hardware adapter completely unresponsive
- System resources exhausted
- Data loss detected

#### Information to Collect

```bash
# System information
./gym-door-bridge diagnostics collect --output diagnostics-$(date +%Y%m%d-%H%M%S).zip

# Include:
# - Configuration files (sanitized)
# - Recent logs
# - System resource usage
# - Network connectivity tests
# - Database schema and statistics
```

#### Contact Information

- **Technical Support**: support@yourdomain.com
- **Emergency Hotline**: +1-800-XXX-XXXX
- **Documentation**: https://docs.yourdomain.com/gym-door-bridge
- **Status Page**: https://status.yourdomain.com

### Prevention Measures

#### Regular Maintenance

```bash
# Weekly tasks
./gym-door-bridge maintenance weekly

# Monthly tasks
./gym-door-bridge maintenance monthly

# Includes:
# - Database optimization
# - Log rotation
# - Configuration validation
# - Update checks
# - Health monitoring
```

#### Monitoring Setup

```yaml
# config.yaml - Enable comprehensive monitoring
monitoring:
  enabled: true
  alerts:
    email: admin@yourdomain.com
    webhook: https://hooks.slack.com/...
  thresholds:
    queueDepth: 1000
    memoryUsage: 80
    diskUsage: 90
    errorRate: 5
```

This troubleshooting guide should help resolve most common issues. For problems not covered here, please contact technical support with the diagnostic information collected using the procedures above.