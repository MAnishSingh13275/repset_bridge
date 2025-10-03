# RepSet Bridge Installation Guide

## Overview

The RepSet Bridge is a Windows service that connects your gym's biometric access control devices (fingerprint scanners, RFID readers, etc.) to the RepSet cloud platform. This guide provides step-by-step installation instructions and troubleshooting tips.

## Quick Installation (Recommended)

### Method 1: One-Click PowerShell Installer

1. **Run PowerShell as Administrator**
   - Press `Windows + X`, select "PowerShell (Admin)" or "Terminal (Admin)"
   - Or search for "PowerShell", right-click, select "Run as administrator"

2. **Run the installer command**
   ```powershell
   Invoke-Expression "& {$(Invoke-RestMethod 'https://raw.githubusercontent.com/your-org/gym-door-bridge/main/public/install-bridge.ps1')} -PairCode 'YOUR_PAIR_CODE'"
   ```
   
   **Replace `YOUR_PAIR_CODE`** with the actual pairing code from your RepSet admin dashboard.

3. **Wait for installation to complete**
   - The installer will automatically download, install, and pair your bridge
   - Configuration files will be created in your Documents folder
   - The installer will attempt to create a Windows service

4. **Service Setup (if needed)**
   If the service creation fails, run this additional command as Administrator:
   ```powershell
   Invoke-Expression "& {$(Invoke-RestMethod 'https://raw.githubusercontent.com/your-org/gym-door-bridge/main/scripts/install.ps1')}"
   ```

## What Gets Installed

### File Locations

- **Executable**: `C:\Program Files\GymDoorBridge\gym-door-bridge.exe`
- **Configuration**: `%USERPROFILE%\Documents\repset-bridge-config.yaml`
- **Database**: `%USERPROFILE%\Documents\bridge.db`
- **Logs**: `%USERPROFILE%\Documents\bridge.log`

### Windows Service

- **Service Name**: `GymDoorBridge`
- **Display Name**: RepSet Gym Door Bridge
- **Startup Type**: Automatic (starts with Windows)

## Manual Installation Steps

If the automated installer doesn't work, follow these manual steps:

### 1. Download Bridge

