# ================================================================
# Gym Door Bridge - Release Builder
# Builds executable and packages everything for GitHub Release
# ================================================================

param(
    [string]$Version = "1.0.0",
    [string]$OutputDir = ".\releases",
    [switch]$Clean = $false,
    [switch]$SkipBuild = $false
)

$ErrorActionPreference = "Stop"

Write-Host "=========================================" -ForegroundColor Green
Write-Host "  Gym Door Bridge Release Builder" -ForegroundColor Green
Write-Host "=========================================" -ForegroundColor Green
Write-Host ""
Write-Host "Version: $Version" -ForegroundColor Cyan
Write-Host "Output: $OutputDir" -ForegroundColor Cyan
Write-Host ""

# Clean previous builds
if ($Clean -and (Test-Path $OutputDir)) {
    Write-Host "[1/7] Cleaning previous builds..." -ForegroundColor Yellow
    Remove-Item $OutputDir -Recurse -Force
    Write-Host "      ✓ Previous builds cleaned" -ForegroundColor Green
} else {
    Write-Host "[1/7] Skipping clean (use -Clean to clean)" -ForegroundColor Gray
}

# Create output directory
if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
}

# Build executable
if (-not $SkipBuild) {
    Write-Host "[2/7] Building executable..." -ForegroundColor Yellow
    
    # Check if Go is available
    $goPath = Get-Command go -ErrorAction SilentlyContinue
    if (-not $goPath) {
        Write-Host "      ✗ Go not found! Please install Go to build." -ForegroundColor Red
        exit 1
    }
    
    # Build for Windows x64
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    $env:CGO_ENABLED = "1"  # Required for SQLite
    
    $buildOutput = "gym-door-bridge.exe"
    $buildCmd = "go build -ldflags `"-s -w -X main.version=$Version`" -o $buildOutput ./cmd"
    
    Write-Host "      Building: $buildCmd" -ForegroundColor Cyan
    Invoke-Expression $buildCmd
    
    if (-not (Test-Path $buildOutput)) {
        Write-Host "      ✗ Build failed - executable not found" -ForegroundColor Red
        exit 1
    }
    
    $fileInfo = Get-Item $buildOutput
    $sizeMB = [math]::Round($fileInfo.Length / 1MB, 2)
    Write-Host "      ✓ Build completed successfully ($sizeMB MB)" -ForegroundColor Green
} else {
    Write-Host "[2/7] Skipping build (use -SkipBuild=`$false to build)" -ForegroundColor Gray
    if (-not (Test-Path "gym-door-bridge.exe")) {
        Write-Host "      ✗ gym-door-bridge.exe not found!" -ForegroundColor Red
        exit 1
    }
}

# Create release directory structure
Write-Host "[3/7] Creating release directory..." -ForegroundColor Yellow
$releaseDir = "$OutputDir\GymDoorBridge-v$Version"
if (Test-Path $releaseDir) {
    Remove-Item $releaseDir -Recurse -Force
}
New-Item -ItemType Directory -Path $releaseDir -Force | Out-Null
Write-Host "      ✓ Release directory created: $releaseDir" -ForegroundColor Green

# Copy files to release directory
Write-Host "[4/7] Copying release files..." -ForegroundColor Yellow

$filesToCopy = @(
    @{ Source = "gym-door-bridge.exe"; Required = $true },
    @{ Source = "GymDoorBridge-Installer.bat"; Required = $true },
    @{ Source = "GymDoorBridge-Installer.ps1"; Required = $true },
    @{ Source = "config.yaml.example"; Required = $false },
    @{ Source = "LICENSE"; Required = $false },
    @{ Source = "README.md"; Required = $false }
)

foreach ($file in $filesToCopy) {
    if (Test-Path $file.Source) {
        Copy-Item $file.Source $releaseDir -Force
        Write-Host "      ✓ Copied: $($file.Source)" -ForegroundColor Green
    } elseif ($file.Required) {
        Write-Host "      ✗ Required file missing: $($file.Source)" -ForegroundColor Red
        exit 1
    } else {
        Write-Host "      ! Optional file missing: $($file.Source)" -ForegroundColor Yellow
    }
}

