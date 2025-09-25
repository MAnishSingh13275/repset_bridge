# RepSet Bridge Administration Tool
param(
    [string]$Action = "help",
    [string]$PairCode = ""
)

$BRIDGE_EXE = "gym-door-bridge.exe"
$CONFIG_FILE = "config.yaml"
$PRODUCTION_SERVER = "https://repset.onezy.in"

function Write-Success { param([string]$msg) Write-Host "✓ $msg" -ForegroundColor Green }
function Write-Warning { param([string]$msg) Write-Host "⚠ $msg" -ForegroundColor Yellow }
function Write-Error { param([string]$msg) Write-Host "✗ $msg" -ForegroundColor Red }
function Write-Info { param([string]$msg) Write-Host "ℹ $msg" -ForegroundColor Cyan }

function Write-Header {
    param([string]$title)
    Write-Host ""
    Write-Host ("=" * 50) -ForegroundColor Cyan
    Write-Host " $title" -ForegroundColor Cyan
    Write-Host ("=" * 50) -ForegroundColor Cyan
    Write-Host ""
}

function Test-Prerequisites {
    if (-not (Test-Path $BRIDGE_EXE)) {
        Write-Error "Bridge executable not found: $BRIDGE_EXE"
        return $false
    }
    if (-not (Test-Path $CONFIG_FILE)) {
        Write-Error "Config file not found: $CONFIG_FILE"
        return $false
    }
    return $true
}

function Get-BridgeProcess {
    Get-Process -Name "gym-door-bridge" -ErrorAction SilentlyContinue
}

function Test-BridgeHealth {
    try {
        $health = Invoke-RestMethod -Uri "http://localhost:8080/health" -TimeoutSec 3
        return @{ Success = $true; Data = $health }
    } catch {
        return @{ Success = $false; Error = $_.Exception.Message }
    }
}

function Start-PairProcess {
    Write-Header "PAIR BRIDGE WITH PRODUCTION"
    
    if (-not (Test-Prerequisites)) { return }
    
    if ([string]::IsNullOrWhiteSpace($PairCode)) {
        Write-Info "To get your pair code:"
        Write-Host "1. Go to: $PRODUCTION_SERVER/{gymId}/admin/dashboard"
        Write-Host "2. Navigate to Bridge Management section"
        Write-Host "3. Create bridge deployment if needed"
        Write-Host "4. Copy the Pair Code (shown in blue)"
        Write-Host ""
        $PairCode = Read-Host "Enter your production pair code"
    }
    
    if ([string]::IsNullOrWhiteSpace($PairCode)) {
        Write-Error "No pair code provided"
        return
    }
    
    Write-Info "Pairing bridge with production server..."
    Write-Host "Pair Code: $PairCode" -ForegroundColor Cyan
    Write-Host "Server: $PRODUCTION_SERVER" -ForegroundColor Cyan
    
    try {
        $result = & ".\$BRIDGE_EXE" pair --pair-code $PairCode --config $CONFIG_FILE 2>&1
        Write-Host ""
        Write-Info "Pairing Result:"
        Write-Host $result
        
        if ($result -match "success|paired") {
            Write-Success "PAIRING SUCCESSFUL!"
            Write-Info "You can now run: .\bridge-admin.ps1 install"
        } else {
            Write-Error "Pairing may have failed. Check output above."
        }
    } catch {
        Write-Error "Pairing command failed: $($_.Exception.Message)"
    }
}

function Start-InstallProcess {
    Write-Header "INSTALL AND START BRIDGE"
    
    if (-not (Test-Prerequisites)) { return }
    
    # Stop existing
    $process = Get-BridgeProcess
    if ($process) {
        Write-Info "Stopping existing bridge..."
        $process.Kill()
        Start-Sleep 2
    }
    
    # Backup database
    if (Test-Path "bridge.db") {
        $backup = "bridge.db.backup.$(Get-Date -Format 'yyyyMMdd-HHmmss')"
        Rename-Item "bridge.db" $backup
        Write-Success "Database backed up to: $backup"
        Remove-Item "bridge.db-*" -ErrorAction SilentlyContinue
    }
    
    # Start bridge
    Write-Info "Starting bridge as background process..."
    Start-Process -FilePath ".\$BRIDGE_EXE" -ArgumentList "--config", $CONFIG_FILE -WindowStyle Hidden
    Start-Sleep 3
    
    # Check health
    $health = Test-BridgeHealth
    if ($health.Success) {
        Write-Success "Bridge started successfully!"
        Write-Success "Device ID: $($health.Data.deviceId)"
        Write-Success "Status: $($health.Data.status)"
        Write-Success "Version: $($health.Data.version)"
        Write-Info "Health endpoint: http://localhost:8080/health"
        Write-Info "API endpoint: http://localhost:8081"
    } else {
        Write-Error "Bridge health check failed: $($health.Error)"
    }
}