1. Go to [GitHub Releases](https://github.com/your-org/gym-door-bridge/releases)
2. Download the latest `GymDoorBridge-v*.zip` file
3. Extract to `C:\Program Files\GymDoorBridge\`

### 2. Create Configuration

Create `%USERPROFILE%\Documents\repset-bridge-config.yaml`:

```yaml
# RepSet Bridge Configuration
device_id: ""
device_key: ""
server_url: "https://repset.onezy.in"
tier: "normal"
queue_max_size: 10000
heartbeat_interval: 60
unlock_duration: 3000
database_path: "%USERPROFILE%/Documents/bridge.db"
log_level: "info"
log_file: "%USERPROFILE%/Documents/bridge.log"
enabled_adapters:
  - "simulator"
```

### 3. Pair Device

```powershell
& "C:\Program Files\GymDoorBridge\gym-door-bridge.exe" pair --pair-code "YOUR_PAIR_CODE" --config "%USERPROFILE%\Documents\repset-bridge-config.yaml"
```

### 4. Create Service

```powershell
sc.exe create GymDoorBridge binpath= "\"C:\Program Files\GymDoorBridge\gym-door-bridge.exe\" --config \"%USERPROFILE%\Documents\repset-bridge-config.yaml\"" start= auto displayname= "RepSet Gym Door Bridge"
```

### 5. Start Service

```powershell
Start-Service -Name "GymDoorBridge"
```

## Verification

### Check Bridge Status

```powershell
& "C:\Program Files\GymDoorBridge\gym-door-bridge.exe" status --config "%USERPROFILE%\Documents\repset-bridge-config.yaml"
```

Expected output should show:
- ✅ Bridge paired with platform
- 🆔 Device ID: bridge_****
- 🔑 Device Key: ****

### Check Service Status

```powershell
Get-Service -Name "GymDoorBridge"
```

Expected status: `Running`

### Check Admin Dashboard

1. Log into your RepSet admin dashboard
2. Go to Bridge Management section
3. Your bridge should appear as "Active" within 1-2 minutes

## Troubleshooting

### Common Issues

#### 1. "Access Denied" or Permission Errors

**Cause**: Not running as Administrator

**Solution**: 
- Always run PowerShell as Administrator
- Right-click PowerShell → "Run as administrator"

#### 2. "Device Already Paired" Error

**Cause**: Bridge was previously paired but config wasn't updated

**Solution**:
```powershell
# Unpair first
& "C:\Program Files\GymDoorBridge\gym-door-bridge.exe" unpair --config "%USERPROFILE%\Documents\repset-bridge-config.yaml"

# Then pair again
& "C:\Program Files\GymDoorBridge\gym-door-bridge.exe" pair --pair-code "YOUR_PAIR_CODE" --config "%USERPROFILE%\Documents\repset-bridge-config.yaml"
```

#### 3. "DeviceId is Required" Error

**Cause**: Bridge is not properly paired

**Solution**:
1. Check pairing status: `& "C:\Program Files\GymDoorBridge\gym-door-bridge.exe" status --config "%USERPROFILE%\Documents\repset-bridge-config.yaml"`
2. If not paired, run pairing command with correct pair code
3. Verify config file has `device_id` and `device_key` populated

#### 4. Service Won't Start

**Cause**: Usually configuration or pairing issues

**Solution**:
```powershell
# Check service status
Get-Service -Name "GymDoorBridge"

# Check Windows Event Log
Get-EventLog -LogName Application -Source "GymDoorBridge" -Newest 10

# Try manual bridge run
& "C:\Program Files\GymDoorBridge\gym-door-bridge.exe" --config "%USERPROFILE%\Documents\repset-bridge-config.yaml"
```

#### 5. Config File Permission Issues

**Cause**: Config file created in Program Files (not writable)

**Solution**:
1. Ensure config is in `%USERPROFILE%\Documents\repset-bridge-config.yaml`
2. Re-run the installer which uses the correct location
3. Update service to use the correct config path

#### 6. "Bridge Not Appearing in Dashboard"

**Cause**: Network connectivity, firewall, or pairing issues

**Solution**:
1. Check internet connectivity
2. Verify bridge is paired: `gym-door-bridge.exe status`
3. Check Windows Firewall settings
4. Try manual bridge run to see error messages
5. Check bridge logs in Documents folder

### Manual Commands Reference

```powershell
# Check bridge status
& "C:\Program Files\GymDoorBridge\gym-door-bridge.exe" status --config "%USERPROFILE%\Documents\repset-bridge-config.yaml"

# Pair device
& "C:\Program Files\GymDoorBridge\gym-door-bridge.exe" pair --pair-code "YOUR_CODE" --config "%USERPROFILE%\Documents\repset-bridge-config.yaml"

# Unpair device
& "C:\Program Files\GymDoorBridge\gym-door-bridge.exe" unpair --config "%USERPROFILE%\Documents\repset-bridge-config.yaml"

# Run bridge manually (for testing)
& "C:\Program Files\GymDoorBridge\gym-door-bridge.exe" --config "%USERPROFILE%\Documents\repset-bridge-config.yaml"

# Service management
Start-Service -Name "GymDoorBridge"
Stop-Service -Name "GymDoorBridge"
Restart-Service -Name "GymDoorBridge"
Get-Service -Name "GymDoorBridge"

# View logs
Get-Content "%USERPROFILE%\Documents\bridge.log" -Tail 20
```

### Getting Help

If you continue to have issues:

1. **Check the logs**: `%USERPROFILE%\Documents\bridge.log`
2. **Run bridge manually** to see real-time error messages
3. **Check Windows Event Viewer**: Look for GymDoorBridge entries
4. **Verify network connectivity** to `https://repset.onezy.in`
5. **Contact support** with:
   - Bridge version
   - Error messages from logs
   - Windows version
   - Network configuration

## Advanced Configuration

### Firewall Settings

If you have strict firewall rules, ensure these are allowed:

- **Outbound HTTPS (443)** to `repset.onezy.in`
- **Outbound HTTP (80)** for device discovery
- **Inbound TCP 8081** for local API (optional)

### Antivirus Exclusions

Add these to your antivirus exclusions:

- `C:\Program Files\GymDoorBridge\`
- `%USERPROFILE%\Documents\repset-bridge-config.yaml`
- `%USERPROFILE%\Documents\bridge.db`
- `%USERPROFILE%\Documents\bridge.log`

### Service Account

The service runs as `LocalSystem` by default. For enhanced security, you can:

1. Create a dedicated service account
2. Grant it "Log on as a service" rights
3. Update service to use the custom account

```powershell
sc.exe config GymDoorBridge obj= ".\ServiceAccount" password= "PASSWORD"
```

## Uninstallation

To completely remove the RepSet Bridge:

```powershell
# Stop and remove service
Stop-Service -Name "GymDoorBridge" -Force
sc.exe delete "GymDoorBridge"

# Remove files
Remove-Item "C:\Program Files\GymDoorBridge" -Recurse -Force
Remove-Item "%USERPROFILE%\Documents\repset-bridge-config.yaml"
Remove-Item "%USERPROFILE%\Documents\bridge.db"
Remove-Item "%USERPROFILE%\Documents\bridge.log"
```

## Version Information

- **Current Version**: v1.3.0
- **Release Notes**: [GitHub Releases](https://github.com/your-org/gym-door-bridge/releases)
- **Update Method**: Re-run installer with new version