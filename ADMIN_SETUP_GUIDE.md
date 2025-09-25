# 🏋️‍♂️ RepSet Bridge Admin Setup Guide

## 📋 **Quick Start for Gym Administrators**

This guide helps gym administrators easily install and manage the RepSet Bridge for biometric access control.

---

## 🚀 **1. Pre-Installation Checklist**

Before starting, ensure you have:

- [ ] **Windows 10/11** with Administrator access
- [ ] **Internet connection** for downloading and connecting to RepSet cloud
- [ ] **Access to your gym's admin dashboard** at https://repset.onezy.in
- [ ] **Bridge software downloaded** and extracted to a folder (e.g., `C:\RepsetBridge`)
- [ ] **Network access** to scan for biometric devices (if any)

---

## 🎯 **2. One-Command Setup**

### **Step 1: Get Your Pair Code**
1. Go to: **https://repset.onezy.in/{yourGymId}/admin/dashboard**
2. Navigate to **"Bridge Management"** section
3. Click **"Create Bridge Deployment"** (if not already created)
4. **Copy the Pair Code** (displayed in blue, format: `XXXX-YYYY-ZZZZ`)

### **Step 2: Open PowerShell**
1. Press `Windows + R`
2. Type `powershell` and press `Ctrl+Shift+Enter` (to run as administrator)
3. Navigate to your bridge folder: `cd "C:\RepsetBridge"`

### **Step 3: Run Setup Commands**
```powershell
# Pair with your gym
.\bridge-admin-tools.ps1 pair -PairCode "YOUR-PAIR-CODE-HERE"

# Install and start the bridge
.\bridge-admin-tools.ps1 install

# Check status
.\bridge-admin-tools.ps1 status
```

**That's it! Your bridge should now be running and connected.**

---

## 🛠️ **3. Admin Tool Commands**

The `bridge-admin-tools.ps1` script provides easy management:

### **Basic Commands:**
```powershell
.\bridge-admin-tools.ps1 status      # Check bridge status
.\bridge-admin-tools.ps1 start       # Start the bridge
.\bridge-admin-tools.ps1 stop        # Stop the bridge  
.\bridge-admin-tools.ps1 restart     # Restart the bridge
.\bridge-admin-tools.ps1 health      # Detailed health check
.\bridge-admin-tools.ps1 logs        # View recent logs
```

### **Setup Commands:**
```powershell
.\bridge-admin-tools.ps1 pair                              # Interactive pairing
.\bridge-admin-tools.ps1 pair -PairCode "XXXX-YYYY-ZZZZ"  # Direct pairing
.\bridge-admin-tools.ps1 install                           # Install bridge
.\bridge-admin-tools.ps1 uninstall -Force                  # Complete removal
```

---

## 📊 **4. Dashboard Integration**

### **What You'll See in Your Dashboard:**

**Before Bridge Connection:**
- Status: `PENDING` ⏳
- Last Heartbeat: `Never`
- Connected Devices: `0`

**After Successful Setup:**
- Status: `ACTIVE` ✅
- Last Heartbeat: `Just now`
- Connected Devices: `X found` (varies by network)

### **Dashboard Features:**
- **Real-time status** monitoring
- **Device discovery** notifications  
- **Access logs** from biometric devices
- **Bridge health** and performance metrics

---

## 🔧 **5. Troubleshooting**

### **Common Issues & Solutions:**

#### **Issue: "Bridge executable not found"**
**Solution:** 
- Download bridge from your dashboard installer
- Extract to folder like `C:\RepsetBridge`
- Ensure `gym-door-bridge.exe` and `config.yaml` are present

#### **Issue: "Pairing failed"**
**Solutions:**
1. **Check pair code:** Get fresh code from dashboard
2. **Check internet:** Ensure connection to `repset.onezy.in`
3. **Check firewall:** Allow bridge through Windows Firewall

#### **Issue: "Bridge won't start"**
**Solutions:**
1. **Clean restart:** `.\bridge-admin-tools.ps1 restart -Force`
2. **Check database:** Script automatically backs up corrupted database
3. **Check ports:** Ensure ports 8080, 8081 are available

#### **Issue: "No devices found"**
**This is normal!** Many gyms don't have biometric devices initially.
- Bridge still provides **manual check-in** capability
- Staff can use **web portal** for member access
- **QR code scanning** available via mobile app

---

## 🚨 **6. Admin Emergency Procedures**

### **If Bridge Stops Working:**
```powershell
# Quick fix attempt
.\bridge-admin-tools.ps1 restart -Force

# If that fails, full reset
.\bridge-admin-tools.ps1 uninstall -Force
.\bridge-admin-tools.ps1 pair -PairCode "YOUR-CODE"
.\bridge-admin-tools.ps1 install
```

### **Get Help Information:**
```powershell
.\bridge-admin-tools.ps1 help     # Show all commands
.\bridge-admin-tools.ps1 status   # Current status
.\bridge-admin-tools.ps1 health   # Detailed diagnostics
.\bridge-admin-tools.ps1 logs     # Recent error logs
```

---

## 📞 **7. Support Information**

### **Self-Service Diagnostics:**
1. **Dashboard Check:** Status should show `ACTIVE` within 2 minutes
2. **Local Check:** `.\bridge-admin-tools.ps1 health`
3. **Process Check:** Look for `gym-door-bridge.exe` in Task Manager

### **What to Include in Support Requests:**
- **Gym ID** from your dashboard URL
- **Bridge status** output: `.\bridge-admin-tools.ps1 status`
- **Recent logs:** `.\bridge-admin-tools.ps1 logs`
- **Windows version** and any firewall/antivirus software

### **Contact Support:**
- **Dashboard:** Help section in your admin dashboard
- **Documentation:** Additional guides available in dashboard
- **Status Page:** Check system status at dashboard

---

## ⚡ **8. Quick Reference Card**

| Task | Command |
|------|---------|
| **First Setup** | `.\bridge-admin-tools.ps1 pair` then `.\bridge-admin-tools.ps1 install` |
| **Check Status** | `.\bridge-admin-tools.ps1 status` |
| **Restart Bridge** | `.\bridge-admin-tools.ps1 restart` |
| **View Logs** | `.\bridge-admin-tools.ps1 logs` |
| **Health Check** | `.\bridge-admin-tools.ps1 health` |
| **Emergency Reset** | `.\bridge-admin-tools.ps1 uninstall -Force` then setup again |

---

## ✅ **9. Success Checklist**

After setup, verify these items:

- [ ] Bridge process shows as **running** in Task Manager
- [ ] Dashboard status shows **ACTIVE** (not pending)
- [ ] Health check returns **healthy** status
- [ ] **Heartbeat timestamp** in dashboard is recent (< 2 minutes ago)
- [ ] Bridge **automatically restarts** after computer reboot (if installed as service)

---

## 🏆 **Congratulations!**

Your RepSet Bridge is now installed and operational! 

- **Member access** is now streamlined through the bridge
- **Biometric devices** will be automatically discovered if present
- **Manual check-in** options remain available for all members
- **Real-time monitoring** available through your admin dashboard

The bridge runs quietly in the background, automatically handling member access and reporting to your dashboard.