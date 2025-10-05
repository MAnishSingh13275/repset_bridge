# Gym Door Bridge - Simple Installation Script
# Compatible with all PowerShell versions

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

    # Check existing service
    $existingService = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($existingService -and -not $Force -and -not $PairCode) {
        Write-Host "WARNING: Gym Door Bridge is already installed!" -ForegroundColor Yellow
        Write-Host "Service Status: $($existingService.Status)" -ForegroundColor White
        Write-Host "Use -Force to reinstall or provide -PairCode to re-pair." -ForegroundColor Yellow
        exit 0
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

    # Stop existing service
    if ($existingService) {
        Write-Host "Stopping existing service..." -ForegroundColor Yellow
        try {
            Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
            Start-Sleep 3
            
            # Force kill processes
            Get-Process -Name "gym-door-bridge*" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
            Start-Sleep 2
            
            # Uninstall old service
            $existingExe = Join-Path $InstallPath "gym-door-bridge.exe"
            if (Test-Path $existingExe) {
                Start-Process -FilePath $existingExe -ArgumentList "uninstall" -Wait -NoNewWindow -ErrorAction SilentlyContinue
                Start-Sleep 2
            }
        } catch {
            Write-Host "WARNING: Could not fully stop existing service" -ForegroundColor Yellow
        }
    }

    # Copy executable
    $targetExe = Join-Path $InstallPath "gym-door-bridge.exe"
    Write-Host "Installing executable..." -ForegroundColor Green
    
    if (Test-Path $targetExe) {
        for ($i = 0; $i -lt 5; $i++) {
            try { 
                Remove-Item $targetExe -Force 
                break 
            } catch { 
                Start-Sleep 2 
            }
        }
    }
    
    try {
        Copy-Item -Path $exe.FullName -Destination $targetExe -Force
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
"@

    $configContent | Set-Content -Path $configPath -Encoding UTF8
    Write-Host "Configuration created successfully" -ForegroundColor Green

    # Install service
    Write-Host "Installing Windows service..." -ForegroundColor Green
    
    try {
        $installResult = Start-Process -FilePath $targetExe -ArgumentList "install" -Wait -PassThru -NoNewWindow
        if ($installResult.ExitCode -ne 0) {
            Write-Host "App install failed, creating service manually..." -ForegroundColor Yellow
            $binPath = "`"$targetExe`" --config `"$configPath`""
            New-Service -Name $ServiceName -BinaryPathName $binPath -DisplayName $ServiceDisp -StartupType Automatic -Description "Gym Door Access Bridge" | Out-Null
        }
        Write-Host "Service installed successfully" -ForegroundColor Green
    } catch {
        Write-Host "WARNING: Service installation had issues, trying manual creation..." -ForegroundColor Yellow
        try {
            $binPath = "`"$targetExe`" --config `"$configPath`""
            New-Service -Name $ServiceName -BinaryPathName $binPath -DisplayName $ServiceDisp -StartupType Automatic -Description "Gym Door Access Bridge" | Out-Null
            Write-Host "Service created manually" -ForegroundColor Green
        } catch {
            Write-Host "ERROR: Failed to create service: $($_.Exception.Message)" -ForegroundColor Red
            exit 1
        }
    }

    # Configure service recovery
    $binPath = "`"$targetExe`" --config `"$configPath`""
    & sc.exe config $ServiceName binPath= $binPath | Out-Null
    & sc.exe failure $ServiceName reset= 86400 actions= restart/5000/restart/10000/restart/30000 | Out-Null

    # Verify service exists
    Start-Sleep 2
    $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if (-not $svc) {
        Write-Host "ERROR: Service not found after installation" -ForegroundColor Red
        exit 1
    }
    Write-Host "Service verification successful: $($svc.Status)" -ForegroundColor Green

    # Pairing
    if ($PairCode) {
        Write-Host "Pairing device with platform..." -ForegroundColor Green
        
        # Clean existing pairing
        Start-Process -FilePath $targetExe -ArgumentList "unpair" -Wait -NoNewWindow -ErrorAction SilentlyContinue | Out-Null
        Start-Sleep 1
        
        # Perform pairing
        $pairResult = Start-Process -FilePath $targetExe -ArgumentList @("pair", $PairCode) -Wait -PassThru -NoNewWindow
        if ($pairResult.ExitCode -eq 0) {
            Write-Host "Device paired successfully!" -ForegroundColor Green
            
            # Verify pairing
            try {
                $cfgContent = Get-Content $configPath -Raw
                if ($cfgContent -match 'device_id:\s*"([^"]+)"') {
                    if ($matches[1]) {
                        Write-Host "Device ID: $($matches[1])" -ForegroundColor Green
                    }
                } else {
                    Write-Host "WARNING: Pairing completed but device_id not found in config" -ForegroundColor Yellow
                }
            } catch {
                Write-Host "WARNING: Could not verify pairing in config file" -ForegroundColor Yellow
            }
        } else {
            Write-Host "WARNING: Pairing failed (exit code: $($pairResult.ExitCode))" -ForegroundColor Yellow
            Write-Host "You can pair manually later with: gym-door-bridge pair $PairCode" -ForegroundColor Yellow
        }
    }

    # Start service
    Write-Host "Starting service..." -ForegroundColor Yellow
    
    # Try multiple methods to start the service
    $serviceStarted = $false
    
    # Method 1: PowerShell Start-Service
    try {
        Start-Service -Name $ServiceName -ErrorAction Stop
        Start-Sleep 3
        $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
        if ($svc.Status -eq "Running") {
            $serviceStarted = $true
            Write-Host "Service started successfully with PowerShell!" -ForegroundColor Green
        }
    } catch {
        Write-Host "PowerShell start failed: $($_.Exception.Message)" -ForegroundColor Yellow
    }
    
    # Method 2: net start command
    if (-not $serviceStarted) {
        try {
            $netResult = & net start $ServiceName 2>&1
            Start-Sleep 3
            $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
            if ($svc.Status -eq "Running") {
                $serviceStarted = $true
                Write-Host "Service started successfully with net command!" -ForegroundColor Green
            } else {
                Write-Host "Net start output: $netResult" -ForegroundColor Yellow
            }
        } catch {
            Write-Host "Net start failed: $($_.Exception.Message)" -ForegroundColor Yellow
        }
    }
    
    # Method 3: sc start command
    if (-not $serviceStarted) {
        try {
            & sc.exe start $ServiceName | Out-Null
            Start-Sleep 5
            $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
            if ($svc.Status -eq "Running") {
                $serviceStarted = $true
                Write-Host "Service started successfully with sc command!" -ForegroundColor Green
            }
        } catch {
            Write-Host "SC start failed: $($_.Exception.Message)" -ForegroundColor Yellow
        }
    }
    
    # Final status check
    $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($svc.Status -eq "Running") {
        Write-Host "Service is running successfully!" -ForegroundColor Green
        
        # Test API
        Write-Host "Testing API connectivity..." -ForegroundColor Yellow
        $apiWorking = $false
        for ($i = 1; $i -le 6; $i++) {
            try {
                Start-Sleep 5
                $response = Invoke-WebRequest -Uri "http://localhost:8081/api/v1/health" -UseBasicParsing -TimeoutSec 10
                Write-Host "API is responding: HTTP $($response.StatusCode)" -ForegroundColor Green
                $apiWorking = $true
                break
            } catch {
                Write-Host "API test attempt $i/6 failed, waiting..." -ForegroundColor Yellow
            }
        }
        
        if (-not $apiWorking) {
            Write-Host "WARNING: API not responding yet, but service is running" -ForegroundColor Yellow
            Write-Host "The API may take a few more minutes to start" -ForegroundColor Yellow
        }
    } else {
        Write-Host "ERROR: Service failed to start (Status: $($svc.Status))" -ForegroundColor Red
        Write-Host "Manual troubleshooting steps:" -ForegroundColor Yellow
        Write-Host "1. Check Windows Event Viewer for errors" -ForegroundColor White
        Write-Host "2. Try: net start $ServiceName" -ForegroundColor White
        Write-Host "3. Check logs at: $DataDir\bridge.log" -ForegroundColor White
        Write-Host "4. Verify config at: $configPath" -ForegroundColor White
    }

    # Installation summary
    Write-Host ""
    Write-Host "Installation Summary:" -ForegroundColor Cyan
    Write-Host "====================" -ForegroundColor Cyan
    Write-Host "Installation Path : $InstallPath" -ForegroundColor White
    Write-Host "Data Path         : $DataDir" -ForegroundColor White
    Write-Host "Service Name      : $ServiceName" -ForegroundColor White
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
    Write-Host "   gym-door-bridge status    - Check bridge status" -ForegroundColor White
    Write-Host "   gym-door-bridge pair CODE - Pair with platform" -ForegroundColor White
    Write-Host "   gym-door-bridge unpair    - Unpair from platform" -ForegroundColor White
    Write-Host "   net start $ServiceName    - Start service" -ForegroundColor White
    Write-Host "   net stop $ServiceName     - Stop service" -ForegroundColor White

    Write-Host ""
    Write-Host "Gym Door Bridge installation completed successfully!" -ForegroundColor Green

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