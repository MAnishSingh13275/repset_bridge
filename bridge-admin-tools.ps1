# Repset Bridge Administration Tool
# This script provides easy management of the Repset Bridge for gym administrators

param(
    [Parameter()]
    [ValidateSet("install", "status", "start", "stop", "restart", "logs", "pair", "health", "uninstall", "help")]
    [string]$Action = "help",
    
    [Parameter()]
    [string]$PairCode = "",
    
    [Parameter()]
    [switch]$Force = $false
)

# Configuration
$BRIDGE_EXE = "gym-door-bridge.exe"
$CONFIG_FILE = "config.yaml"
$LOG_FILE = "bridge.log"
$PRODUCTION_SERVER = "https://repset.onezy.in"

# Colors for output
$Colors = @{
    Success = "Green"
    Warning = "Yellow"
    Error = "Red"
    Info = "Cyan"
    Header = "Magenta"
}

function Write-Header {
    param([string]$Text)
    Write-Host ""
    Write-Host "=" * 60 -ForegroundColor $Colors.Header
    Write-Host " $Text" -ForegroundColor $Colors.Header
    Write-Host "=" * 60 -ForegroundColor $Colors.Header
    Write-Host ""
}

function Write-Success {
    param([string]$Text)
    Write-Host "✓ $Text" -ForegroundColor $Colors.Success
}

function Write-Warning {
    param([string]$Text)
    Write-Host "⚠ $Text" -ForegroundColor $Colors.Warning
}

function Write-Error {
    param([string]$Text)
    Write-Host "✗ $Text" -ForegroundColor $Colors.Error
}

function Write-Info {
    param([string]$Text)
    Write-Host "ℹ $Text" -ForegroundColor $Colors.Info
}

function Test-Prerequisites {
    Write-Info "Checking prerequisites..."
    
    # Check if bridge executable exists
    if (-not (Test-Path $BRIDGE_EXE)) {
        Write-Error "Bridge executable not found: $BRIDGE_EXE"
        Write-Info "Please ensure the bridge is downloaded and extracted to this directory."
        return $false
    }
    Write-Success "Bridge executable found"
    
    # Check if config file exists
    if (-not (Test-Path $CONFIG_FILE)) {
        Write-Error "Configuration file not found: $CONFIG_FILE"
        Write-Info "Please ensure config.yaml is present in this directory."
        return $false
    }
    Write-Success "Configuration file found"
    
    return $true
}

function Get-BridgeProcess {
    return Get-Process -Name "gym-door-bridge" -ErrorAction SilentlyContinue
}

function Get-BridgeService {
    return Get-Service -Name "*bridge*" -ErrorAction SilentlyContinue
}

function Test-BridgeHealth {
    try {
        $health = Invoke-RestMethod -Uri "http://localhost:8080/health" -TimeoutSec 5
        return @{
            Healthy = $true
            Status = $health.status
            DeviceId = $health.deviceId
            Uptime = $health.uptime
            Version = $health.version
            Tier = $health.tier
        }
    } catch {
        return @{ Healthy = $false; Error = $_.Exception.Message }
    }
}

function Install-Bridge {
    Write-Header "INSTALLING REPSET BRIDGE"
    
    if (-not (Test-Prerequisites)) {
        return
    }
    
    # Check if already running
    $process = Get-BridgeProcess
    if ($process) {
        Write-Warning "Bridge is already running (PID: $($process.Id))"
        if (-not $Force) {
            Write-Info "Use -Force to restart the bridge"
            return
        }
        Stop-Bridge
    }
    
    # Clean up old database if corrupted
    if (Test-Path "bridge.db") {
        Write-Info "Backing up existing database..."
        $backupName = "bridge.db.backup.$(Get-Date -Format 'yyyyMMdd-HHmmss')"
        Rename-Item "bridge.db" $backupName
        Write-Success "Database backed up to: $backupName"
        
        # Clean up associated files
        Remove-Item "bridge.db-shm" -ErrorAction SilentlyContinue
        Remove-Item "bridge.db-wal" -ErrorAction SilentlyContinue
    }
    
    # Start bridge as background process
    Write-Info "Starting bridge as background process..."
    try {
        Start-Process -FilePath ".\$BRIDGE_EXE" -ArgumentList "--config", $CONFIG_FILE -WindowStyle Hidden
        Start-Sleep -Seconds 3
        
        $health = Test-BridgeHealth
        if ($health.Healthy) {
            Write-Success "Bridge started successfully!"
            Write-Success "Device ID: $($health.DeviceId)"
            Write-Success "Status: $($health.Status)"
            Write-Success "Version: $($health.Version)"
            Write-Info "Health endpoint: http://localhost:8080/health"
            Write-Info "API endpoint: http://localhost:8081"
        } else {
            Write-Error "Bridge started but health check failed: $($health.Error)"
        }
    } catch {
        Write-Error "Failed to start bridge: $($_.Exception.Message)"
    }
}