# Create installation README
Write-Host "[5/7] Creating installation guide..." -ForegroundColor Yellow
$installReadme = @"
# Gym Door Bridge Installation

## Quick Install (Recommended)

1. **Right-click** on `GymDoorBridge-Installer.bat`
2. Select **"Run as administrator"**
3. **Wait** for automatic device discovery (1-2 minutes)
4. **Enter your pairing code** when prompted
5. **Done!** Service installed and running

## Alternative Install (PowerShell)

1. **Right-click** on `GymDoorBridge-Installer.ps1`
2. Select **"Run with PowerShell"** (as administrator)
3. Follow the prompts

## What Gets Installed

- ✅ Windows Service (auto-starts on boot)
- ✅ Automatic biometric device discovery
- ✅ Start Menu management shortcuts
- ✅ Event log integration
- ✅ Offline queue with retry logic

## Supported Devices (Auto-Discovered)

- **ZKTeco** fingerprint scanners (K40, K50, F18, F19, etc.)
- **ESSL** biometric devices (X990, Biomax, etc.)
- **Realtime** access control (T502, T503, etc.)
- Most network-connected biometric hardware

## After Installation

### Check Status
- Start Menu → Gym Door Bridge → Check Status
- Or run: `gym-door-bridge.exe service status`

### Pair Device
- Start Menu → Gym Door Bridge → Pair Device
- Or run: `gym-door-bridge.exe pair --pair-code YOUR_CODE`

### Service Management
- Use Windows Services (services.msc)
- Or use Start Menu shortcuts
- Service Name: **GymDoorBridge**

## Troubleshooting

### Installation Issues
- Run as Administrator
- Temporarily disable antivirus
- Check Windows Event Viewer

### Device Discovery Issues
- Ensure devices are on same network
- Check device IP addresses are accessible
- Verify devices are powered on

### Service Issues
- Check Services.msc for status
- View Event Viewer → Applications and Services Logs
- Restart service: Start Menu → Gym Door Bridge → Restart Service

## Support

- **Config File**: `C:\Program Files\GymDoorBridge\config.yaml`
- **Logs**: Windows Event Viewer
- **Service**: GymDoorBridge in services.msc

For support, include:
- Windows version
- Error messages
- Event log entries
- Service status output

## Uninstall

- Start Menu → Gym Door Bridge → Uninstall
- Or run: `gym-door-bridge.exe service uninstall`
"@

$installReadme | Out-File "$releaseDir\INSTALL.txt" -Encoding UTF8 -Force
Write-Host "      ✓ Installation guide created" -ForegroundColor Green

# Create changelog
Write-Host "[6/7] Creating changelog..." -ForegroundColor Yellow
$changelog = @"
# Gym Door Bridge v$Version

## Features

- ✅ **Automatic Device Discovery** - Finds ZKTeco, ESSL, Realtime devices
- ✅ **Windows Service** - Auto-starts on boot, runs in background
- ✅ **One-Click Installation** - Simple batch/PowerShell installers
- ✅ **Offline Queue** - Stores events when network unavailable
- ✅ **Start Menu Integration** - Easy management shortcuts
- ✅ **Event Log Integration** - Windows Event Viewer support
- ✅ **HMAC Authentication** - Secure cloud communication
- ✅ **Multi-Device Support** - Handles multiple biometric devices
- ✅ **Performance Tiers** - Adapts to system resources
- ✅ **Health Monitoring** - Continuous system health checks

## Installation

1. Download and extract this ZIP file
2. Right-click `GymDoorBridge-Installer.bat`
3. Select "Run as administrator"
4. Follow prompts for automatic installation

## System Requirements

- Windows 10/11 or Windows Server 2016+
- Administrator privileges for installation
- Network access to biometric devices
- Internet connectivity for cloud sync

## Files Included

