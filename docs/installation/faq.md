# RepSet Bridge Installation FAQ

## General Questions

### Q: What is the RepSet Bridge?
**A:** The RepSet Bridge is a Windows service that connects biometric devices (fingerprint scanners, RFID readers) and access control hardware to the RepSet cloud platform. It enables real-time member check-ins and access control for gyms and fitness facilities.

### Q: Do I need technical expertise to install the bridge?
**A:** No. The automated installation system is designed for non-technical users. You only need to run a single PowerShell command as an administrator. The installer handles all technical configuration automatically.

### Q: How long does installation take?
**A:** Typically 2-5 minutes, depending on your internet connection speed and system performance. The installer downloads the bridge executable, configures the service, and tests connectivity automatically.

### Q: Can I install multiple bridges on the same computer?
**A:** No. Each computer can only run one bridge instance. However, a single bridge can connect to multiple biometric devices on the same network.

## System Requirements

### Q: What Windows versions are supported?
**A:** 
- **Supported:** Windows 10 (1809+), Windows 11, Windows Server 2019/2022
- **Not Supported:** Windows 7, Windows 8, Windows Server 2012/2016 (without updates)

### Q: Do I need administrator privileges?
**A:** Yes. Administrator privileges are required to:
- Install the Windows service
- Create directories in Program Files
- Configure Windows Firewall rules
- Modify system registry entries

### Q: What if I don't have .NET Framework installed?
**A:** The installer will detect missing .NET Framework and attempt to install it automatically. If automatic installation fails, you'll receive instructions to download and install .NET Framework 4.7.2 manually.

### Q: How much disk space is required?
**A:** Minimum 500 MB, recommended 2 GB. This includes:
- Bridge executable (~50 MB)
- Configuration files (~1 MB)
- Log files (grows over time, auto-rotated)
- Local database for offline operation (~100 MB)

## Installation Process

### Q: What does the installation command do?
**A:** The installation command:
1. Downloads the latest bridge executable from GitHub
2. Creates installation directory structure
3. Generates configuration file with your gym's credentials
4. Installs and starts the Windows service
5. Tests connectivity to your biometric devices
6. Verifies connection to the RepSet platform

### Q: Can I customize the installation directory?
**A:** By default, the bridge installs to `C:\Program Files\RepSet\Bridge\`. This location is recommended for security and compatibility. Custom locations are not officially supported.

### Q: What happens if installation fails?
**A:** The installer includes automatic rollback functionality. If installation fails:
- Partial installations are automatically cleaned up
- Error messages provide specific troubleshooting steps
- Log files are created for technical support
- You can safely retry installation after resolving issues

### Q: Can I run the installer multiple times?
**A:** Yes. Running the installer on an existing installation will:
- Update the bridge to the latest version
- Preserve existing configuration settings
- Restart the service with new version
- Maintain device connections and data

## Network and Connectivity

### Q: What network ports does the bridge use?
**A:** 
- **Outbound HTTPS (443):** Communication with RepSet platform
- **Inbound/Outbound TCP:** Communication with biometric devices (ports vary by device type)
- **Common device ports:** 4370 (ZKTeco), 80/8080 (ESSL), 5005/9999 (Realtime)

### Q: Do I need to configure my firewall?
**A:** The installer automatically configures Windows Firewall rules. For corporate firewalls, you may need to:
- Allow outbound HTTPS to *.repset.com and github.com
- Allow communication with biometric device IP addresses
- Configure proxy settings if required

### Q: Can the bridge work behind a corporate proxy?
**A:** Yes. The installer can detect and use system proxy settings automatically. For manual configuration:
```powershell
# Set proxy credentials
[System.Net.WebRequest]::DefaultWebProxy.Credentials = [System.Net.CredentialCache]::DefaultCredentials
```

### Q: What if my internet connection is unstable?
**A:** The bridge includes offline operation capabilities:
- Stores access events locally when offline
- Automatically syncs data when connection is restored
- Continues to operate with cached member data
- Queues up to 10,000 events for later synchronization

## Device Compatibility

### Q: What biometric devices are supported?
**A:** The bridge supports most network-connected biometric devices:
- **ZKTeco:** K40, K50, F18, F19, F22 series and compatible models
- **ESSL:** X990, Biomax, K21 series and compatible models  
- **Realtime:** T502, T503, RS10 series and compatible models
- **Generic:** Wiegand 26/34 devices with serial adapters

### Q: How does device auto-discovery work?
**A:** During installation, the bridge:
1. Scans your local network for biometric devices
2. Tests common ports for each device type
3. Attempts to identify device models and capabilities
4. Automatically generates configuration for discovered devices
5. Tests connectivity and adds working devices to the configuration

### Q: What if my devices aren't auto-discovered?
**A:** You can manually add devices by editing the configuration file:
```yaml
enabled_adapters:
  - my_device

