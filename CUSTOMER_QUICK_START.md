# RepSet Bridge - Quick Start Guide

## Installation Complete! 🎉

Your RepSet Bridge has been successfully installed and is ready to connect your gym door hardware to the RepSet platform.

## What Happens Next?

1. **Bridge Starts Automatically** - The bridge will start automatically when your computer boots up
2. **Dashboard Updates** - Check your RepSet admin dashboard - the bridge should appear as "Active" within 1-2 minutes
3. **Door Access Ready** - Your gym door access system is now connected to RepSet

## If You Need Help

### Quick Status Check

Open PowerShell and run:

```powershell
cd "C:\Program Files\GymDoorBridge"
.\gym-door-bridge.exe status
```

### Common Solutions

**Bridge Not Showing as Active?**

```powershell
Start-Service -Name "GymDoorBridge"
```

**Need to Restart the Bridge?**

```powershell
Restart-Service -Name "GymDoorBridge"
```

**View Recent Logs?**

```powershell
Get-Content "%USERPROFILE%\Documents\bridge.log" -Tail 20
```

### Manual Start (if needed)

If automatic startup isn't working, you can start the bridge manually:

```powershell
cd "C:\Program Files\GymDoorBridge"
.\gym-door-bridge.exe --config "C:\Users\[USERNAME]\Documents\repset-bridge-config.yaml"
```

_Replace [USERNAME] with your actual username_

## Support

If you need assistance:

1. **Run Diagnostics**:

   ```powershell
   .\gym-door-bridge.exe status
   ```

2. **Contact Support** with the diagnostic output

3. **Emergency Manual Start**: If all else fails, you can always run the bridge manually using the command above

## File Locations

- **Bridge Program**: `C:\Program Files\GymDoorBridge\`
- **Configuration**: `C:\Users\[USERNAME]\Documents\repset-bridge-config.yaml`
- **Logs**: `C:\Users\[USERNAME]\Documents\bridge.log`
- **Main Executable**: `C:\Program Files\GymDoorBridge\gym-door-bridge.exe`

## Your Bridge Details

- **Device ID**: Check your RepSet dashboard or configuration file
- **Platform**: https://repset.onezy.in
- **Status**: Should show as "Active" in your dashboard

---

**Need immediate help?** Contact RepSet support with your gym ID and any error messages.
