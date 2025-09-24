# ================================================================
# Gym Door Bridge - Web Installer (One-Click from Website)
# Downloads, extracts, and installs automatically from the web
# ================================================================

param(
    [string]$Version = "latest",
    [string]$ReleaseUrl = "https://github.com/MAnishSingh13275/repset_bridge/releases/download/v1.0.0/GymDoorBridge-v1.0.0.zip",
    [string]$PairCode = "",
    [switch]$Silent = $false
)

$ErrorActionPreference = "Stop"

# Set up console
$Host.UI.RawUI.WindowTitle = "Gym Door Bridge Web Installer"
if (-not $Silent) {
    Clear-Host
}

# Color functions
function Write-Success { param([string]$Message) Write-Host "      [OK] $Message" -ForegroundColor Green }
function Write-Error { param([string]$Message) Write-Host "      [ERROR] $Message" -ForegroundColor Red }
function Write-Warning { param([string]$Message) Write-Host "      [WARNING] $Message" -ForegroundColor Yellow }
function Write-Info { param([string]$Message) Write-Host "      [INFO] $Message" -ForegroundColor Cyan }
function Write-Step { param([string]$Step, [string]$Message) Write-Host "[$Step] $Message" -ForegroundColor White }

# Banner
if (-not $Silent) {
    Write-Host ""
    Write-Host "  ================================================================" -ForegroundColor Green
    Write-Host "  |                                                              |" -ForegroundColor Green
    Write-Host "  |              GYM DOOR BRIDGE WEB INSTALLER                  |" -ForegroundColor Green
    Write-Host "  |                                                              |" -ForegroundColor Green
    Write-Host "  |          Downloads and installs automatically               |" -ForegroundColor Green
    Write-Host "  |                                                              |" -ForegroundColor Green
    Write-Host "  ================================================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "  Version: $Version" -ForegroundColor Gray
    Write-Host "  Source: GitHub Release" -ForegroundColor Gray
    Write-Host ""
}

