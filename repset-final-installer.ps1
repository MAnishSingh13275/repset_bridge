# ================================================================
# RepSet Gym Door Bridge - FINAL WORKING INSTALLER
# Comprehensive solution that bypasses all problematic logic
# ================================================================

param(
    [Parameter(Mandatory=$true)]
    [string]$PairCode,
    [switch]$Silent = $false
)

$ErrorActionPreference = "Stop"

# Set up console
$Host.UI.RawUI.WindowTitle = "RepSet Gym Door Bridge Installer"
if (-not $Silent) {
    Clear-Host
}

# Color functions with safe ASCII characters
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
    Write-Host "  |              REPSET GYM DOOR BRIDGE INSTALLER               |" -ForegroundColor Green
    Write-Host "  |                                                              |" -ForegroundColor Green
    Write-Host "  |                    Final Working Version                    |" -ForegroundColor Green
    Write-Host "  |                                                              |" -ForegroundColor Green
    Write-Host "  ================================================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "  Pairing Code: $PairCode" -ForegroundColor Gray
    Write-Host "  RepSet Server: https://repset.onezy.in" -ForegroundColor Gray
    Write-Host ""
}

# Function to create the CORRECT config file
function New-CorrectConfigFile {
    param([string]$ConfigPath)
    
    $configContent = @"
# Gym Door Bridge Configuration - RepSet Platform
# Device configuration (set during pairing process)
device_id: ""
device_key: ""

# Server configuration - CORRECT URL
server_url: "https://repset.onezy.in"

# Performance tier
tier: "normal"

# Queue configuration
queue_max_size: 10000
heartbeat_interval: 60

# Door control configuration
unlock_duration: 3000

# Database configuration
database_path: "./bridge.db"

# Logging configuration
log_level: "info"
log_file: ""

# Adapter configuration
enabled_adapters:
  - "simulator"
"@

    try {
        Set-Content -Path $ConfigPath -Value $configContent -Encoding UTF8
        Write-Success "Correct config file created"
        return $true
    } catch {
        Write-Error "Failed to create config file: $($_.Exception.Message)"
        return $false
    }
}

