# RepSet Bridge Manager - Simple Admin Tool
param(
    [Parameter(Position=0)]
    [ValidateSet("pair", "install", "start", "stop", "restart", "status", "health", "logs", "help")]
    [string]$Action = "help",
    
    [Parameter()]
    [string]$PairCode = ""
)

# Configuration
$BRIDGE_EXE = "gym-door-bridge.exe"
$CONFIG_FILE = "config.yaml"
$PRODUCTION_SERVER = "https://repset.onezy.in"

function Write-ColorText {
    param([string]$Text, [string]$Color = "White")
    Write-Host $Text -ForegroundColor $Color
}

function Write-Header {
    param([string]$Text)
    Write-Host ""
    Write-ColorText ("=" * 50) "Cyan"
    Write-ColorText " $Text" "Cyan"  
    Write-ColorText ("=" * 50) "Cyan"
    Write-Host ""
}

function Test-BridgeFiles {
    $missing = @()
    if (-not (Test-Path $BRIDGE_EXE)) { $missing += $BRIDGE_EXE }
    if (-not (Test-Path $CONFIG_FILE)) { $missing += $CONFIG_FILE }
    
    if ($missing.Count -gt 0) {
        Write-ColorText "ERROR: Missing files:" "Red"
        $missing | ForEach-Object { Write-ColorText "  - $_" "Red" }
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
        return @{
            Healthy = $true
            Data = $health
        }
    } catch {
        return @{
            Healthy = $false
            Error = $_.Exception.Message
        }
    }
}

function Invoke-PairBridge {
    Write-Header "PAIR BRIDGE WITH PRODUCTION"
    
    if (-not (Test-BridgeFiles)) { return }
    
    if ([string]::IsNullOrWhiteSpace($PairCode)) {
        Write-ColorText "To get your pair code:" "Yellow"
        Write-ColorText "1. Go to: $PRODUCTION_SERVER/{gymId}/admin/dashboard" "White"
        Write-ColorText "2. Navigate to Bridge Management section" "White"
        Write-ColorText "3. Create bridge deployment if needed" "White" 
        Write-ColorText "4. Copy the Pair Code (shown in blue)" "White"
        Write-Host ""
        $PairCode = Read-Host "Enter your production pair code"
    }
    
    if ([string]::IsNullOrWhiteSpace($PairCode)) {
        Write-ColorText "No pair code provided. Cancelled." "Red"
        return
    }
    
    Write-ColorText "Pairing bridge..." "Yellow"
    Write-ColorText "Pair Code: $PairCode" "Cyan"
    Write-ColorText "Server: $PRODUCTION_SERVER" "Cyan"
    
    try {
        $result = & ".\$BRIDGE_EXE" pair --pair-code $PairCode --config $CONFIG_FILE 2>&1
        Write-Host ""
        Write-ColorText "Pairing Result:" "Yellow"
        Write-Host $result
        
        if ($result -match "success|paired") {
            Write-ColorText "✓ PAIRING SUCCESSFUL!" "Green"
        } else {
            Write-ColorText "✗ Pairing may have failed. Check output above." "Red"
        }
    } catch {
        Write-ColorText "✗ Pairing command failed: $($_.Exception.Message)" "Red"
    }
}

function Invoke-InstallBridge {
    Write-Header "INSTALL AND START BRIDGE"
    
    if (-not (Test-BridgeFiles)) { return }
    
    # Stop existing process
    $process = Get-BridgeProcess
    if ($process) {
        Write-ColorText "Stopping existing bridge process..." "Yellow"
        $process.Kill()
        Start-Sleep 2
    }
    
    # Backup old database
    if (Test-Path "bridge.db") {
        $backup = "bridge.db.backup.$(Get-Date -Format 'yyyyMMdd-HHmmss')"
        Rename-Item "bridge.db" $backup
        Write-ColorText "✓ Database backed up to: $backup" "Green"
        
        Remove-Item "bridge.db-*" -ErrorAction SilentlyContinue
    }
    
    # Start bridge
    Write-ColorText "Starting bridge as background process..." "Yellow"
    Start-Process -FilePath ".\$BRIDGE_EXE" -ArgumentList "--config", $CONFIG_FILE -WindowStyle Hidden
    
    Start-Sleep 3
    
    # Check health
    $health = Test-BridgeHealth
    if ($health.Healthy) {
        Write-ColorText "✓ Bridge started successfully!" "Green"
        Write-ColorText "✓ Device ID: $($health.Data.deviceId)" "Green"
        Write-ColorText "✓ Status: $($health.Data.status)" "Green"
        Write-ColorText "✓ Version: $($health.Data.version)" "Green"
        Write-ColorText "ℹ Health endpoint: http://localhost:8080/health" "Cyan"
        Write-ColorText "ℹ API endpoint: http://localhost:8081" "Cyan"
    } else {
        Write-ColorText "✗ Bridge started but health check failed" "Red"
        Write-ColorText "Error: $($health.Error)" "Red"
    }
}

