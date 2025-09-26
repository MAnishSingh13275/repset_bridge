# Complete RepSet Bridge Setup - Fix Pairing and Service
# Run as Administrator

Write-Host "RepSet Bridge Complete Setup Fix" -ForegroundColor Blue
Write-Host "=================================" -ForegroundColor Blue

# Check admin privileges
$currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
$principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Host "[ERROR] Administrator privileges required" -ForegroundColor Red
    exit 1
}

$exePath = "C:\Program Files\GymDoorBridge\gym-door-bridge.exe"
$configPath = "C:\Program Files\GymDoorBridge\config.yaml"
$pairCode = "0A99-03C8-6460"

# Verify files exist
if (-not (Test-Path $exePath) -or -not (Test-Path $configPath)) {
    Write-Host "[ERROR] Bridge files not found" -ForegroundColor Red
    exit 1
}

Write-Host "[OK] Bridge files found" -ForegroundColor Green
Write-Host "[INFO] Pair Code: $pairCode" -ForegroundColor Cyan

# Step 1: Check current config file content
Write-Host "[INFO] Checking current configuration..." -ForegroundColor Cyan
$configContent = Get-Content $configPath -Raw
Write-Host "[INFO] Current config preview:" -ForegroundColor Gray
$configLines = Get-Content $configPath | Select-Object -First 10
foreach ($line in $configLines) {
    Write-Host "  $line" -ForegroundColor Gray
}

# Check if device_id is empty
$hasDeviceId = $configContent -match 'device_id:\s*"[^"]+"' -and $configContent -notmatch 'device_id:\s*""'
$hasDeviceKey = $configContent -match 'device_key:\s*"[^"]+"' -and $configContent -notmatch 'device_key:\s*""'

if ($hasDeviceId -and $hasDeviceKey) {
    Write-Host "[OK] Config file already has device credentials" -ForegroundColor Green
} else {
    Write-Host "[WARNING] Config file missing device credentials" -ForegroundColor Yellow
    Write-Host "[INFO] Device ID found: $hasDeviceId" -ForegroundColor Gray
    Write-Host "[INFO] Device Key found: $hasDeviceKey" -ForegroundColor Gray
}

# Step 2: Stop any existing service
Write-Host "[INFO] Stopping existing service..." -ForegroundColor Cyan
$existing = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
if ($existing -and $existing.Status -eq "Running") {
    Stop-Service -Name "GymDoorBridge" -Force -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 2
}

# Step 3: Fix pairing - try multiple approaches
Write-Host "[INFO] Completing bridge pairing..." -ForegroundColor Cyan

