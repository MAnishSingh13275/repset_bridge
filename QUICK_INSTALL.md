# 🚀 Gym Door Bridge - One-Click Installation Guide

The Gym Door Bridge connects your gym's door access hardware (fingerprint scanners, RFID readers) to your management platform automatically and runs in the background forever.

## 📥 Download & Install

### Windows (Recommended)

1. **Download** the latest `gym-door-bridge-windows.exe` from the [releases page](https://github.com/yourorg/repset-bridge/releases/latest)

2. **Right-click** the downloaded file and select **"Run as administrator"**

3. **Run the installer:**
   ```cmd
   gym-door-bridge-windows.exe install
   ```

4. **Pair with your platform:**
   ```cmd
   gym-door-bridge-windows.exe pair YOUR_PAIR_CODE
   ```

**That's it!** 🎉 The bridge is now installed and will:
- ✅ Run automatically when Windows starts
- ✅ Restart automatically if it crashes  
- ✅ Update itself automatically in the background
- ✅ Discover and connect to your door hardware automatically

### macOS

1. **Download** the latest `gym-door-bridge-macos` from the [releases page](https://github.com/yourorg/repset-bridge/releases/latest)

2. **Open Terminal** and run:
   ```bash
   sudo ./gym-door-bridge-macos install
   ```

3. **Pair with your platform:**
   ```bash
   ./gym-door-bridge-macos pair YOUR_PAIR_CODE
   ```

**Done!** 🎉 The bridge will now run as a background daemon.

## 🔧 Management Commands

Once installed, you can manage the bridge service:

```bash
# Check status
gym-door-bridge status

# Start service
gym-door-bridge start

# Stop service  
gym-door-bridge stop

# Restart service
gym-door-bridge restart

# Unpair from platform
gym-door-bridge unpair

# Uninstall completely
gym-door-bridge uninstall
```

## 🔗 Getting Your Pair Code

1. **Log in** to your gym management platform
2. **Navigate** to Settings → Integrations → Door Bridge
3. **Click** "Add New Bridge" 
4. **Copy** the generated pair code
5. **Use** the pair code with the `gym-door-bridge pair` command

## 🔍 Checking if it's Working

After installation and pairing:

1. **Check status:** Run `gym-door-bridge status`
2. **Check logs:** Look in your installation directory under `logs/bridge.log`
3. **Check your platform:** The bridge should appear as "Connected" in your dashboard
4. **Test hardware:** Use your fingerprint scanner or RFID reader - events should appear in your platform

## 🚨 Troubleshooting

### "Access Denied" or "Permission Denied"
- **Windows:** Right-click Command Prompt and "Run as administrator"
- **macOS:** Use `sudo` before commands

### Service Won't Start
```bash
# Check detailed status
gym-door-bridge status

# Check logs
# Windows: C:\Program Files\GymDoorBridge\logs\bridge.log
# macOS: /Applications/GymDoorBridge/logs/bridge.log
```

### Hardware Not Detected
The bridge automatically scans your network for supported devices. If your hardware isn't found:

1. **Check network connection** - Bridge and hardware must be on same network
2. **Check firewall settings** - Allow the bridge through your firewall
3. **Check hardware IP settings** - Ensure hardware has valid IP address
4. **Supported devices:** ZKTeco, ESSL, and most standard TCP/IP fingerprint/RFID devices

### Can't Connect to Platform
1. **Check internet connection**
2. **Verify pair code** - Generate a new one if needed
3. **Check firewall** - Allow outbound HTTPS connections

## 📞 Support

- **Email:** support@repset.onezy.in
- **Documentation:** [Full documentation](https://docs.repset.onezy.in/bridge)
- **Status:** [System status page](https://status.repset.onezy.in)

## 🔄 Updates

The bridge updates itself automatically! You can also force an update:

```bash
# Check for updates manually
gym-door-bridge check-updates

# Force update (if available)
gym-door-bridge force-update
```

## 📊 What Happens After Installation

1. **Service starts automatically** on boot
2. **Discovers hardware** on your network automatically
3. **Connects to your platform** using the pair code
4. **Forwards door events** (fingerprint scans, RFID taps) to your platform
5. **Receives door control commands** from your platform
6. **Updates automatically** when new versions are available
7. **Restarts automatically** if it encounters any issues

The bridge is designed to be completely hands-off after the initial setup! 🙌