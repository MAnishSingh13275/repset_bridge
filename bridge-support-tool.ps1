# RepSet Bridge Support Tool
# This script helps diagnose and fix common bridge issues for customers

param(
    [Parameter(Mandatory=$false)]
    [ValidateSet("status", "start", "restart", "reinstall-service", "logs", "test", "help")]
    [string]$Action = "help"
)

# Configuration
$bridgePath = "C:\Program Files\GymDoorBridge\gym-door-bridge.exe"
$configPath = "$env:USERPROFILE\Documents\repset-bridge-config.yaml"
$serviceName = "GymDoorBridge"

# Helper functions
function Write-Success { param([string]$Message) Write-Host "[✓] $Message" -ForegroundColor Green }
function Write-Error { param([string]$Message) Write-Host "[✗] $Message" -ForegroundColor Red }
function Write-Warning { param([string]$Message) Write-Host "[!] $Message" -ForegroundColor Yellow }
function Write-Info { param([string]$Message) Write-Host "[i] $Message" -ForegroundColor Cyan }

function Test-AdminRights {
    return ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")
}

function Show-BridgeStatus {
    Write-Host "=== RepSet Bridge Status ===" -ForegroundColor Blue
    Write-Host ""
    
    # Check if files exist
    if (Test-Path $bridgePath) {
        Write-Success "Bridge executable found"
    } else {
        Write-Error "Bridge executable not found at $bridgePath"
        return
    }
    
    if (Test-Path $configPath) {
        Write-Success "Configuration file found"
    } else {
        Write-Error "Configuration file not found at $configPath"
        return
    }
    
    # Check service status
    $service = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
    if ($service) {
        Write-Info "Service Status: $($service.Status)"
        if ($service.Status -eq 'Running') {
            Write-Success "Bridge service is running"
        } else {
            Write-Warning "Bridge service is not running"
        }
    } else {
        Write-Warning "Bridge service not found"
        
        # Check scheduled task
        $task = Get-ScheduledTask -TaskName "RepSetBridge" -ErrorAction SilentlyContinue
        if ($task) {
            Write-Info "Scheduled Task Status: $($task.State)"
            if ($task.State -eq 'Running') {
                Write-Success "Bridge is running as scheduled task"
            } else {
                Write-Warning "Bridge scheduled task is not running"
            }
        } else {
            Write-Warning "No automatic startup configured"
        }
    }
    
    # Test bridge directly
    Write-Info "Testing bridge executable..."
    try {
        $result = & $bridgePath status --config $configPath 2>&1
        if ($result -match "Bridge paired with platform") {
            Write-Success "Bridge is paired and configured correctly"
        } else {
            Write-Warning "Bridge may have configuration issues"
        }
    } catch {
        Write-Error "Failed to run bridge executable"
    }
}

function Start-Bridge {
    Write-Host "=== Starting RepSet Bridge ===" -ForegroundColor Blue
    Write-Host ""
    
    # Try service first
    $service = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
    if ($service) {
        try {
            Start-Service -Name $serviceName -ErrorAction Stop
            Write-Success "Bridge service started successfully"
            return
        } catch {
            Write-Warning "Failed to start service: $($_.Exception.Message)"
        }
    }
    
    # Try scheduled task
    $task = Get-ScheduledTask -TaskName "RepSetBridge" -ErrorAction SilentlyContinue
    if ($task) {
        try {
            Start-ScheduledTask -TaskName "RepSetBridge" -ErrorAction Stop
            Write-Success "Bridge scheduled task started successfully"
            return
        } catch {
            Write-Warning "Failed to start scheduled task: $($_.Exception.Message)"
        }
    }
    
    # Manual start
    Write-Info "Starting bridge manually..."
    Write-Warning "This will keep the bridge running only while this window is open"
    Write-Info "Press Ctrl+C to stop the bridge"
    Write-Host ""
    
    try {
        & $bridgePath --config $configPath
    } catch {
        Write-Error "Failed to start bridge manually: $($_.Exception.Message)"
    }
}

function Restart-Bridge {
    Write-Host "=== Restarting RepSet Bridge ===" -ForegroundColor Blue
    Write-Host ""
    
    # Stop service
    $service = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
    if ($service -and $service.Status -eq 'Running') {
        Stop-Service -Name $serviceName -Force
        Write-Info "Service stopped"
    }
    
    # Stop scheduled task
    $task = Get-ScheduledTask -TaskName "RepSetBridge" -ErrorAction SilentlyContinue
    if ($task -and $task.State -eq 'Running') {
        Stop-ScheduledTask -TaskName "RepSetBridge"
        Write-Info "Scheduled task stopped"
    }
    
    Start-Sleep -Seconds 2
    Start-Bridge
}