# Approach 1: Force re-pair
Write-Host "[INFO] Attempting forced re-pairing..." -ForegroundColor Cyan
try {
    $forceArgs = @("pair", "--pair-code", $pairCode, "--config", $configPath, "--force")
    $pairResult = & $exePath @forceArgs 2>&1
    Write-Host "[INFO] Force pair result: $pairResult" -ForegroundColor Gray
    
    # Check if config was updated
    Start-Sleep -Seconds 1
    $newConfig = Get-Content $configPath -Raw
    $nowHasId = $newConfig -match 'device_id:\s*"[^"]+"' -and $newConfig -notmatch 'device_id:\s*""'
    $nowHasKey = $newConfig -match 'device_key:\s*"[^"]+"' -and $newConfig -notmatch 'device_key:\s*""'
    
    if ($nowHasId -and $nowHasKey) {
        Write-Host "[OK] Force pairing updated config file successfully" -ForegroundColor Green
    } else {
        Write-Host "[WARNING] Force pairing did not update config file" -ForegroundColor Yellow
        
        # Approach 2: Unpair and re-pair
        Write-Host "[INFO] Trying unpair and re-pair..." -ForegroundColor Cyan
        try {
            & $exePath unpair --config $configPath 2>&1 | Out-Null
            Start-Sleep -Seconds 2
            
            $pairArgs = @("pair", "--pair-code", $pairCode, "--config", $configPath)
            $pairResult2 = & $exePath @pairArgs 2>&1
            Write-Host "[INFO] Re-pair result: $pairResult2" -ForegroundColor Gray
            
            # Check again
            Start-Sleep -Seconds 1
            $finalConfig = Get-Content $configPath -Raw
            $finalHasId = $finalConfig -match 'device_id:\s*"[^"]+"' -and $finalConfig -notmatch 'device_id:\s*""'
            $finalHasKey = $finalConfig -match 'device_key:\s*"[^"]+"' -and $finalConfig -notmatch 'device_key:\s*""'
            
            if ($finalHasId -and $finalHasKey) {
                Write-Host "[OK] Re-pairing updated config file successfully" -ForegroundColor Green
            } else {
                Write-Host "[ERROR] Pairing still not updating config file" -ForegroundColor Red
                Write-Host "[INFO] This may be a permissions issue with the config file" -ForegroundColor Yellow
            }
        } catch {
            Write-Host "[WARNING] Unpair/re-pair failed: $($_.Exception.Message)" -ForegroundColor Yellow
        }
    }
} catch {
    Write-Host "[ERROR] Pairing process failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Step 4: Verify the final config state
Write-Host "[INFO] Verifying final configuration..." -ForegroundColor Cyan
$finalConfigContent = Get-Content $configPath -Raw
$finalHasDeviceId = $finalConfigContent -match 'device_id:\s*"[^"]+"' -and $finalConfigContent -notmatch 'device_id:\s*""'
$finalHasDeviceKey = $finalConfigContent -match 'device_key:\s*"[^"]+"' -and $finalConfigContent -notmatch 'device_key:\s*""'

if ($finalHasDeviceId -and $finalHasDeviceKey) {
    Write-Host "[OK] Config file has valid device credentials" -ForegroundColor Green
    
    # Test the bridge manually to ensure it starts
    Write-Host "[INFO] Testing bridge startup..." -ForegroundColor Cyan
    try {
        $testProcess = Start-Process -FilePath $exePath -ArgumentList "--config", $configPath -PassThru -NoNewWindow
        Start-Sleep -Seconds 3
        
        if (-not $testProcess.HasExited) {
            Write-Host "[OK] Bridge starts successfully" -ForegroundColor Green
            Stop-Process -Id $testProcess.Id -Force -ErrorAction SilentlyContinue
        } else {
            Write-Host "[ERROR] Bridge exits immediately (exit code: $($testProcess.ExitCode))" -ForegroundColor Red
        }
    } catch {
        Write-Host "[ERROR] Bridge test failed: $($_.Exception.Message)" -ForegroundColor Red
    }
    
} else {
    Write-Host "[ERROR] Config file still missing device credentials" -ForegroundColor Red
    Write-Host "[INFO] Manual pairing may be required from the RepSet admin dashboard" -ForegroundColor Yellow
    
    # Show current config for debugging
    Write-Host "[INFO] Current config file content:" -ForegroundColor Cyan
    Get-Content $configPath | ForEach-Object { Write-Host "  $_" -ForegroundColor Gray }
}

# Step 5: Set up or fix the Windows service
Write-Host "[INFO] Setting up Windows service..." -ForegroundColor Cyan

# Remove existing service
if ($existing) {
    try {
        sc.exe delete "GymDoorBridge" | Out-Null
        Start-Sleep -Seconds 2
        Write-Host "[OK] Removed existing service" -ForegroundColor Green
    } catch {
        Write-Host "[WARNING] Could not remove existing service" -ForegroundColor Yellow
    }
}

# Create service with proper configuration
$serviceName = "GymDoorBridge"
$serviceDisplay = "RepSet Gym Door Bridge"
$serviceBinPath = "`"$exePath`" --config `"$configPath`""

try {
    $createResult = sc.exe create $serviceName binPath= $serviceBinPath start= auto DisplayName= $serviceDisplay obj= "LocalSystem" type= own error= normal 2>&1
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "[OK] Service created successfully" -ForegroundColor Green
        
        # Set service description and failure actions
        sc.exe description $serviceName "RepSet Gym Door Access Bridge - Manages biometric device connectivity" | Out-Null
        sc.exe failure $serviceName reset= 86400 actions= restart/5000/restart/10000/restart/30000 | Out-Null
        
        # Try to start the service
        if ($finalHasDeviceId -and $finalHasDeviceKey) {
            Write-Host "[INFO] Starting service..." -ForegroundColor Cyan
            try {
                Start-Service -Name $serviceName -ErrorAction Stop
                Start-Sleep -Seconds 3
                
                $serviceStatus = Get-Service -Name $serviceName
                if ($serviceStatus.Status -eq "Running") {
                    Write-Host "[OK] Service started successfully!" -ForegroundColor Green
                } else {
                    Write-Host "[WARNING] Service status: $($serviceStatus.Status)" -ForegroundColor Yellow
                }
            } catch {
                Write-Host "[ERROR] Service startup failed: $($_.Exception.Message)" -ForegroundColor Red
                Write-Host "[INFO] Check Windows Event Viewer for detailed error information" -ForegroundColor Cyan
            }
        } else {
            Write-Host "[WARNING] Service created but not started due to missing device credentials" -ForegroundColor Yellow
        }
        
    } else {
        Write-Host "[ERROR] Service creation failed: $createResult" -ForegroundColor Red
    }
} catch {
    Write-Host "[ERROR] Service setup failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Step 6: Final status report
Write-Host ""
Write-Host "=== FINAL STATUS REPORT ===" -ForegroundColor Blue

# Config status
if ($finalHasDeviceId -and $finalHasDeviceKey) {
    Write-Host "[OK] Configuration: Device credentials present" -ForegroundColor Green
} else {
    Write-Host "[ERROR] Configuration: Missing device credentials" -ForegroundColor Red
}

# Service status
$finalService = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
if ($finalService) {
    Write-Host "[OK] Service: $($finalService.Status) (Start Type: $($finalService.StartType))" -ForegroundColor Green
    
    if ($finalService.Status -eq "Running") {
        Write-Host ""
        Write-Host "[SUCCESS] RepSet Bridge is fully operational!" -ForegroundColor Green
        Write-Host "[INFO] The bridge will automatically start with Windows" -ForegroundColor Cyan
        Write-Host "[INFO] You can check the RepSet dashboard to verify connectivity" -ForegroundColor Cyan
    } else {
        Write-Host ""
        Write-Host "[WARNING] Service exists but not running" -ForegroundColor Yellow
        Write-Host "[INFO] Try manually: net start GymDoorBridge" -ForegroundColor Cyan
    }
} else {
    Write-Host "[ERROR] Service: Not found" -ForegroundColor Red
}

# Next steps
Write-Host ""
Write-Host "=== NEXT STEPS ===" -ForegroundColor Blue
if (-not $finalHasDeviceId -or -not $finalHasDeviceKey) {
    Write-Host "[INFO] Bridge pairing incomplete - please check RepSet admin dashboard" -ForegroundColor Cyan
    Write-Host "[INFO] You may need to generate a new pair code and run:" -ForegroundColor Cyan
    Write-Host "  & '$exePath' pair --pair-code NEW_CODE --config '$configPath'" -ForegroundColor Yellow
} elseif ($finalService -and $finalService.Status -ne "Running") {
    Write-Host "[INFO] Configuration looks good - try starting the service:" -ForegroundColor Cyan
    Write-Host "  net start GymDoorBridge" -ForegroundColor Yellow
} else {
    Write-Host "[INFO] Setup complete! Check your RepSet dashboard for bridge status." -ForegroundColor Cyan
}

Write-Host ""
Write-Host "Press Enter to exit..."
Read-Host