adapter_configs:
  my_device:
    device_type: zkteco  # or essl, realtime
    connection: tcp
    device_config:
      ip: "192.168.1.100"
      port: "4370"
```

### Q: Can I connect devices on different network segments?
**A:** Yes, but devices must be reachable from the bridge computer. You may need to:
- Configure routing between network segments
- Adjust firewall rules for cross-segment communication
- Use static routes or VLAN configuration

## Service Management

### Q: How do I check if the bridge service is running?
**A:** Use these commands in PowerShell (as Administrator):
```powershell
# Check service status
Get-Service -Name "RepSetBridge"

# View detailed service information
Get-Service -Name "RepSetBridge" | Select-Object Name, Status, StartType

# Check service in Services.msc
services.msc
```

### Q: How do I restart the bridge service?
**A:** 
```powershell
# Restart service
Restart-Service -Name "RepSetBridge"

# Or stop and start separately
Stop-Service -Name "RepSetBridge"
Start-Service -Name "RepSetBridge"
```

### Q: What if the service won't start?
**A:** Common solutions:
1. **Check Event Viewer:** Look for error messages in Windows Event Log
2. **Verify configuration:** Ensure config.yaml is valid
3. **Test connectivity:** Verify network access to devices and platform
4. **Check permissions:** Ensure service has necessary privileges
5. **Reinstall service:** Run installation command again

### Q: Can I change the service startup type?
**A:** The service is configured for automatic startup by default. To change:
```powershell
# Set to manual startup
Set-Service -Name "RepSetBridge" -StartupType Manual

# Set back to automatic
Set-Service -Name "RepSetBridge" -StartupType Automatic
```

## Configuration and Pairing

### Q: How do I pair the bridge with my gym?
**A:** Pairing happens automatically during installation using the embedded pair code. For manual pairing:
```cmd
gym-door-bridge.exe pair --pair-code YOUR_PAIR_CODE
```

### Q: Where do I find my pair code?
**A:** 
1. Log into your RepSet admin dashboard
2. Navigate to Settings → Bridge Management
3. Click "Generate Installation Command"
4. The pair code is embedded in the generated command

### Q: What if pairing fails?
**A:** Common solutions:
- Verify internet connectivity
- Check that the pair code hasn't expired (24-hour limit)
- Ensure firewall allows HTTPS connections
- Try generating a new pair code

### Q: Can I change configuration after installation?
**A:** Yes. Edit the configuration file at `C:\Program Files\RepSet\Bridge\config.yaml` and restart the service:
```powershell
Restart-Service -Name "RepSetBridge"
```

## Troubleshooting

### Q: The installation command won't run - "execution of scripts is disabled"
**A:** This is a PowerShell execution policy restriction. Solutions:
```powershell
# Option 1: Temporary bypass
Set-ExecutionPolicy -ExecutionPolicy Bypass -Scope Process -Force

