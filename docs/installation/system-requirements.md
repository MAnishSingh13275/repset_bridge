# RepSet Bridge System Requirements

## Overview

This document outlines the system requirements for installing and running the RepSet Bridge. The bridge connects biometric devices and access control hardware to the RepSet platform.

## Minimum System Requirements

### Operating System
- **Windows 10** (version 1809 or later) - 64-bit
- **Windows 11** - 64-bit  
- **Windows Server 2019** or later
- **Windows Server 2016** (with latest updates)

### Hardware Requirements
- **CPU:** 1 GHz or faster processor
- **RAM:** 2 GB minimum, 4 GB recommended
- **Storage:** 500 MB available disk space
- **Network:** Ethernet or Wi-Fi connection

### Software Dependencies
- **.NET Framework 4.7.2** or later
- **PowerShell 5.1** or later (included with Windows 10/11)
- **Windows Management Framework 5.1** (for older systems)

### Network Requirements
- **Internet Connection:** Required for installation and updates
- **Outbound HTTPS (443):** Access to github.com and api.repset.com
- **Local Network Access:** For biometric device communication
- **Firewall:** Windows Firewall or third-party firewall configured

### User Privileges
- **Administrator Rights:** Required for installation and service management
- **Local System Account:** Service runs under Local System by default

## Recommended System Requirements

### Operating System
- **Windows 10 Pro/Enterprise** (latest version)
- **Windows 11 Pro/Enterprise**
- **Windows Server 2022**

### Hardware Requirements
- **CPU:** Multi-core processor (2+ cores)
- **RAM:** 8 GB or more
- **Storage:** 2 GB available disk space (for logs and data)
- **Network:** Gigabit Ethernet connection

### Additional Recommendations
- **UPS (Uninterruptible Power Supply):** For continuous operation
- **Dedicated Network:** Separate VLAN for biometric devices
- **Static IP Address:** For the bridge computer
- **Antivirus Exclusions:** Configure exclusions for bridge files

## Network Configuration

### Required Outbound Connections

| Destination | Port | Protocol | Purpose |
|-------------|------|----------|---------|
| github.com | 443 | HTTPS | Download bridge executable |
| api.repset.com | 443 | HTTPS | Platform communication |
| *.repset.com | 443 | HTTPS | API endpoints and updates |

### Local Network Requirements

| Device Type | Ports | Protocol | Notes |
|-------------|-------|----------|-------|
| ZKTeco Devices | 4370 | TCP | Default communication port |
| ESSL Devices | 80, 8080 | HTTP/TCP | Web-based communication |
| Realtime Devices | 5005, 9999 | TCP | Device-specific ports |
| Generic Devices | Various | TCP/UDP | Depends on manufacturer |

### Firewall Configuration

#### Windows Firewall Rules
```powershell
# Allow outbound HTTPS
New-NetFirewallRule -DisplayName "RepSet Bridge HTTPS" -Direction Outbound -Protocol TCP -RemotePort 443 -Action Allow

# Allow inbound connections from biometric devices (adjust IP range as needed)
New-NetFirewallRule -DisplayName "RepSet Bridge Devices" -Direction Inbound -Protocol TCP -LocalPort 4370,80,8080,5005,9999 -RemoteAddress 192.168.1.0/24 -Action Allow
```

#### Corporate Firewall
- Whitelist github.com and *.repset.com domains
- Allow HTTPS (443) outbound traffic
- Configure proxy settings if required

## Biometric Device Compatibility

### Supported Device Types

#### ZKTeco Devices
- **Models:** K40, K50, F18, F19, F22, ZK4500, and compatible models
- **Connection:** TCP/IP (Port 4370)
- **Protocol:** ZKTeco SDK protocol
- **Requirements:** Device firmware 6.60 or later

#### ESSL Devices  
- **Models:** X990, Biomax N-BM5, eSSL K21, and compatible models
- **Connection:** HTTP/TCP (Ports 80, 8080)
- **Protocol:** HTTP API
- **Requirements:** Web interface enabled

