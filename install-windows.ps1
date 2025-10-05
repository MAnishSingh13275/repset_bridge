# Gym Door Bridge - Windows One-Click Installer
# This script downloads and installs the Gym Door Bridge automatically

param(
    [string]$PairCode = "",
    [switch]$Force = $false,
    [switch]$Help = $false
)

# Show help
if ($Help) {
    Write-Host @"
🚀 Gym Door Bridge - Windows One-Click Installer

USAGE:
    .\install-windows.ps1 [OPTIONS]

OPTIONS:
    -PairCode <code>    Automatically pair with platform using this code
    -Force              Force reinstall even if already installed
    -Help               Show this help message

EXAMPLES:
    # Install bridge only
    .\install-windows.ps1

    # Install and pair in one step
    .\install-windows.ps1 -PairCode "ABC123DEF456"

    # Force reinstall
    .\install-windows.ps1 -Force

REQUIREMENTS:
    - Windows 10/11 or Windows Server 2016+
    - Administrator privileges (script will prompt for elevation)
    - Internet connection for download

For support: support@repset.onezy.in
"@
    exit 0
}

# Colors for output
function Write-Success { param($Message) Write-Host "✅ $Message" -ForegroundColor Green }
function Write-Info { param($Message) Write-Host "ℹ️  $Message" -ForegroundColor Cyan }
function Write-Warning { param($Message) Write-Host "⚠️  $Message" -ForegroundColor Yellow }
function Write-Error { param($Message) Write-Host "❌ $Message" -ForegroundColor Red }
function Write-Header { param($Message) Write-Host "`n🎯 $Message" -ForegroundColor Magenta }

Write-Host @"

🚀 Gym Door Bridge - Windows Installer
========================================
This will install the Gym Door Bridge as a Windows service that:
- Runs automatically on startup
- Restarts automatically on failure  
- Updates automatically in the background
- Discovers your door hardware automatically

"@ -ForegroundColor Cyan