function Get-BridgeStatus {
    Write-Header "BRIDGE STATUS"
    
    # Check process
    $process = Get-BridgeProcess
    if ($process) {
        Write-Success "Bridge process is running"
        Write-Info "Process ID: $($process.Id)"
        Write-Info "CPU Time: $($process.CPU)"
        Write-Info "Memory: $([math]::Round($process.WorkingSet64 / 1MB, 2)) MB"
    } else {
        Write-Warning "Bridge process is not running"
    }
    
    # Check service
    $service = Get-BridgeService
    if ($service) {
        Write-Info "Windows Service Status: $($service.Status)"
        Write-Info "Service Name: $($service.Name)"
    } else {
        Write-Info "Windows Service: Not installed"
    }
    
    # Check health
    $health = Test-BridgeHealth
    if ($health.Healthy) {
        Write-Success "Bridge is healthy and responding"
        Write-Info "Device ID: $($health.DeviceId)"
        Write-Info "Status: $($health.Status)"
        Write-Info "Uptime: $([math]::Round($health.Uptime / 1000000000, 0)) seconds"
        Write-Info "Performance Tier: $($health.Tier)"
        Write-Info "Version: $($health.Version)"
    } else {
        Write-Warning "Bridge health check failed: $($health.Error)"
    }
}

function Start-Bridge {
    Write-Header "STARTING BRIDGE"
    
    if (-not (Test-Prerequisites)) {
        return
    }
    
    $process = Get-BridgeProcess
    if ($process) {
        Write-Warning "Bridge is already running (PID: $($process.Id))"
        return
    }
    
    Write-Info "Starting bridge..."
    Install-Bridge
}

function Stop-Bridge {
    Write-Header "STOPPING BRIDGE"
    
    $process = Get-BridgeProcess
    if (-not $process) {
        Write-Warning "Bridge process is not running"
        return
    }
    
    Write-Info "Stopping bridge process (PID: $($process.Id))..."
    try {
        $process.Kill()
        Start-Sleep -Seconds 2
        Write-Success "Bridge stopped successfully"
    } catch {
        Write-Error "Failed to stop bridge: $($_.Exception.Message)"
    }
}

function Restart-Bridge {
    Write-Header "RESTARTING BRIDGE"
    Stop-Bridge
    Start-Sleep -Seconds 3
    Start-Bridge
}

function Show-BridgeLogs {
    Write-Header "BRIDGE LOGS"
    
    if (Test-Path $LOG_FILE) {
        Write-Info "Showing last 50 lines from ${LOG_FILE}:"
        Get-Content $LOG_FILE -Tail 50
    } else {
        Write-Warning "Log file not found: $LOG_FILE"
        Write-Info "Bridge might be logging to console only."
    }
    
    # Show recent Windows Event Log entries
    Write-Info "Recent Windows Event Log entries:"
    try {
        Get-EventLog -LogName System -Source "Service Control Manager" -Newest 5 | 
        Where-Object {$_.Message -like "*Bridge*" -or $_.Message -like "*gym-door*"} |
        Format-Table TimeGenerated, EntryType, Message -Wrap
    } catch {
        Write-Warning "Could not access Windows Event Log"
    }
}

function Pair-Bridge {
    Write-Header "PAIRING BRIDGE WITH PRODUCTION"
    
    if (-not (Test-Prerequisites)) {
        return
    }
    
    if ([string]::IsNullOrWhiteSpace($PairCode)) {
        Write-Info "To get your pair code:"
        Write-Info "1. Go to: $PRODUCTION_SERVER/{gymId}/admin/dashboard"
        Write-Info "2. Navigate to Bridge Management section"
        Write-Info "3. Create bridge deployment if none exists"
        Write-Info "4. Copy the Pair Code (shown in blue)"
        Write-Host ""
        $PairCode = Read-Host "Enter your production pair code"
    }
    
    if ([string]::IsNullOrWhiteSpace($PairCode)) {
        Write-Error "No pair code provided. Pairing cancelled."
        return
    }
    
    Write-Info "Pairing bridge with production server..."
    Write-Info "Pair Code: $PairCode"
    Write-Info "Server: $PRODUCTION_SERVER"
    
    try {
        $result = & ".\$BRIDGE_EXE" pair --pair-code $PairCode --config $CONFIG_FILE 2>&1
        
        Write-Host ""
        Write-Info "Pairing Result:"
        Write-Host "$result" -ForegroundColor White
        
        if ($result -match "success" -or $result -match "paired") {
            Write-Success "PAIRING SUCCESSFUL!"
            Write-Info "Bridge is now paired with production server"
            Write-Info "You can now install and start the bridge"
        } else {
            Write-Error "Pairing may have failed. Check the output above."
        }
    } catch {
        Write-Error "Pairing command failed: $($_.Exception.Message)"
    }
}

