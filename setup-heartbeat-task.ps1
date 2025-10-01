# Setup Scheduled Task for Bridge Heartbeats
# Run this as Administrator to create a scheduled task that sends heartbeats every 60 seconds

Write-Host "Setting up Bridge Heartbeat Scheduled Task" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan

# Check if running as Administrator
$currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
$principal = New-Object Security.Principal.WindowsPrincipal($currentUser)

if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Host "ERROR: This script must be run as Administrator" -ForegroundColor Red
    exit 1
}

# Paths
$scriptPath = "C:\Program Files\GymDoorBridge\bridge-heartbeat-service.ps1"
$taskName = "RepSet Bridge Heartbeat"

try {
    # Copy the heartbeat script to the bridge directory
    Write-Host "Copying heartbeat script to bridge directory..." -ForegroundColor Yellow
    Copy-Item "bridge-heartbeat-service.ps1" -Destination $scriptPath -Force
    Write-Host "✓ Script copied successfully" -ForegroundColor Green
    
    # Remove existing task if it exists
    $existingTask = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
    if ($existingTask) {
        Write-Host "Removing existing scheduled task..." -ForegroundColor Yellow
        Unregister-ScheduledTask -TaskName $taskName -Confirm:$false
    }
    
    # Create scheduled task action
    $action = New-ScheduledTaskAction -Execute "powershell.exe" -Argument "-ExecutionPolicy Bypass -File `"$scriptPath`""
    
    # Create scheduled task trigger (every 60 seconds)
    $trigger = New-ScheduledTaskTrigger -Once -At (Get-Date) -RepetitionInterval (New-TimeSpan -Seconds 60) -RepetitionDuration (New-TimeSpan -Days 365)
    
    # Create scheduled task settings
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -RunOnlyIfNetworkAvailable
    
    # Create scheduled task principal (run as SYSTEM)
    $principal = New-ScheduledTaskPrincipal -UserId "SYSTEM" -LogonType ServiceAccount -RunLevel Highest
    
    # Register the scheduled task
    Write-Host "Creating scheduled task..." -ForegroundColor Yellow
    Register-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger -Settings $settings -Principal $principal -Description "Sends heartbeats from RepSet Bridge to platform every 60 seconds"
    
    Write-Host "✓ Scheduled task created successfully" -ForegroundColor Green
    
    # Start the task immediately
    Write-Host "Starting heartbeat task..." -ForegroundColor Yellow
    Start-ScheduledTask -TaskName $taskName
    
    Write-Host ""
    Write-Host "=== SETUP COMPLETE ===" -ForegroundColor Green
    Write-Host "✓ Bridge heartbeat service is now running" -ForegroundColor Green
    Write-Host "✓ Heartbeats will be sent every 60 seconds" -ForegroundColor Green
    Write-Host "✓ Logs will be written to: C:\Program Files\GymDoorBridge\heartbeat.log" -ForegroundColor Green
    Write-Host ""
    Write-Host "To check the task status:" -ForegroundColor Cyan
    Write-Host "Get-ScheduledTask -TaskName '$taskName'" -ForegroundColor Gray
    Write-Host ""
    Write-Host "To view heartbeat logs:" -ForegroundColor Cyan
    Write-Host "Get-Content 'C:\Program Files\GymDoorBridge\heartbeat.log' -Tail 20" -ForegroundColor Gray
    
} catch {
    Write-Host "ERROR: Failed to setup scheduled task: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}