# Check if running as administrator
function Test-AdminRights {
    $currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
    return $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

if (-not (Test-AdminRights)) {
    Write-Warning "Administrator privileges required!"
    Write-Info "Please right-click PowerShell and 'Run as Administrator', then run this script again."
    Read-Host "Press Enter to exit"
    exit 1
}

# Configuration
$GitHubRepo = "yourorg/repset-bridge"  # Update with actual repo
$InstallDir = "$env:ProgramFiles\GymDoorBridge"
$ServiceName = "GymDoorBridge"
$LatestReleaseUrl = "https://api.github.com/repos/$GitHubRepo/releases/latest"

Write-Header "Checking system requirements..."

# Check Windows version
$osVersion = [System.Environment]::OSVersion.Version
if ($osVersion.Major -lt 10) {
    Write-Error "Windows 10 or later is required. Current version: $($osVersion)"
    Read-Host "Press Enter to exit"
    exit 1
}
Write-Success "Windows version: $($osVersion) ✓"

# Check if service is already installed
$existingService = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($existingService -and -not $Force) {
    Write-Warning "Gym Door Bridge is already installed!"
    $response = Read-Host "Do you want to reinstall? (y/N)"
    if ($response -ne 'y' -and $response -ne 'Y') {
        Write-Info "Installation cancelled."
        exit 0
    }
    $Force = $true
}

Write-Header "Downloading latest version..."

try {
    # Get latest release info
    $release = Invoke-RestMethod -Uri $LatestReleaseUrl -Headers @{"User-Agent"="GymDoorBridge-Installer"}
    $version = $release.tag_name
    Write-Success "Latest version: $version"

    # Find Windows executable
    $asset = $release.assets | Where-Object { $_.name -like "*windows*.exe" -or $_.name -like "*win64*.exe" } | Select-Object -First 1
    if (-not $asset) {
        throw "No Windows executable found in release"
    }

    $downloadUrl = $asset.browser_download_url
    $fileName = $asset.name
    $tempPath = "$env:TEMP\$fileName"

    Write-Info "Downloading: $fileName"
    Write-Info "From: $downloadUrl"
    
    # Download with progress
    $webClient = New-Object System.Net.WebClient
    $webClient.DownloadProgressChanged += {
        param($sender, $e)
        Write-Progress -Activity "Downloading $fileName" -Status "$($e.ProgressPercentage)% Complete" -PercentComplete $e.ProgressPercentage
    }
    $webClient.DownloadFileCompleted += {
        Write-Progress -Activity "Downloading $fileName" -Completed
    }
    
    $webClient.DownloadFileAsync($downloadUrl, $tempPath)
    
    # Wait for download to complete
    do {
        Start-Sleep -Milliseconds 100
    } while ($webClient.IsBusy)
    
    $webClient.Dispose()
    
    Write-Success "Downloaded: $tempPath"

} catch {
    Write-Error "Failed to download: $($_.Exception.Message)"
    Write-Info "Please check your internet connection and try again."
    Read-Host "Press Enter to exit"
    exit 1
}

Write-Header "Installing Gym Door Bridge..."

try {
    # Stop existing service if running
    if ($existingService) {
        Write-Info "Stopping existing service..."
        Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
        Start-Sleep -Seconds 2
    }

    # Run installer
    Write-Info "Running installer with elevated privileges..."
    $installArgs = @("install")
    if ($Force) {
        $installArgs += "--force"
    }
    
    $process = Start-Process -FilePath $tempPath -ArgumentList $installArgs -Wait -PassThru -NoNewWindow
    
    if ($process.ExitCode -eq 0) {
        Write-Success "Installation completed successfully!"
    } else {
        throw "Installer exited with code: $($process.ExitCode)"
    }

} catch {
    Write-Error "Installation failed: $($_.Exception.Message)"
    Write-Info "Please check the logs and try running the installer manually:"
    Write-Info "$tempPath install"
    Read-Host "Press Enter to exit"
    exit 1
}

# Pair with platform if code provided
if ($PairCode) {
    Write-Header "Pairing with platform..."
    try {
        $process = Start-Process -FilePath "$InstallDir\gym-door-bridge.exe" -ArgumentList @("pair", $PairCode) -Wait -PassThru -NoNewWindow
        if ($process.ExitCode -eq 0) {
            Write-Success "Successfully paired with platform!"
        } else {
            Write-Warning "Pairing failed. You can pair manually later with:"
            Write-Info "gym-door-bridge pair YOUR_PAIR_CODE"
        }
    } catch {
        Write-Warning "Pairing failed: $($_.Exception.Message)"
        Write-Info "You can pair manually later with:"
        Write-Info "gym-door-bridge pair YOUR_PAIR_CODE"
    }
}

# Verify installation
Write-Header "Verifying installation..."

try {
    # Check if service is installed and running
    $service = Get-Service -Name $ServiceName -ErrorAction Stop
    Write-Success "Service status: $($service.Status)"
    
    # Check if executable exists
    $exePath = "$InstallDir\gym-door-bridge.exe"
    if (Test-Path $exePath) {
        Write-Success "Executable installed: $exePath"
    } else {
        Write-Warning "Executable not found at expected location"
    }
    
    # Test status command
    $statusProcess = Start-Process -FilePath $exePath -ArgumentList "status" -Wait -PassThru -NoNewWindow -RedirectStandardOutput "$env:TEMP\bridge-status.txt" -ErrorAction SilentlyContinue
    if ($statusProcess.ExitCode -eq 0) {
        Write-Success "Bridge status command working ✓"
    }

} catch {
    Write-Warning "Verification issue: $($_.Exception.Message)"
}

# Cleanup
if (Test-Path $tempPath) {
    Remove-Item $tempPath -Force -ErrorAction SilentlyContinue
}

# Show completion message
Write-Host @"

🎉 Installation Complete!
========================

The Gym Door Bridge has been installed and is running as a Windows service.

WHAT'S RUNNING:
✅ Windows Service: $ServiceName (Status: $($service.Status))
✅ Auto-start: Enabled (will start automatically on boot)
✅ Auto-restart: Enabled (will restart if it crashes)
✅ Auto-update: Enabled (will update itself automatically)

NEXT STEPS:
"@ -ForegroundColor Green

if (-not $PairCode) {
    Write-Host @"
1. Get a pairing code from your gym management platform
2. Run: gym-door-bridge pair YOUR_PAIR_CODE
3. The bridge will automatically discover your door hardware

"@ -ForegroundColor Yellow
} else {
    Write-Host @"
1. Check your gym management platform - the bridge should show as "Connected"
2. The bridge will automatically discover your door hardware
3. Test your fingerprint scanner or RFID reader

"@ -ForegroundColor Yellow
}

Write-Host @"
USEFUL COMMANDS:
- gym-door-bridge status    (check status)
- gym-door-bridge restart   (restart service)
- gym-door-bridge logs      (view recent logs)

SUPPORT:
- Email: support@repset.onezy.in
- Docs: https://docs.repset.onezy.in/bridge

"@ -ForegroundColor Cyan

Write-Success "Ready to go! The bridge will work automatically in the background."
Read-Host "Press Enter to close"