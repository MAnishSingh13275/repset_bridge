# Ultra-Simple Gym Door Bridge Installer
param([string]$PairCode = "")

# Check admin
$user = [Security.Principal.WindowsIdentity]::GetCurrent()
$principal = New-Object Security.Principal.WindowsPrincipal($user)
if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Host "ERROR: Run as Administrator required!" -ForegroundColor Red
    Read-Host "Press ENTER to exit"
    exit 1
}

Clear-Host
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "  GYM DOOR BRIDGE - INSTALLER" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Installing your gym door access system..." -ForegroundColor Green
Write-Host ""

$INSTALL_PATH = "$env:ProgramFiles\GymDoorBridge"
$SERVICE_NAME = "GymDoorBridge"
$DOWNLOAD_URL = "https://github.com/MAnishSingh13275/repset_bridge/releases/download/v2.1.0/gym-door-bridge-windows-v2.0.0-5-gd731fe9.zip"

# Create directories
Write-Host "Creating directories..." -ForegroundColor White
try {
    if (Test-Path $INSTALL_PATH) { Remove-Item -Recurse -Force $INSTALL_PATH }
    New-Item -ItemType Directory -Force -Path $INSTALL_PATH | Out-Null
    New-Item -ItemType Directory -Force -Path "$INSTALL_PATH\logs" | Out-Null
    New-Item -ItemType Directory -Force -Path "$INSTALL_PATH\data" | Out-Null
    New-Item -ItemType Directory -Force -Path "$INSTALL_PATH\config" | Out-Null
    Write-Host "✓ Directories created" -ForegroundColor Green
} catch {
    Write-Host "ERROR: $($_.Exception.Message)" -ForegroundColor Red
    Read-Host "Press ENTER to exit"
    exit 1
}

# Download
Write-Host "Downloading software..." -ForegroundColor White
try {
    $tempZip = "$env:TEMP\gymdoor.zip"
    (New-Object System.Net.WebClient).DownloadFile($DOWNLOAD_URL, $tempZip)
    Write-Host "✓ Download completed" -ForegroundColor Green
} catch {
    Write-Host "ERROR: Download failed - $($_.Exception.Message)" -ForegroundColor Red
    Read-Host "Press ENTER to exit"
    exit 1
}

# Extract
Write-Host "Extracting files..." -ForegroundColor White
try {
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $tempDir = "$env:TEMP\gymdoor"
    if (Test-Path $tempDir) { Remove-Item -Recurse -Force $tempDir }
    [System.IO.Compression.ZipFile]::ExtractToDirectory($tempZip, $tempDir)
    
    $sourceExe = "$tempDir\gym-door-bridge-windows.exe"
    $targetExe = "$INSTALL_PATH\gym-door-bridge.exe"
    Copy-Item -Path $sourceExe -Destination $targetExe -Force
    
    Write-Host "✓ Files installed" -ForegroundColor Green
} catch {
    Write-Host "ERROR: Extraction failed - $($_.Exception.Message)" -ForegroundColor Red
    Read-Host "Press ENTER to exit"
    exit 1
}

# Create simple config
Write-Host "Creating configuration..." -ForegroundColor White
try {
    $configPath = "$INSTALL_PATH\config\config.yaml"
    $pathForConfig = $INSTALL_PATH -replace '\\', '/'
    
    # Build config line by line to avoid parsing issues
    $lines = @()
    $lines += "server_url: https://repset.onezy.in"
    $lines += "log_level: info" 
    $lines += "log_file: $pathForConfig/logs/bridge.log"
    $lines += "database_path: $pathForConfig/data/bridge.db"
    $lines += "device_id: """""
    $lines += "device_key: """""
    $lines += "tier: normal"
    $lines += "queue_max_size: 10000"
    $lines += "heartbeat_interval: 60"
    $lines += "unlock_duration: 3000"
    $lines += "enabled_adapters:"
    $lines += "  - simulator"
    $lines += "adapter_configs:"
    $lines += "  simulator:"
    $lines += "    device_type: simulator"
    $lines += "    connection: memory"
    $lines += "    sync_interval: 10"
    
    $lines | Out-File -FilePath $configPath -Encoding UTF8
    Write-Host "✓ Configuration created" -ForegroundColor Green
} catch {
    Write-Host "ERROR: Config creation failed - $($_.Exception.Message)" -ForegroundColor Red
    Read-Host "Press ENTER to exit"
    exit 1
}

# Install service
Write-Host "Installing Windows service..." -ForegroundColor White
try {
    # Remove existing
    $existing = Get-Service -Name $SERVICE_NAME -ErrorAction SilentlyContinue
    if ($existing) {
        Stop-Service -Name $SERVICE_NAME -Force -ErrorAction SilentlyContinue
        & sc.exe delete $SERVICE_NAME | Out-Null
        Start-Sleep 2
    }
    
    # Create new service
    $binPath = "`"$targetExe`" --config `"$configPath`""
    $result = & sc.exe create $SERVICE_NAME binPath= $binPath DisplayName= "Gym Door Access Bridge" start= auto
    
    if ($LASTEXITCODE -eq 0) {
        & sc.exe description $SERVICE_NAME "Gym door access bridge" | Out-Null
        Write-Host "✓ Service installed" -ForegroundColor Green
        
        try {
            Start-Service -Name $SERVICE_NAME -ErrorAction SilentlyContinue
            Write-Host "✓ Service started" -ForegroundColor Green
        } catch {
            Write-Host "Service will start after pairing" -ForegroundColor Yellow
        }
    } else {
        Write-Host "Service install failed - software can run manually" -ForegroundColor Yellow
    }
} catch {
    Write-Host "Service error - software still works manually" -ForegroundColor Yellow
}

# Pairing
if (-not $PairCode) {
    Write-Host ""
    Write-Host "DEVICE PAIRING:" -ForegroundColor Cyan
    Write-Host "Enter pairing code (or ENTER to skip):" -ForegroundColor White
    $PairCode = Read-Host
}

if ($PairCode -and $PairCode.Trim()) {
    Write-Host "Pairing device..." -ForegroundColor White
    try {
        & $targetExe --config $configPath pair $PairCode.Trim()
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✓ Paired successfully!" -ForegroundColor Green
            Restart-Service -Name $SERVICE_NAME -Force -ErrorAction SilentlyContinue
        } else {
            Write-Host "Pair later with: `"$targetExe`" pair CODE" -ForegroundColor Yellow
        }
    } catch {
        Write-Host "Pair later with: `"$targetExe`" pair CODE" -ForegroundColor Yellow
    }
}

# Cleanup
Remove-Item $tempZip -Force -ErrorAction SilentlyContinue
Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue

# Done
Write-Host ""
Write-Host "============================================================" -ForegroundColor Green
Write-Host "  INSTALLATION COMPLETE!" -ForegroundColor Green  
Write-Host "============================================================" -ForegroundColor Green
Write-Host ""
Write-Host "🎉 Gym Door Bridge installed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "Location: $INSTALL_PATH" -ForegroundColor White
Write-Host "Support: support@repset.onezy.in" -ForegroundColor Cyan
Write-Host ""
Write-Host "Your gym door system is ready!" -ForegroundColor Green
Write-Host ""

Write-Host "Press any key to close..." -ForegroundColor Yellow
try { $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown") | Out-Null } catch { Read-Host }