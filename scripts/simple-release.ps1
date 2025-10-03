# Simple Build and Release Script for Gym Door Bridge
param(
    [string]$Version = "v1.0.0",
    [string]$ReleaseNotes = "Initial release of Gym Door Bridge"
)

Write-Host "Building and Releasing Gym Door Bridge $Version" -ForegroundColor Cyan

try {
    # Check GitHub CLI
    $ghVersion = gh --version 2>$null
    if (-not $ghVersion) {
        Write-Host "GitHub CLI (gh) is not installed!" -ForegroundColor Red
        Write-Host "Please install it from: https://cli.github.com/" -ForegroundColor Yellow
        exit 1
    }
    Write-Host "GitHub CLI found" -ForegroundColor Green

    # Clean and create build directory
    if (Test-Path "build") {
        Remove-Item -Recurse -Force "build"
    }
    New-Item -ItemType Directory -Path "build" -Force | Out-Null

    # Build Windows executable
    Write-Host "Building Windows executable..." -ForegroundColor Green
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    $env:CGO_ENABLED = "0"
    
    go build -ldflags "-s -w" -o "build/gym-door-bridge.exe" ./cmd
    
    if (-not (Test-Path "build/gym-door-bridge.exe")) {
        throw "Build failed - executable not found"
    }
    Write-Host "Build completed successfully" -ForegroundColor Green

    # Copy additional files
    Copy-Item "README.md" -Destination "build/" -ErrorAction SilentlyContinue
    
    # Create simple config template
    $configTemplate = @"
device_id: ""
device_key: ""
server_url: "https://repset.onezy.in"
tier: "normal"
queue_max_size: 10000
heartbeat_interval: 60
unlock_duration: 3000
database_path: "./bridge.db"
log_level: "info"
enabled_adapters:
  - "simulator"
api_server:
  enabled: true
  port: 8081
  host: "0.0.0.0"
"@
    $configTemplate | Out-File -FilePath "build/config.yaml.template" -Encoding UTF8

    # Create installation guide
    $installGuide = @"
# Gym Door Bridge Installation

## Quick Install
Run PowerShell as Administrator:
iex (iwr -useb https://raw.githubusercontent.com/MAnishSingh13275/repset_bridge/main/scripts/install-bridge.ps1).Content

## Manual Install
1. Extract files to C:\GymDoorBridge
2. Run PowerShell as Administrator
3. Run: .\gym-door-bridge.exe install
4. Pair: .\gym-door-bridge.exe pair YOUR_PAIR_CODE

## Service Management
- Start: net start GymDoorBridge
- Stop: net stop GymDoorBridge
- Status: sc query GymDoorBridge

API: http://localhost:8081
"@
    $installGuide | Out-File -FilePath "build/INSTALL.txt" -Encoding UTF8

    # Create zip package
    Compress-Archive -Path "build/*" -DestinationPath "build/gym-door-bridge-windows.zip" -Force
    Write-Host "Package created successfully" -ForegroundColor Green

    # Reset environment
    Remove-Item Env:GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
    Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue

    # Create release notes
    $releaseNotesContent = @"
# Gym Door Bridge $Version

## Features
- One-Click Installation with PowerShell
- Auto-Discovery of biometric devices
- Multi-Device Support (ZKTeco, ESSL, Realtime)
- Secure Pairing with cloud platform
- Offline Operation with event queuing
- Windows Service with auto-restart
- REST API for remote control
- Health Monitoring

## Installation
Run PowerShell as Administrator:
``````
iex (iwr -useb https://raw.githubusercontent.com/MAnishSingh13275/repset_bridge/main/scripts/install-bridge.ps1).Content
``````

## With Pair Code
``````
`$pairCode = "YOUR_PAIR_CODE"
`$script = iwr -useb https://raw.githubusercontent.com/MAnishSingh13275/repset_bridge/main/scripts/install-bridge.ps1
Invoke-Expression "& { `$(`$script.Content) } -PairCode '`$pairCode'"
``````

$ReleaseNotes
"@
    $releaseNotesContent | Out-File -FilePath "build/release-notes.md" -Encoding UTF8

    # Create GitHub release
    Write-Host "Creating GitHub release..." -ForegroundColor Green
    
    gh release create $Version "build/gym-door-bridge-windows.zip" --title "Gym Door Bridge $Version" --notes-file "build/release-notes.md"
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "GitHub release created successfully!" -ForegroundColor Green
        Write-Host "Release URL: https://github.com/MAnishSingh13275/repset_bridge/releases/tag/$Version" -ForegroundColor Cyan
        
        Write-Host "`nInstallation Command:" -ForegroundColor Yellow
        Write-Host "iex (iwr -useb https://raw.githubusercontent.com/MAnishSingh13275/repset_bridge/main/scripts/install-bridge.ps1).Content" -ForegroundColor White
    } else {
        throw "GitHub release creation failed"
    }

} catch {
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

Write-Host "`nRelease completed successfully!" -ForegroundColor Green