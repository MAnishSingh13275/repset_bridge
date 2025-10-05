# Windows Defender False Positive Solution

## 🛡️ Problem
Windows Defender is flagging `gym-door-bridge.exe` as a threat and removing it. This is a **false positive** because:

- The executable is not digitally signed (common for open-source software)
- It installs as a Windows service (triggers security alerts)
- It makes network connections (normal for bridge software)
- It's a new/unknown executable to Windows Defender

## 🚀 Solutions (Choose One)

### **Solution 1: Automatic Exclusions (Recommended)**

The installation script now automatically adds Windows Defender exclusions. Just run:

```powershell
# Run as Administrator - exclusions are added automatically
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/MAnishSingh13275/repset_bridge/main/install-bridge.ps1" -OutFile "install-bridge.ps1"
.\install-bridge.ps1 -PairCode "YOUR_PAIR_CODE" -ServerUrl "https://repset.onezy.in" -Force
```

### **Solution 2: Manual Exclusions Setup**

If you want to set up exclusions first:

```powershell
# Step 1: Download exclusions script
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/MAnishSingh13275/repset_bridge/main/setup-defender-exclusions.ps1" -OutFile "setup-defender-exclusions.ps1"

# Step 2: Run exclusions script (as Administrator)
.\setup-defender-exclusions.ps1

# Step 3: Install bridge
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/MAnishSingh13275/repset_bridge/main/install-bridge.ps1" -OutFile "install-bridge.ps1"
.\install-bridge.ps1 -PairCode "YOUR_PAIR_CODE" -ServerUrl "https://repset.onezy.in" -Force
```

### **Solution 3: Temporary Disable (Quick Fix)**

```powershell
# Temporarily disable real-time protection (as Administrator)
Set-MpPreference -DisableRealtimeMonitoring $true

# Install bridge
.\install-bridge.ps1 -PairCode "YOUR_PAIR_CODE" -ServerUrl "https://repset.onezy.in" -Force

# Re-enable protection
Set-MpPreference -DisableRealtimeMonitoring $false
```

### **Solution 4: Manual Windows Security Exclusions**

1. Open **Windows Security** (Windows Defender)
2. Go to **Virus & threat protection**
3. Click **Manage settings** under "Virus & threat protection settings"
4. Scroll down to **Exclusions** and click **Add or remove exclusions**
5. Click **Add an exclusion** and add:
   - **Folder**: `C:\Program Files\GymDoorBridge`
   - **Folder**: `C:\ProgramData\GymDoorBridge`
   - **Process**: `gym-door-bridge.exe`

## 🔍 Verification

After adding exclusions, verify they're working:

```powershell
# Check current exclusions
Get-MpPreference | Select-Object -ExpandProperty ExclusionPath
Get-MpPreference | Select-Object -ExpandProperty ExclusionProcess
```

## 🔒 Security Notes

**This is safe because:**
- ✅ The software is open source (code is publicly available)
- ✅ Downloaded from official GitHub repository
- ✅ No malicious behavior (just connects gym hardware to cloud)
- ✅ Exclusions are specific to the bridge software only
- ✅ Windows Defender will still protect against real threats

**The exclusions only affect:**
- `C:\Program Files\GymDoorBridge` folder
- `C:\ProgramData\GymDoorBridge` folder  
- `gym-door-bridge.exe` process

## 🎯 Why This Happens

This is extremely common with:
- New software installations
- Service-based applications
- Network-connected software
- Unsigned executables

Major software companies face this same issue until they get code signing certificates and build reputation with antivirus vendors.

## 🔄 Removing Exclusions (If Needed)

If you ever want to remove the exclusions:

```powershell
# Run the exclusions script with -Remove flag
.\setup-defender-exclusions.ps1 -Remove
```

## 📞 Support

If you continue having issues:
1. Check Windows Event Viewer for specific error messages
2. Ensure you're running PowerShell as Administrator
3. Try the temporary disable method as a last resort
4. Contact your IT administrator if in a corporate environment

The bridge software is legitimate and safe - this is just Windows Defender being overly cautious with new software.