function Show-Status {
    Write-Header "BRIDGE STATUS"
    
    # Process check
    $process = Get-BridgeProcess
    if ($process) {
        Write-Success "Bridge process is running"
        Write-Host "  Process ID: $($process.Id)"
        Write-Host "  Memory: $([math]::Round($process.WorkingSet64 / 1MB, 1)) MB"
    } else {
        Write-Error "Bridge process is not running"
    }
    
    # Service check
    $service = Get-Service -Name "*bridge*" -ErrorAction SilentlyContinue
    if ($service) {
        Write-Info "Windows Service: $($service.Status)"
    } else {
        Write-Info "Windows Service: Not installed"
    }
    
    # Health check
    $health = Test-BridgeHealth
    if ($health.Success) {
        Write-Success "Bridge is healthy and responding"
        $data = $health.Data
        Write-Host "  Device ID: $($data.deviceId)"
        Write-Host "  Status: $($data.status)"
        Write-Host "  Version: $($data.version)"
        Write-Host "  Uptime: $([math]::Round($data.uptime / 1000000000, 0)) seconds"
        Write-Host "  Performance Tier: $($data.tier)"
    } else {
        Write-Error "Bridge health check failed: $($health.Error)"
    }
}

function Show-Health {
    Write-Header "BRIDGE HEALTH CHECK"
    
    $health = Test-BridgeHealth
    if ($health.Success) {
        Write-Success "Bridge is healthy and responding"
        Write-Host ""
        $data = $health.Data
        Write-Host "Health Details:" -ForegroundColor Cyan
        Write-Host "- Device ID: $($data.deviceId)"
        Write-Host "- Status: $($data.status)"
        Write-Host "- Version: $($data.version)"
        Write-Host "- Performance Tier: $($data.tier)"
        Write-Host "- Uptime: $([math]::Round($data.uptime / 1000000000, 0)) seconds"
        
        # Test API
        try {
            Invoke-WebRequest -Uri "http://localhost:8081" -TimeoutSec 3 | Out-Null
            Write-Success "API server responding on port 8081"
        } catch {
            Write-Warning "API server on port 8081 not responding"
        }
    } else {
        Write-Error "Bridge is not responding"
        Write-Host "Error: $($health.Error)"
        Write-Info "Try: .\bridge-admin.ps1 start"
    }
}

function Start-Bridge {
    Write-Header "START BRIDGE"
    $process = Get-BridgeProcess
    if ($process) {
        Write-Warning "Bridge is already running (PID: $($process.Id))"
    } else {
        Start-InstallProcess
    }
}

function Stop-Bridge {
    Write-Header "STOP BRIDGE"
    $process = Get-BridgeProcess
    if ($process) {
        Write-Info "Stopping bridge process (PID: $($process.Id))..."
        $process.Kill()
        Start-Sleep 2
        Write-Success "Bridge stopped"
    } else {
        Write-Warning "Bridge is not running"
    }
}

function Restart-Bridge {
    Write-Header "RESTART BRIDGE"
    Stop-Bridge
    Start-Sleep 2
    Start-Bridge
}

function Show-Help {
    Write-Host ""
    Write-Host "REPSET BRIDGE ADMIN TOOL" -ForegroundColor Cyan
    Write-Host "========================" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "USAGE:" -ForegroundColor Yellow
    Write-Host "  .\bridge-admin.ps1 [ACTION] [-PairCode CODE]"
    Write-Host ""
    Write-Host "ACTIONS:" -ForegroundColor Yellow
    Write-Host "  pair         Pair bridge with production server"
    Write-Host "  install      Install and start the bridge"
    Write-Host "  start        Start the bridge"
    Write-Host "  stop         Stop the bridge"
    Write-Host "  restart      Restart the bridge"
    Write-Host "  status       Show bridge status"
    Write-Host "  health       Detailed health check"
    Write-Host "  help         Show this help"
    Write-Host ""
    Write-Host "EXAMPLES:" -ForegroundColor Yellow
    Write-Host "  .\bridge-admin.ps1 pair -PairCode 'ABCD-1234-EFGH'" -ForegroundColor Gray
    Write-Host "  .\bridge-admin.ps1 install" -ForegroundColor Gray
    Write-Host "  .\bridge-admin.ps1 status" -ForegroundColor Gray
    Write-Host ""
    Write-Host "SETUP WORKFLOW:" -ForegroundColor Green
    Write-Host "1. Get pair code from: $PRODUCTION_SERVER/{gymId}/admin/dashboard"
    Write-Host "2. .\bridge-admin.ps1 pair"
    Write-Host "3. .\bridge-admin.ps1 install"
    Write-Host "4. .\bridge-admin.ps1 status"
    Write-Host ""
}

# Main execution
switch ($Action.ToLower()) {
    "pair"     { Start-PairProcess }
    "install"  { Start-InstallProcess }
    "start"    { Start-Bridge }
    "stop"     { Stop-Bridge }
    "restart"  { Restart-Bridge }
    "status"   { Show-Status }
    "health"   { Show-Health }
    default    { Show-Help }
}