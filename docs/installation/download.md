# Download & Install Gym Door Bridge

## 🚀 Quick Installation for Gym Owners

### NEW: Enhanced Web Installer ⭐

The fastest and most reliable way to install:

**Run PowerShell as Administrator, then:**
```powershell
iex (iwr -useb https://raw.githubusercontent.com/your-org/gym-door-bridge/main/public/install-bridge.ps1).Content
```

**Enhanced Features:**
- ✅ Automatic download of latest version
- ✅ Smart pairing with auto-unpair capability  
- ✅ Handles "already paired" devices automatically
- ✅ No manual unpair commands needed
- ✅ Reduces administrator support requests

### Step 1: Download (Manual Method)
Choose one of these options:

**Option A: Download Release Package**
1. Go to: https://github.com/your-org/gym-door-bridge/releases
2. Download: `gym-door-bridge-v1.0.0-windows.zip`
3. Extract to any folder (e.g., Desktop)

**Option B: Direct Download**
Download just the installer:
- [install.ps1](https://github.com/your-org/gym-door-bridge/releases/download/v1.0.0/scripts/install.ps1)

### Step 2: Install
1. **Right-click** on PowerShell and select **"Run as administrator"**
2. **Navigate** to the extracted folder and run `.\scripts\install.ps1`
3. Wait for automatic device discovery (2-3 minutes)
4. Done! ✅

### What Gets Downloaded

The release package contains:
```
gym-door-bridge-v1.0.0-windows.zip
├── gym-door-bridge.exe     # Main service executable
├── scripts/
│   └── install.ps1         # PowerShell installer
├── README.md               # Quick start guide
└── INSTALLATION.md         # Detailed installation guide
```

### System Requirements
- ✅ Windows 10/11 or Windows Server 2016+
- ✅ Administrator privileges
- ✅ Network connection to biometric devices
- ✅ Internet connection (for platform pairing)

### Supported Devices (Auto-Detected)
- ✅ ZKTeco fingerprint devices
- ✅ ESSL biometric devices  
- ✅ Realtime access control devices
- ✅ Most TCP/IP based biometric hardware

## 🔧 For IT Administrators

### Silent Installation
```cmd
gym-door-bridge.exe install --silent
```

### Custom Installation Path
```cmd
gym-door-bridge.exe install --install-path "C:\MyGymBridge"
```

### Bulk Deployment
Use PowerShell for multiple installations:
```powershell
.\install.ps1 -InstallPath "C:\GymBridge" -Force
```

## 📞 Support

If installation fails:
1. Check you're running as Administrator
2. Ensure Windows Defender/antivirus isn't blocking
3. Verify network connectivity to biometric devices
4. Contact support with error messages

## 🔄 Updates

To update an existing installation:
1. Download new version
2. Run `.\scripts\install.ps1` again (will update automatically)
3. Or use: `gym-door-bridge.exe install --force`