function Test-BridgeConnection {
    Write-Header "BRIDGE HEALTH CHECK"
    
    $health = Test-BridgeHealth
    if ($health.Healthy) {
        Write-Success "Bridge is healthy and responding"
        Write-Host ""
        Write-Host "Health Details:" -ForegroundColor $Colors.Info
        Write-Host "- Device ID: $($health.DeviceId)" -ForegroundColor White
        Write-Host "- Status: $($health.Status)" -ForegroundColor White
        Write-Host "- Version: $($health.Version)" -ForegroundColor White
        Write-Host "- Performance Tier: $($health.Tier)" -ForegroundColor White
        Write-Host "- Uptime: $([math]::Round($health.Uptime / 1000000000, 0)) seconds" -ForegroundColor White
        
        # Test API endpoint
        try {
            $response = Invoke-WebRequest -Uri "http://localhost:8081" -TimeoutSec 5
            Write-Success "API server is responding on port 8081"
        } catch {
            Write-Warning "API server on port 8081 is not responding"
        }
    } else {
        Write-Error "Bridge is not responding to health checks"
        Write-Info "Error: $($health.Error)"
        Write-Info "Make sure the bridge is running and try: .\bridge-admin-tools.ps1 start"
    }
}

function Uninstall-Bridge {
    Write-Header "UNINSTALLING BRIDGE"
    
    # Stop process
    Stop-Bridge
    
    # Try to uninstall service (requires admin)
    Write-Info "Attempting to uninstall Windows service..."
    try {
        & ".\$BRIDGE_EXE" service uninstall 2>&1 | Out-Null
        Write-Success "Windows service uninstalled"
    } catch {
        Write-Warning "Service uninstall requires administrator privileges"
    }
    
    # Clean up files
    if ($Force) {
        Write-Info "Cleaning up database and log files..."
        Remove-Item "bridge.db*" -Force -ErrorAction SilentlyContinue
        Remove-Item $LOG_FILE -Force -ErrorAction SilentlyContinue
        Write-Success "Cleanup completed"
    } else {
        Write-Info "Use -Force to also remove database and log files"
    }
}

function Show-Help {
    Write-Host ""
    Write-Host "REPSET BRIDGE ADMINISTRATION TOOL" -ForegroundColor $Colors.Header
    Write-Host "=================================" -ForegroundColor $Colors.Header
    Write-Host ""
    Write-Host "USAGE:" -ForegroundColor $Colors.Info
    Write-Host "  .\bridge-admin-tools.ps1 [ACTION] [OPTIONS]" -ForegroundColor White
    Write-Host ""
    Write-Host "ACTIONS:" -ForegroundColor $Colors.Info
    Write-Host "  pair         - Pair bridge with production server" -ForegroundColor White
    Write-Host "  install      - Install and start the bridge" -ForegroundColor White
    Write-Host "  start        - Start the bridge" -ForegroundColor White
    Write-Host "  stop         - Stop the bridge" -ForegroundColor White
    Write-Host "  restart      - Restart the bridge" -ForegroundColor White
    Write-Host "  status       - Show bridge status and health" -ForegroundColor White
    Write-Host "  health       - Test bridge health and connectivity" -ForegroundColor White
    Write-Host "  logs         - Show recent bridge logs" -ForegroundColor White
    Write-Host "  uninstall    - Uninstall and clean up bridge" -ForegroundColor White
    Write-Host "  help         - Show this help message" -ForegroundColor White
    Write-Host ""
    Write-Host "OPTIONS:" -ForegroundColor $Colors.Info
    Write-Host "  -PairCode    - Specify pair code for pairing" -ForegroundColor White
    Write-Host "  -Force       - Force action (skip confirmations)" -ForegroundColor White
    Write-Host ""
    Write-Host "EXAMPLES:" -ForegroundColor $Colors.Info
    Write-Host "  .\bridge-admin-tools.ps1 pair -PairCode 'ABC1-DEF2-GHI3'" -ForegroundColor Gray
    Write-Host "  .\bridge-admin-tools.ps1 install" -ForegroundColor Gray
    Write-Host "  .\bridge-admin-tools.ps1 status" -ForegroundColor Gray
    Write-Host "  .\bridge-admin-tools.ps1 restart -Force" -ForegroundColor Gray
    Write-Host ""
    Write-Host "TYPICAL SETUP WORKFLOW:" -ForegroundColor $Colors.Warning
    Write-Host "1. Get pair code from dashboard: $PRODUCTION_SERVER/{gymId}/admin/dashboard" -ForegroundColor White
    Write-Host "2. .\bridge-admin-tools.ps1 pair" -ForegroundColor White
    Write-Host "3. .\bridge-admin-tools.ps1 install" -ForegroundColor White
    Write-Host "4. .\bridge-admin-tools.ps1 status" -ForegroundColor White
    Write-Host ""
}

# Main script logic
switch ($Action.ToLower()) {
    "install" { Install-Bridge }
    "status" { Get-BridgeStatus }
    "start" { Start-Bridge }
    "stop" { Stop-Bridge }
    "restart" { Restart-Bridge }
    "logs" { Show-BridgeLogs }
    "pair" { Pair-Bridge }
    "health" { Test-BridgeConnection }
    "uninstall" { Uninstall-Bridge }
    default { Show-Help }
}