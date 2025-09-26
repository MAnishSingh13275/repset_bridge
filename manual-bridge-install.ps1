# ================================================================
# RepSet Bridge - MANUAL INSTALLATION (No Service Complications)
# ================================================================

param(
    [Parameter(Mandatory=$true)]
    [string]$PairCode
)

$ErrorActionPreference = "Continue"  # Don't stop on errors

# Color functions
function Write-Success { param([string]$Message) Write-Host "✅ $Message" -ForegroundColor Green }
function Write-Error { param([string]$Message) Write-Host "❌ $Message" -ForegroundColor Red }
function Write-Warning { param([string]$Message) Write-Host "⚠️  $Message" -ForegroundColor Yellow }
function Write-Info { param([string]$Message) Write-Host "ℹ️  $Message" -ForegroundColor Cyan }
function Write-Step { param([string]$Step, [string]$Message) Write-Host "[$Step] $Message" -ForegroundColor White }

Write-Host "=================================================" -ForegroundColor Blue
Write-Host "    RepSet Bridge Manual Installer" -ForegroundColor Blue  
Write-Host "    Pair Code: $PairCode" -ForegroundColor Blue
Write-Host "=================================================" -ForegroundColor Blue
Write-Host ""

try {
    # Step 1: Check privileges
    Write-Step "1/6" "Checking administrator privileges..."
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    
    if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        Write-Warning "Not running as administrator - some features may not work"
        Write-Info "For best results, run PowerShell as Administrator"
    } else {
        Write-Success "Running with administrator privileges"
    }

    # Step 2: Setup workspace
    Write-Step "2/6" "Setting up workspace..."
    $tempDir = "$env:TEMP\RepSetBridge-Manual-$(Get-Random)"
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
    Write-Success "Workspace created: $tempDir"

    # Step 3: Download bridge
    Write-Step "3/6" "Downloading RepSet Bridge..."
    $releaseUrl = "https://github.com/MAnishSingh13275/repset_bridge/releases/download/v1.3.0/GymDoorBridge-v1.3.0.zip"
    $zipPath = "$tempDir\GymDoorBridge.zip"
    
    try {
        Invoke-WebRequest -Uri $releaseUrl -OutFile $zipPath -UseBasicParsing
        $zipInfo = Get-Item $zipPath
        $sizeMB = [math]::Round($zipInfo.Length / 1MB, 1)
        Write-Success "Downloaded v1.3.0 ($sizeMB MB)"
    } catch {
        Write-Error "Download failed: $($_.Exception.Message)"
        throw "Cannot continue without bridge executable"
    }

    # Step 4: Extract files
    Write-Step "4/6" "Extracting files..."
    $extractDir = "$tempDir\extracted"
    try {
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        [System.IO.Compression.ZipFile]::ExtractToDirectory($zipPath, $extractDir)
        
        $executableFile = Get-ChildItem -Path $extractDir -Filter "gym-door-bridge.exe" -Recurse -File | Select-Object -First 1
        if (-not $executableFile) {
            throw "Executable not found in downloaded package"
        }
        Write-Success "Bridge executable found: $($executableFile.Name)"
    } catch {
        Write-Error "Extraction failed: $($_.Exception.Message)"
        throw "Cannot continue without bridge executable"
    }

    # Step 5: Install files
    Write-Step "5/6" "Installing bridge files..."
    $installPath = "$env:ProgramFiles\GymDoorBridge"
    $targetExe = "$installPath\gym-door-bridge.exe"
    $configPath = "$installPath\config.yaml"
    
    # Create installation directory
    try {
        New-Item -ItemType Directory -Path $installPath -Force | Out-Null
        Write-Success "Installation directory created: $installPath"
    } catch {
        Write-Warning "Could not create Program Files directory, trying user directory..."
        $installPath = "$env:USERPROFILE\RepSetBridge"
        $targetExe = "$installPath\gym-door-bridge.exe"
        $configPath = "$installPath\config.yaml"
        New-Item -ItemType Directory -Path $installPath -Force | Out-Null
        Write-Success "Using user directory: $installPath"
    }
    
    # Copy executable
    Copy-Item -Path $executableFile.FullName -Destination $targetExe -Force
    Write-Success "Bridge executable installed"
    
    # Create config file
    $configContent = @"
# RepSet Bridge Configuration
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
    
    Set-Content -Path $configPath -Value $configContent -Encoding UTF8
    Write-Success "Configuration file created"

    # Step 6: Pair the bridge
    Write-Step "6/6" "Pairing with RepSet platform..."
    Write-Info "Pair Code: $PairCode"
    Write-Info "Server: https://repset.onezy.in"
    
    try {
        $pairArgs = @("pair", "--pair-code", $PairCode, "--config", $configPath)
        $pairProcess = Start-Process -FilePath $targetExe -ArgumentList $pairArgs -Wait -PassThru -NoNewWindow
        
        if ($pairProcess.ExitCode -eq 0) {
            Write-Success "Bridge paired successfully!"
        } else {
            Write-Warning "Pairing may have failed (exit code: $($pairProcess.ExitCode))"
            Write-Info "This might be OK if the bridge was already paired"
        }
    } catch {
        Write-Warning "Pairing process error: $($_.Exception.Message)"
        Write-Info "You can try pairing manually later"
    }

    # Installation complete
    Write-Host ""
    Write-Host "=================================================" -ForegroundColor Green
    Write-Host "    ✅ INSTALLATION COMPLETE!" -ForegroundColor Green
    Write-Host "=================================================" -ForegroundColor Green
    Write-Host ""
    Write-Success "Bridge installed to: $installPath"
    Write-Success "Configuration: $configPath"
    Write-Success "Executable: $targetExe"
    Write-Host ""
    Write-Info "To start the bridge manually:"
    Write-Host "  & '$targetExe' --config '$configPath'" -ForegroundColor Yellow
    Write-Host ""
    Write-Info "To set up auto-start, run these commands as Administrator:"
    Write-Host "  sc.exe create GymDoorBridge binPath= '`"$targetExe`" --config `"$configPath`"' start= auto" -ForegroundColor Yellow
    Write-Host "  Start-Service -Name GymDoorBridge" -ForegroundColor Yellow
    Write-Host ""

} catch {
    Write-Host ""
    Write-Host "=================================================" -ForegroundColor Red
    Write-Host "    ❌ INSTALLATION FAILED" -ForegroundColor Red
    Write-Host "=================================================" -ForegroundColor Red
    Write-Error "Error: $($_.Exception.Message)"
    Write-Host ""
} finally {
    # Cleanup
    if ($tempDir -and (Test-Path $tempDir)) {
        try {
            Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
            Write-Info "Temporary files cleaned up"
        } catch {}
    }
}

Write-Host ""
Read-Host "Press Enter to continue"