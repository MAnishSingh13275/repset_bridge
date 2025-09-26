# ================================================================
# RepSet Bridge - Windows Service Setup Script
# Run this AFTER the bridge is installed to enable auto-start
# ================================================================

$ErrorActionPreference = "Continue"

# Color functions
function Write-Success { param([string]$Message) Write-Host "✅ $Message" -ForegroundColor Green }
function Write-Error { param([string]$Message) Write-Host "❌ $Message" -ForegroundColor Red }
function Write-Warning { param([string]$Message) Write-Host "⚠️  $Message" -ForegroundColor Yellow }
function Write-Info { param([string]$Message) Write-Host "ℹ️  $Message" -ForegroundColor Cyan }

Write-Host "=================================================" -ForegroundColor Blue
Write-Host "    RepSet Bridge Service Setup" -ForegroundColor Blue
Write-Host "=================================================" -ForegroundColor Blue
Write-Host ""

try {
    # Check for administrator privileges
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    
    if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        Write-Error "This script requires administrator privileges"
        Write-Info "Please run PowerShell as Administrator and try again"
        exit 1
    }
    Write-Success "Running with administrator privileges"

    # Look for bridge installation
    $possiblePaths = @(
        "$env:ProgramFiles\GymDoorBridge",
        "$env:USERPROFILE\RepSetBridge",
        "$env:ProgramFiles\RepSetBridge"
    )
    
    $installPath = $null
    $targetExe = $null
    $configPath = $null
    
    foreach ($path in $possiblePaths) {
        $exe = "$path\gym-door-bridge.exe"
        $config = "$path\config.yaml"
        
        if ((Test-Path $exe) -and (Test-Path $config)) {
            $installPath = $path
            $targetExe = $exe
            $configPath = $config
            break
        }
    }
    
    if (-not $installPath) {
        Write-Error "Bridge installation not found in common locations"
        Write-Info "Expected locations:"
        foreach ($path in $possiblePaths) {
            Write-Info "  - $path\gym-door-bridge.exe"
        }
        Write-Info "Please run the installer first"
        exit 1
    }
    
    Write-Success "Found bridge installation: $installPath"
    Write-Info "Executable: $targetExe"
    Write-Info "Config: $configPath"

    # Remove existing service if it exists
    $existingService = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
    if ($existingService) {
        Write-Info "Removing existing service..."
        try {
            Stop-Service -Name "GymDoorBridge" -Force -ErrorAction SilentlyContinue
            & sc.exe delete "GymDoorBridge" | Out-Null
            Start-Sleep -Seconds 2
            Write-Success "Existing service removed"
        } catch {
            Write-Warning "Could not remove existing service cleanly"
        }
    }

    # Method 1: Try using sc.exe (most reliable)
    Write-Info "Creating Windows service using sc.exe..."
    $serviceName = "GymDoorBridge"
    $serviceDisplayName = "RepSet Gym Door Bridge"
    $servicePath = "`"$targetExe`" --config `"$configPath`""
    
    try {
        $scResult = & sc.exe create $serviceName binPath= $servicePath start= auto DisplayName= $serviceDisplayName 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Service created successfully using sc.exe"
            
            # Try to start the service
            Write-Info "Starting service..."
            try {
                Start-Service -Name $serviceName -ErrorAction Stop
                Write-Success "Service started successfully!"
                
                # Verify service is running
                $service = Get-Service -Name $serviceName
                Write-Success "Service Status: $($service.Status)"
                
                # Test bridge status
                Write-Info "Testing bridge status..."
                try {
                    $statusResult = & $targetExe status --config $configPath 2>&1
                    Write-Info "Bridge status: $statusResult"
                } catch {
                    Write-Warning "Could not get bridge status, but service is running"
                }
                
            } catch {
                Write-Warning "Service created but failed to start: $($_.Exception.Message)"
                Write-Info "You can try starting it manually from Services.msc"
            }
        } else {
            throw "sc.exe failed with exit code $LASTEXITCODE: $scResult"
        }
    } catch {
        Write-Warning "sc.exe method failed: $($_.Exception.Message)"
        
        # Method 2: Try PowerShell New-Service as fallback
        Write-Info "Trying PowerShell New-Service method..."
        try {
            New-Service -Name $serviceName -BinaryPathName $servicePath -DisplayName $serviceDisplayName -StartupType Automatic -Description "RepSet Gym Door Access Bridge Service" -ErrorAction Stop
            Write-Success "Service created using PowerShell New-Service"
            
            try {
                Start-Service -Name $serviceName -ErrorAction Stop
                Write-Success "Service started successfully!"
            } catch {
                Write-Warning "Service created but failed to start: $($_.Exception.Message)"
            }
        } catch {
            Write-Error "Both service creation methods failed"
            Write-Info "Manual service creation commands:"
            Write-Host "  sc.exe create GymDoorBridge binPath= '$servicePath' start= auto DisplayName= '$serviceDisplayName'" -ForegroundColor Yellow
            Write-Host "  Start-Service -Name GymDoorBridge" -ForegroundColor Yellow
            exit 1
        }
    }

    # Final status check
    Write-Host ""
    Write-Host "=================================================" -ForegroundColor Green
    Write-Host "    ✅ SERVICE SETUP COMPLETE!" -ForegroundColor Green
    Write-Host "=================================================" -ForegroundColor Green
    Write-Host ""
    
    $finalService = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
    if ($finalService) {
        Write-Success "Service Name: $($finalService.Name)"
        Write-Success "Display Name: $($finalService.DisplayName)"
        Write-Success "Status: $($finalService.Status)"
        Write-Success "Start Type: $($finalService.StartType)"
        Write-Host ""
        Write-Info "The RepSet Bridge will now start automatically with Windows!"
        Write-Info "You can manage it through Services.msc or PowerShell commands:"
        Write-Host "  Get-Service -Name GymDoorBridge" -ForegroundColor Yellow
        Write-Host "  Start-Service -Name GymDoorBridge" -ForegroundColor Yellow
        Write-Host "  Stop-Service -Name GymDoorBridge" -ForegroundColor Yellow
    } else {
        Write-Warning "Service creation may have failed"
    }

} catch {
    Write-Host ""
    Write-Host "=================================================" -ForegroundColor Red
    Write-Host "    ❌ SERVICE SETUP FAILED" -ForegroundColor Red
    Write-Host "=================================================" -ForegroundColor Red
    Write-Error "Error: $($_.Exception.Message)"
    Write-Host ""
}

Write-Host ""
Read-Host "Press Enter to continue"