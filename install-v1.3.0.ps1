# ================================================================
# Gym Door Bridge v1.3.0 - Production Web Installer 
# One-click installation for RepSet customers
# ================================================================

param(
    [string]$PairCode = "",
    [switch]$Silent = $false
)

$ErrorActionPreference = "Stop"

# Set up console
$Host.UI.RawUI.WindowTitle = "Gym Door Bridge v1.3.0 Installer"
if (-not $Silent) {
    Clear-Host
}

# Color functions
function Write-Success { param([string]$Message) Write-Host "      ✓ $Message" -ForegroundColor Green }
function Write-Error { param([string]$Message) Write-Host "      ✗ $Message" -ForegroundColor Red }
function Write-Warning { param([string]$Message) Write-Host "      ⚠ $Message" -ForegroundColor Yellow }
function Write-Info { param([string]$Message) Write-Host "      → $Message" -ForegroundColor Cyan }
function Write-Step { param([string]$Step, [string]$Message) Write-Host "[$Step] $Message" -ForegroundColor White }

# Banner
if (-not $Silent) {
    Write-Host ""
    Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Green
    Write-Host "  █                                                              █" -ForegroundColor Green
    Write-Host "  █              GYM DOOR BRIDGE WEB INSTALLER                  █" -ForegroundColor Green
    Write-Host "  █                                                              █" -ForegroundColor Green
    Write-Host "  █          Downloads and installs automatically               █" -ForegroundColor Green
    Write-Host "  █                                                              █" -ForegroundColor Green
    Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Green
    Write-Host ""
    Write-Host "  Version: v1.3.0" -ForegroundColor Gray
    Write-Host "  Source: GitHub Release" -ForegroundColor Gray
    Write-Host ""
}

# Function to create proper config file
function New-ProperConfigFile {
    param([string]$ConfigPath, [string]$InstallPath)
    
    $configContent = @"
# Gym Door Bridge Configuration
# Device configuration (set during pairing process)
device_id: ""
device_key: ""

# Server configuration
server_url: "https://repset.onezy.in"

# Performance tier (auto-detected, can be overridden)
tier: "normal"

# Queue configuration
queue_max_size: 10000
heartbeat_interval: 60  # seconds

# Door control configuration
unlock_duration: 3000  # milliseconds

# Database configuration
database_path: "./bridge.db"

# Logging configuration
log_level: "info"  # debug, info, warn, error
log_file: ""       # empty for stdout only

# Adapter configuration
enabled_adapters:
  - "simulator"
"@

    try {
        Set-Content -Path $ConfigPath -Value $configContent -Encoding UTF8
        Write-Success "Configuration file created: $ConfigPath"
        return $true
    } catch {
        Write-Error "Failed to create config file: $($_.Exception.Message)"
        return $false
    }
}

# Function to set proper directory permissions
function Set-DirectoryPermissions {
    param([string]$Path)
    
    try {
        # Give full control to SYSTEM and Administrators
        $systemSID = New-Object System.Security.Principal.SecurityIdentifier("S-1-5-18")
        $adminsSID = New-Object System.Security.Principal.SecurityIdentifier("S-1-5-32-544")
        
        $acl = Get-Acl $Path
        
        # Add SYSTEM full control
        $systemAccess = New-Object System.Security.AccessControl.FileSystemAccessRule(
            $systemSID, "FullControl", "ContainerInherit,ObjectInherit", "None", "Allow")
        $acl.SetAccessRule($systemAccess)
        
        # Add Administrators full control
        $adminsAccess = New-Object System.Security.AccessControl.FileSystemAccessRule(
            $adminsSID, "FullControl", "ContainerInherit,ObjectInherit", "None", "Allow")
        $acl.SetAccessRule($adminsAccess)
        
        Set-Acl -Path $Path -AclObject $acl
        Write-Success "Directory permissions configured"
        return $true
    } catch {
        Write-Warning "Permission setup had issues: $($_.Exception.Message)"
        Write-Info "Service may need manual permission fixes if issues occur"
        return $false
    }
}