function Invoke-StartBridge {
    Write-Header "START BRIDGE"
    
    $process = Get-BridgeProcess
    if ($process) {
        Write-ColorText "⚠ Bridge is already running (PID: $($process.Id))" "Yellow"
        return
    }
    
    Invoke-InstallBridge
}

function Invoke-StopBridge {
    Write-Header "STOP BRIDGE"
    
    $process = Get-BridgeProcess
    if (-not $process) {
        Write-ColorText "⚠ Bridge is not running" "Yellow"
        return
    }
    
    Write-ColorText "Stopping bridge process (PID: $($process.Id))..." "Yellow"
    $process.Kill()
    Start-Sleep 2
    Write-ColorText "✓ Bridge stopped" "Green"
}

function Invoke-RestartBridge {
    Write-Header "RESTART BRIDGE"
    Invoke-StopBridge
    Start-Sleep 2
    Invoke-StartBridge
}

function Show-BridgeStatus {
    Write-Header "BRIDGE STATUS"
    
    # Process status
    $process = Get-BridgeProcess
    if ($process) {
        Write-ColorText "✓ Bridge process is running" "Green"
        Write-ColorText "  Process ID: $($process.Id)" "White"
        Write-ColorText "  Memory: $([math]::Round($process.WorkingSet64 / 1MB, 1)) MB" "White"
    } else {
        Write-ColorText "✗ Bridge process is not running" "Red"
    }
    
    # Service status  
    $service = Get-Service -Name "*bridge*" -ErrorAction SilentlyContinue
    if ($service) {
        Write-ColorText "ℹ Windows Service: $($service.Status)" "Cyan"
    } else {
        Write-ColorText "ℹ Windows Service: Not installed" "Cyan"
    }
    
    # Health status
    $health = Test-BridgeHealth
    if ($health.Healthy) {
        Write-ColorText "✓ Bridge is healthy and responding" "Green"
        $data = $health.Data
        Write-ColorText "  Device ID: $($data.deviceId)" "White"
        Write-ColorText "  Status: $($data.status)" "White" 
        Write-ColorText "  Version: $($data.version)" "White"
        Write-ColorText "  Uptime: $([math]::Round($data.uptime / 1000000000, 0)) seconds" "White"
        Write-ColorText "  Performance Tier: $($data.tier)" "White"
    } else {
        Write-ColorText "✗ Bridge health check failed" "Red"
        Write-ColorText "  Error: $($health.Error)" "Red"
    }
}

function Show-BridgeHealth {
    Write-Header "BRIDGE HEALTH CHECK"
    
    $health = Test-BridgeHealth
    if ($health.Healthy) {
        Write-ColorText "✓ Bridge is healthy and responding" "Green"
        Write-Host ""
        
        $data = $health.Data
        Write-ColorText "Health Details:" "Cyan"
        Write-ColorText "- Device ID: $($data.deviceId)" "White"
        Write-ColorText "- Status: $($data.status)" "White"
        Write-ColorText "- Version: $($data.version)" "White"
        Write-ColorText "- Performance Tier: $($data.tier)" "White"
        Write-ColorText "- Uptime: $([math]::Round($data.uptime / 1000000000, 0)) seconds" "White"
        
        # Test API
        try {
            Invoke-WebRequest -Uri "http://localhost:8081" -TimeoutSec 3 | Out-Null
            Write-ColorText "✓ API server responding on port 8081" "Green"
        } catch {
            Write-ColorText "⚠ API server on port 8081 not responding" "Yellow"
        }
    } else {
        Write-ColorText "✗ Bridge is not responding" "Red"
        Write-ColorText "Error: $($health.Error)" "Red"
        Write-ColorText "Try: .\bridge-manager.ps1 start" "Cyan"
    }
}