#### Realtime Devices
- **Models:** T502, T503, RS10, and compatible models
- **Connection:** TCP (Ports 5005, 9999)
- **Protocol:** Realtime SDK protocol
- **Requirements:** Network communication enabled

#### Generic Wiegand Devices
- **Connection:** Serial/USB converter required
- **Protocol:** Wiegand 26/34 bit
- **Requirements:** Compatible serial interface

### Device Network Configuration

#### IP Address Requirements
- Devices must be on the same network as the bridge computer
- Static IP addresses recommended for devices
- IP range: Typically 192.168.1.x or 10.0.0.x

#### Device Settings
- **Network Mode:** TCP/IP enabled
- **Communication Password:** Default or configured
- **Time Synchronization:** Enabled
- **Real-time Events:** Enabled for immediate data transfer

## Installation Environment

### Development/Testing Environment
- **OS:** Windows 10 Pro (minimum)
- **RAM:** 4 GB minimum
- **Network:** Local network with test devices
- **Internet:** Required for downloads and updates

### Production Environment
- **OS:** Windows Server 2019/2022 (recommended)
- **RAM:** 8 GB or more
- **Storage:** SSD recommended for better performance
- **Network:** Dedicated VLAN with UPS-backed network equipment
- **Backup:** Regular system and configuration backups

### High Availability Setup
- **Primary Server:** Main bridge installation
- **Secondary Server:** Backup bridge (manual failover)
- **Load Balancer:** Not required (single bridge per location)
- **Database:** Local SQLite (included)

## Security Requirements

### System Security
- **Windows Updates:** Keep system updated with latest security patches
- **Antivirus:** Compatible antivirus with bridge exclusions configured
- **User Accounts:** Dedicated service account (optional)
- **Audit Logging:** Windows Event Log integration

### Network Security
- **TLS 1.2+:** Required for platform communication
- **Certificate Validation:** Automatic certificate validation
- **Encrypted Storage:** Configuration files encrypted at rest
- **Access Control:** Restrict physical access to bridge computer

### Compliance Considerations
- **GDPR:** Biometric data processing compliance
- **HIPAA:** Healthcare facility requirements (if applicable)
- **SOC 2:** Security controls for service organizations
- **Local Regulations:** Country-specific data protection laws

## Performance Considerations

### Expected Load
- **Concurrent Users:** Up to 1000 members per location
- **Daily Transactions:** Up to 10,000 access events
- **Device Count:** Up to 20 biometric devices per bridge
- **Data Retention:** 90 days local storage (configurable)

### Performance Metrics
- **Response Time:** < 2 seconds for access verification
- **Throughput:** 100+ transactions per minute
- **Uptime:** 99.9% availability target
- **Recovery Time:** < 5 minutes after system restart

### Optimization Tips
- Use SSD storage for better I/O performance
- Ensure adequate RAM for caching
- Configure devices for optimal polling intervals
- Monitor network latency to devices

## Compatibility Matrix

### Windows Version Compatibility

| Windows Version | .NET 4.7.2 | PowerShell 5.1 | Bridge Support | Notes |
|----------------|-------------|-----------------|----------------|-------|
| Windows 10 1809+ | ✅ | ✅ | ✅ | Fully supported |
| Windows 11 | ✅ | ✅ | ✅ | Recommended |
| Server 2019 | ✅ | ✅ | ✅ | Production ready |
| Server 2022 | ✅ | ✅ | ✅ | Latest features |
| Server 2016 | ⚠️ | ⚠️ | ⚠️ | Requires updates |
| Windows 10 1803 | ❌ | ❌ | ❌ | Not supported |

### Device Compatibility

