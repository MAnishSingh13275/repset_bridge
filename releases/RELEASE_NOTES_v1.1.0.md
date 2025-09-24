# Gym Door Bridge v1.1.0 Release Notes

## 🚀 **Major Updates - Production Ready**

This release includes significant improvements to make the Bridge fully production-ready with your gym management platform.

### ✅ **What's New**

#### 🌐 **Production Server Configuration**
- **Default Server URL**: Now points to `https://repset.onezy.in` (production)  
- **API Integration**: Fully compatible with production admin dashboard
- **No Manual Configuration**: Works out-of-the-box with your platform

#### 🔧 **Installer Improvements**
- **Fixed Download URLs**: Corrected GitHub repository paths
- **Better Error Handling**: More informative error messages  
- **Robust File Discovery**: Works across different PowerShell versions
- **Graceful Service Startup**: Installation succeeds even if service needs pairing first

#### 🛠 **New Commands**
- `gym-door-bridge install` - Install as Windows service with auto-discovery
- `gym-door-bridge uninstall` - Clean removal of service and files  
- `gym-door-bridge status` - Enhanced status reporting

### 🔄 **Updated Components**

#### **Pairing System**
- ✅ Connects to production API endpoints
- ✅ Proper JSON error handling  
- ✅ Clear success/failure messaging
- ✅ Automatic configuration updates

#### **Configuration**
- ✅ Production URLs by default
- ✅ Optimized for `repset.onezy.in` platform
- ✅ Backward compatible with existing setups

#### **Service Management**
- ✅ Improved Windows service reliability
- ✅ Better startup/shutdown handling
- ✅ Enhanced logging and monitoring

### 🐛 **Bug Fixes**

- Fixed executable search issues in installer
- Corrected service command syntax (`service install` vs `install`)
- Resolved placeholder URL references  
- Fixed type conversion errors in configuration
- Improved error messages and user guidance

### 📦 **Installation Methods**

#### **From Admin Dashboard** (Recommended)
1. Navigate to your admin dashboard bridge page
2. Generate installer with pairing code
3. Copy PowerShell command to target machine
4. Run as Administrator

#### **Manual Installation**
1. Download `GymDoorBridge-v1.1.0.zip`
2. Extract to desired location
3. Run `GymDoorBridge-Installer.ps1` as Administrator
4. Enter pairing code from admin dashboard

### 🎯 **Supported Devices**

- **ZKTeco** fingerprint devices (K40, K50, F18, F19, etc.)
- **ESSL** biometric systems (X990, Biomax series)  
- **Realtime** access control (T502, T503, etc.)
- **Generic** TCP/IP biometric devices

### ⚙️ **System Requirements**

- **OS**: Windows 10 / Windows Server 2019+
- **Privileges**: Administrator rights for installation
- **Network**: Internet connection for pairing and operation
- **Hardware**: Any network-connected biometric device

### 🔗 **Integration**

This version is fully integrated with:
- ✅ **Admin Dashboard**: `/[gymId]/admin/bridge` page
- ✅ **API Endpoints**: `/api/v1/devices/pair`, `/api/v1/bridge/events`
- ✅ **Production Platform**: `https://repset.onezy.in`

### 📈 **Performance**

- **Executable Size**: ~13.9 MB (optimized)
- **Installation Time**: 2-3 minutes (including device discovery)
- **Memory Usage**: ~10-20 MB during operation
- **Network Impact**: Minimal (heartbeat every 60 seconds)

### 🚨 **Breaking Changes**

- Default server URL changed from localhost to production
- Installation command syntax updated (`gym-door-bridge service install`)
- Configuration file structure improvements

### 🔄 **Migration from v1.0.0**

If upgrading from v1.0.0:
1. **Unpair** existing device: `gym-door-bridge unpair`
2. **Uninstall** old version: `gym-door-bridge service uninstall`  
3. **Install** v1.1.0 using new installer
4. **Pair** with new code from admin dashboard

### 🎉 **Ready for Production**

This release is **production-ready** and can be deployed to gym locations immediately. The installer will:

1. ✅ Download and install successfully
2. ✅ Discover biometric devices automatically  
3. ✅ Connect to your production platform
4. ✅ Handle device pairing seamlessly
5. ✅ Run as a reliable Windows service

---

## 📞 **Support**

For technical support:
- Check Windows Event Viewer for detailed logs
- Use `gym-door-bridge status` for diagnostics
- Contact support with error messages and system info

**File**: GymDoorBridge-v1.1.0.zip  
**Size**: 5.44 MB  
**SHA256**: [Calculate after upload]  
**Compatible**: repset.onezy.in platform