function Show-BridgeLogs {
    Write-Header "BRIDGE LOGS"
    
    # Check for log file
    $logFiles = @("bridge.log", "gym-door-bridge.log")
    $foundLogs = $false
    
    foreach ($logFile in $logFiles) {
        if (Test-Path $logFile) {
            Write-ColorText "Last 20 lines from ${logFile}:" "Cyan"
            Get-Content $logFile -Tail 20 | ForEach-Object { Write-Host "  $_" }
            $foundLogs = $true
            Write-Host ""
        }
    }
    
    if (-not $foundLogs) {
        Write-ColorText "⚠ No log files found" "Yellow"
        Write-ColorText "Bridge might be logging to console only" "White"
    }
    
    # Recent Windows events
    try {
        $events = Get-EventLog -LogName System -Source "Service Control Manager" -Newest 3 -ErrorAction SilentlyContinue |
                  Where-Object { $_.Message -like "*Bridge*" -or $_.Message -like "*gym-door*" }
        
        if ($events) {
            Write-ColorText "Recent Windows Events:" "Cyan"
            $events | ForEach-Object {
                Write-ColorText "  $($_.TimeGenerated): $($_.EntryType) - $($_.Message.Substring(0, [Math]::Min(100, $_.Message.Length)))..." "White"
            }
        }
    } catch {
        Write-ColorText "⚠ Could not access Windows Event Log" "Yellow"
    }
}

function Show-Help {
    Write-Host ""
    Write-ColorText "REPSET BRIDGE MANAGER" "Cyan"
    Write-ColorText "=====================" "Cyan"
    Write-Host ""
    Write-ColorText "USAGE:" "Yellow"
    Write-ColorText "  .\bridge-manager.ps1 [ACTION] [-PairCode CODE]" "White"
    Write-Host ""
    Write-ColorText "ACTIONS:" "Yellow"
    Write-ColorText "  pair         Pair bridge with production server" "White"
    Write-ColorText "  install      Install and start the bridge" "White" 
    Write-ColorText "  start        Start the bridge" "White"
    Write-ColorText "  stop         Stop the bridge" "White"
    Write-ColorText "  restart      Restart the bridge" "White"
    Write-ColorText "  status       Show bridge status" "White"
    Write-ColorText "  health       Detailed health check" "White"
    Write-ColorText "  logs         Show recent logs" "White"
    Write-ColorText "  help         Show this help" "White"
    Write-Host ""
    Write-ColorText "EXAMPLES:" "Yellow"
    Write-ColorText "  .\bridge-manager.ps1 pair -PairCode 'ABCD-1234-EFGH'" "Gray"
    Write-ColorText "  .\bridge-manager.ps1 install" "Gray"
    Write-ColorText "  .\bridge-manager.ps1 status" "Gray"
    Write-Host ""
    Write-ColorText "SETUP WORKFLOW:" "Green"
    Write-ColorText "1. Get pair code from: $PRODUCTION_SERVER/{gymId}/admin/dashboard" "White"
    Write-ColorText "2. .\bridge-manager.ps1 pair" "White"
    Write-ColorText "3. .\bridge-manager.ps1 install" "White"
    Write-ColorText "4. .\bridge-manager.ps1 status" "White"
    Write-Host ""
}

# Main execution
switch ($Action.ToLower()) {
    "pair"     { Invoke-PairBridge }
    "install"  { Invoke-InstallBridge }
    "start"    { Invoke-StartBridge }
    "stop"     { Invoke-StopBridge }
    "restart"  { Invoke-RestartBridge }
    "status"   { Show-BridgeStatus }
    "health"   { Show-BridgeHealth }
    "logs"     { Show-BridgeLogs }
    default    { Show-Help }
}