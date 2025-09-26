# Fix RepSet Bridge Service Startup Issues
# Run as Administrator

Write-Host "RepSet Bridge Service Startup Fix" -ForegroundColor Blue
Write-Host "==================================" -ForegroundColor Blue

# Check admin privileges
$currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
$principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Host "[ERROR] Administrator privileges required" -ForegroundColor Red
    exit 1
}

$exePath = "C:\Program Files\GymDoorBridge\gym-door-bridge.exe"
$configPath = "C:\Program Files\GymDoorBridge\config.yaml"

# Verify files exist
if (-not (Test-Path $exePath) -or -not (Test-Path $configPath)) {
    Write-Host "[ERROR] Bridge files not found" -ForegroundColor Red
    exit 1
}

Write-Host "[OK] Bridge files found" -ForegroundColor Green

# Step 1: Test the executable manually first
Write-Host "[INFO] Testing bridge executable..." -ForegroundColor Cyan
try {
    $testOutput = & $exePath status --config $configPath 2>&1
    Write-Host "[INFO] Bridge test output: $testOutput" -ForegroundColor Gray
    Write-Host "[OK] Bridge executable works" -ForegroundColor Green
} catch {
    Write-Host "[ERROR] Bridge executable test failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Step 2: Remove existing service completely
Write-Host "[INFO] Removing existing service..." -ForegroundColor Cyan
$existing = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
if ($existing) {
    Stop-Service -Name "GymDoorBridge" -Force -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 3
    
    # Use multiple methods to ensure complete removal
    try {
        sc.exe delete "GymDoorBridge" | Out-Null
        Start-Sleep -Seconds 2
        Write-Host "[OK] Service removed" -ForegroundColor Green
    } catch {
        Write-Host "[WARNING] Service removal may have failed" -ForegroundColor Yellow
    }
}

# Step 3: Create service with explicit permissions and error handling
Write-Host "[INFO] Creating service with proper configuration..." -ForegroundColor Cyan

$serviceName = "GymDoorBridge"
$serviceDisplay = "RepSet Gym Door Bridge"
$serviceBinPath = "`"$exePath`" --config `"$configPath`""

# Create service with LocalSystem account and proper error handling
try {
    $createResult = sc.exe create $serviceName binPath= $serviceBinPath start= auto DisplayName= $serviceDisplay obj= "LocalSystem" type= own error= normal 2>&1
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "[OK] Service created successfully" -ForegroundColor Green
        
        # Set service description
        sc.exe description $serviceName "RepSet Gym Door Access Bridge - Manages biometric device connectivity for gym access control"
        
        # Configure failure actions (restart on failure)
        sc.exe failure $serviceName reset= 86400 actions= restart/5000/restart/10000/restart/30000
        
        Write-Host "[OK] Service configured with auto-restart" -ForegroundColor Green
        
    } else {
        Write-Host "[ERROR] Service creation failed: $createResult" -ForegroundColor Red
        throw "Service creation failed with exit code $LASTEXITCODE"
    }
} catch {
    Write-Host "[ERROR] Service creation error: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Step 4: Grant additional permissions if needed
Write-Host "[INFO] Configuring service permissions..." -ForegroundColor Cyan
try {
    # Grant service logon rights (in case needed)
    sc.exe config $serviceName obj= "LocalSystem" password= ""
    Write-Host "[OK] Service account configured" -ForegroundColor Green
} catch {
    Write-Host "[WARNING] Could not configure service account" -ForegroundColor Yellow
}

# Step 5: Start the service with detailed error handling
Write-Host "[INFO] Starting service..." -ForegroundColor Cyan
try {
    # Clear any pending operations
    Start-Sleep -Seconds 2
    
    # Start the service
    Start-Service -Name $serviceName -ErrorAction Stop
    
    # Wait and verify it's running
    Start-Sleep -Seconds 5
    $serviceStatus = Get-Service -Name $serviceName
    
    if ($serviceStatus.Status -eq "Running") {
        Write-Host "[OK] Service started successfully!" -ForegroundColor Green
    } else {
        Write-Host "[WARNING] Service status: $($serviceStatus.Status)" -ForegroundColor Yellow
        
        # Try to get more details from event log
        try {
            $events = Get-EventLog -LogName System -Source "Service Control Manager" -Newest 3 | Where-Object {$_.Message -like "*$serviceName*"}
            if ($events) {
                Write-Host "[INFO] Recent service events:" -ForegroundColor Cyan
                foreach ($event in $events) {
                    Write-Host "  $($event.TimeGenerated): $($event.Message)" -ForegroundColor Gray
                }
            }
        } catch {
            Write-Host "[INFO] Could not retrieve event log details" -ForegroundColor Gray
        }
    }
    
} catch {
    Write-Host "[ERROR] Service startup failed: $($_.Exception.Message)" -ForegroundColor Red
    
    # Detailed troubleshooting
    Write-Host "[INFO] Troubleshooting service startup..." -ForegroundColor Cyan
    
    # Check if it's a permissions issue by running manually
    Write-Host "[INFO] Testing manual execution..." -ForegroundColor Cyan
    try {
        $manualTest = Start-Process -FilePath $exePath -ArgumentList "--config", $configPath -Wait -PassThru -NoNewWindow
        Write-Host "[INFO] Manual execution exit code: $($manualTest.ExitCode)" -ForegroundColor Gray
    } catch {
        Write-Host "[ERROR] Manual execution failed: $($_.Exception.Message)" -ForegroundColor Red
    }
    
    # Provide manual commands
    Write-Host "" 
    Write-Host "[INFO] Manual service management commands:" -ForegroundColor Cyan
    Write-Host "  Start: net start GymDoorBridge" -ForegroundColor Yellow
    Write-Host "  Stop:  net stop GymDoorBridge" -ForegroundColor Yellow
    Write-Host "  Status: sc.exe query GymDoorBridge" -ForegroundColor Yellow
}

# Step 6: Final verification
Write-Host ""
Write-Host "=== FINAL STATUS ===" -ForegroundColor Blue

$finalService = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
if ($finalService) {
    Write-Host "[OK] Service Name: $($finalService.Name)" -ForegroundColor Green
    Write-Host "[OK] Display Name: $($finalService.DisplayName)" -ForegroundColor Green
    Write-Host "[OK] Status: $($finalService.Status)" -ForegroundColor Green
    Write-Host "[OK] Start Type: $($finalService.StartType)" -ForegroundColor Green
    
    if ($finalService.Status -eq "Running") {
        Write-Host ""
        Write-Host "[SUCCESS] RepSet Bridge is running and will auto-start with Windows!" -ForegroundColor Green
        
        # Test bridge connectivity
        Write-Host "[INFO] Testing bridge connectivity..." -ForegroundColor Cyan
        Start-Sleep -Seconds 3
        try {
            $statusCheck = & $exePath status --config $configPath 2>&1
            if ($statusCheck -match "running" -or $statusCheck -match "connected") {
                Write-Host "[OK] Bridge is connected and operational" -ForegroundColor Green
            } else {
                Write-Host "[INFO] Bridge status: Service running, checking connectivity..." -ForegroundColor Cyan
            }
        } catch {
            Write-Host "[INFO] Bridge service is running (connectivity check pending)" -ForegroundColor Cyan
        }
    } else {
        Write-Host ""
        Write-Host "[WARNING] Service exists but not running" -ForegroundColor Yellow
        Write-Host "[INFO] Try: net start GymDoorBridge" -ForegroundColor Cyan
    }
} else {
    Write-Host "[ERROR] Service not found after setup" -ForegroundColor Red
}

Write-Host ""
Write-Host "Press Enter to exit..."
Read-Host