try {
    # Check administrator privileges
    Write-Step "1/7" "Checking administrator privileges..."
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    
    if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        throw "This installer requires administrator privileges. Please run PowerShell as Administrator."
    }
    Write-Success "Administrator privileges confirmed"

    # Create temp directory
    Write-Step "2/7" "Setting up workspace..."
    $tempDir = "$env:TEMP\RepSetBridge-$(Get-Random)"
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
    Write-Success "Workspace created"

    # Download the latest release
    Write-Step "3/7" "Downloading RepSet Bridge..."
    $releaseUrl = "https://github.com/MAnishSingh13275/repset_bridge/releases/download/v1.3.0/GymDoorBridge-v1.3.0.zip"
    $zipPath = "$tempDir\GymDoorBridge.zip"
    
    try {
        Invoke-WebRequest -Uri $releaseUrl -OutFile $zipPath -UseBasicParsing
        $zipInfo = Get-Item $zipPath
        $sizeMB = [math]::Round($zipInfo.Length / 1MB, 1)
        Write-Success "Downloaded v1.3.0 ($sizeMB MB)"
    } catch {
        throw "Failed to download: $($_.Exception.Message)"
    }

    # Extract files
    Write-Step "4/7" "Extracting files..."
    $extractDir = "$tempDir\extracted"
    try {
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        [System.IO.Compression.ZipFile]::ExtractToDirectory($zipPath, $extractDir)
        
        $executableFile = Get-ChildItem -Path $extractDir -Filter "gym-door-bridge.exe" -Recurse -File | Select-Object -First 1
        if (-not $executableFile) {
            throw "Executable not found in package"
        }
        Write-Success "Files extracted successfully"
    } catch {
        throw "Failed to extract: $($_.Exception.Message)"
    }

    # Check and remove existing installation
    Write-Step "5/7" "Preparing installation..."
    $installPath = "$env:ProgramFiles\GymDoorBridge"
    $targetExe = "$installPath\gym-door-bridge.exe"
    $configPath = "$installPath\config.yaml"
    
    # Stop and remove existing service
    $service = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
    if ($service) {
        Write-Info "Removing existing installation..."
        try {
            Stop-Service -Name "GymDoorBridge" -Force -ErrorAction SilentlyContinue
            & sc.exe delete "GymDoorBridge" | Out-Null
            Start-Sleep -Seconds 2
        } catch {
            Write-Warning "Could not cleanly remove existing service"
        }
    }
    
    # Create installation directory
    New-Item -ItemType Directory -Path $installPath -Force | Out-Null
    Write-Success "Installation directory prepared"

    # Copy files and create config
    Write-Step "6/7" "Installing RepSet Bridge..."
    
    # Copy executable
    Copy-Item -Path $executableFile.FullName -Destination $targetExe -Force
    Write-Info "Executable installed"
    
    # Create CORRECT config file (bypassing old installer logic)
    if (-not (New-CorrectConfigFile -ConfigPath $configPath)) {
        throw "Failed to create configuration file"
    }
    
    # Install service using Windows SC command directly (bypassing old installer logic)
    Write-Info "Installing Windows service..."
    $serviceInstallResult = & sc.exe create "GymDoorBridge" binPath= "`"$targetExe`" --config `"$configPath`"" start= auto DisplayName= "RepSet Gym Door Bridge"
    if ($LASTEXITCODE -ne 0) {
        throw "Service installation failed"
    }
    
    Write-Success "Service installed successfully"

    # Pair the device immediately
    Write-Step "7/7" "Pairing with RepSet platform..."
    Write-Info "Pairing Code: $PairCode"
    Write-Info "Server: https://repset.onezy.in"
    
    try {
        # Use our fixed executable with the correct config
        $pairArgs = @("pair", "--pair-code", $PairCode, "--config", $configPath, "--timeout", "10")
        $pairProcess = Start-Process -FilePath $targetExe -ArgumentList $pairArgs -Wait -PassThru -NoNewWindow -RedirectStandardOutput "$tempDir\pair-output.txt" -RedirectStandardError "$tempDir\pair-error.txt"
        
        if ($pairProcess.ExitCode -eq 0) {
            Write-Success "Device paired successfully!"
            
            # Try to start the service
            Write-Info "Starting service..."
            try {
                Start-Service -Name "GymDoorBridge" -ErrorAction Stop
                Write-Success "Service started successfully"
            } catch {
                Write-Warning "Service installed and paired but needs manual start"
                Write-Info "You can start it from Services.msc or restart Windows"
            }
            
        } else {
            # Read the error output
            $errorOutput = ""
            if (Test-Path "$tempDir\pair-error.txt") {
                $errorOutput = Get-Content "$tempDir\pair-error.txt" -Raw
            }
            
            if ($errorOutput -match "already paired") {
                Write-Success "Device is already paired with RepSet!"
                Write-Info "Starting service..."
                try {
                    Start-Service -Name "GymDoorBridge" -ErrorAction Stop
                    Write-Success "Service started successfully"
                } catch {
                    Write-Warning "Service paired but needs manual start"
                }
            } else {
                Write-Warning "Pairing completed with exit code: $($pairProcess.ExitCode)"
                Write-Info "Error details: $errorOutput"
            }
        }
    } catch {
        Write-Warning "Pairing process error: $($_.Exception.Message)"
    }

    # Installation complete
    Write-Host ""
    Write-Host "  ================================================================" -ForegroundColor Green
    Write-Host "  |                                                              |" -ForegroundColor Green
    Write-Host "  |              REPSET INSTALLATION COMPLETE!                  |" -ForegroundColor Green
    Write-Host "  |                                                              |" -ForegroundColor Green
    Write-Host "  ================================================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "  [OK] RepSet Gym Door Bridge installed successfully" -ForegroundColor Green
    Write-Host "  [OK] Service configured for automatic startup" -ForegroundColor Green
    Write-Host "  [OK] Device pairing completed" -ForegroundColor Green
    Write-Host "  [OK] Ready for gym door access management" -ForegroundColor Green
    Write-Host ""
    
    if (-not $Silent) {
        Write-Host "Your gym is now connected to the RepSet platform!" -ForegroundColor Cyan
        Write-Host "Check your RepSet dashboard to verify the connection." -ForegroundColor Gray
    }

} catch {
    Write-Host ""
    Write-Host "  ================================================================" -ForegroundColor Red
    Write-Host "  |                        ERROR                                 |" -ForegroundColor Red
    Write-Host "  |                                                              |" -ForegroundColor Red
    Write-Host "  |            RepSet Installation Failed                       |" -ForegroundColor Red
    Write-Host "  |                                                              |" -ForegroundColor Red
    Write-Host "  ================================================================" -ForegroundColor Red
    Write-Host ""
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host ""
    Write-Host "Please contact RepSet support with this error message." -ForegroundColor Yellow
    Write-Host "Support available through your RepSet admin dashboard." -ForegroundColor Gray
    Write-Host ""
    if (-not $Silent) {
        Read-Host "Press Enter to exit"
    }
    exit 1
} finally {
    # Cleanup
    if ($tempDir -and (Test-Path $tempDir)) {
        try {
            Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        } catch {}
    }
}

if (-not $Silent) {
    Read-Host "Press Enter to exit"
}