| Device Brand | Model Range | Firmware | Status | Notes |
|--------------|-------------|----------|--------|-------|
| ZKTeco | K40, K50 series | 6.60+ | ✅ | Fully tested |
| ZKTeco | F18, F19 series | 6.60+ | ✅ | Fully tested |
| ESSL | X990, Biomax | Latest | ✅ | HTTP API required |
| Realtime | T502, T503 | Latest | ✅ | TCP communication |
| Generic | Wiegand 26/34 | N/A | ⚠️ | Serial adapter needed |

## Pre-Installation Checklist

### System Preparation
- [ ] Verify Windows version and architecture (64-bit)
- [ ] Install Windows updates
- [ ] Verify .NET Framework 4.7.2 or later
- [ ] Confirm PowerShell 5.1 or later
- [ ] Ensure administrator privileges
- [ ] Configure antivirus exclusions

### Network Preparation
- [ ] Test internet connectivity
- [ ] Verify access to github.com and api.repset.com
- [ ] Configure firewall rules
- [ ] Document device IP addresses
- [ ] Test device network connectivity
- [ ] Configure proxy settings (if required)

### Device Preparation
- [ ] Configure device IP addresses
- [ ] Enable TCP/IP communication
- [ ] Set communication passwords
- [ ] Enable real-time events
- [ ] Test device web interfaces
- [ ] Document device credentials

### Security Preparation
- [ ] Review security policies
- [ ] Configure Windows Firewall
- [ ] Set up audit logging
- [ ] Plan backup procedures
- [ ] Review compliance requirements
- [ ] Document security configurations

## Post-Installation Verification

### System Verification
```powershell
# Verify service installation
Get-Service -Name "RepSetBridge"

# Check service startup type
Get-Service -Name "RepSetBridge" | Select-Object Name, Status, StartType

# Verify configuration file
Test-Path "C:\Program Files\RepSet\Bridge\config.yaml"

# Check log files
Get-ChildItem "C:\Program Files\RepSet\Bridge\logs\"
```

### Network Verification
```powershell
# Test platform connectivity
Test-NetConnection -ComputerName api.repset.com -Port 443

# Test device connectivity (example)
Test-NetConnection -ComputerName 192.168.1.100 -Port 4370
```

### Performance Verification
- Monitor CPU and memory usage
- Check response times for device communication
- Verify log file rotation
- Test automatic service restart

## Troubleshooting System Requirements

### Common Issues
1. **Insufficient .NET Framework version**
   - Download and install .NET Framework 4.7.2 or later
   - Restart system after installation

2. **PowerShell execution policy restrictions**
   - Set execution policy to RemoteSigned or Bypass
   - Run installation as administrator

3. **Network connectivity issues**
   - Check firewall settings
   - Verify proxy configuration
   - Test DNS resolution

4. **Device communication failures**
   - Verify device IP addresses
   - Check network connectivity
   - Confirm device settings

### Diagnostic Commands
```powershell
# System information
Get-ComputerInfo | Select-Object WindowsProductName, WindowsVersion, TotalPhysicalMemory

# .NET Framework version
Get-ChildItem "HKLM:SOFTWARE\Microsoft\NET Framework Setup\NDP\v4\Full\" | Get-ItemPropertyValue -Name Release

# PowerShell version
$PSVersionTable.PSVersion

# Network connectivity
Test-NetConnection -ComputerName api.repset.com -Port 443 -InformationLevel Detailed
```

## Support and Resources

### Documentation
- [Installation Guide](./README.md)
- [Troubleshooting Guide](./troubleshooting-guide.md)
- [FAQ](./faq.md)

### Technical Support
- **Email:** bridge-support@repset.com
- **Documentation:** https://docs.repset.com/bridge
- **GitHub Issues:** https://github.com/your-org/gym-door-bridge/issues

### Community Resources
- **User Forum:** https://community.repset.com
- **Video Tutorials:** https://docs.repset.com/bridge/videos
- **Best Practices:** https://docs.repset.com/bridge/best-practices

---

*Last updated: $(Get-Date -Format 'yyyy-MM-dd')*
*Version: 1.0*