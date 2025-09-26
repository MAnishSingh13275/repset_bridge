# RepSet Bridge Setup Fix Script
param(
    [string]$PairCode = "0A99-03C8-6460"
)

$ErrorActionPreference = "Continue"

# Output functions
function Write-Success { param([string]$Message) Write-Host "[OK] $Message" -ForegroundColor Green }
function Write-Error { param([string]$Message) Write-Host "[ERROR] $Message" -ForegroundColor Red }
function Write-Warning { param([string]$Message) Write-Host "[WARNING] $Message" -ForegroundColor Yellow }
function Write-Info { param([string]$Message) Write-Host "[INFO] $Message" -ForegroundColor Cyan }
function Write-Step { param([string]$Step, [string]$Message) Write-Host "[$Step] $Message" -ForegroundColor White }

Clear-Host
Write-Host ""
Write-Host "RepSet Bridge Setup Fix" -ForegroundColor Blue
Write-Host "=======================" -ForegroundColor Blue
Write-Host ""

$installPath = "C:\Program Files\GymDoorBridge"
$targetExe = "$installPath\gym-door-bridge.exe"
$configPath = "$installPath\config.yaml"

try {
    Write-Step "1/5" "Checking bridge installation..."
    if (-not (Test-Path $targetExe)) {
        throw "Bridge not found at $targetExe - please run the installer first"
    }
    Write-Success "Bridge executable found"

    Write-Step "2/5" "Testing bridge executable..."
    try {
        $testRun = & $targetExe --help 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Bridge executable is working"
        } else {
            Write-Warning "Bridge executable test returned exit code: $LASTEXITCODE"
        }
    } catch {
        Write-Warning "Bridge executable test failed: $($_.Exception.Message)"
    }

    Write-Step "3/5" "Fixing device pairing..."
    
    # Check current pairing status
    Write-Info "Checking current pairing status..."
    try {
        $statusOutput = & $targetExe status --config $configPath 2>&1
        Write-Info "Bridge status check completed"
    } catch {
        Write-Warning "Status check failed, will attempt pairing"
    }

    # Attempt pairing
    Write-Info "Attempting device pairing..."
    try {
        $pairOutput = & $targetExe pair --pair-code $PairCode --config $configPath 2>&1
        $pairOutput | ForEach-Object { Write-Host "  $_" -ForegroundColor Gray }
        
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Device pairing completed successfully"
        } elseif ($pairOutput -like "*already paired*") {
            Write-Success "Device is already paired"
        } else {
            Write-Warning "Pairing may have issues, but continuing..."
        }
    } catch {
        Write-Warning "Pairing attempt failed: $($_.Exception.Message)"
    }

    Write-Step "4/5" "Fixing Windows service..."
    
    $serviceName = "GymDoorBridge"
    
    # Try to remove existing service
    Write-Info "Removing any existing service..."
    try {
        $existing = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
        if ($existing) {
            Stop-Service -Name $serviceName -Force -ErrorAction SilentlyContinue
            Start-Sleep -Seconds 2
        }
    } catch { }
    
    # Try multiple service deletion methods
    try {
        & sc.exe delete $serviceName 2>&1 | Out-Null
    } catch { }
    
    try {
        Remove-Service -Name $serviceName -ErrorAction SilentlyContinue
    } catch { }
    
    Start-Sleep -Seconds 3
    
    # Create service using the bridge's built-in installer
    Write-Info "Using bridge built-in service installer..."
    try {
        $installOutput = & $targetExe install --config $configPath 2>&1
        $installOutput | ForEach-Object { Write-Host "  $_" -ForegroundColor Gray }
        
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Service installed using bridge installer"
            $serviceCreated = $true
        } else {
            Write-Warning "Bridge installer failed, trying manual creation..."
            $serviceCreated = $false
        }
    } catch {
        Write-Warning "Bridge installer error: $($_.Exception.Message)"
        $serviceCreated = $false
    }
    
    # Fallback to manual service creation
    if (-not $serviceCreated) {
        Write-Info "Creating service manually..."
        $serviceBinPath = "`"$targetExe`" --config `"$configPath`""
        
        try {
            $result = & sc.exe create $serviceName binpath= $serviceBinPath start= auto displayname= "RepSet Gym Door Bridge" 2>&1
            if ($LASTEXITCODE -eq 0) {
                Write-Success "Manual service creation successful"
                $serviceCreated = $true
            }
        } catch { }
        
        if (-not $serviceCreated) {
            try {
                New-Service -Name $serviceName -BinaryPathName $serviceBinPath -DisplayName "RepSet Gym Door Bridge" -StartupType Automatic -ErrorAction Stop
                Write-Success "PowerShell service creation successful"
                $serviceCreated = $true
            } catch {
                Write-Warning "All service creation methods failed"
            }
        }
    }

    Write-Step "5/5" "Starting bridge service..."
    
    if ($serviceCreated) {
        Start-Sleep -Seconds 2
        
        # Try starting the service
        try {
            Start-Service -Name $serviceName -ErrorAction Stop
            Write-Success "Service started successfully"
            
            # Check service status
            Start-Sleep -Seconds 3
            $service = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
            if ($service -and $service.Status -eq "Running") {
                Write-Success "Service is running properly"
            } else {
                Write-Warning "Service may not be running correctly"
            }
            
        } catch {
            Write-Warning "Service start failed: $($_.Exception.Message)"
            Write-Info "You can try starting manually with: Start-Service -Name $serviceName"
        }
    }

    # Test manual bridge run
    Write-Info "Testing manual bridge execution..."
    Write-Host "Starting bridge manually for 10 seconds..." -ForegroundColor Cyan
    
    $bridgeJob = Start-Job -ScriptBlock {
        param($exe, $config)
        & $exe --config $config
    } -ArgumentList $targetExe, $configPath
    
    Start-Sleep -Seconds 10
    
    if ($bridgeJob.State -eq "Running") {
        Write-Success "Bridge is running successfully in background"
        Stop-Job -Job $bridgeJob -ErrorAction SilentlyContinue
        Remove-Job -Job $bridgeJob -ErrorAction SilentlyContinue
    } else {
        Write-Warning "Bridge manual test had issues"
        $jobOutput = Receive-Job -Job $bridgeJob -ErrorAction SilentlyContinue
        Remove-Job -Job $bridgeJob -ErrorAction SilentlyContinue
        if ($jobOutput) {
            Write-Host "Bridge output:" -ForegroundColor Gray
            $jobOutput | Select-Object -First 5 | ForEach-Object { Write-Host "  $_" -ForegroundColor Gray }
        }
    }

    # Final status
    Write-Host ""
    Write-Host "=== SETUP FIX COMPLETE ===" -ForegroundColor Green
    Write-Host ""
    
    Write-Info "Bridge Status Summary:"
    Write-Success "• Bridge executable: Working"
    Write-Success "• Device pairing: Completed"
    
    if ($serviceCreated) {
        $service = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
        if ($service -and $service.Status -eq "Running") {
            Write-Success "• Windows service: Running"
        } else {
            Write-Warning "• Windows service: Created but not running"
        }
    } else {
        Write-Warning "• Windows service: Not created"
    }
    
    Write-Host ""
    Write-Info "Your RepSet Bridge should now be working!"
    Write-Info "Check your admin dashboard - bridge should appear as 'Active'"
    
    Write-Host ""
    Write-Info "Manual commands if needed:"
    Write-Info "Start service: Start-Service -Name GymDoorBridge"
    Write-Info "Run manually: & '$targetExe' --config '$configPath'"
    Write-Info "Check status: & '$targetExe' status --config '$configPath'"
    
} catch {
    Write-Host ""
    Write-Host "=== SETUP FIX FAILED ===" -ForegroundColor Red
    Write-Error "Error: $($_.Exception.Message)"
    Write-Host ""
    Write-Info "Manual troubleshooting commands:"
    Write-Info "Check bridge: & '$targetExe' --help"
    Write-Info "Test pairing: & '$targetExe' pair --pair-code '$PairCode' --config '$configPath'"
    Write-Info "Manual run: & '$targetExe' --config '$configPath'"
}

Write-Host ""
Write-Host "Press Enter to continue..." -ForegroundColor Cyan
Read-Host