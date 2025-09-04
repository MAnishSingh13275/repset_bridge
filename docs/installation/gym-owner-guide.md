# Gym Door Bridge - Installation Guide for Gym Owners

## 🏋️ What is this?

The Gym Door Bridge connects your existing biometric devices (fingerprint scanners, card readers) to your gym management software. It automatically detects your devices and handles member check-ins.

## 🚀 Super Simple Installation

### Method 1: One-Click Web Install (Easiest)

1. **Right-click** on Windows PowerShell and select **"Run as administrator"**
2. **Copy and paste** this command:
   ```powershell
   iex (iwr -useb https://raw.githubusercontent.com/your-org/gym-door-bridge/main/web-install.ps1).Content
   ```
3. **Press Enter** and wait 2-3 minutes
4. **Done!** ✅

### Method 2: Download and Install

1. **Download**: [gym-door-bridge-installer.zip](https://github.com/your-org/gym-door-bridge/releases/latest)
2. **Extract** the zip file to your Desktop
3. **Right-click** on `install.bat` → **"Run as administrator"**
4. **Wait** for automatic setup (2-3 minutes)
5. **Done!** ✅

## 🔍 What Happens During Installation?

The installer automatically:

1. **🔍 Scans your network** for biometric devices
2. **⚙️ Configures** found devices automatically  
3. **🔧 Installs** as a Windows service
4. **🚀 Starts** the service automatically
5. **📝 Creates** log files for troubleshooting

### Supported Devices (Auto-Detected)
- ✅ **ZKTeco** fingerprint scanners (K40, K50, F18, F19, etc.)
- ✅ **ESSL** biometric devices (X990, Biomax, etc.)
- ✅ **Realtime** access control (T502, T503, etc.)
- ✅ Most network-connected biometric hardware

## 📱 After Installation

### Step 1: Check Status
Open Command Prompt as Administrator and run:
```cmd
gym-door-bridge status
```

You should see:
- ✅ Service is running
- ✅ X device(s) discovered
- ✅ Bridge paired with platform

### Step 2: Pair with Your Gym Software
```cmd
gym-door-bridge pair
```
Follow the prompts to connect to your gym management system.

### Step 3: Test Member Check-in
Have a member use their fingerprint on any connected device. Check-ins should appear in your gym software within seconds.

## 🛠️ Troubleshooting

### "No devices discovered"
- ✅ Ensure biometric devices are powered on
- ✅ Check devices are on the same network as your computer
- ✅ Verify device IP addresses are accessible
- ✅ Temporarily disable Windows Firewall to test

### "Service won't start"
- ✅ Run Command Prompt as Administrator
- ✅ Check Windows Event Viewer for errors
- ✅ Restart your computer and try again

### "Installation failed"
- ✅ Ensure you're running as Administrator
- ✅ Temporarily disable antivirus software
- ✅ Check you have internet connection
- ✅ Try the manual download method

## 📞 Need Help?

### Quick Checks
1. **Service Status**: Open Services.msc → Look for "Gym Door Access Bridge"
2. **Logs**: Check `C:\Program Files\GymDoorBridge\logs\bridge.log`
3. **Device Test**: Try `telnet [device-ip] [device-port]` to test connectivity

### Contact Support
- 📧 Email: support@yourgym.com
- 📱 Phone: 1-800-GYM-HELP
- 💬 Include log files and error messages

## 🔄 Updates

The bridge automatically checks for updates. To manually update:

1. Download the latest version
2. Run `install.bat` again (it will update automatically)
3. Or use: `gym-door-bridge install --force`

## 🗑️ Uninstall

If you need to remove the bridge:

```cmd
gym-door-bridge uninstall
```

Or run the installer with uninstall option.

## 💡 Pro Tips

- **Backup**: The installer creates automatic backups of your configuration
- **Multiple Devices**: The system handles multiple biometric devices automatically
- **Network Changes**: If you move devices, restart the service to re-discover
- **Performance**: The bridge uses minimal system resources
- **Security**: All communication is encrypted and secure

## 🏆 Success!

Once installed, your biometric devices will automatically sync member check-ins with your gym software. Members can use fingerprints, cards, or PINs as configured on your devices.

**The bridge runs 24/7 in the background - no daily maintenance required!**