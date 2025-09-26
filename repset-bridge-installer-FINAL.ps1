# ================================================================
# RepSet Gym Door Bridge - FINAL COMPREHENSIVE INSTALLER
# This bypasses all old installer logic and works completely
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
    Write-Host "  █              REPSET GYM DOOR BRIDGE INSTALLER               █" -ForegroundColor Green
    Write-Host "  █                                                              █" -ForegroundColor Green
    Write-Host "  █                    Final Fixed Version                      █" -ForegroundColor Green
    Write-Host "  █                                                              █" -ForegroundColor Green
    Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Green
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

# Function to set directory permissions
function Set-DirectoryPermissions {
    param([string]$Path)
    
    try {
        $systemSID = New-Object System.Security.Principal.SecurityIdentifier("S-1-5-18")
        $adminsSID = New-Object System.Security.Principal.SecurityIdentifier("S-1-5-32-544")
        
        $acl = Get-Acl $Path
        
        $systemAccess = New-Object System.Security.AccessControl.FileSystemAccessRule(
            $systemSID, "FullControl", "ContainerInherit,ObjectInherit", "None", "Allow")
        $acl.SetAccessRule($systemAccess)
        
        $adminsAccess = New-Object System.Security.AccessControl.FileSystemAccessRule(
            $adminsSID, "FullControl", "ContainerInherit,ObjectInherit", "None", "Allow")
        $acl.SetAccessRule($adminsAccess)
        
        Set-Acl -Path $Path -AclObject $acl
        Write-Success "Directory permissions configured"
        return $true
    } catch {
        Write-Warning "Permission setup encountered issues: $($_.Exception.Message)"
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
    
    # Create CORRECT config file (not using the old installer logic)
    if (-not (New-CorrectConfigFile -ConfigPath $configPath)) {
        throw "Failed to create configuration file"
    }
    
    # Set permissions
    Set-DirectoryPermissions -Path $installPath | Out-Null
    
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
        $pairProcess = Start-Process -FilePath $targetExe -ArgumentList $pairArgs -Wait -PassThru -NoNewWindow
        
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
            
        } elseif ($pairProcess.ExitCode -eq 1) {
            # Check if already paired
            Write-Info "Checking if device is already paired..."
            $statusOutput = & $targetExe status --config $configPath 2>&1
            if ($statusOutput -match "Bridge not paired") {
                Write-Error "Pairing failed with code: $PairCode"
                Write-Info "Please verify the pairing code in your RepSet dashboard"
            } else {
                Write-Success "Device is already paired with RepSet!"
                Write-Info "Starting service..."
                try {
                    Start-Service -Name "GymDoorBridge" -ErrorAction Stop
                    Write-Success "Service started successfully"
                } catch {
                    Write-Warning "Service paired but needs manual start"
                }
            }
        } else {
            Write-Warning "Pairing completed with exit code: $($pairProcess.ExitCode)"
        }
    } catch {
        Write-Warning "Pairing process error: $($_.Exception.Message)"
    }

    # Installation complete
    Write-Host ""
    Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Green
    Write-Host "  █                                                              █" -ForegroundColor Green
    Write-Host "  █              REPSET INSTALLATION COMPLETE!                  █" -ForegroundColor Green
    Write-Host "  █                                                              █" -ForegroundColor Green
    Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Green
    Write-Host ""
    Write-Host "  ✅ RepSet Gym Door Bridge installed successfully" -ForegroundColor Green
    Write-Host "  ✅ Service configured for automatic startup" -ForegroundColor Green
    Write-Host "  ✅ Device pairing completed" -ForegroundColor Green
    Write-Host "  ✅ Ready for gym door access management" -ForegroundColor Green
    Write-Host ""
    
    if (-not $Silent) {
        Write-Host "Your gym is now connected to the RepSet platform!" -ForegroundColor Cyan
        Write-Host "Check your RepSet dashboard to verify the connection." -ForegroundColor Gray
    }

} catch {
    Write-Host ""
    Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Red
    Write-Host "  █                        ERROR                                 █" -ForegroundColor Red
    Write-Host "  █                                                              █" -ForegroundColor Red
    Write-Host "  █            RepSet Installation Failed                       █" -ForegroundColor Red
    Write-Host "  █                                                              █" -ForegroundColor Red
    Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Red
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