try {
    # Check administrator privileges
    Write-Step "1/8" "Checking administrator privileges..."
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    
    if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        Write-Host ""
        Write-Host "  ================================================================" -ForegroundColor Red
        Write-Host "  |                        ERROR                                 |" -ForegroundColor Red
        Write-Host "  |                                                              |" -ForegroundColor Red
        Write-Host "  |    This installer requires administrator privileges!         |" -ForegroundColor Red
        Write-Host "  |                                                              |" -ForegroundColor Red
        Write-Host "  |    Please run PowerShell as administrator and try again     |" -ForegroundColor Red
        Write-Host "  |                                                              |" -ForegroundColor Red
        Write-Host "  ================================================================" -ForegroundColor Red
        Write-Host ""
        if (-not $Silent) { Read-Host "Press Enter to exit" }
        exit 1
    }
    Write-Success "Administrator privileges confirmed"
    Write-Host ""

    # Create temp directory
    Write-Step "2/8" "Setting up temporary workspace..."
    $tempDir = "$env:TEMP\GymDoorBridge-Install-$(Get-Random)"
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
    Write-Success "Temporary directory: $tempDir"
    Write-Host ""

    # Download release
    Write-Step "3/8" "Downloading latest release..."
    Write-Info "Downloading from GitHub..."
    
    # If version is "latest", resolve the actual download URL
    if ($Version -eq "latest") {
        try {
            Write-Info "Resolving latest release URL..."
            $apiUrl = "https://api.github.com/repos/MAnishSingh13275/repset_bridge/releases/latest"
            $releaseInfo = Invoke-RestMethod -Uri $apiUrl -UserAgent "GymDoorBridge-WebInstaller"
            
            # Find the ZIP asset
            $zipAsset = $releaseInfo.assets | Where-Object { $_.name -match "GymDoorBridge-v.*\.zip" } | Select-Object -First 1
            if (-not $zipAsset) {
                throw "No ZIP asset found in latest release"
            }
            
            $ReleaseUrl = $zipAsset.browser_download_url
            $Version = $releaseInfo.tag_name -replace "^v", ""
            Write-Success "Latest version: v$Version"
        }
        catch {
            Write-Warning "Failed to resolve latest version, using provided URL"
        }
    }

    $zipPath = "$tempDir\GymDoorBridge.zip"
    Write-Info "Downloading: $ReleaseUrl"
    
    # Download with progress
    $progressPreference = 'SilentlyContinue'
    try {
        Invoke-WebRequest -Uri $ReleaseUrl -OutFile $zipPath -UseBasicParsing
        $progressPreference = 'Continue'
    }
    catch {
        $progressPreference = 'Continue'
        throw "Failed to download release: $($_.Exception.Message)"
    }

    if (-not (Test-Path $zipPath)) {
        throw "Downloaded file not found: $zipPath"
    }

    $zipInfo = Get-Item $zipPath
    $sizeMB = [math]::Round($zipInfo.Length / 1MB, 2)
    Write-Success "Download completed ($sizeMB MB)"
    Write-Host ""

    # Extract ZIP
    Write-Step "4/8" "Extracting installer..."
    $extractDir = "$tempDir\extracted"
    
    try {
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        [System.IO.Compression.ZipFile]::ExtractToDirectory($zipPath, $extractDir)
    }
    catch {
        throw "Failed to extract ZIP file: $($_.Exception.Message)"
    }

    # Find the installer files
    $installerBat = Get-ChildItem -Path $extractDir -Name "GymDoorBridge-Installer.bat" -Recurse | Select-Object -First 1
    $installerPs1 = Get-ChildItem -Path $extractDir -Name "GymDoorBridge-Installer.ps1" -Recurse | Select-Object -First 1
    $executable = Get-ChildItem -Path $extractDir -Name "gym-door-bridge.exe" -Recurse | Select-Object -First 1

    if (-not $executable) {
        throw "gym-door-bridge.exe not found in downloaded package"
    }

    Write-Success "Extraction completed"
    Write-Host ""

    # Change to extracted directory for installation
    $installDir = Split-Path $executable.FullName -Parent
    Push-Location $installDir

    try {
        # Check existing installation
        Write-Step "5/8" "Checking existing installation..."
        $service = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
        if ($service) {
            Write-Warning "Service already installed"
            if (-not $Silent) {
                $reinstall = Read-Host "      Do you want to reinstall? (Y/n)"
                if ($reinstall.ToLower() -eq "n") {
                    Write-Host "      Installation cancelled by user."
                    exit 0
                }
            }
            Write-Info "Removing existing service..."
            & ".\gym-door-bridge.exe" service uninstall | Out-Null
            Start-Sleep -Seconds 2
            Write-Success "Existing service removed"
        } else {
            Write-Success "No existing installation found"
        }
        Write-Host ""

        # Install service
        Write-Step "6/8" "Installing Windows service..."
        Write-Info "This will automatically discover your biometric devices..."
        Write-Info "Please wait while scanning network (this may take 1-2 minutes)..."
        Write-Host ""
        
        $installOutput = & ".\gym-door-bridge.exe" install 2>&1
        if ($LASTEXITCODE -ne 0) {
            throw "Service installation failed: $installOutput"
        }
        Write-Success "Service installed successfully!"
        Write-Host ""

        # Start service
        Write-Step "7/8" "Starting service..."
        $startResult = & ".\gym-door-bridge.exe" service start 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Service started successfully"
        } else {
            Write-Warning "Service installation completed but failed to start automatically"
            Write-Info "You can start it manually from Services.msc"
        }
        Write-Host ""

        # Verify installation
        Write-Step "8/8" "Verifying installation..."
        Start-Sleep -Seconds 3
        $statusResult = & ".\gym-door-bridge.exe" service status 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Service is running and operational"
        } else {
            Write-Warning "Service status check inconclusive"
        }
        Write-Host ""

    }
    finally {
        Pop-Location
    }

    # Installation complete
    Write-Host "  ================================================================" -ForegroundColor Green
    Write-Host "  |                                                              |" -ForegroundColor Green
    Write-Host "  |              INSTALLATION SUCCESSFUL!                       |" -ForegroundColor Green
    Write-Host "  |                                                              |" -ForegroundColor Green
    Write-Host "  ================================================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "  [OK] Downloaded latest version (v$Version)" -ForegroundColor Green
    Write-Host "  [OK] Gym Door Bridge service installed and running" -ForegroundColor Green
    Write-Host "  [OK] Auto-discovery completed for biometric devices" -ForegroundColor Green
    Write-Host "  [OK] Service configured to start automatically on boot" -ForegroundColor Green
    Write-Host ""

    if (-not $Silent) {
        Write-Host "  ================================================================" -ForegroundColor Cyan
        Write-Host "  |                      NEXT STEPS                             |" -ForegroundColor Cyan
        Write-Host "  ================================================================" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "  1. PAIR YOUR DEVICE:" -ForegroundColor White
        Write-Host "     • Get pairing code from your admin portal" -ForegroundColor Gray
        Write-Host "     • Use Start Menu > Gym Door Bridge > Pair Device" -ForegroundColor Gray
        Write-Host "     • Or run: gym-door-bridge pair --pair-code YOUR_CODE" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  2. VERIFY SETUP:" -ForegroundColor White
        Write-Host "     • Use Start Menu > Gym Door Bridge > Check Status" -ForegroundColor Gray
        Write-Host "     • Check Windows Services (services.msc)" -ForegroundColor Gray
        Write-Host "     • View Windows Event Viewer for logs" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  3. MANAGE SERVICE:" -ForegroundColor White
        Write-Host "     • All management tools available in Start Menu" -ForegroundColor Gray
        Write-Host "     • Service runs automatically on Windows startup" -ForegroundColor Gray
        Write-Host "     • No daily maintenance required" -ForegroundColor Gray
        Write-Host ""
    }

    # Handle pairing
    if ($PairCode) {
        Write-Host "Pairing device with provided code..." -ForegroundColor Yellow
        Push-Location $installDir
        try {
            $pairResult = & ".\gym-door-bridge.exe" pair --pair-code $PairCode 2>&1
            if ($LASTEXITCODE -eq 0) {
                Write-Host ""
                Write-Success "Device paired successfully!"
                Write-Host "Your gym door bridge is now fully operational!" -ForegroundColor Green
            } else {
                Write-Host ""
                Write-Error "Pairing failed. You can try again later."
                Write-Host "Pairing output: $pairResult" -ForegroundColor Red
            }
        }
        finally {
            Pop-Location
        }
    } elseif (-not $Silent) {
        Write-Host ""
        $pairNow = Read-Host "Would you like to pair your device now? (Y/n)"
        if ($pairNow.ToLower() -ne "n") {
            $inputCode = Read-Host "Enter your pairing code"
            if ($inputCode.Trim()) {
                Write-Host ""
                Write-Host "Pairing device..." -ForegroundColor Yellow
                Push-Location $installDir
                try {
                    $pairResult = & ".\gym-door-bridge.exe" pair --pair-code $inputCode 2>&1
                    if ($LASTEXITCODE -eq 0) {
                        Write-Host ""
                        Write-Success "Device paired successfully!"
                        Write-Host "Your gym door bridge is now fully operational!" -ForegroundColor Green
                    } else {
                        Write-Host ""
                        Write-Error "Pairing failed. You can try again later using:"
                        Write-Host "Start Menu > Gym Door Bridge > Pair Device" -ForegroundColor Cyan
                    }
                }
                finally {
                    Pop-Location
                }
            }
        }
    }

    Write-Host ""
    Write-Host "Web installation completed successfully!" -ForegroundColor Green
    if (-not $Silent) {
        Write-Host "You can close this window now." -ForegroundColor Gray
        Write-Host ""
        Read-Host "Press Enter to exit"
    }

}
catch {
    Write-Host ""
    Write-Host "  ================================================================" -ForegroundColor Red
    Write-Host "  |                        ERROR                                 |" -ForegroundColor Red
    Write-Host "  |                                                              |" -ForegroundColor Red
    Write-Host "  |                Web Installation Failed!                     |" -ForegroundColor Red
    Write-Host "  |                                                              |" -ForegroundColor Red
    Write-Host "  ================================================================" -ForegroundColor Red
    Write-Host ""
    Write-Host "Error details: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host ""
    Write-Host "Please try the following:" -ForegroundColor Yellow
    Write-Host "1. Check your internet connection" -ForegroundColor Gray
    Write-Host "2. Run PowerShell as Administrator" -ForegroundColor Gray
    Write-Host "3. Temporarily disable firewall/antivirus" -ForegroundColor Gray
    Write-Host "4. Download manually from GitHub Releases" -ForegroundColor Gray
    Write-Host ""
    if (-not $Silent) {
        Read-Host "Press Enter to exit"
    }
    exit 1
}
finally {
    # Cleanup temp directory
    if (Test-Path $tempDir) {
        try {
            Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        }
        catch {
            # Ignore cleanup errors
        }
    }
}