- `gym-door-bridge.exe` - Main executable
- `GymDoorBridge-Installer.bat` - Batch installer
- `GymDoorBridge-Installer.ps1` - PowerShell installer
- `config.yaml.example` - Example configuration
- `INSTALL.txt` - Installation guide
- `CHANGELOG.txt` - This file

## Version Information

- Version: $Version
- Platform: Windows x64
- Go Version: $(go version)
- Build Date: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss UTC')
"@

$changelog | Out-File "$releaseDir\CHANGELOG.txt" -Encoding UTF8 -Force
Write-Host "      ✓ Changelog created" -ForegroundColor Green

# Create ZIP package
Write-Host "[7/7] Creating ZIP package..." -ForegroundColor Yellow
$zipPath = "$OutputDir\GymDoorBridge-v$Version.zip"
if (Test-Path $zipPath) {
    Remove-Item $zipPath -Force
}

# Use .NET compression
Add-Type -AssemblyName System.IO.Compression.FileSystem
[System.IO.Compression.ZipFile]::CreateFromDirectory($releaseDir, $zipPath)

$zipInfo = Get-Item $zipPath
$zipSizeMB = [math]::Round($zipInfo.Length / 1MB, 2)
Write-Host "      ✓ ZIP package created: $zipPath ($zipSizeMB MB)" -ForegroundColor Green

# Generate release notes for GitHub
$releaseNotes = @"
## 🚀 Gym Door Bridge v$Version

### One-Click Installation
- Download the ZIP file below
- Extract and run `GymDoorBridge-Installer.bat` as Administrator
- Automatic device discovery and service installation

### Features
- ✅ Windows Service with auto-start on boot
- ✅ Automatic biometric device discovery (ZKTeco, ESSL, Realtime)
- ✅ Offline queue with retry logic
- ✅ Start Menu management shortcuts
- ✅ HMAC authentication for secure cloud sync
- ✅ Performance tier auto-detection
- ✅ Health monitoring and logging

### System Requirements
- Windows 10/11 or Windows Server 2016+
- Administrator privileges for installation
- Network access to biometric devices

### Installation Steps
1. Download `GymDoorBridge-v$Version.zip`
2. Extract to any folder
3. Right-click `GymDoorBridge-Installer.bat` → "Run as administrator"
4. Follow prompts (auto-discovery takes 1-2 minutes)
5. Enter pairing code when prompted
6. Done! Service runs automatically

### Support
- Service Name: **GymDoorBridge**  
- Config: `C:\Program Files\GymDoorBridge\config.yaml`
- Logs: Windows Event Viewer
- Management: Start Menu → Gym Door Bridge

---
**Package Contents:**
- Pre-built executable for Windows x64
- One-click batch and PowerShell installers
- Installation guide and changelog
- Example configuration file
"@

$releaseNotes | Out-File "$OutputDir\release-notes.md" -Encoding UTF8 -Force

# Summary
Write-Host ""
Write-Host "=========================================" -ForegroundColor Green  
Write-Host "  BUILD COMPLETED SUCCESSFULLY!" -ForegroundColor Green
Write-Host "=========================================" -ForegroundColor Green
Write-Host ""
Write-Host "📦 Release Package: $zipPath" -ForegroundColor Cyan
Write-Host "📝 Release Notes:   $OutputDir\release-notes.md" -ForegroundColor Cyan  
Write-Host "📂 Files Directory: $releaseDir" -ForegroundColor Cyan
Write-Host ""
Write-Host "Next Steps:" -ForegroundColor Yellow
Write-Host "1. Upload $zipPath to GitHub Release" -ForegroundColor White
Write-Host "2. Copy contents of release-notes.md to release description" -ForegroundColor White
Write-Host "3. Tag release as v$Version" -ForegroundColor White
Write-Host ""
Write-Host "Users can then:" -ForegroundColor Yellow
Write-Host "• Download the ZIP from GitHub Releases" -ForegroundColor White
Write-Host "• Extract and run GymDoorBridge-Installer.bat as Admin" -ForegroundColor White
Write-Host "• Service installs automatically with device discovery" -ForegroundColor White
Write-Host ""