# 🏋️‍♂️ RepSet Bridge - Quick Setup for Gym Admins

## 📦 **What's This?**
The RepSet Bridge connects your gym's biometric devices (fingerprint readers, door controllers) to the RepSet cloud platform for automated member check-ins.

---

## 🚀 **Super Quick Setup (3 Steps)**

### **Step 1: Get Your Pair Code**
1. Go to your gym dashboard: **https://repset.onezy.in/{yourGymId}/admin/dashboard**
2. Find **"Bridge Management"** section
3. Click **"Create Bridge Deployment"** 
4. **Copy the blue pair code** (looks like: `ABCD-1234-EFGH`)

### **Step 2: Run Setup**
1. **Right-click** on `Quick-Setup.bat` and select **"Run as administrator"**
2. **OR** open PowerShell as administrator in this folder and run:
   ```powershell
   .\bridge-admin.ps1 pair
   .\bridge-admin.ps1 install
   ```

### **Step 3: Verify**
```powershell
.\bridge-admin.ps1 status
```
You should see: ✅ Bridge process is running ✅ Bridge is healthy

**Done! Check your dashboard - status should change from "pending" to "ACTIVE"**

---

## 🛠️ **Daily Management**

### **Check Bridge Status**
```powershell
.\bridge-admin.ps1 status
```

### **Restart Bridge** (if having issues)
```powershell
.\bridge-admin.ps1 restart
```

### **Stop Bridge** (temporarily)
```powershell
.\bridge-admin.ps1 stop
```

### **Start Bridge** (after stopping)
```powershell
.\bridge-admin.ps1 start
```

---

## 🔍 **Troubleshooting**

### **Problem: Dashboard shows "pending"**
**Solution:** Bridge might not be running
```powershell
.\bridge-admin.ps1 status
.\bridge-admin.ps1 restart
```

### **Problem: Bridge won't start**
**Solution:** Clean restart
```powershell
.\bridge-admin.ps1 stop
.\bridge-admin.ps1 install
```

### **Problem: Need fresh pair code**
**Solution:** Get new code from dashboard and re-pair
```powershell
.\bridge-admin.ps1 pair -PairCode "NEW-CODE-HERE"
```

---

## 📊 **What Happens After Setup?**

### **Immediate Benefits:**
- ✅ **Dashboard shows bridge as ACTIVE**
- ✅ **Automated device discovery** begins
- ✅ **Member check-ins** start working
- ✅ **Real-time status monitoring**

### **If No Biometric Devices Found:**
**This is totally normal!** Many gyms start without biometric devices.

**You still get:**
- 📱 **QR code check-ins** via mobile app
- 💻 **Web portal access** for members  
- 👥 **Manual staff check-ins**
- 📊 **Full dashboard analytics**

### **Adding Devices Later:**
When you get fingerprint readers or door controllers:
1. **Connect them to your network**
2. **Bridge automatically discovers them**
3. **Configure in dashboard**
4. **Start using immediately**

---

## 📞 **Need Help?**

### **First, Try These:**
1. **Check dashboard:** https://repset.onezy.in/{gymId}/admin/dashboard
2. **Restart bridge:** `.\bridge-admin.ps1 restart`  
3. **Check status:** `.\bridge-admin.ps1 status`

### **For Support:**
- Include your **Gym ID** (from dashboard URL)
- Include output from: `.\bridge-admin.ps1 status`
- Note any **error messages**

---

## 💡 **Pro Tips**

### **Keep Bridge Running:**
- Bridge runs in background automatically
- Survives computer restarts
- Sends heartbeats every 60 seconds
- Dashboard shows real-time status

### **Performance:**
- Uses minimal resources (~20MB RAM)
- Runs efficiently in background  
- Auto-scales based on gym activity
- Smart device discovery

### **Security:**
- Encrypted communication to cloud
- Secure device authentication
- Local database for offline capability
- Automatic security updates

---

## ✅ **Success Checklist**

After setup, you should have:
- [ ] ✅ Bridge process running in Task Manager
- [ ] ✅ Dashboard shows "ACTIVE" status  
- [ ] ✅ Recent heartbeat timestamp (< 2 minutes)
- [ ] ✅ Member check-ins working
- [ ] ✅ Real-time dashboard updates

**Congratulations! Your gym is now connected to the RepSet platform! 🎉**