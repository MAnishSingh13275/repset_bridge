# Fix RepSet Bridge Config and Create Service
# Run as Administrator

Write-Host "RepSet Bridge Fix and Service Setup" -ForegroundColor Blue
Write-Host "====================================" -ForegroundColor Blue

# Check admin privileges
$currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
$principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Host "[ERROR] Administrator privileges required" -ForegroundColor Red
    Write-Host "Please run PowerShell as Administrator" -ForegroundColor Yellow
    exit 1
}

# Paths
$installPath = "C:\Program Files\GymDoorBridge"
$exePath = "$installPath\gym-door-bridge.exe"
$configPath = "$installPath\config.yaml"

# Check if bridge is installed
if (-not (Test-Path $exePath)) {
    Write-Host "[ERROR] Bridge not found at $exePath" -ForegroundColor Red
    Write-Host "Please run the installer first" -ForegroundColor Yellow
    exit 1
}

Write-Host "[OK] Found bridge installation" -ForegroundColor Green

# Step 1: Fix config file (remove BOM, fix format)
Write-Host "[INFO] Fixing config file..." -ForegroundColor Cyan

$configContent = @"
# RepSet Bridge Configuration
device_id: ""
device_key: ""
server_url: "https://repset.onezy.in"
tier: "normal"
queue_max_size: 10000
heartbeat_interval: 60
unlock_duration: 3000
database_path: "./bridge.db"
log_level: "info"
log_file: ""
enabled_adapters:
  - "simulator"
"@

try {
    # Use UTF8 without BOM
    $utf8NoBomEncoding = New-Object System.Text.UTF8Encoding($false)
    [System.IO.File]::WriteAllText($configPath, $configContent, $utf8NoBomEncoding)
    Write-Host "[OK] Config file fixed" -ForegroundColor Green
} catch {
    Write-Host "[ERROR] Failed to fix config: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Step 2: Test the fixed config
Write-Host "[INFO] Testing bridge with fixed config..." -ForegroundColor Cyan
try {
    $testResult = & $exePath pair --pair-code "0A99-03C8-6460" --config $configPath 2>&1
    Write-Host "[INFO] Pairing output: $testResult" -ForegroundColor Gray
    
    if ($testResult -match "successfully" -or $testResult -match "already") {
        Write-Host "[OK] Bridge paired successfully!" -ForegroundColor Green
    } else {
        Write-Host "[WARNING] Pairing may have issues, but continuing..." -ForegroundColor Yellow
    }
} catch {
    Write-Host "[WARNING] Pairing test failed, but continuing with service setup..." -ForegroundColor Yellow
}

# Step 3: Create Windows service
Write-Host "[INFO] Creating Windows service..." -ForegroundColor Cyan

# Remove existing service first
$existing = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
if ($existing) {
    Write-Host "[INFO] Removing existing service..." -ForegroundColor Cyan
    Stop-Service -Name "GymDoorBridge" -Force -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 2
    & sc.exe delete "GymDoorBridge" | Out-Null
    Start-Sleep -Seconds 2
}

# Create service with correct syntax
$serviceName = "GymDoorBridge"
$serviceDisplay = "RepSet Gym Door Bridge"
$serviceBinPath = "`"$exePath`" --config `"$configPath`""

Write-Host "[INFO] Service command: sc.exe create $serviceName binPath= '$serviceBinPath' start= auto DisplayName= '$serviceDisplay'" -ForegroundColor Gray

try {
    $scOutput = & sc.exe create $serviceName binPath= $serviceBinPath start= auto DisplayName= $serviceDisplay 2>&1
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "[OK] Service created successfully" -ForegroundColor Green
        
        # Start the service
        Write-Host "[INFO] Starting service..." -ForegroundColor Cyan
        try {
            Start-Service -Name $serviceName
            Write-Host "[OK] Service started successfully!" -ForegroundColor Green
        } catch {
            Write-Host "[WARNING] Service created but failed to start: $($_.Exception.Message)" -ForegroundColor Yellow
            Write-Host "[INFO] Try starting manually: Start-Service -Name GymDoorBridge" -ForegroundColor Cyan
        }
        
    } else {
        Write-Host "[ERROR] Service creation failed (exit code: $LASTEXITCODE)" -ForegroundColor Red
        Write-Host "[INFO] Output: $scOutput" -ForegroundColor Gray
        
        # Try alternative method
        Write-Host "[INFO] Trying PowerShell New-Service..." -ForegroundColor Cyan
        try {
            New-Service -Name $serviceName -BinaryPathName $serviceBinPath -DisplayName $serviceDisplay -StartupType Automatic
            Write-Host "[OK] Service created with PowerShell" -ForegroundColor Green
            Start-Service -Name $serviceName
            Write-Host "[OK] Service started!" -ForegroundColor Green
        } catch {
            Write-Host "[ERROR] Both service creation methods failed: $($_.Exception.Message)" -ForegroundColor Red
        }
    }
    
} catch {
    Write-Host "[ERROR] Service creation error: $($_.Exception.Message)" -ForegroundColor Red
}

# Step 4: Verify everything
Write-Host "" 
Write-Host "=== FINAL STATUS ===" -ForegroundColor Blue

# Check service
$service = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
if ($service) {
    Write-Host "[OK] Service Status: $($service.Status)" -ForegroundColor Green
    Write-Host "[OK] Start Type: $($service.StartType)" -ForegroundColor Green
    
    if ($service.Status -eq "Running") {
        Write-Host "[OK] RepSet Bridge is running and will auto-start with Windows!" -ForegroundColor Green
    } else {
        Write-Host "[WARNING] Service exists but not running. Try: Start-Service -Name GymDoorBridge" -ForegroundColor Yellow
    }
} else {
    Write-Host "[ERROR] Service not found" -ForegroundColor Red
}

# Test bridge status
try {
    Write-Host "[INFO] Testing bridge status..." -ForegroundColor Cyan
    $status = & $exePath status --config $configPath 2>&1
    Write-Host "[INFO] Bridge status: $status" -ForegroundColor Gray
} catch {
    Write-Host "[WARNING] Could not get bridge status" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "Setup complete! Press Enter to exit..."
Read-Host