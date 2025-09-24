# 📦 Manual GitHub Release Upload Instructions

Since GitHub CLI needs a PowerShell restart to work, here are the **step-by-step manual instructions** to upload your release:

## 🚀 **Quick Upload Steps**

### 1. **Open GitHub Releases**
Go to: **https://github.com/MAnishSingh13275/repset_bridge/releases**

### 2. **Create New Release**
- Click **"Create a new release"** button
- Or go directly to: **https://github.com/MAnishSingh13275/repset_bridge/releases/new**

### 3. **Fill Release Information**

**Tag version:** `v1.1.0`
- ✅ Make sure to type exactly: `v1.1.0`
- ✅ Select "Create new tag: v1.1.0 on publish"

**Release title:** `Gym Door Bridge v1.1.0`

**Description:** Copy and paste from `RELEASE_NOTES_v1.1.0.md` or use this short version:
```markdown
🚀 **Production-Ready Release**

Major improvements for production deployment with your gym management platform:

✅ **Production Configuration**: Default server URL now points to `https://repset.onezy.in`
✅ **Fixed Installer**: Corrected GitHub repository URLs and improved error handling  
✅ **Enhanced Commands**: New `install` and `uninstall` commands for Windows service
✅ **Better Compatibility**: Works across different PowerShell versions

### Key Changes:
- Production server URL configured by default
- Robust installation process with fallback methods
- Improved pairing system with production API endpoints
- Fixed all placeholder URLs and build errors

### Installation:
1. Download `GymDoorBridge-v1.1.0.zip` 
2. Extract and run `GymDoorBridge-Installer.ps1` as Administrator
3. Enter pairing code from admin dashboard

**Ready for deployment to gym locations!** 🏋️‍♂️
```

### 4. **Upload ZIP File**
- **Drag and drop** `GymDoorBridge-v1.1.0.zip` into the "Attach binaries" area
- Or click **"Choose your files"** and select the ZIP
- **Location**: `G:\repset_onezy\repset_bridge\releases\GymDoorBridge-v1.1.0.zip`

### 5. **Publish Release**  
- ✅ Check **"Set as the latest release"**
- ✅ Leave **"Set as a pre-release"** unchecked
- Click **"Publish release"**

## 🎉 **After Publishing**

### **Verify the Release**
1. Go to: https://github.com/MAnishSingh13275/repset_bridge/releases
2. Confirm `v1.1.0` appears as "Latest"
3. Test download link: https://github.com/MAnishSingh13275/repset_bridge/releases/download/v1.1.0/GymDoorBridge-v1.1.0.zip

### **Update Your Admin Dashboard**  
Your admin dashboard installer API has already been updated to use v1.1.0, so:
1. Deploy your updated Next.js app
2. Test installer generation from admin dashboard  
3. Verify it downloads v1.1.0 automatically

## 🔧 **Alternative: GitHub CLI Method**

If you want to use GitHub CLI later:

1. **Restart PowerShell** (to refresh PATH)
2. **Authenticate**: `gh auth login --web`
3. **Run upload script**: 
   ```powershell
   cd G:\repset_onezy\repset_bridge\releases
   .\upload-to-github.ps1
   ```

## ✅ **Verification Checklist**

After upload, verify:
- [ ] Release appears at: https://github.com/MAnishSingh13275/repset_bridge/releases
- [ ] Tag is exactly: `v1.1.0`
- [ ] ZIP file is downloadable  
- [ ] Marked as "Latest release"
- [ ] Admin dashboard generates correct installer URLs

## 🎯 **Expected Result**

Once uploaded, your admin dashboard will:
1. ✅ Generate installers that download `GymDoorBridge-v1.1.0.zip`
2. ✅ Bridge connects to `https://repset.onezy.in` automatically
3. ✅ Gym locations can install and pair seamlessly

**Your bridge system will be production-ready!** 🚀