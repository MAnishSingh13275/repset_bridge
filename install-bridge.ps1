# Gym Door Bridge - Fixed Installation Script
# Handles permissions, service configuration, and startup properly

param(
    [string]$PairCode = "",
    [string]$ServerUrl = "https://repset.onezy.in",
    [string]$InstallPath = "$env:ProgramFiles\GymDoorBridge",
    [switch]$Force = $false
)

# Check admin privileges
if (-NOT ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    Write-Host "ERROR: This script requires Administrator privileges!" -ForegroundColor Red
    Write-Host "Please run PowerShell as Administrator and try again." -ForegroundColor Yellow
    exit 1
}

Write-Host "Gym Door Bridge Installation" -ForegroundColor Cyan
Write-Host "============================" -ForegroundColor Cyan

# Constants
$ServiceName = "GymDoorBridge"
$ServiceDisp = "Gym Door Access Bridge"
$DataDir = Join-Path $env:ProgramData "GymDoorBridge"
$TempZip = Join-Path $env:TEMP "gym-door-bridge.zip"
$TempExtract = Join-Path $env:TEMP "gym-door-bridge"
$DownloadUrl = "https://github.com/MAnishSingh13275/repset_bridge/releases/latest/download/gym-door-bridge-windows.zip"

try {
    # Create directories
    Write-Host "Creating directories..." -ForegroundColor Green
    New-Item -ItemType Directory -Force -Path $DataDir | Out-Null
    New-Item -ItemType Directory -Force -Path $InstallPath | Out-Null

    # Set proper permissions on directories
    Write-Host "Setting directory permissions..." -ForegroundColor Green
    try {
        # Grant full control to SYSTEM and Local Service on data directory
        & icacls $DataDir /grant "NT AUTHORITY\SYSTEM:(OI)(CI)F" /T | Out-Null
        & icacls $DataDir /grant "NT AUTHORITY\LOCAL SERVICE:(OI)(CI)F" /T | Out-Null
        & icacls $DataDir /grant "BUILTIN\Administrators:(OI)(CI)F" /T | Out-Null
        
        # Grant read/execute to Local Service on install directory
        & icacls $InstallPath /grant "NT AUTHORITY\LOCAL SERVICE:(OI)(CI)RX" /T | Out-Null
        & icacls $InstallPath /grant "NT AUTHORITY\SYSTEM:(OI)(CI)F" /T | Out-Null
        
        Write-Host "Permissions set successfully" -ForegroundColor Green
    } catch {
        Write-Host "WARNING: Could not set all permissions: $($_.Exception.Message)" -ForegroundColor Yellow
    }

    # Check existing service and stop it properly
    $existingService = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($existingService) {
        if (-not $Force -and -not $PairCode) {
            Write-Host "WARNING: Gym Door Bridge is already installed!" -ForegroundColor Yellow
            Write-Host "Service Status: $($existingService.Status)" -ForegroundColor White
            Write-Host "Use -Force to reinstall or provide -PairCode to re-pair." -ForegroundColor Yellow
            exit 0
        }
        
        Write-Host "Stopping and removing existing service..." -ForegroundColor Yellow
        try {
            # Stop service
            Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
            Start-Sleep 3
            
            # Kill any remaining processes
            Get-Process -Name "gym-door-bridge*" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
            Start-Sleep 2
            
            # Remove service
            & sc.exe delete $ServiceName | Out-Null
            Start-Sleep 2
            
            Write-Host "Existing service removed" -ForegroundColor Green
        } catch {
            Write-Host "WARNING: Could not fully remove existing service" -ForegroundColor Yellow
        }
    }

    # Download
    Write-Host "Downloading latest version..." -ForegroundColor Green
    
    if (Test-Path $TempExtract) { 
        Remove-Item $TempExtract -Recurse -Force -ErrorAction SilentlyContinue 
    }
    New-Item -ItemType Directory -Path $TempExtract -Force | Out-Null

    try {
        Invoke-WebRequest -Uri $DownloadUrl -OutFile $TempZip -UseBasicParsing -TimeoutSec 60
        Write-Host "Download successful" -ForegroundColor Green
    } catch {
        Write-Host "ERROR: Download failed: $($_.Exception.Message)" -ForegroundColor Red
        exit 1
    }

    # Extract
    Write-Host "Extracting files..." -ForegroundColor Green
    try {
        Expand-Archive -Path $TempZip -DestinationPath $TempExtract -Force
    } catch {
        Write-Host "ERROR: Extraction failed: $($_.Exception.Message)" -ForegroundColor Red
        exit 1
    }

    # Find executable
    Write-Host "Locating executable..." -ForegroundColor Yellow
    $exe = Get-ChildItem -Path $TempExtract -Filter "gym-door-bridge.exe" -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
    if (-not $exe) {
        $exe = Get-ChildItem -Path $TempExtract -Filter "*.exe" -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
    }
    if (-not $exe) {
        Write-Host "ERROR: No executable found in package" -ForegroundColor Red
        exit 1
    }
    Write-Host "Found executable: $($exe.Name)" -ForegroundColor Green

    # Copy executable with proper handling
    $targetExe = Join-Path $InstallPath "gym-door-bridge.exe"
    Write-Host "Installing executable..." -ForegroundColor Green
    
    # Remove existing executable if it exists
    if (Test-Path $targetExe) {
        for ($i = 0; $i -lt 10; $i++) {
            try { 
                Remove-Item $targetExe -Force 
                break 
            } catch { 
                Write-Host "Waiting for file unlock... ($($i+1)/10)" -ForegroundColor Yellow
                Start-Sleep 3 
            }
        }
    }
    
    try {
        Copy-Item -Path $exe.FullName -Destination $targetExe -Force
        
        # Set executable permissions
        & icacls $targetExe /grant "NT AUTHORITY\LOCAL SERVICE:RX" | Out-Null
        & icacls $targetExe /grant "NT AUTHORITY\SYSTEM:F" | Out-Null
        
        Write-Host "Executable installed successfully" -ForegroundColor Green
    } catch {
        Write-Host "ERROR: Failed to install executable: $($_.Exception.Message)" -ForegroundColor Red
        exit 1
    }

    # Create configuration
    Write-Host "Creating configuration..." -ForegroundColor Green
    $configPath = Join-Path $InstallPath "config.yaml"
    $absDataPath = $DataDir.Replace('\', '/')
    
    $configContent = @"
server_url: "$ServerUrl"
tier: "normal"
queue_max_size: 10000
heartbeat_interval: 60
unlock_duration: 3000
device_id: ""
device_key: ""
database_path: "$absDataPath/bridge.db"
log_level: "info"
log_file: "$absDataPath/bridge.log"
enabled_adapters:
  - "simulator"
api_server:
  enabled: true
  port: 8081
  host: "0.0.0.0"
"@

    $configContent | Set-Content -Path $configPath -Encoding UTF8
    
    # Set config file permissions
    & icacls $configPath /grant "NT AUTHORITY\LOCAL SERVICE:R" | Out-Null
    
    Write-Host "Configuration created successfully" -ForegroundColor Green

    # Create service with proper configuration
    Write-Host "Creating Windows service..." -ForegroundColor Green
    
    try {
        # Create service with proper settings
        $binPath = "`"$targetExe`" --config `"$configPath`""
        
        # Create the service
        & sc.exe create $ServiceName binPath= $binPath DisplayName= $ServiceDisp start= auto | Out-Null
        
        # Configure service
        & sc.exe description $ServiceName "Gym Door Access Bridge - integrates RepSet with door controllers" | Out-Null
        & sc.exe config $ServiceName obj= "NT AUTHORITY\LocalService" | Out-Null
        & sc.exe failure $ServiceName reset= 86400 actions= restart/5000/restart/10000/restart/30000 | Out-Null
        
        Write-Host "Service created successfully" -ForegroundColor Green
    } catch {
        Write-Host "ERROR: Failed to create service: $($_.Exception.Message)" -ForegroundColor Red
        exit 1
    }

    # Verify service exists
    Start-Sleep 2
    $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if (-not $svc) {
        Write-Host "ERROR: Service not found after creation" -ForegroundColor Red
        exit 1
    }
    Write-Host "Service verification successful: $($svc.Status)" -ForegroundColor Green

    # Pairing (before starting service)
    if ($PairCode) {
        Write-Host "Pairing device with platform..." -ForegroundColor Green
        
        # Clean existing pairing
        Start-Process -FilePath $targetExe -ArgumentList "unpair" -Wait -NoNewWindow -ErrorAction SilentlyContinue | Out-Null
        Start-Sleep 1
        
        # Perform pairing
        $pairResult = Start-Process -FilePath $targetExe -ArgumentList @("pair", $PairCode) -Wait -PassThru -NoNewWindow
        if ($pairResult.ExitCode -eq 0) {
            Write-Host "Device paired successfully!" -ForegroundColor Green
        } else {
            Write-Host "WARNING: Pairing failed (exit code: $($pairResult.ExitCode))" -ForegroundColor Yellow
            Write-Host "You can pair manually later with: gym-door-bridge pair $PairCode" -ForegroundColor Yellow
        }
    }

    # Start service with multiple attempts
    Write-Host "Starting service..." -ForegroundColor Yellow
    
    $serviceStarted = $false
    $maxAttempts = 3
    
    for ($attempt = 1; $attempt -le $maxAttempts; $attempt++) {
        Write-Host "Start attempt $attempt/$maxAttempts..." -ForegroundColor Yellow
        
        try {
            # Try starting with net command
            $startResult = & net start $ServiceName 2>&1
            Start-Sleep 5
            
            $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
            if ($svc.Status -eq "Running") {
                $serviceStarted = $true
                Write-Host "Service started successfully!" -ForegroundColor Green
                break
            } else {
                Write-Host "Service status: $($svc.Status)" -ForegroundColor Yellow
                if ($attempt -lt $maxAttempts) {
                    Write-Host "Retrying in 5 seconds..." -ForegroundColor Yellow
                    Start-Sleep 5
                }
            }
        } catch {
            Write-Host "Start attempt $attempt failed: $($_.Exception.Message)" -ForegroundColor Yellow
            if ($attempt -lt $maxAttempts) {
                Start-Sleep 5
            }
        }
    }
    
    # Final status check and API test
    $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($svc.Status -eq "Running") {
        Write-Host "Service is running successfully!" -ForegroundColor Green
        
        # Test API connectivity
        Write-Host "Testing API connectivity..." -ForegroundColor Yellow
        $apiWorking = $false
        
        for ($i = 1; $i -le 12; $i++) {
            try {
                Start-Sleep 5
                $response = Invoke-WebRequest -Uri "http://localhost:8081/api/v1/health" -UseBasicParsing -TimeoutSec 10
                Write-Host "API is responding: HTTP $($response.StatusCode)" -ForegroundColor Green
                $apiWorking = $true
                break
            } catch {
                Write-Host "API test $i/12: waiting for API to start..." -ForegroundColor Yellow
            }
        }
        
        if (-not $apiWorking) {
            Write-Host "WARNING: API not responding yet" -ForegroundColor Yellow
            Write-Host "The service is running but API may need more time to start" -ForegroundColor Yellow
        }
        
        # Test bridge status if paired
        if ($PairCode -and $apiWorking) {
            Write-Host "Testing bridge status..." -ForegroundColor Yellow
            try {
                $statusResult = Start-Process -FilePath $targetExe -ArgumentList "status" -Wait -PassThru -NoNewWindow
                if ($statusResult.ExitCode -eq 0) {
                    Write-Host "Bridge status check successful!" -ForegroundColor Green
                }
            } catch {
                Write-Host "Bridge status check failed, but service is running" -ForegroundColor Yellow
            }
        }
        
    } else {
        Write-Host "ERROR: Service failed to start (Status: $($svc.Status))" -ForegroundColor Red
        Write-Host ""
        Write-Host "Troubleshooting steps:" -ForegroundColor Yellow
        Write-Host "1. Check Windows Event Viewer (Windows Logs > Application)" -ForegroundColor White
        Write-Host "2. Check service logs: $DataDir\bridge.log" -ForegroundColor White
        Write-Host "3. Try manual start: net start $ServiceName" -ForegroundColor White
        Write-Host "4. Run directly: `"$targetExe`" --config `"$configPath`"" -ForegroundColor White
        
        # Try to get more error information
        Write-Host ""
        Write-Host "Attempting to run bridge directly for error diagnosis..." -ForegroundColor Yellow
        try {
            $directResult = Start-Process -FilePath $targetExe -ArgumentList @("--config", $configPath, "--log-level", "debug") -Wait -PassThru -NoNewWindow -RedirectStandardError "$env:TEMP\bridge-error.log"
            if (Test-Path "$env:TEMP\bridge-error.log") {
                $errorContent = Get-Content "$env:TEMP\bridge-error.log" -Raw
                if ($errorContent) {
                    Write-Host "Error details: $errorContent" -ForegroundColor Red
                }
            }
        } catch {
            Write-Host "Could not run direct diagnosis" -ForegroundColor Yellow
        }
    }

    # Installation summary
    Write-Host ""
    Write-Host "Installation Summary:" -ForegroundColor Cyan
    Write-Host "====================" -ForegroundColor Cyan
    Write-Host "Installation Path : $InstallPath" -ForegroundColor White
    Write-Host "Data Path         : $DataDir" -ForegroundColor White
    Write-Host "Service Name      : $ServiceName" -ForegroundColor White
    Write-Host "Service Status    : $($svc.Status)" -ForegroundColor $(if($svc.Status -eq 'Running'){'Green'}else{'Red'})
    Write-Host "API Endpoint      : http://localhost:8081" -ForegroundColor White
    Write-Host "Server URL        : $ServerUrl" -ForegroundColor White
    if ($PairCode) { 
        Write-Host "Pair Code Used    : $PairCode" -ForegroundColor White 
        Write-Host ""
        Write-Host "NOTE: Device credentials are stored securely in Windows Credential Manager," -ForegroundColor Cyan
        Write-Host "      not in the config file. This is normal and secure behavior." -ForegroundColor Cyan
    }

    Write-Host ""
    Write-Host "Useful Commands:" -ForegroundColor Cyan
    Write-Host "   gym-door-bridge status           - Check bridge status" -ForegroundColor White
    Write-Host "   gym-door-bridge trigger-heartbeat - Test connectivity" -ForegroundColor White
    Write-Host "   gym-door-bridge device-status    - Check platform status" -ForegroundColor White
    Write-Host "   gym-door-bridge pair CODE        - Pair with platform" -ForegroundColor White
    Write-Host "   net start $ServiceName           - Start service" -ForegroundColor White
    Write-Host "   net stop $ServiceName            - Stop service" -ForegroundColor White

    Write-Host ""
    if ($svc.Status -eq "Running") {
        Write-Host "Gym Door Bridge installation completed successfully!" -ForegroundColor Green
    } else {
        Write-Host "Gym Door Bridge installed but service needs manual start" -ForegroundColor Yellow
    }

} catch {
    Write-Host "ERROR: Installation failed: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "Please check logs and try again." -ForegroundColor Yellow
    exit 1
} finally {
    # Cleanup
    if (Test-Path $TempZip) { 
        Remove-Item $TempZip -Force -ErrorAction SilentlyContinue 
    }
    if (Test-Path $TempExtract) { 
        Remove-Item $TempExtract -Recurse -Force -ErrorAction SilentlyContinue 
    }
}