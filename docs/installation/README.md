# RepSet Bridge Installation Guide

This guide walks you through installing the RepSet Bridge as a Windows service with automatic biometric device discovery and platform integration.

## 📖 Documentation Quick Links

- **🚀 [Quick Start](#-quick-start)** - Get started immediately
- **📋 [System Requirements](./system-requirements.md)** - Check compatibility
- **🔧 [Troubleshooting Guide](./troubleshooting-guide.md)** - Resolve issues
- **❓ [FAQ](./faq.md)** - Common questions
- **👥 [Gym Owner Guide](./gym-owner-guide.md)** - Non-technical guide
- **📚 [Documentation Index](./documentation-index.md)** - All documentation

## 🚀 Quick Start

### Option 1: Enhanced Web Installation (Recommended) ⭐

**One-Click Smart Install:**
```powershell
# Run PowerShell as Administrator, then:
iex (iwr -useb https://raw.githubusercontent.com/your-org/gym-door-bridge/main/public/install-bridge.ps1).Content
```

**Enhanced Features:**
- 🚀 Downloads latest version automatically
- 🔄 Smart pairing with auto-unpair capability
- ⚡ Handles "already paired" devices automatically
- 🛠️ No manual unpair commands needed
- 📞 Reduces administrator support requests

### Option 2: Local Installation

1. **Download or Build**

   ```bash
   # If you have Go installed:
   build.bat

   # Or download from GitHub releases
   ```

2. **Install as Service**
   ```powershell
   # Run PowerShell as Administrator, then:
   .\scripts\install.ps1
   ```

That's it! The service will automatically:

- Scan your network for biometric devices
- Configure supported devices automatically
- Install as a Windows service
- Start running immediately

### Option 3: PowerShell Installation (Local)

```powershell
# Run PowerShell as Administrator
.\install.ps1

# Enhanced installer with smart pairing
.\enhanced-bridge-installer.ps1

# With custom options
.\install.ps1 -InstallPath "C:\MyGymBridge" -Force
```

### Option 3: Manual Installation

```cmd
# Run as Administrator
gym-door-bridge.exe install
```

## 📋 What Gets Installed

The installer creates:

```
C:\Program Files\GymDoorBridge\
├── gym-door-bridge.exe     # Main service executable
├── config.yaml             # Auto-generated configuration
├── bridge.db               # Local database
└── logs\                   # Log files
    └── bridge.log
```

**Windows Service:**

- **Name:** GymDoorBridge
- **Display Name:** Gym Door Access Bridge
- **Startup:** Automatic
- **Account:** Local System

**Registry Entries:**

- `HKLM\SOFTWARE\GymDoorBridge\InstallPath`
- `HKLM\SOFTWARE\GymDoorBridge\ConfigPath`

## 🔍 Device Auto-Discovery

The installer automatically scans your network and configures:

### Supported Devices

| Brand        | Model Examples     | Ports      | Protocol |
| ------------ | ------------------ | ---------- | -------- |
| **ZKTeco**   | K40, K50, F18, F19 | 4370       | TCP      |
| **ESSL**     | X990, Biomax N-BM5 | 80, 8080   | HTTP     |
| **Realtime** | T502, T503         | 5005, 9999 | TCP      |

### Discovery Process

1. **Network Scan:** Scans all local network interfaces
2. **Port Probing:** Tests common biometric device ports
3. **Device Identification:** Attempts to identify device type
4. **Configuration Generation:** Creates adapter configs automatically
5. **Service Installation:** Installs with discovered devices enabled

### Example Auto-Generated Config

```yaml
enabled_adapters:
  - zkteco_192_168_1_100_4370
  - essl_192_168_1_101_80

adapter_configs:
  zkteco_192_168_1_100_4370:
    device_type: zkteco
    connection: tcp
    device_config:
      ip: "192.168.1.100"
      port: "4370"
      comm_password: "0"
    sync_interval: 10

  essl_192_168_1_101_80:
    device_type: essl
    connection: tcp
    device_config:
      ip: "192.168.1.101"
      port: "80"
      username: "admin"
      password: "admin"
    sync_interval: 10
```

## 🔧 Post-Installation

### 1. Pair with Platform (Smart Pairing Enabled ⭐)

```cmd
gym-door-bridge.exe pair --pair-code YOUR_CODE
```

**Smart Pairing Features:**
- 🔄 Automatically detects if device is already paired
- 🔧 Runs `unpair --force` if needed
- ⚡ Retries pairing with current code automatically
- 🎉 Works seamlessly during reinstallations

This will:

- Register the bridge with your SaaS platform
- Obtain device credentials  
- Update configuration with platform details
- Handle existing pairings automatically

### 2. Verify Installation

```cmd
# Check service status
sc query GymDoorBridge

# View service in Services.msc
services.msc

# Check logs
type "C:\Program Files\GymDoorBridge\logs\bridge.log"
```

### 3. Test Device Connection

The service automatically tests device connections on startup. Check logs for:

```
INFO Biometric adapter started
INFO Found ZKTeco device at 192.168.1.100:4370 (Model: K40)
INFO Device connected successfully
```

## 🛠️ Troubleshooting

### Service Won't Start

1. **Check logs:**

   ```cmd
   type "C:\Program Files\GymDoorBridge\logs\bridge.log"
   ```

2. **Verify configuration:**

   ```cmd
   gym-door-bridge.exe --config "C:\Program Files\GymDoorBridge\config.yaml" --log-level debug
   ```

3. **Test device connectivity:**
   ```cmd
   telnet 192.168.1.100 4370
   ```

### No Devices Discovered

1. **Check network connectivity:**

   - Ensure devices are on the same network
   - Verify device IP addresses
   - Check firewall settings

2. **Manual device addition:**
   Edit `config.yaml` to add devices manually:

   ```yaml
   enabled_adapters:
     - my_device

   adapter_configs:
     my_device:
       device_type: zkteco # or essl, realtime
       connection: tcp
       device_config:
         ip: "192.168.1.100"
         port: "4370"
   ```

3. **Restart service:**
   ```cmd
   net stop GymDoorBridge
   net start GymDoorBridge
   ```

### Permission Issues

1. **Run as Administrator:**
   All installation commands must run as Administrator

2. **Check service account:**
   Service runs as Local System by default

3. **Firewall configuration:**
   Ensure Windows Firewall allows the service

## 🔄 Updates and Maintenance

### Updating the Service

1. **Stop service:**

   ```cmd
   net stop GymDoorBridge
   ```

2. **Replace executable:**

   ```cmd
   copy new-gym-door-bridge.exe "C:\Program Files\GymDoorBridge\gym-door-bridge.exe"
   ```

3. **Start service:**
   ```cmd
   net start GymDoorBridge
   ```

### Adding New Devices

The service automatically discovers new devices every 5 minutes. To force immediate discovery:

```cmd
net restart GymDoorBridge
```

### Backup Configuration

```cmd
copy "C:\Program Files\GymDoorBridge\config.yaml" config-backup.yaml
copy "C:\Program Files\GymDoorBridge\bridge.db" bridge-backup.db
```

## 🗑️ Uninstallation

### Option 1: Automated Uninstall

```cmd
# Run as Administrator
gym-door-bridge.exe uninstall
```

### Option 2: PowerShell Uninstall

```powershell
# Run as Administrator
.\install.ps1 -Uninstall
```

### Manual Cleanup (if needed)

```cmd
# Stop and remove service
sc stop GymDoorBridge
sc delete GymDoorBridge

# Remove installation directory
rmdir /s "C:\Program Files\GymDoorBridge"

# Remove registry entries
reg delete "HKLM\SOFTWARE\GymDoorBridge" /f
```

## 📞 Support

If you encounter issues:

1. **Check logs** in the installation directory
2. **Run with debug logging:** `--log-level debug`
3. **Verify network connectivity** to devices
4. **Check Windows Event Viewer** for service errors
5. **Contact support** with log files and error messages

## 🔒 Security Notes

- Service runs as Local System account
- Configuration files contain device credentials
- Network traffic is unencrypted (device limitation)
- Logs may contain sensitive information
- Regular security updates recommended

## 📚 Additional Resources

### 📖 Documentation
- **[System Requirements](./system-requirements.md)** - Detailed compatibility information
- **[Troubleshooting Guide](./troubleshooting-guide.md)** - Comprehensive problem resolution
- **[FAQ](./faq.md)** - Frequently asked questions
- **[Gym Owner Guide](./gym-owner-guide.md)** - Simplified installation guide
- **[Documentation Index](./documentation-index.md)** - Complete documentation overview

### 🆘 Getting Help
- **Email Support:** bridge-support@repset.com
- **GitHub Issues:** [Report bugs or request features](https://github.com/your-org/gym-door-bridge/issues)
- **Community Forum:** https://community.repset.com
- **Video Tutorials:** https://docs.repset.com/bridge/videos

### 🔧 Quick Diagnostic Commands
```powershell
# Check service status
Get-Service -Name "RepSetBridge"

# Test platform connectivity
Test-NetConnection -ComputerName api.repset.com -Port 443

# View recent logs
Get-Content "C:\Program Files\RepSet\Bridge\logs\bridge.log" | Select-Object -Last 20

# Check system requirements
Get-ComputerInfo | Select-Object WindowsProductName, WindowsVersion, TotalPhysicalMemory
```

### 📋 Before Contacting Support
Please collect this information:
1. **System Information:** Windows version, PowerShell version, .NET Framework version
2. **Error Messages:** Exact error text and screenshots
3. **Log Files:** Installation and service logs
4. **Network Configuration:** Connectivity test results
5. **Installation Command:** The exact command used