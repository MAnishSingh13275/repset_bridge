# RepSet Bridge - Multi-Step Installation Guide

## 🎯 Overview

This guide provides a smooth, step-by-step installation process for the RepSet Bridge. Each step is separate, verified, and foolproof.

**Total Time:** 5-10 minutes  
**Difficulty:** Easy (just follow the steps)  
**Requirements:** Windows computer with Administrator access

---

## 📋 What You Need Before Starting

✅ **Windows computer** (Windows 10/11 or Server)  
✅ **Administrator access** to your computer  
✅ **Internet connection** (for download)  
✅ **Pair code** from your RepSet admin dashboard (you can get this during Step 4)

---

## 🚀 Installation Process

### Step 1: Download Bridge Files

**What this does:** Downloads the RepSet Bridge software from GitHub

1. **Right-click** on the Windows Start button
2. Select **"Terminal (Admin)"** or **"PowerShell (Admin)"**
3. **Copy and paste** this command:
   ```powershell
   Invoke-WebRequest -Uri "https://raw.githubusercontent.com/MAnishSingh13275/repset_bridge/main/step1-download.ps1" -OutFile "step1-download.ps1"; .\step1-download.ps1
   ```
4. **Press Enter** and wait for download to complete
5. **Result:** Bridge files downloaded and verified ✅

---

### Step 2: Install Bridge Files

**What this does:** Sets up directories, copies files, and creates configuration

1. **In the same PowerShell window**, run:
   ```powershell
   .\step2-install.ps1
   ```
2. **Wait** for installation to complete (1-2 minutes)
3. **Result:** Bridge installed to `C:\Program Files\GymDoorBridge` ✅

---

### Step 3: Setup Windows Service

**What this does:** Creates and configures the Windows service

1. **In the same PowerShell window**, run:
   ```powershell
   .\step3-service.ps1
   ```
2. **Wait** for service setup to complete
3. **Result:** Windows service "RepSet Gym Door Bridge" created ✅

---

### Step 4: Pair with RepSet Platform

**What this does:** Connects your bridge to your RepSet account

#### Option A: If you have your pair code ready
```powershell
.\step4-pair.ps1 -PairCode "YOUR-PAIR-CODE"
```

#### Option B: If you need to get your pair code
```powershell
.\step4-pair.ps1
```
Then follow the prompts to enter your pair code.

**To get your pair code:**
1. Log into your RepSet admin dashboard
2. Go to **Bridge Management** section  
3. Click **"Add New Bridge"** or **"Generate Pair Code"**
4. Copy the code (format: XXXX-XXXX-XXXX)

**Result:** Bridge paired and service running ✅

---

## 🎉 Installation Complete!

After Step 4, you should see:
- ✅ Bridge paired with RepSet platform
- ✅ Windows service running  
- ✅ API endpoint responding
- ✅ Ready for device discovery

---

## 🔍 Verify Everything is Working

Run these commands to check status:

```powershell
# Check Windows service
Get-Service -Name "GymDoorBridge"

# Check bridge status
& "C:\Program Files\GymDoorBridge\gym-door-bridge.exe" status

# Check if API is responding
Invoke-WebRequest -Uri "http://localhost:8081/health" -UseBasicParsing
```

**Expected Results:**
- Service Status: **Running**
- Bridge Status: **Connected to RepSet**
- API Status: **HTTP 200 OK**

---

## 🎛️ Managing Your Bridge

### Service Management
```powershell
# Start the service
net start GymDoorBridge

# Stop the service  
net stop GymDoorBridge

# Restart the service
net stop GymDoorBridge && net start GymDoorBridge

# Check service status
Get-Service -Name "GymDoorBridge"
```

### Bridge Commands
```powershell
# Check bridge status
gym-door-bridge status

# Check discovered devices
gym-door-bridge devices

# View recent logs
Get-Content "C:\ProgramData\GymDoorBridge\bridge.log" -Tail 20
```

---

## 📂 Important File Locations

| Type | Location |
|------|----------|
| **Program Files** | `C:\Program Files\GymDoorBridge\` |
| **Configuration** | `C:\Program Files\GymDoorBridge\config.yaml` |
| **Data Directory** | `C:\ProgramData\GymDoorBridge\` |
| **Log Files** | `C:\ProgramData\GymDoorBridge\bridge.log` |
| **Database** | `C:\ProgramData\GymDoorBridge\bridge.db` |

---

## 🔧 Troubleshooting

### If a Step Fails

**Step 1 (Download) Issues:**
- Check internet connection
- Disable antivirus temporarily
- Try different network/hotspot

**Step 2 (Install) Issues:**
- Make sure PowerShell is running as Administrator
- Check available disk space
- Ensure no other RepSet software is running

**Step 3 (Service) Issues:**
- Verify Steps 1 & 2 completed successfully
- Check Windows Event Viewer for service errors
- Try restarting computer and running Step 3 again

**Step 4 (Pairing) Issues:**
- Verify pair code is correct and not expired
- Check internet connectivity to `https://repset.onezy.in`
- Generate a new pair code if needed

### Common Solutions

**"Access Denied" Errors:**
- Always run PowerShell as Administrator
- Right-click PowerShell → "Run as administrator"

**"Service won't start" Errors:**
- Check bridge pairing status first
- Look at logs in `C:\ProgramData\GymDoorBridge\bridge.log`
- Try manual start: `Start-Service -Name "GymDoorBridge"`

**"Can't download scripts" Errors:**
- Check firewall settings
- Try from different network
- Download scripts manually from GitHub

---

## 🆘 Getting Help

### Quick Diagnostics
Run this to gather diagnostic info:
```powershell
Write-Host "=== RepSet Bridge Diagnostics ==="
Get-Service -Name "GymDoorBridge" | Format-List
& "C:\Program Files\GymDoorBridge\gym-door-bridge.exe" status
Get-Content "C:\ProgramData\GymDoorBridge\bridge.log" -Tail 10
```

### Contact Support
When contacting support, include:
- Windows version
- Error messages from installation
- Service status
- Recent log entries
- Your gym/location ID

---

## 🔄 Updates and Maintenance

### Updating the Bridge
To update to a newer version:
1. Stop the current service: `net stop GymDoorBridge`
2. Run the installation steps again (they will update existing installation)
3. Start the service: `net start GymDoorBridge`

### Regular Maintenance
The bridge is designed to run 24/7 with minimal maintenance:
- **Logs** are automatically rotated
- **Updates** can be installed without losing configuration
- **Service** automatically restarts on system reboot
- **Devices** are automatically rediscovered after network changes

---

## 🎯 What Happens After Installation

1. **Automatic Startup** - Bridge starts with Windows
2. **Device Discovery** - Automatically finds biometric devices on your network
3. **Dashboard Connection** - Your RepSet dashboard shows the bridge as "Active"
4. **Member Sync** - Check-ins from devices sync to RepSet in real-time
5. **Self-Monitoring** - Bridge monitors itself and restarts if needed

---

## 🏆 Success Indicators

Your installation is successful when:
- ✅ Windows service shows as "Running"
- ✅ RepSet dashboard shows bridge as "Active"  
- ✅ API endpoint responds at `http://localhost:8081`
- ✅ Bridge status shows "Connected"
- ✅ Log files show successful startup messages

**The bridge will now run 24/7 automatically connecting your gym hardware to RepSet!**