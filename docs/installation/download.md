# Download & Install Gym Door Bridge

## 🚀 Quick Installation for Gym Owners

### Step 1: Download
Choose one of these options:

**Option A: Download Release Package**
1. Go to: https://github.com/your-org/gym-door-bridge/releases
2. Download: `gym-door-bridge-v1.0.0-windows.zip`
3. Extract to any folder (e.g., Desktop)

**Option B: Direct Download**
Download just the installer:
- [install.bat](https://github.com/your-org/gym-door-bridge/releases/download/v1.0.0/install.bat)

### Step 2: Install
1. **Right-click** on `install.bat`
2. Select **"Run as administrator"**
3. Wait for automatic device discovery (2-3 minutes)
4. Done! ✅

### What Gets Downloaded

The release package contains:
```
gym-door-bridge-v1.0.0-windows.zip
├── gym-door-bridge.exe     # Main service executable
├── install.bat             # Simple installer
├── install.ps1             # PowerShell installer (advanced)
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
2. Run `install.bat` again (will update automatically)
3. Or use: `gym-door-bridge.exe install --force`