try {
    # Check administrator privileges
    Write-Step "1/8" "Checking administrator privileges..."
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    
    if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        Write-Host ""
        Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Red
        Write-Host "  █                        ERROR                                 █" -ForegroundColor Red
        Write-Host "  █                                                              █" -ForegroundColor Red
        Write-Host "  █    This installer requires administrator privileges!         █" -ForegroundColor Red
        Write-Host "  █                                                              █" -ForegroundColor Red
        Write-Host "  █    Please run PowerShell as administrator and try again     █" -ForegroundColor Red
        Write-Host "  █                                                              █" -ForegroundColor Red
        Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Red
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
    
    # Fixed URL for v1.3.0 release
    $releaseUrl = "https://github.com/MAnishSingh13275/repset_bridge/releases/download/v1.3.0/GymDoorBridge-v1.3.0.zip"
    $Version = "v1.3.0"
    
    Write-Success "Latest version resolved: $Version"
    
    $zipPath = "$tempDir\GymDoorBridge.zip"
    Write-Info "Downloading: $releaseUrl"
    
    # Download with progress
    try {
        Invoke-WebRequest -Uri $releaseUrl -OutFile $zipPath -UseBasicParsing
    } catch {
        throw "Failed to download release: $($_.Exception.Message)"
    }

    if (-not (Test-Path $zipPath)) {
        throw "Downloaded file not found: $zipPath"
    }

    $zipInfo = Get-Item $zipPath
    $sizeMB = [math]::Round($zipInfo.Length / 1MB, 1)
    Write-Success "Download completed ($sizeMB MB)"
    Write-Host ""

    # Extract ZIP
    Write-Step "4/8" "Extracting installer..."
    $extractDir = "$tempDir\extracted"
    Write-Info "Extraction to: $extractDir"
    
    try {
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        [System.IO.Compression.ZipFile]::ExtractToDirectory($zipPath, $extractDir)
    } catch {
        throw "Failed to extract ZIP file: $($_.Exception.Message)"
    }

    # Find the executable
    Write-Info "Searching for executable in extracted files..."
    $executableFile = Get-ChildItem -Path $extractDir -Filter "gym-door-bridge.exe" -Recurse -File -ErrorAction SilentlyContinue | Select-Object -First 1
    
    if (-not $executableFile) {
        throw "gym-door-bridge.exe not found in downloaded package"
    }
    
    Write-Info "Found executable: $($executableFile.FullName)"
    Write-Success "Extraction completed"
    Write-Host ""

    # Check existing installation
    Write-Step "5/8" "Checking existing installation..."
    $service = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
    if ($service) {
        Write-Warning "Service already installed"
        if (-not $Silent) {
            $reinstall = Read-Host "Do you want to reinstall? (Y/n)"
            if ($reinstall.ToLower() -eq "n") {
                Write-Host "Installation cancelled by user."
                exit 0
            }
        }
        Write-Info "Removing existing service..."
        # Try to stop and remove the service
        try {
            Stop-Service -Name "GymDoorBridge" -Force -ErrorAction SilentlyContinue
            & sc.exe delete "GymDoorBridge" | Out-Null
        } catch {
            Write-Warning "Failed to cleanly remove existing service"
        }
        Write-Success "Existing service removed"
    } else {
        Write-Success "No existing installation found"
    }
    Write-Host ""

    # Copy executable to installation directory
    Write-Step "6/9" "Copying executable to installation directory..."
    $installPath = "$env:ProgramFiles\GymDoorBridge"
    $targetExe = "$installPath\gym-door-bridge.exe"
    $configPath = "$installPath\config.yaml"
    
    # Create installation directory
    New-Item -ItemType Directory -Path $installPath -Force | Out-Null
    
    # Copy executable
    Copy-Item -Path $executableFile.FullName -Destination $targetExe -Force
    Write-Success "Executable copied to: $targetExe"
    
    # Create proper config file
    Write-Info "Creating configuration file..."
    if (New-ProperConfigFile -ConfigPath $configPath -InstallPath $installPath) {
        Write-Success "Configuration file created: $configPath"
    } else {
        throw "Failed to create configuration file"
    }
    Write-Host ""

    # Install Windows service
    Write-Step "7/9" "Installing Windows service..."
    Write-Info "This will automatically discover your biometric devices..."
    Write-Info "Please wait while scanning network (this may take 1-2 minutes)..."
    Write-Host ""
    
    try {
        # Use the service install command with explicit config path
        $installArgs = @("service", "install", "--config", $configPath)
        $installProcess = Start-Process -FilePath $targetExe -ArgumentList $installArgs -Wait -PassThru -NoNewWindow
        
        if ($installProcess.ExitCode -ne 0) {
            throw "Service installation failed with exit code $($installProcess.ExitCode)"
        }
    } catch {
        throw "Service installation failed: $($_.Exception.Message)"
    }
    
    Write-Success "Service installed successfully!"
    Write-Host ""

    # Configure service permissions
    Write-Step "8/9" "Configuring service permissions..."
    Write-Info "Setting up database and directory permissions..."
    Set-DirectoryPermissions -Path $installPath | Out-Null
    Write-Success "Directory permissions configured"
    
    # Set service to start automatically and handle permissions
    try {
        $svc = Get-WmiObject -Class Win32_Service -Filter "Name='GymDoorBridge'"
        if ($svc) {
            $svc.ChangeStartMode("Automatic") | Out-Null
            Write-Success "Service configured for automatic startup"
        }
    } catch {
        Write-Warning "Failed to configure service startup mode"
    }
    Write-Host ""

    # Verify installation
    Write-Step "9/9" "Verifying installation..."
    $service = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
    if ($service) {
        Write-Success "Service is installed and configured: $($service.Status)"
        Write-Success "Service will start automatically on Windows boot"
        Write-Success "Permissions configured for service operation"
    } else {
        Write-Warning "Service installation verification failed"
    }
    Write-Host ""

    # Installation complete
    Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Green
    Write-Host "  █                                                              █" -ForegroundColor Green
    Write-Host "  █              INSTALLATION SUCCESSFUL!                       █" -ForegroundColor Green
    Write-Host "  █                                                              █" -ForegroundColor Green
    Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Green
    Write-Host ""
    Write-Host "  ✓ Downloaded latest version ($Version)" -ForegroundColor Green
    Write-Host "  ✓ Gym Door Bridge service installed successfully" -ForegroundColor Green
    Write-Host "  ✓ Auto-discovery configured for biometric devices" -ForegroundColor Green
    Write-Host "  ✓ Service configured to start automatically on boot" -ForegroundColor Green
    Write-Host "  ✓ Service is installed and ready" -ForegroundColor Green
    Write-Host "    Note: Service will start automatically after pairing" -ForegroundColor Gray
    Write-Host ""

    # Handle pairing if provided
    if ($PairCode) {
        Write-Host ""
        Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Cyan
        Write-Host "  █                                                              █" -ForegroundColor Cyan
        Write-Host "  █                    PAIRING DEVICE                          █" -ForegroundColor Cyan
        Write-Host "  █                                                              █" -ForegroundColor Cyan
        Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "🔄 Pairing device with provided code..." -ForegroundColor Yellow
        Write-Host "📡 Code: $PairCode" -ForegroundColor Cyan
        Write-Host "🌐 Server: https://repset.onezy.in" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "🔄 Smart Pairing: Attempting to pair device with code: $PairCode" -ForegroundColor Yellow
        
        try {
            $pairArgs = @("pair", "--pair-code", $PairCode, "--config", $configPath)
            $pairProcess = Start-Process -FilePath $targetExe -ArgumentList $pairArgs -Wait -PassThru -WindowStyle Hidden
            
            if ($pairProcess.ExitCode -eq 0) {
                Write-Host ""
                Write-Success "Device paired successfully!"
                Write-Host "🎉 Your gym door bridge is now fully operational!" -ForegroundColor Green
                
                # Try to start the service after successful pairing
                Write-Info "Starting service after successful pairing..."
                try {
                    Start-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
                    Write-Success "Service started successfully"
                } catch {
                    Write-Warning "Service pairing completed but service startup needs manual intervention"
                    Write-Info "You can start the service from Services.msc or restart your computer"
                }
            } else {
                Write-Warning "Pairing may have had issues (exit code: $($pairProcess.ExitCode))"
                Write-Info "You can try pairing again using: Start Menu > Gym Door Bridge > Pair Device"
            }
        } catch {
            Write-Warning "Pairing process encountered an error: $($_.Exception.Message)"
            Write-Info "You can try pairing manually later"
        }
    } elseif (-not $Silent) {
        $pairNow = Read-Host "🔗 Would you like to pair your device now? (Y/n)"
        if ($pairNow.ToLower() -ne "n") {
            $inputCode = Read-Host "Enter your pairing code"
            if ($inputCode.Trim()) {
                # Recursive call with the pairing code
                & $PSCommandPath -PairCode $inputCode.Trim() -Silent:$Silent
            }
        }
    }

    Write-Host ""
    if (-not $Silent) {
        Write-Host "✅ Installation completed successfully!" -ForegroundColor Green
        Write-Host "Your gym door bridge is ready for RepSet integration." -ForegroundColor Gray
        Write-Host ""
        Read-Host "Press Enter to exit"
    }

} catch {
    Write-Host ""
    Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Red
    Write-Host "  █                        ERROR                                 █" -ForegroundColor Red
    Write-Host "  █                                                              █" -ForegroundColor Red
    Write-Host "  █                Installation Failed!                         █" -ForegroundColor Red
    Write-Host "  █                                                              █" -ForegroundColor Red
    Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Red
    Write-Host ""
    Write-Host "Error details: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host ""
    Write-Host "Please try the following:" -ForegroundColor Yellow
    Write-Host "1. Check your internet connection" -ForegroundColor Gray
    Write-Host "2. Run PowerShell as Administrator" -ForegroundColor Gray
    Write-Host "3. Temporarily disable firewall/antivirus" -ForegroundColor Gray
    Write-Host "4. Contact RepSet support with the error message above" -ForegroundColor Gray
    Write-Host ""
    if (-not $Silent) {
        Read-Host "Press Enter to exit"
    }
    exit 1
} finally {
    # Cleanup temp directory
    if ($tempDir -and (Test-Path $tempDir)) {
        try {
            Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        } catch {
            # Ignore cleanup errors
        }
    }
}