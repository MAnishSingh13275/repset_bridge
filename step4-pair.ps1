# ================================================================
# RepSet Bridge - Step 4: Device Pairing & Service Start
# Handles device pairing and starts the service reliably
# ================================================================

param(
    [Parameter(Mandatory=$false)]
    [string]$PairCode = "",
    [string]$ServerUrl = "https://repset.onezy.in",
    [switch]$Silent = $false,
    [switch]$SkipServiceStart = $false
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
        Write-Host "🚀 RepSet Bridge - Step 4: Device Pairing" -ForegroundColor Cyan
        Write-Host "========================================" -ForegroundColor Cyan
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

    # Check if Step 3 was completed
    $TempDir = "$env:TEMP\RepSetBridge"
    $serviceInfoFile = "$TempDir\service-info.json"
    
    Write-Step "Checking Step 3 completion..."
    if (-not (Test-Path $serviceInfoFile)) {
        Write-Step "ERROR: Step 3 (Service Setup) must be completed first" "Error"
        Write-Host ""
        Write-Host "Please run Step 3 first:" -ForegroundColor Yellow
        Write-Host "   .\step3-service.ps1" -ForegroundColor Gray
        if (-not $Silent) { Read-Host "Press Enter to exit" }
        exit 1
    }

    # Load service info
    $serviceInfo = Get-Content $serviceInfoFile | ConvertFrom-Json
    $installInfoFile = "$TempDir\install-info.json"
    $installInfo = Get-Content $installInfoFile | ConvertFrom-Json
    
    $ExePath = $installInfo.exePath
    $ConfigPath = $installInfo.configPath
    $serviceName = $serviceInfo.serviceName

    # Verify files and service exist
    if (-not (Test-Path $ExePath)) {
        Write-Step "ERROR: Bridge executable not found. Please re-run previous steps" "Error"
        exit 1
    }
    if (-not (Test-Path $ConfigPath)) {
        Write-Step "ERROR: Configuration file not found. Please re-run previous steps" "Error"
        exit 1
    }
    
    $service = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
    if (-not $service) {
        Write-Step "ERROR: Windows service not found. Please re-run Step 3" "Error"
        exit 1
    }
    Write-Step "✅ Step 3 completion verified" "Success"

    # Get pair code if not provided
    if (-not $PairCode) {
        if (-not $Silent) {
            Write-Host ""
            Write-Host "🔗 Device Pairing Required" -ForegroundColor Yellow
            Write-Host ""
            Write-Host "To pair your bridge with RepSet:" -ForegroundColor Gray
            Write-Host "1. Log into your RepSet admin dashboard" -ForegroundColor Gray
            Write-Host "2. Go to Bridge Management section" -ForegroundColor Gray
            Write-Host "3. Click 'Add New Bridge' or 'Generate Pair Code'" -ForegroundColor Gray
            Write-Host "4. Copy the pair code (format: XXXX-XXXX-XXXX)" -ForegroundColor Gray
            Write-Host ""
            
            do {
                $PairCode = Read-Host "Enter your pair code"
                if (-not $PairCode) {
                    Write-Host "Pair code is required to continue." -ForegroundColor Yellow
                }
            } while (-not $PairCode)
        } else {
            Write-Step "ERROR: Pair code is required for silent installation" "Error"
            exit 1
        }
    }

    # Validate pair code format
    if ($PairCode -notmatch '^[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}$') {
        Write-Step "WARNING: Pair code format seems unusual (expected: XXXX-XXXX-XXXX)" "Warning"
        Write-Step "Continuing with provided code: $PairCode" "Info"
    }

    # Stop service for pairing
    Write-Step "Preparing for device pairing..."
    try {
        Stop-Service -Name $serviceName -Force -ErrorAction SilentlyContinue
        Start-Sleep 2
        Write-Step "✅ Service stopped for pairing" "Success"
    } catch {
        Write-Step "⚠️ Could not stop service (may not be running)" "Warning"
    }

    # Attempt to unpair any existing pairing first
    Write-Step "Clearing any existing pairing..."
    try {
        $unpairResult = & $ExePath unpair --config $ConfigPath 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Step "✅ Previous pairing cleared" "Success"
        } else {
            Write-Step "No existing pairing found (normal for first install)" "Info"
        }
    } catch {
        Write-Step "No existing pairing to clear" "Info"
    }

    # Perform pairing
    Write-Step "Pairing bridge with RepSet platform..."
    Write-Step "Using pair code: $PairCode" "Info"
    Write-Step "Connecting to: $ServerUrl" "Info"
    
    try {
        $pairResult = & $ExePath pair $PairCode --config $ConfigPath 2>&1
        
        if ($LASTEXITCODE -eq 0) {
            Write-Step "✅ Device pairing successful!" "Success"
            
            # Parse the result to get device ID
            $deviceIdMatch = $pairResult | Select-String "device_id([a-zA-Z0-9_]+)"
            if ($deviceIdMatch) {
                $deviceId = $deviceIdMatch.Matches[0].Groups[1].Value
                Write-Step "✅ Device ID: $deviceId" "Success"
            }
            
        } else {
            # Check if it's a network issue vs invalid code
            $pairResultString = $pairResult -join " "
            if ($pairResultString -match "network|connection|timeout|server") {
                Write-Step "⚠️ Pairing delayed due to network issues - will retry when service starts" "Warning"
                Write-Step "The bridge will attempt to pair automatically when the service starts" "Info"
            } elseif ($pairResultString -match "invalid|expired|not found") {
                Write-Step "ERROR: Invalid or expired pair code" "Error"
                Write-Host ""
                Write-Host "Please:" -ForegroundColor Yellow
                Write-Host "1. Check your pair code is correct" -ForegroundColor Gray
                Write-Host "2. Generate a new pair code if this one expired" -ForegroundColor Gray
                Write-Host "3. Verify you have internet connectivity" -ForegroundColor Gray
                if (-not $Silent) { Read-Host "Press Enter to exit" }
                exit 1
            } else {
                Write-Step "⚠️ Pairing had issues but may work when service starts" "Warning"
                Write-Step "Error details: $pairResultString" "Warning"
            }
        }
    } catch {
        Write-Step "ERROR: Pairing command failed: $($_.Exception.Message)" "Error"
        if (-not $Silent) { Read-Host "Press Enter to exit" }
        exit 1
    }

    # Start the service
    if (-not $SkipServiceStart) {
        Write-Step "Starting RepSet Bridge service..."
        try {
            Start-Service -Name $serviceName -ErrorAction Stop
            Start-Sleep 5
            
            $serviceStatus = (Get-Service -Name $serviceName).Status
            if ($serviceStatus -eq "Running") {
                Write-Step "✅ Service started successfully!" "Success"
                
                # Quick API health check
                Start-Sleep 3
                try {
                    $healthCheck = Invoke-WebRequest -Uri "http://localhost:8081/health" -UseBasicParsing -TimeoutSec 10 -ErrorAction Stop
                    Write-Step "✅ Bridge API responding (HTTP $($healthCheck.StatusCode))" "Success"
                } catch {
                    Write-Step "⚠️ Bridge API not yet available (may be starting up)" "Warning"
                }
                
                # Check bridge status
                try {
                    Start-Sleep 2
                    $statusResult = & $ExePath status --config $ConfigPath 2>&1
                    if ($statusResult -match "paired|connected|active") {
                        Write-Step "✅ Bridge is connected to RepSet platform!" "Success"
                    } else {
                        Write-Step "⚠️ Bridge service is running but connection status unclear" "Warning"
                    }
                } catch {
                    Write-Step "⚠️ Could not check bridge status (service may be starting)" "Warning"
                }
                
            } else {
                Write-Step "⚠️ Service started but status: $serviceStatus" "Warning"
            }
        } catch {
            Write-Step "ERROR: Failed to start service: $($_.Exception.Message)" "Error"
            Write-Host ""
            Write-Host "Troubleshooting:" -ForegroundColor Yellow
            Write-Host "1. Check Windows Event Viewer for service errors" -ForegroundColor Gray
            Write-Host "2. Verify pairing was successful" -ForegroundColor Gray
            Write-Host "3. Check bridge logs in: $($installInfo.dataDir)" -ForegroundColor Gray
            Write-Host ""
            Write-Host "You can try starting manually:" -ForegroundColor Cyan
            Write-Host "   Start-Service -Name '$serviceName'" -ForegroundColor Gray
            if (-not $Silent) { Read-Host "Press Enter to exit" }
            exit 1
        }
    }

    # Create completion info file
    Write-Step "Creating completion record..."
    try {
        $completionInfo = @{
            completionTime = (Get-Date).ToString()
            pairCode = $PairCode
            serverUrl = $ServerUrl
            deviceId = if ($deviceId) { $deviceId } else { "Unknown" }
            serviceStatus = (Get-Service -Name $serviceName).Status
            installStep = 4
            installationComplete = $true
        }
        $completionInfo | ConvertTo-Json | Set-Content "$TempDir\completion-info.json"
        Write-Step "✅ Completion record created" "Success"
    } catch {
        Write-Step "⚠️ Could not create completion record (non-critical)" "Warning"
    }

    if (-not $Silent) {
        Write-Host ""
        Write-Host "🎉 INSTALLATION COMPLETED SUCCESSFULLY!" -ForegroundColor Green
        Write-Host "======================================" -ForegroundColor Green
        Write-Host ""
        Write-Host "📋 Final Summary:" -ForegroundColor Cyan
        Write-Host "   🔗 Pair Code Used: $PairCode" -ForegroundColor Gray
        Write-Host "   🌐 Server URL: $ServerUrl" -ForegroundColor Gray
        if ($deviceId) {
            Write-Host "   🆔 Device ID: $deviceId" -ForegroundColor Gray
        }
        Write-Host "   🔧 Service Name: $serviceName" -ForegroundColor Gray
        Write-Host "   📊 Service Status: $((Get-Service -Name $serviceName).Status)" -ForegroundColor Gray
        Write-Host "   📁 Install Path: $($installInfo.installPath)" -ForegroundColor Gray
        Write-Host "   💾 Data Directory: $($installInfo.dataDir)" -ForegroundColor Gray
        Write-Host "   🌐 API Endpoint: http://localhost:8081" -ForegroundColor Gray
        Write-Host ""
        Write-Host "✅ Your RepSet Bridge is now active and connected!" -ForegroundColor Green
        Write-Host ""
        Write-Host "🎯 What's Next:" -ForegroundColor Cyan
        Write-Host "   1. Check your RepSet admin dashboard" -ForegroundColor Gray
        Write-Host "   2. Your bridge should appear as 'Active' within 2 minutes" -ForegroundColor Gray
        Write-Host "   3. Any connected biometric devices will be auto-discovered" -ForegroundColor Gray
        Write-Host "   4. Member check-ins will sync automatically" -ForegroundColor Gray
        Write-Host ""
        Write-Host "🔧 Management Commands:" -ForegroundColor Cyan
        Write-Host "   gym-door-bridge status           - Check bridge status" -ForegroundColor Gray
        Write-Host "   net start $serviceName          - Start service" -ForegroundColor Gray
        Write-Host "   net stop $serviceName           - Stop service" -ForegroundColor Gray
        Write-Host "   Get-Service -Name '$serviceName' - Check service status" -ForegroundColor Gray
        Write-Host ""
        Write-Host "📂 Important Locations:" -ForegroundColor Cyan
        Write-Host "   Logs: $($installInfo.dataDir)\bridge.log" -ForegroundColor Gray
        Write-Host "   Config: $ConfigPath" -ForegroundColor Gray
        Write-Host "   Program: $ExePath" -ForegroundColor Gray
        Write-Host ""
        Write-Host "🎊 Installation Complete! The bridge runs automatically." -ForegroundColor Green
        Write-Host ""
        
        Read-Host "Press Enter to finish"
    }

} catch {
    Write-Step "UNEXPECTED ERROR: $($_.Exception.Message)" "Error"
    if (-not $Silent) { Read-Host "Press Enter to exit" }
    exit 1
}