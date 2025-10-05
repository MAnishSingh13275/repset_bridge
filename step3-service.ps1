# ================================================================
# RepSet Bridge - Step 3: Windows Service Setup
# Creates and configures the Windows service with robust error handling
# ================================================================

param(
    [switch]$Silent = $false
)

$ProgressPreference = 'SilentlyContinue'
$ErrorActionPreference = 'Stop'

function Write-Step {
    param([string]$Message, [string]$Level = "Info")
    $timestamp = Get-Date -Format "HH:mm:ss"
    $color = switch ($Level) {
        "Error" { "Red" }
        "Success" { "Green" }
        "Warning" { "Yellow" }
        default { "Cyan" }
    }
    if (-not $Silent) {
        Write-Host "[$timestamp] $Message" -ForegroundColor $color
    }
}

try {
    if (-not $Silent) {
        Clear-Host
        Write-Host ""
        Write-Host "🚀 RepSet Bridge - Step 3: Service Setup" -ForegroundColor Cyan
        Write-Host "=======================================" -ForegroundColor Cyan
        Write-Host ""
    }

    # Check admin privileges
    Write-Step "Checking administrator privileges..."
    if (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
        Write-Step "ERROR: Administrator privileges required" "Error"
        Write-Host ""
        Write-Host "Please:" -ForegroundColor Yellow
        Write-Host "1. Right-click PowerShell" -ForegroundColor Gray
        Write-Host "2. Select 'Run as Administrator'" -ForegroundColor Gray
        Write-Host "3. Run this script again" -ForegroundColor Gray
        if (-not $Silent) { Read-Host "Press Enter to exit" }
        exit 1
    }
    Write-Step "✅ Administrator privileges confirmed" "Success"

    # Check if Step 2 was completed
    $TempDir = "$env:TEMP\RepSetBridge"
    $installInfoFile = "$TempDir\install-info.json"
    
    Write-Step "Checking Step 2 completion..."
    if (-not (Test-Path $installInfoFile)) {
        Write-Step "ERROR: Step 2 (Installation) must be completed first" "Error"
        Write-Host ""
        Write-Host "Please run Step 2 first:" -ForegroundColor Yellow
        Write-Host "   .\step2-install.ps1" -ForegroundColor Gray
        if (-not $Silent) { Read-Host "Press Enter to exit" }
        exit 1
    }

    # Load installation info
    $installInfo = Get-Content $installInfoFile | ConvertFrom-Json
    $ExePath = $installInfo.exePath
    $ConfigPath = $installInfo.configPath

    # Verify files exist
    if (-not (Test-Path $ExePath)) {
        Write-Step "ERROR: Bridge executable not found. Please re-run Step 2" "Error"
        exit 1
    }
    if (-not (Test-Path $ConfigPath)) {
        Write-Step "ERROR: Configuration file not found. Please re-run Step 2" "Error"
        exit 1
    }
    Write-Step "✅ Step 2 completion verified" "Success"

    # Remove existing service if present
    Write-Step "Checking for existing service..."
    $existingService = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
    if ($existingService) {
        Write-Step "Removing existing service..." "Warning"
        try {
            Stop-Service -Name "GymDoorBridge" -Force -ErrorAction SilentlyContinue
            Start-Sleep 2
            
            # Use sc.exe to remove service
            $removeResult = sc.exe delete "GymDoorBridge" 2>&1
            if ($LASTEXITCODE -eq 0) {
                Write-Step "✅ Existing service removed successfully" "Success"
            } else {
                Write-Step "⚠️ Service removal had issues but continuing..." "Warning"
            }
            Start-Sleep 2
        } catch {
            Write-Step "⚠️ Issue removing existing service: $($_.Exception.Message)" "Warning"
        }
    } else {
        Write-Step "✅ No existing service found" "Success"
    }

    # Create Windows service using multiple methods
    Write-Step "Creating Windows service..."
    $serviceName = "GymDoorBridge"
    $displayName = "RepSet Gym Door Bridge"
    $description = "RepSet Bridge service for gym door access hardware integration"
    $binaryPath = "`"$ExePath`" --config `"$ConfigPath`""

    $serviceCreated = $false
    $creationMethod = ""

    # Method 1: Use bridge's built-in installer
    try {
        Write-Step "Trying service creation method 1 (built-in installer)..."
        $result = & $ExePath install --config $ConfigPath 2>&1
        if ($LASTEXITCODE -eq 0) {
            $serviceCreated = $true
            $creationMethod = "Built-in installer"
            Write-Step "✅ Service created using built-in installer" "Success"
        } else {
            Write-Step "Method 1 failed, trying alternative..." "Warning"
        }
    } catch {
        Write-Step "Method 1 failed: $($_.Exception.Message)" "Warning"
    }

    # Method 2: Use PowerShell New-Service
    if (-not $serviceCreated) {
        try {
            Write-Step "Trying service creation method 2 (PowerShell)..."
            New-Service -Name $serviceName -BinaryPathName $binaryPath -DisplayName $displayName -Description $description -StartupType Automatic
            $serviceCreated = $true
            $creationMethod = "PowerShell New-Service"
            Write-Step "✅ Service created using PowerShell" "Success"
        } catch {
            Write-Step "Method 2 failed: $($_.Exception.Message)" "Warning"
        }
    }

    # Method 3: Use sc.exe
    if (-not $serviceCreated) {
        try {
            Write-Step "Trying service creation method 3 (sc.exe)..."
            $scResult = sc.exe create $serviceName binpath= $binaryPath start= auto displayname= $displayName 2>&1
            if ($LASTEXITCODE -eq 0) {
                $serviceCreated = $true
                $creationMethod = "sc.exe command"
                Write-Step "✅ Service created using sc.exe" "Success"
                
                # Set description separately
                sc.exe description $serviceName $description | Out-Null
            } else {
                Write-Step "Method 3 failed: $scResult" "Warning"
            }
        } catch {
            Write-Step "Method 3 failed: $($_.Exception.Message)" "Warning"
        }
    }

    if (-not $serviceCreated) {
        Write-Step "ERROR: All service creation methods failed" "Error"
        Write-Host ""
        Write-Host "Troubleshooting steps:" -ForegroundColor Yellow
        Write-Host "1. Ensure no other RepSet Bridge services exist" -ForegroundColor Gray
        Write-Host "2. Check Windows Event Viewer for service errors" -ForegroundColor Gray
        Write-Host "3. Try rebooting and running this script again" -ForegroundColor Gray
        if (-not $Silent) { Read-Host "Press Enter to exit" }
        exit 1
    }

    # Configure service settings
    Write-Step "Configuring service settings..."
    try {
        # Set service to restart on failure
        sc.exe failure $serviceName reset= 86400 actions= restart/5000/restart/10000/restart/30000 | Out-Null
        
        # Set service recovery options
        sc.exe config $serviceName start= auto | Out-Null
        
        Write-Step "✅ Service configuration completed" "Success"
    } catch {
        Write-Step "⚠️ Service configuration had issues (non-critical)" "Warning"
    }

    # Verify service creation
    Write-Step "Verifying service installation..."
    Start-Sleep 2
    $newService = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
    if (-not $newService) {
        Write-Step "ERROR: Service verification failed - service not found" "Error"
        exit 1
    }
    Write-Step "✅ Service verification successful" "Success"

    # Test starting the service (but don't require it to stay running if not paired)
    Write-Step "Testing service startup..."
    try {
        Start-Service -Name $serviceName -ErrorAction Stop
        Start-Sleep 5
        
        $serviceStatus = (Get-Service -Name $serviceName).Status
        if ($serviceStatus -eq "Running") {
            Write-Step "✅ Service started successfully" "Success"
        } else {
            Write-Step "⚠️ Service installed but not running (normal if not paired yet)" "Warning"
        }
        
        # Stop the service for now - it will be started properly after pairing
        Stop-Service -Name $serviceName -Force -ErrorAction SilentlyContinue
        Write-Step "Service stopped (will be started after pairing)" "Info"
        
    } catch {
        Write-Step "⚠️ Service startup test failed (normal if not paired yet)" "Warning"
        Write-Step "Service will start properly after pairing is completed" "Info"
    }

    # Create service info file
    Write-Step "Creating service record..."
    try {
        $serviceInfo = @{
            serviceTime = (Get-Date).ToString()
            serviceName = $serviceName
            displayName = $displayName
            binaryPath = $binaryPath
            creationMethod = $creationMethod
            status = $newService.Status
            installStep = 3
        }
        $serviceInfo | ConvertTo-Json | Set-Content "$TempDir\service-info.json"
        Write-Step "✅ Service record created" "Success"
    } catch {
        Write-Step "⚠️ Could not create service record (non-critical)" "Warning"
    }

    if (-not $Silent) {
        Write-Host ""
        Write-Host "🎉 STEP 3 COMPLETED SUCCESSFULLY!" -ForegroundColor Green
        Write-Host "=================================" -ForegroundColor Green
        Write-Host ""
        Write-Host "📋 Service Summary:" -ForegroundColor Cyan
        Write-Host "   🔧 Service Name: $serviceName" -ForegroundColor Gray
        Write-Host "   📱 Display Name: $displayName" -ForegroundColor Gray
        Write-Host "   📁 Binary Path: $binaryPath" -ForegroundColor Gray
        Write-Host "   ⚙️ Creation Method: $creationMethod" -ForegroundColor Gray
        Write-Host "   📊 Current Status: $($newService.Status)" -ForegroundColor Gray
        Write-Host "   🚀 Startup Type: Automatic" -ForegroundColor Gray
        Write-Host ""
        Write-Host "✅ Ready for Step 4: Device Pairing" -ForegroundColor Green
        Write-Host ""
        Write-Host "Next: Run the pairing script with your pair code:" -ForegroundColor Yellow
        Write-Host "   .\step4-pair.ps1 -PairCode \"YOUR_PAIR_CODE\"" -ForegroundColor Gray
        Write-Host ""
        Write-Host "💡 If you don't have a pair code yet:" -ForegroundColor Cyan
        Write-Host "   1. Log into your RepSet admin dashboard" -ForegroundColor Gray
        Write-Host "   2. Go to Bridge Management section" -ForegroundColor Gray
        Write-Host "   3. Generate a new pair code for this location" -ForegroundColor Gray
        Write-Host ""
        
        Read-Host "Press Enter to continue"
    }

} catch {
    Write-Step "UNEXPECTED ERROR: $($_.Exception.Message)" "Error"
    if (-not $Silent) { Read-Host "Press Enter to exit" }
    exit 1
}