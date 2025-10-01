# Bridge Heartbeat Service
# This script sends periodic heartbeats to the RepSet platform
# Run this as a scheduled task every 60 seconds

param(
    [string]$ConfigPath = "C:\Program Files\GymDoorBridge\config.yaml",
    [string]$LogPath = "C:\Program Files\GymDoorBridge\heartbeat.log"
)

function Write-Log {
    param([string]$Message)
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    "$timestamp - $Message" | Add-Content -Path $LogPath
    Write-Host "$timestamp - $Message"
}

function Read-BridgeConfig {
    param([string]$ConfigPath)
    
    try {
        $config = @{}
        $content = Get-Content $ConfigPath -Raw
        
        # Parse YAML-like config (simple parsing)
        if ($content -match 'device_id:\s*(.+)') {
            $config.device_id = $matches[1].Trim()
        }
        if ($content -match 'device_key:\s*(.+)') {
            $config.device_key = $matches[1].Trim()
        }
        if ($content -match 'server_url:\s*(.+)') {
            $config.server_url = $matches[1].Trim()
        }
        
        return $config
    } catch {
        Write-Log "ERROR: Failed to read config file: $($_.Exception.Message)"
        return $null
    }
}

function Send-Heartbeat {
    param(
        [string]$DeviceId,
        [string]$DeviceKey,
        [string]$ServerUrl
    )
    
    try {
        $heartbeatPayload = @{
            device_id = $DeviceId
            device_key = $DeviceKey
            status = @{
                version = "1.4.0"
                uptime = [int]((Get-Date) - (Get-Process -Name "gym-door-bridge" -ErrorAction SilentlyContinue | Select-Object -First 1).StartTime).TotalSeconds
                connected_devices = 1
                last_event_time = (Get-Date).ToString("yyyy-MM-ddTHH:mm:ssZ")
                system_info = @{
                    platform = "windows"
                    arch = "amd64"
                    memory = [math]::Round((Get-WmiObject -Class Win32_ComputerSystem).TotalPhysicalMemory / 1MB)
                    cpu = (Get-WmiObject -Class Win32_Processor | Measure-Object -Property LoadPercentage -Average).Average
                }
            }
        } | ConvertTo-Json -Depth 3
        
        $heartbeatUrl = "$ServerUrl/api/v1/bridge/heartbeat"
        Write-Log "Sending heartbeat to: $heartbeatUrl"
        
        $response = Invoke-RestMethod -Uri $heartbeatUrl -Method POST -Body $heartbeatPayload -ContentType "application/json" -UseBasicParsing -TimeoutSec 30
        
        Write-Log "SUCCESS: Heartbeat sent successfully"
        Write-Log "Response: $($response | ConvertTo-Json -Compress)"
        
        return $true
        
    } catch {
        Write-Log "ERROR: Heartbeat failed - $($_.Exception.Message)"
        
        if ($_.Exception.Response) {
            $statusCode = $_.Exception.Response.StatusCode
            Write-Log "ERROR: HTTP Status Code: $statusCode"
        }
        
        return $false
    }
}

# Main execution
Write-Log "=== Bridge Heartbeat Service Starting ==="

# Read configuration
$config = Read-BridgeConfig -ConfigPath $ConfigPath

if (-not $config -or -not $config.device_id -or -not $config.device_key -or -not $config.server_url) {
    Write-Log "ERROR: Invalid configuration. Missing device_id, device_key, or server_url"
    exit 1
}

Write-Log "Device ID: $($config.device_id)"
Write-Log "Server URL: $($config.server_url)"

# Send heartbeat
$success = Send-Heartbeat -DeviceId $config.device_id -DeviceKey $config.device_key -ServerUrl $config.server_url

if ($success) {
    Write-Log "=== Heartbeat completed successfully ==="
    exit 0
} else {
    Write-Log "=== Heartbeat failed ==="
    exit 1
}