# Option 2: Set RemoteSigned policy
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser -Force
```

### Q: I get "Access Denied" errors during installation
**A:** You need administrator privileges:
1. Right-click PowerShell and select "Run as Administrator"
2. Click "Yes" on the UAC prompt
3. Re-run the installation command

### Q: The bridge downloads but won't start
**A:** This usually indicates missing .NET Framework:
1. Download .NET Framework 4.7.2 from Microsoft
2. Install and restart your computer
3. Re-run the bridge installation

### Q: Devices are discovered but don't sync data
**A:** Check device configuration:
- Verify device IP addresses are correct
- Ensure devices have real-time events enabled
- Check device communication passwords
- Test device connectivity manually

### Q: The bridge was working but stopped
**A:** Common causes and solutions:
1. **Service stopped:** Restart the service
2. **Network changes:** Verify device IP addresses
3. **Platform connectivity:** Check internet connection
4. **Device issues:** Restart biometric devices
5. **Configuration corruption:** Restore from backup or reinstall

## Security and Privacy

### Q: Is biometric data stored locally?
**A:** Yes, but in processed form:
- Biometric templates (mathematical representations) are stored locally
- Raw biometric images are never stored or transmitted
- Templates cannot be reverse-engineered to recreate biometric data
- Local storage enables offline operation

### Q: How is data transmitted to the platform?
**A:** All communication uses industry-standard security:
- HTTPS/TLS 1.2+ encryption for all data transmission
- HMAC-SHA256 signatures for message authentication
- No raw biometric data is ever transmitted
- Only access events and member IDs are sent to the platform

### Q: What data is logged?
**A:** The bridge logs:
- Service startup/shutdown events
- Device connection status
- Access events (member ID, timestamp, device)
- Error messages and diagnostics
- No biometric data or personal information

### Q: How do I ensure GDPR compliance?
**A:** The bridge is designed with privacy in mind:
- Biometric data is processed locally only
- Member data is encrypted in transit and at rest
- Access logs can be configured for retention periods
- Data can be deleted on member request
- Audit trails are maintained for compliance

## Updates and Maintenance

### Q: How do I update the bridge?
**A:** Updates are handled automatically:
- The bridge checks for updates daily
- Critical updates are applied automatically
- Major updates may require manual approval
- You can also run the installation command again to force an update

### Q: How do I backup the bridge configuration?
**A:** 
```powershell
# Backup configuration and database
Copy-Item "C:\Program Files\RepSet\Bridge\config.yaml" "C:\Backup\config.yaml.backup"
Copy-Item "C:\Program Files\RepSet\Bridge\bridge.db" "C:\Backup\bridge.db.backup"
```

### Q: What maintenance is required?
**A:** Minimal maintenance is needed:
- **Automatic:** Log rotation, database cleanup, updates
- **Monthly:** Check service status and review logs
- **Quarterly:** Verify device connectivity and test backup procedures
- **Annually:** Review security settings and update documentation

### Q: How do I monitor bridge health?
**A:** Several monitoring options:
- **Windows Services:** Check service status in services.msc
- **Event Viewer:** Review Application and System logs
- **Log Files:** Check bridge.log for operational status
- **Platform Dashboard:** View bridge status in RepSet admin panel

## Uninstallation

### Q: How do I uninstall the bridge?
**A:** 
```powershell
# Stop and remove service
Stop-Service -Name "RepSetBridge" -Force
sc.exe delete "RepSetBridge"

# Remove installation directory
Remove-Item -Path "C:\Program Files\RepSet" -Recurse -Force

# Clean registry entries (optional)
Remove-Item -Path "HKLM:\SOFTWARE\RepSet" -Recurse -Force
```

### Q: Will uninstalling delete my data?
**A:** Yes, uninstalling removes:
- Bridge executable and configuration
- Local database with member templates
- Log files and cached data
- Windows service registration

**Important:** Backup any data you want to preserve before uninstalling.

### Q: Can I reinstall after uninstalling?
**A:** Yes. After uninstalling, you can run the installation command again. You'll need:
- A new pair code from the admin dashboard
- To reconfigure any custom device settings
- To re-enroll member biometric data on devices

## Support and Resources

### Q: Where can I get help?
**A:** Multiple support channels are available:
- **Documentation:** Comprehensive guides and troubleshooting
- **Email Support:** bridge-support@repset.com
- **GitHub Issues:** Report bugs and feature requests
- **Community Forum:** User discussions and tips
- **Video Tutorials:** Step-by-step installation guides

### Q: What information should I include when requesting support?
**A:** Please provide:
- Exact error messages
- Windows version and system specifications
- Installation command used
- Log files from `C:\Program Files\RepSet\Bridge\logs\`
- Network configuration details
- Device types and models

### Q: Is phone support available?
**A:** Email support is the primary channel for technical issues. Phone support may be available for enterprise customers or critical production issues.

### Q: How quickly will I get a response?
**A:** Response times vary by support channel:
- **Critical Issues:** Within 4 hours
- **General Support:** Within 24 hours
- **Feature Requests:** Within 1 week
- **Community Forum:** Community-driven response times

---

*Last updated: $(Get-Date -Format 'yyyy-MM-dd')*
*Version: 1.0*