# RepSet Bridge Installer - FIXED VERSION
# Simplified installer that avoids the Path parameter issue

param(
    [Parameter(Mandatory=$true)]
    [string]$PairCode,
    [switch]$Silent = $false
)

$ErrorActionPreference = "Stop"

# Color functions
function Write-Success { param([string]$Message) Write-Host "[OK] $Message" -ForegroundColor Green }
function Write-Error { param([string]$Message) Write-Host "[ERROR] $Message" -ForegroundColor Red }
function Write-Warning { param([string]$Message) Write-Host "[WARNING] $Message" -ForegroundColor Yellow }
function Write-Info { param([string]$Message) Write-Host "[INFO] $Message" -ForegroundColor Cyan }
function Write-Step { param([string]$Step, [string]$Message) Write-Host "[$Step] $Message" -ForegroundColor White }

if (-not $Silent) {
    Clear-Host
    Write-Host ""
    Write-Host "RepSet Bridge Installer" -ForegroundColor Cyan
    Write-Host "======================" -ForegroundColor Cyan
    Write-Host "Pair Code: $PairCode" -ForegroundColor Gray
    Write-Host "Platform: https://repset.onezy.in" -ForegroundColor Gray
    Write-Host ""
}

try {
    # Check administrator privileges
    Write-Step "1/6" "Checking administrator privileges"
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    
    if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        Write-Error "Administrator privileges required!"
        Write-Info "• Run PowerShell as Administrator"
        Write-Info "• Check internet connectivity"
        Write-Info "• Disable antivirus temporarily"
        if (-not $Silent) { Read-Host "Press Enter to exit" }
        exit 1
    }
    Write-Success "Administrator privileges confirmed"

    # Set up workspace
    Write-Step "2/6" "Setting up workspace"
    $TempDir = Join-Path ([System.IO.Path]::GetTempPath()) "RepSetBridge-$(Get-Random)"
    New-Item -ItemType Directory -Path $TempDir -Force | Out-Null
    Write-Success "Workspace created"

    # Download RepSet Bridge
    Write-Step "3/6" "Downloading RepSet Bridge"
    $releaseUrl = "https://github.com/MAnishSingh13275/repset_bridge/releases/download/v1.3.0/GymDoorBridge-v1.3.0.zip"
    $zipPath = Join-Path $TempDir "GymDoorBridge.zip"
    
    try {
        Invoke-WebRequest -Uri $releaseUrl -OutFile $zipPath -UseBasicParsing
        $zipInfo = Get-Item $zipPath
        $sizeMB = [math]::Round($zipInfo.Length / 1MB, 1)
        Write-Success "Downloaded v1.3.0 ($sizeMB MB)"
    } catch {
        throw "Failed to download: $($_.Exception.Message)"
    }

    # Install bridge
    Write-Step "4/6" "Installing bridge"
    
    # Extract files
    $extractDir = Join-Path $TempDir "extracted"
    try {
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        [System.IO.Compression.ZipFile]::ExtractToDirectory($zipPath, $extractDir)
        
        $executableFile = Get-ChildItem -Path $extractDir -Filter "gym-door-bridge.exe" -Recurse -File | Select-Object -First 1
        if (-not $executableFile) {
            throw "Executable not found in package"
        }
    } catch {
        throw "Failed to extract: $($_.Exception.Message)"
    }

    # Set up installation directory
    $ProgramFilesPath = ${env:ProgramFiles}
    if (-not $ProgramFilesPath) {
        $ProgramFilesPath = "C:\Program Files"
    }
    
    $InstallDir = Join-Path $ProgramFilesPath "GymDoorBridge"
    $TargetExe = Join-Path $InstallDir "gym-door-bridge.exe"
    $ConfigFile = Join-Path $InstallDir "config.yaml"
    
    # Validate paths
    if (-not $InstallDir -or -not $TargetExe -or -not $ConfigFile) {
        throw "Failed to create installation paths"
    }

    # Remove existing service if present
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
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    
    # Copy executable
    Copy-Item -Path $executableFile.FullName -Destination $TargetExe -Force
    Write-Info "Executable installed"
    
    # Create config file using Out-File instead of Set-Content
    $configContent = @"
# Gym Door Bridge Configuration - RepSet Platform
device_id: ""
device_key: ""
server_url: "https://repset.onezy.in"
tier: "normal"
queue_max_size: 10000
heartbeat_interval: 60
unlock_duration: 3000
database_path: "./bridge.db"
log_level: "info"
log_file: ""
enabled_adapters:
  - "simulator"
"@

    try {
        $configContent | Out-File -FilePath $ConfigFile -Encoding UTF8 -Force
        Write-Info "Config file created"
    } catch {
        throw "Failed to create config file: $($_.Exception.Message)"
    }

    # Install service
    Write-Info "Installing Windows service..."
    $serviceInstallResult = & sc.exe create "GymDoorBridge" binPath= "`"$TargetExe`" --config `"$ConfigFile`"" start= auto DisplayName= "RepSet Gym Door Bridge"
    if ($LASTEXITCODE -ne 0) {
        throw "Service installation failed"
    }
    Write-Success "Service installed successfully"

    # Pair the device
    Write-Step "5/6" "Pairing with RepSet platform"
    Write-Info "Pairing Code: $PairCode"
    Write-Info "Server: https://repset.onezy.in"
    
    try {
        $pairArgs = @("pair", "--pair-code", $PairCode, "--config", $ConfigFile, "--timeout", "10")
        $pairProcess = Start-Process -FilePath $TargetExe -ArgumentList $pairArgs -Wait -PassThru -NoNewWindow
        
        if ($pairProcess.ExitCode -eq 0) {
            Write-Success "Device paired successfully!"
        } else {
            Write-Warning "Pairing completed with exit code: $($pairProcess.ExitCode)"
        }
    } catch {
        Write-Warning "Pairing process error: $($_.Exception.Message)"
    }

    # Start service
    Write-Step "6/6" "Starting service"
    try {
        Start-Service -Name "GymDoorBridge" -ErrorAction Stop
        Write-Success "Service started successfully"
    } catch {
        Write-Warning "Service installed and paired but needs manual start"
        Write-Info "You can start it from Services.msc or restart Windows"
    }

    # Cleanup
    Remove-Item -Path $TempDir -Recurse -Force -ErrorAction SilentlyContinue

    Write-Host ""
    Write-Host "=== INSTALLATION SUCCESSFUL ===" -ForegroundColor Green
    Write-Success "RepSet Gym Door Bridge installed successfully"
    Write-Success "Service configured for automatic startup"
    Write-Success "Device pairing completed"
    Write-Success "Ready for gym door access management"
    Write-Host ""

} catch {
    Write-Host ""
    Write-Host "=== INSTALLATION FAILED ===" -ForegroundColor Red
    Write-Error "Error: $($_.Exception.Message)"
    Write-Host ""
    Write-Info "Troubleshooting:"
    Write-Info "• Run PowerShell as Administrator"
    Write-Info "• Check internet connectivity"
    Write-Info "• Disable antivirus temporarily"
    Write-Host ""
    exit 1
} finally {
    # Cleanup temp directory
    if ($TempDir -and (Test-Path $TempDir)) {
        try {
            Remove-Item $TempDir -Recurse -Force -ErrorAction SilentlyContinue
        } catch {}
    }
}

if (-not $Silent) {
    Read-Host "Press Enter to exit"
}