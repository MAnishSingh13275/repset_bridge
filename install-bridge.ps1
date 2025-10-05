# Gym Door Bridge - Enhanced Installation Script
# Handles permissions, service configuration, pairing, and startup properly
# Version 1.1 - Enhanced with better error handling and validation

param(
    [string]$PairCode = "",
    [string]$ServerUrl = "https://repset.onezy.in",
    [string]$InstallPath = "$env:ProgramFiles\GymDoorBridge",
    [switch]$Force = $false
)

# Helper function to test if bridge is properly configured
function Test-BridgeConfiguration {
    param(
        [string]$ExePath,
        [string]$ConfigPath
    )
    
    try {
        Write-Host "Testing bridge configuration..." -ForegroundColor Yellow
        
        # Test if bridge can load config without errors
        $testArgs = @("status", "--config", "`"$ConfigPath`"")
        $testResult = Start-Process -FilePath $ExePath -ArgumentList $testArgs -Wait -PassThru -NoNewWindow -WindowStyle Hidden
        
        if ($testResult.ExitCode -eq 0) {
            Write-Host "✅ Bridge configuration test passed" -ForegroundColor Green
            return $true
        } else {
            Write-Host "⚠️  Bridge configuration test failed (exit code: $($testResult.ExitCode))" -ForegroundColor Yellow
            return $false
        }
    } catch {
        Write-Host "❌ Bridge configuration test error: $($_.Exception.Message)" -ForegroundColor Red
        return $false
    }
}

# Helper function to check service health after startup
function Wait-ForServiceHealth {
    param(
        [string]$ServiceName,
        [int]$TimeoutSeconds = 120
    )
    
    Write-Host "Waiting for service to become healthy..." -ForegroundColor Yellow
    
    $maxAttempts = [math]::Ceiling($TimeoutSeconds / 5)
    $attempt = 0
    
    while ($attempt -lt $maxAttempts) {
        $attempt++
        Start-Sleep 5
        
        $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
        if (-not $svc) {
            Write-Host "Service not found on attempt $attempt/$maxAttempts" -ForegroundColor Yellow
            continue
        }
        
        if ($svc.Status -eq "Running") {
            # Test if API is responding
            try {
                $healthResponse = Invoke-WebRequest -Uri "http://localhost:8081/api/v1/health" -UseBasicParsing -TimeoutSec 10 -ErrorAction Stop
                if ($healthResponse.StatusCode -eq 200) {
                    Write-Host "✅ Service is healthy and API is responding" -ForegroundColor Green
                    return $true
                }
            } catch {
                # API not ready yet, continue waiting
            }
            
            Write-Host "Service running but API not ready yet... ($($attempt * 5)s)" -ForegroundColor Yellow
        } else {
            Write-Host "Service status: $($svc.Status) on attempt $attempt/$maxAttempts" -ForegroundColor Yellow
        }
    }
    
    Write-Host "⚠️  Service health check timed out after $TimeoutSeconds seconds" -ForegroundColor Yellow
    return $false
}

# Check admin privileges
if (-NOT ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    Write-Host "ERROR: This script requires Administrator privileges!" -ForegroundColor Red
    Write-Host "Please run PowerShell as Administrator and try again." -ForegroundColor Yellow
    exit 1
}

Write-Host "Gym Door Bridge Installation" -ForegroundColor Cyan
Write-Host "============================" -ForegroundColor Cyan

# Constants (define before using)
$ServiceName = "GymDoorBridge"
$ServiceDisp = "Gym Door Access Bridge"
$DataDir = Join-Path $env:ProgramData "GymDoorBridge"
$TempZip = Join-Path $env:TEMP "gym-door-bridge.zip"
$TempExtract = Join-Path $env:TEMP "gym-door-bridge"
$DownloadUrl = "https://github.com/MAnishSingh13275/repset_bridge/releases/latest/download/gym-door-bridge-windows.zip"

# Add Windows Defender exclusions to prevent false positives
Write-Host "Setting up Windows Defender exclusions..." -ForegroundColor Green
try {
    Add-MpPreference -ExclusionPath $InstallPath -ErrorAction SilentlyContinue
    Add-MpPreference -ExclusionPath $DataDir -ErrorAction SilentlyContinue
    Add-MpPreference -ExclusionProcess "gym-door-bridge.exe" -ErrorAction SilentlyContinue
    Write-Host "Windows Defender exclusions added successfully" -ForegroundColor Green
} catch {
    Write-Host "WARNING: Could not add Windows Defender exclusions: $($_.Exception.Message)" -ForegroundColor Yellow
    Write-Host "You may need to add exclusions manually if Windows Defender interferes" -ForegroundColor Yellow
}

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
        
        # Also ensure the credential storage directory has proper permissions
        $credDir = Join-Path $DataDir "credentials"
        New-Item -ItemType Directory -Force -Path $credDir | Out-Null
        & icacls $credDir /grant "NT AUTHORITY\SYSTEM:(OI)(CI)F" /T | Out-Null
        & icacls $credDir /grant "NT AUTHORITY\LOCAL SERVICE:(OI)(CI)F" /T | Out-Null
        & icacls $credDir /grant "BUILTIN\Administrators:(OI)(CI)F" /T | Out-Null
        
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
    
    # Remove any existing service first
    & sc.exe delete $ServiceName 2>$null | Out-Null
    Start-Sleep 2
    
    try {
        # Create service with proper settings - use PowerShell method for better quote handling
        $binPath = "`"$targetExe`" --config `"$configPath`""
        
        Write-Host "Creating service with binary path: $binPath" -ForegroundColor Yellow
        
        # Use PowerShell New-Service instead of sc.exe for better quote handling
        New-Service -Name $ServiceName -BinaryPathName $binPath -DisplayName $ServiceDisp -StartupType Automatic -Description "Gym Door Access Bridge - integrates RepSet with door controllers" | Out-Null
        Write-Host "Service created with PowerShell New-Service" -ForegroundColor Green
        

        
    } catch {
        Write-Host "ERROR: Failed to create service: $($_.Exception.Message)" -ForegroundColor Red
        exit 1
    }
    
    # Configure service account and failure recovery using sc.exe
    try {
        Write-Host "Configuring service settings..." -ForegroundColor Yellow
        
        # Configure service to run as LocalService
        & sc.exe config $ServiceName obj= "NT AUTHORITY\LocalService" 2>&1 | Out-Null
        
        # Configure failure recovery
        & sc.exe failure $ServiceName reset= 86400 actions= restart/5000/restart/10000/restart/30000 2>&1 | Out-Null
        
        Write-Host "Service configuration completed" -ForegroundColor Green
    } catch {
        Write-Host "WARNING: Could not configure all service settings: $($_.Exception.Message)" -ForegroundColor Yellow
    }

    # Verify service exists
    Write-Host "Verifying service creation..." -ForegroundColor Yellow
    Start-Sleep 3
    
    $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if (-not $svc) {
        Write-Host "ERROR: Service not found after creation" -ForegroundColor Red
        
        # Try to get more information
        Write-Host "Checking all services for partial matches..." -ForegroundColor Yellow
        $allServices = Get-Service | Where-Object { $_.Name -like "*Gym*" -or $_.DisplayName -like "*Gym*" }
        if ($allServices) {
            Write-Host "Found related services:" -ForegroundColor Yellow
            $allServices | ForEach-Object { Write-Host "  - $($_.Name): $($_.DisplayName)" -ForegroundColor Gray }
        } else {
            Write-Host "No related services found" -ForegroundColor Yellow
        }
        
        # Check with sc.exe
        Write-Host "Checking with sc.exe query..." -ForegroundColor Yellow
        $scResult = & sc.exe query $ServiceName 2>&1
        Write-Host "SC query result: $scResult" -ForegroundColor Gray
        
        exit 1
    }
    Write-Host "Service verification successful: $($svc.Status)" -ForegroundColor Green

    # Pairing (before starting service)
    if ($PairCode) {
        Write-Host "Pairing device with platform..." -ForegroundColor Green
        
        # Clean existing pairing
        Start-Process -FilePath $targetExe -ArgumentList "unpair" -Wait -NoNewWindow -ErrorAction SilentlyContinue | Out-Null
        Start-Sleep 2
        
        # Perform pairing with proper argument format
        $pairArguments = @("pair", $PairCode, "--config", "`"$configPath`"")
        Write-Host "Running pairing command: $targetExe $($pairArguments -join ' ')" -ForegroundColor Yellow
        
        $pairResult = Start-Process -FilePath $targetExe -ArgumentList $pairArguments -Wait -PassThru -NoNewWindow
        if ($pairResult.ExitCode -eq 0) {
            Write-Host "✅ Device paired successfully!" -ForegroundColor Green
            
            # Verify pairing and test configuration
            Write-Host "Verifying pairing and testing configuration..." -ForegroundColor Yellow
            
            Start-Sleep 2  # Give time for credential storage
            
            $configTest = Test-BridgeConfiguration -ExePath $targetExe -ConfigPath $configPath
            if ($configTest) {
                Write-Host "✅ Pairing and configuration verification successful" -ForegroundColor Green
            } else {
                Write-Host "⚠️  Configuration test failed after pairing - service may have startup issues" -ForegroundColor Yellow
            }
        } else {
            Write-Host "❌ Pairing failed (exit code: $($pairResult.ExitCode))" -ForegroundColor Red
            Write-Host "You can pair manually later with: gym-door-bridge pair $PairCode --config `"$configPath`"" -ForegroundColor Yellow
            
            # Continue with service creation anyway, user can pair later
            Write-Host "Continuing with service installation..." -ForegroundColor Yellow
        }
    }

    # Start service with multiple attempts and better error handling
    Write-Host "Starting service..." -ForegroundColor Yellow
    
    $serviceStarted = $false
    $maxAttempts = 5  # Increased attempts
    
    # Wait a bit for service registration to complete
    Start-Sleep 3
    
    for ($attempt = 1; $attempt -le $maxAttempts; $attempt++) {
        Write-Host "Start attempt $attempt/$maxAttempts..." -ForegroundColor Yellow
        
        try {
            # First check if service exists and is not already running
            $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
            if (-not $svc) {
                Write-Host "Service $ServiceName not found, waiting for registration..." -ForegroundColor Yellow
                Start-Sleep 5
                continue
            }
            
            if ($svc.Status -eq "Running") {
                Write-Host "Service is already running!" -ForegroundColor Green
                $serviceStarted = $true
                break
            }
            
            # Try starting with PowerShell Start-Service first
            try {
                Start-Service -Name $ServiceName -ErrorAction Stop
                Write-Host "Service start command issued via PowerShell" -ForegroundColor Green
            } catch {
                Write-Host "PowerShell Start-Service failed: $($_.Exception.Message)" -ForegroundColor Yellow
                # Fallback to net command
                $netResult = & net start $ServiceName 2>&1
                Write-Host "net start result: $netResult" -ForegroundColor Gray
            }
            
            # Wait for service to start and check multiple times
            $waitAttempts = 0
            $maxWaitAttempts = 12 # 60 seconds total
            while ($waitAttempts -lt $maxWaitAttempts) {
                Start-Sleep 5
                $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
                if ($svc -and $svc.Status -eq "Running") {
                    $serviceStarted = $true
                    Write-Host "Service started successfully after $($waitAttempts * 5) seconds!" -ForegroundColor Green
                    break
                }
                $waitAttempts++
                Write-Host "Waiting for service to start... ($($waitAttempts * 5)s)" -ForegroundColor Yellow
            }
            
            if ($serviceStarted) {
                break
            } else {
                $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
                if ($svc) {
                    Write-Host "Service status after wait: $($svc.Status)" -ForegroundColor Yellow
                } else {
                    Write-Host "Service not found after wait" -ForegroundColor Red
                }
            }
            
        } catch {
            Write-Host "Start attempt $attempt failed: $($_.Exception.Message)" -ForegroundColor Yellow
        }
        
        if (-not $serviceStarted -and $attempt -lt $maxAttempts) {
            Write-Host "Retrying in 10 seconds..." -ForegroundColor Yellow
            Start-Sleep 10
        }
    }
    
    # Final status check using enhanced health monitoring
    $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($svc.Status -eq "Running") {
        Write-Host "Service is running, checking health..." -ForegroundColor Green
        
        # Use the health check function with longer timeout for first startup
        $isHealthy = Wait-ForServiceHealth -ServiceName $ServiceName -TimeoutSeconds 90
        
        if ($isHealthy) {
            Write-Host "✅ Service is healthy and fully operational!" -ForegroundColor Green
            
            # Test bridge status if paired
            if ($PairCode) {
                Write-Host "Testing paired bridge status..." -ForegroundColor Yellow
                try {
                    $statusArgs = @("status", "--config", "`"$configPath`"")
                    $statusResult = Start-Process -FilePath $targetExe -ArgumentList $statusArgs -Wait -PassThru -NoNewWindow
                    if ($statusResult.ExitCode -eq 0) {
                        Write-Host "✅ Bridge status check successful!" -ForegroundColor Green
                    } else {
                        Write-Host "⚠️  Bridge status check returned exit code: $($statusResult.ExitCode)" -ForegroundColor Yellow
                    }
                } catch {
                    Write-Host "⚠️  Bridge status check failed: $($_.Exception.Message)" -ForegroundColor Yellow
                }
            }
        } else {
            Write-Host "⚠️  Service is running but not healthy - API may still be starting up" -ForegroundColor Yellow
        }
        
    } else {
        Write-Host "ERROR: Service failed to start (Status: $($svc.Status))" -ForegroundColor Red
        Write-Host ""
        Write-Host "Troubleshooting steps:" -ForegroundColor Yellow
        Write-Host "1. Check Windows Event Viewer (Windows Logs > Application)" -ForegroundColor White
        Write-Host "2. Check service logs: $DataDir\bridge.log" -ForegroundColor White
        Write-Host "3. Try manual start: net start $ServiceName" -ForegroundColor White
        Write-Host "4. Run directly: `"$targetExe`" --config `"$configPath`"" -ForegroundColor White
        
        # Enhanced error diagnosis
        Write-Host ""
        Write-Host "Running enhanced diagnostics..." -ForegroundColor Yellow
        
        # Check Windows Event Log for service errors
        try {
            Write-Host "Checking Windows Event Log for recent service errors..." -ForegroundColor Yellow
            $eventLogEntries = Get-EventLog -LogName Application -Source "Service Control Manager" -Newest 5 -ErrorAction SilentlyContinue | Where-Object { $_.Message -like "*$ServiceName*" }
            if ($eventLogEntries) {
                Write-Host "Recent service-related events:" -ForegroundColor Yellow
                $eventLogEntries | ForEach-Object {
                    Write-Host "  [$($_.TimeGenerated)] $($_.EntryType): $($_.Message)" -ForegroundColor Gray
                }
            } else {
                Write-Host "No recent service events found in Event Log" -ForegroundColor Gray
            }
        } catch {
            Write-Host "Could not check Event Log: $($_.Exception.Message)" -ForegroundColor Yellow
        }
        
        # Check if log file exists and has content
        $logPath = Join-Path $DataDir "bridge.log"
        if (Test-Path $logPath) {
            Write-Host "Checking bridge log file: $logPath" -ForegroundColor Yellow
            try {
                $logContent = Get-Content $logPath -Tail 20 -ErrorAction SilentlyContinue
                if ($logContent) {
                    Write-Host "Last 20 lines of bridge log:" -ForegroundColor Yellow
                    $logContent | ForEach-Object { Write-Host "  $_" -ForegroundColor Gray }
                } else {
                    Write-Host "Log file exists but is empty" -ForegroundColor Gray
                }
            } catch {
                Write-Host "Could not read log file: $($_.Exception.Message)" -ForegroundColor Yellow
            }
        } else {
            Write-Host "Bridge log file does not exist: $logPath" -ForegroundColor Gray
        }
        
        # Try to run bridge directly with timeout
        Write-Host "Attempting to run bridge directly for error diagnosis..." -ForegroundColor Yellow
        try {
            # Create temporary log files
            $tempErrorLog = Join-Path $env:TEMP "bridge-diagnostic-error.log"
            $tempOutputLog = Join-Path $env:TEMP "bridge-diagnostic-output.log"
            
            # Use proper argument array with quoted config path
            $arguments = @("--config", "`"$configPath`"", "--log-level", "debug")
            Write-Host "Running: $targetExe $($arguments -join ' ')" -ForegroundColor Gray
            Write-Host "This will run for 10 seconds to capture startup errors..." -ForegroundColor Gray
            
            # Start the process but kill it after 10 seconds
            $process = Start-Process -FilePath $targetExe -ArgumentList $arguments -PassThru -NoNewWindow -RedirectStandardError $tempErrorLog -RedirectStandardOutput $tempOutputLog
            
            # Wait 10 seconds then kill if still running
            if (!$process.WaitForExit(10000)) {
                Write-Host "Stopping diagnostic run after 10 seconds..." -ForegroundColor Gray
                $process.Kill()
                $process.WaitForExit(5000)
            }
            
            # Read the captured output
            Start-Sleep 1
            
            if (Test-Path $tempErrorLog) {
                $errorContent = Get-Content $tempErrorLog -Raw -ErrorAction SilentlyContinue
                if ($errorContent -and $errorContent.Trim() -ne "") {
                    Write-Host "Diagnostic Error Output:" -ForegroundColor Red
                    $errorContent -split "\n" | ForEach-Object { if ($_.Trim()) { Write-Host "  $_" -ForegroundColor Red } }
                }
                Remove-Item $tempErrorLog -Force -ErrorAction SilentlyContinue
            }
            
            if (Test-Path $tempOutputLog) {
                $outputContent = Get-Content $tempOutputLog -Raw -ErrorAction SilentlyContinue
                if ($outputContent -and $outputContent.Trim() -ne "") {
                    Write-Host "Diagnostic Standard Output:" -ForegroundColor Yellow
                    $outputContent -split "\n" | ForEach-Object { if ($_.Trim()) { Write-Host "  $_" -ForegroundColor Yellow } }
                }
                Remove-Item $tempOutputLog -Force -ErrorAction SilentlyContinue
            }
        } catch {
            Write-Host "Could not run direct diagnosis: $($_.Exception.Message)" -ForegroundColor Yellow
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

    # Final validation and recommendations
    Write-Host ""
    Write-Host "Final Installation Status:" -ForegroundColor Cyan
    Write-Host "========================" -ForegroundColor Cyan
    
    $finalSvc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($finalSvc -and $finalSvc.Status -eq "Running") {
        Write-Host "✅ Installation completed successfully!" -ForegroundColor Green
        Write-Host "✅ Service is running" -ForegroundColor Green
        
        if ($PairCode) {
            Write-Host "✅ Device pairing was attempted" -ForegroundColor Green
        }
        
        # Test API accessibility
        Write-Host ""
        Write-Host "Testing API accessibility..." -ForegroundColor Yellow
        try {
            $apiResponse = Invoke-WebRequest -Uri "http://localhost:8081/api/v1/health" -UseBasicParsing -TimeoutSec 15 -ErrorAction Stop
            Write-Host "✅ API is accessible at http://localhost:8081" -ForegroundColor Green
        } catch {
            Write-Host "⚠️  API not yet accessible (this is normal, may take a few minutes)" -ForegroundColor Yellow
        }
        
        Write-Host ""
        Write-Host "🎉 Gym Door Bridge is installed and running!" -ForegroundColor Green
        
    } elseif ($finalSvc) {
        Write-Host "⚠️  Installation completed but service is not running" -ForegroundColor Yellow
        Write-Host "   Service Status: $($finalSvc.Status)" -ForegroundColor Yellow
        Write-Host ""
        Write-Host "Next Steps:" -ForegroundColor Yellow
        Write-Host "1. Check the diagnostic output above for errors" -ForegroundColor White
        Write-Host "2. Try starting manually: net start $ServiceName" -ForegroundColor White
        Write-Host "3. Check Event Viewer: Windows Logs > Application" -ForegroundColor White
        Write-Host "4. Check bridge logs: $DataDir\bridge.log" -ForegroundColor White
        if (-not $PairCode) {
            Write-Host "5. Pair the device: gym-door-bridge pair YOUR_PAIR_CODE --config `"$configPath`"" -ForegroundColor White
        }
        
    } else {
        Write-Host "❌ Installation failed - service not found" -ForegroundColor Red
        Write-Host ""
        Write-Host "Please check the diagnostic output above and contact support if needed." -ForegroundColor Yellow
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