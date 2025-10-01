# RepSet Bridge Simple Installer - GitHub Raw Version
# Direct download and install from GitHub release v1.4.0

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
    Write-Host "RepSet Bridge Installer v1.4.0" -ForegroundColor Cyan
    Write-Host "==============================" -ForegroundColor Cyan
    Write-Host "Pair Code: $PairCode" -ForegroundColor Gray
    Write-Host "Platform: https://repset.onezy.in" -ForegroundColor Gray
    Write-Host "Version: v1.4.0" -ForegroundColor Gray
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

    # Download RepSet Bridge from GitHub release
    Write-Step "3/6" "Downloading RepSet Bridge v1.4.0"
    $exePath = Join-Path $TempDir "gym-door-bridge.exe"
    $downloadUrl = "https://github.com/MAnishSingh13275/repset_bridge/releases/download/v1.4.0/gym-door-bridge.exe"
    
    try {
        Write-Info "Downloading from GitHub release..."
        Invoke-WebRequest -Uri $downloadUrl -OutFile $exePath -UseBasicParsing
        $exeInfo = Get-Item $exePath
        $sizeMB = [math]::Round($exeInfo.Length / 1MB, 1)
        Write-Success "Downloaded v1.4.0 ($sizeMB MB)"
    } catch {
        throw "Failed to download RepSet Bridge: $($_.Exception.Message)"
    }

    # Verify download
    if (-not (Test-Path $exePath) -or (Get-Item $exePath).Length -eq 0) {
        throw "Downloaded file is invalid or empty"
    }

    # Install bridge
    Write-Step "4/6" "Installing bridge"
    
    # Set up installation directory
    $ProgramFilesPath = ${env:ProgramFiles}
    if (-not $ProgramFilesPath -or $ProgramFilesPath.Trim() -eq "") {
        $ProgramFilesPath = "C:\Program Files"
    }
    
    $InstallDir = Join-Path $ProgramFilesPath "GymDoorBridge"
    $TargetExe = Join-Path $InstallDir "gym-door-bridge.exe"
    $ConfigFile = Join-Path $InstallDir "config.yaml"

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
    
    # Create installation directory and copy executable
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Copy-Item -Path $exePath -Destination $TargetExe -Force
    Write-Info "Executable installed"
    
    # Create config file
    $configContent = @"
# Gym Door Bridge Configuration - RepSet Platform v1.4.0
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

    $configContent | Out-File -FilePath $ConfigFile -Encoding UTF8 -Force
    Write-Info "Config file created"

    # Install Windows service
    Write-Step "5/6" "Installing Windows service"
    try {
        $serviceArgs = @("create", "GymDoorBridge", "binPath=", "`"$TargetExe`" --config `"$ConfigFile`"", "start=", "auto", "DisplayName=", "RepSet Gym Door Bridge")
        $serviceResult = & sc.exe @serviceArgs 2>&1
        if ($LASTEXITCODE -ne 0) {
            throw "Service installation failed: $serviceResult"
        }
        Write-Success "Service installed successfully"
    } catch {
        throw "Service installation error: $($_.Exception.Message)"
    }

    # Pair the device
    Write-Step "6/6" "Pairing with RepSet platform"
    Write-Info "Pairing Code: $PairCode"
    Write-Info "Server: https://repset.onezy.in"
    
    try {
        $pairArgs = @("pair", "--pair-code", $PairCode, "--config", $ConfigFile, "--timeout", "10")
        $pairProcess = Start-Process -FilePath $TargetExe -ArgumentList $pairArgs -Wait -PassThru -NoNewWindow -RedirectStandardOutput "$TempDir\pair_output.txt" -RedirectStandardError "$TempDir\pair_error.txt"
        
        if ($pairProcess.ExitCode -eq 0) {
            Write-Success "Device paired successfully!"
        } else {
            # Check if already paired
            $statusArgs = @("status", "--config", $ConfigFile)
            $statusProcess = Start-Process -FilePath $TargetExe -ArgumentList $statusArgs -Wait -PassThru -NoNewWindow -RedirectStandardOutput "$TempDir\status_output.txt" -RedirectStandardError "$TempDir\status_error.txt"
            
            if ($statusProcess.ExitCode -eq 0) {
                $statusOutput = Get-Content "$TempDir\status_output.txt" -ErrorAction SilentlyContinue
                if ($statusOutput -and $statusOutput -notmatch "Bridge not paired") {
                    Write-Success "Device is already paired with RepSet!"
                } else {
                    Write-Warning "Pairing may have failed. Please verify pairing code: $PairCode"
                }
            } else {
                Write-Warning "Pairing completed with exit code: $($pairProcess.ExitCode)"
            }
        }
    } catch {
        Write-Warning "Pairing process error: $($_.Exception.Message)"
        Write-Info "You can pair manually later using the RepSet dashboard"
    }

    # Start service
    try {
        Start-Service -Name "GymDoorBridge" -ErrorAction Stop
        Write-Success "Service started successfully"
    } catch {
        Write-Warning "Service installed and paired but needs manual start"
        Write-Info "You can start it from Services.msc or restart Windows"
    }

    Write-Host ""
    Write-Host "=== INSTALLATION SUCCESSFUL ===" -ForegroundColor Green
    Write-Success "RepSet Gym Door Bridge v1.4.0 installed successfully"
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
    Write-Info "• Verify pairing code is correct"
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