function Reinstall-Service {
    if (-not (Test-AdminRights)) {
        Write-Error "Administrator privileges required for service installation"
        Write-Info "Please run PowerShell as Administrator and try again"
        return
    }
    
    Write-Host "=== Reinstalling Bridge Service ===" -ForegroundColor Blue
    Write-Host ""
    
    # Remove existing service
    $service = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
    if ($service) {
        Stop-Service -Name $serviceName -Force -ErrorAction SilentlyContinue
        & sc.exe delete $serviceName | Out-Null
        Write-Info "Existing service removed"
        Start-Sleep -Seconds 2
    }
    
    # Try bridge's built-in installer
    try {
        $result = & $bridgePath service install --config $configPath 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Service installed successfully"
            
            # Start the service
            $startResult = & $bridgePath service start 2>&1
            if ($LASTEXITCODE -eq 0) {
                Write-Success "Service started successfully"
            } else {
                Start-Service -Name $serviceName
                Write-Success "Service started with PowerShell"
            }
            return
        }
    } catch { }
    
    # Fallback to manual service creation
    try {
        $serviceBinPath = "`"$bridgePath`" --config `"$configPath`""
        New-Service -Name $serviceName -BinaryPathName $serviceBinPath -DisplayName "RepSet Gym Door Bridge" -StartupType Automatic
        Start-Service -Name $serviceName
        Write-Success "Service created and started successfully"
    } catch {
        Write-Error "Failed to create service: $($_.Exception.Message)"
        
        # Create scheduled task as fallback
        Write-Info "Creating scheduled task as fallback..."
        try {
            $action = New-ScheduledTaskAction -Execute $bridgePath -Argument "--config `"$configPath`""
            $trigger = New-ScheduledTaskTrigger -AtStartup
            $principal = New-ScheduledTaskPrincipal -UserId "SYSTEM" -LogonType ServiceAccount -RunLevel Highest
            
            Register-ScheduledTask -TaskName "RepSetBridge" -Action $action -Trigger $trigger -Principal $principal -Description "RepSet Gym Door Bridge"
            Start-ScheduledTask -TaskName "RepSetBridge"
            
            Write-Success "Scheduled task created and started successfully"
        } catch {
            Write-Error "Failed to create scheduled task: $($_.Exception.Message)"
        }
    }
}

function Show-Logs {
    Write-Host "=== RepSet Bridge Logs ===" -ForegroundColor Blue
    Write-Host ""
    
    $logPath = "$env:USERPROFILE\Documents\bridge.log"
    if (Test-Path $logPath) {
        Write-Info "Log file: $logPath"
        Write-Host ""
        Get-Content $logPath -Tail 20 | ForEach-Object {
            if ($_ -match "error|failed|exception") {
                Write-Host $_ -ForegroundColor Red
            } elseif ($_ -match "warning|warn") {
                Write-Host $_ -ForegroundColor Yellow
            } elseif ($_ -match "success|started|paired") {
                Write-Host $_ -ForegroundColor Green
            } else {
                Write-Host $_ -ForegroundColor White
            }
        }
    } else {
        Write-Warning "Log file not found at $logPath"
    }
    
    # Show Windows Event Log entries
    Write-Host ""
    Write-Info "Recent Windows Event Log entries:"
    try {
        Get-EventLog -LogName Application -Source "GymDoorBridge" -Newest 5 -ErrorAction SilentlyContinue | 
            Format-Table TimeGenerated, EntryType, Message -Wrap
    } catch {
        Write-Info "No Windows Event Log entries found"
    }
}

function Test-Bridge {
    Write-Host "=== Testing RepSet Bridge ===" -ForegroundColor Blue
    Write-Host ""
    
    # Test configuration
    if (Test-Path $configPath) {
        $config = Get-Content $configPath | ConvertFrom-Yaml -ErrorAction SilentlyContinue
        if ($config -and $config.device_id) {
            Write-Success "Configuration file is valid"
            Write-Info "Device ID: $($config.device_id)"
            Write-Info "Server URL: $($config.server_url)"
        } else {
            Write-Warning "Configuration file may be invalid"
        }
    }
    
    # Test network connectivity
    Write-Info "Testing network connectivity..."
    try {
        $response = Invoke-WebRequest -Uri "https://repset.onezy.in/health" -TimeoutSec 10 -ErrorAction Stop
        Write-Success "Network connectivity to RepSet platform: OK"
    } catch {
        Write-Error "Network connectivity issue: $($_.Exception.Message)"
    }
    
    # Test bridge executable
    Write-Info "Testing bridge executable..."
    try {
        $result = & $bridgePath --help 2>&1
        if ($result -match "Usage:") {
            Write-Success "Bridge executable is working"
        } else {
            Write-Warning "Bridge executable may have issues"
        }
    } catch {
        Write-Error "Bridge executable failed: $($_.Exception.Message)"
    }
}

function Show-Help {
    Write-Host "=== RepSet Bridge Support Tool ===" -ForegroundColor Blue
    Write-Host ""
    Write-Host "Usage: .\bridge-support-tool.ps1 -Action <action>" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Available Actions:" -ForegroundColor Yellow
    Write-Host "  status          - Show bridge status and configuration"
    Write-Host "  start           - Start the bridge service/task"
    Write-Host "  restart         - Restart the bridge"
    Write-Host "  reinstall-service - Reinstall the Windows service (requires Admin)"
    Write-Host "  logs            - Show recent bridge logs"
    Write-Host "  test            - Test bridge configuration and connectivity"
    Write-Host "  help            - Show this help message"
    Write-Host ""
    Write-Host "Examples:" -ForegroundColor Green
    Write-Host "  .\bridge-support-tool.ps1 -Action status"
    Write-Host "  .\bridge-support-tool.ps1 -Action start"
    Write-Host "  .\bridge-support-tool.ps1 -Action logs"
    Write-Host ""
    Write-Host "For support, contact RepSet technical support with the output of:" -ForegroundColor Cyan
    Write-Host "  .\bridge-support-tool.ps1 -Action test"
}

# Main execution
switch ($Action.ToLower()) {
    "status" { Show-BridgeStatus }
    "start" { Start-Bridge }
    "restart" { Restart-Bridge }
    "reinstall-service" { Reinstall-Service }
    "logs" { Show-Logs }
    "test" { Test-Bridge }
    "help" { Show-Help }
    default { Show-Help }
}

if ($Action -ne "start") {
    Write-Host ""
    Write-Host "Press Enter to close..." -ForegroundColor Gray
    Read-Host
}