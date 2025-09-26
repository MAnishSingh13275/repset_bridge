# RepSet Bridge Windows Service Setup
# Run this script as Administrator to create and start the Windows service

param(
    [string]$ServiceName = "GymDoorBridge",
    [string]$DisplayName = "RepSet Gym Door Bridge",
    [string]$ExePath = "$env:ProgramFiles\GymDoorBridge\gym-door-bridge.exe",
    [string]$ConfigPath = "$env:USERPROFILE\Documents\repset-bridge-config.yaml"
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
Write-Host "RepSet Bridge Service Setup" -ForegroundColor Blue
Write-Host "===========================" -ForegroundColor Blue
Write-Host ""

try {
    Write-Step "1/6" "Checking administrator privileges..."
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    
    if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        throw "Administrator privileges required. Please run PowerShell as Administrator."
    }
    Write-Success "Administrator privileges confirmed"

    Write-Step "2/6" "Checking bridge installation..."
    if (-not (Test-Path $ExePath)) {
        throw "Bridge executable not found at: $ExePath"
    }
    Write-Success "Bridge executable found"

    Write-Step "3/6" "Checking bridge configuration..."
    if (-not (Test-Path $ConfigPath)) {
        throw "Bridge configuration not found at: $ConfigPath"
    }
    
    # Verify bridge is paired
    try {
        $statusOutput = & $ExePath status --config $ConfigPath 2>&1
        if ($statusOutput -like "*Bridge paired with platform*") {
            Write-Success "Bridge is properly paired"
        } else {
            Write-Warning "Bridge may not be paired - service will still be created"
        }
    } catch {
        Write-Warning "Could not verify bridge pairing status"
    }

    Write-Step "4/6" "Removing existing service if present..."
    try {
        $existing = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
        if ($existing) {
            Write-Info "Stopping existing service..."
            Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
            Start-Sleep -Seconds 3
            
            Write-Info "Removing existing service..."
            & sc.exe delete $ServiceName | Out-Null
            Start-Sleep -Seconds 2
            Write-Success "Existing service removed"
        } else {
            Write-Info "No existing service found"
        }
    } catch {
        Write-Warning "Error during service cleanup: $($_.Exception.Message)"
    }

    Write-Step "5/6" "Creating Windows service..."
    $serviceBinPath = "`"$ExePath`" --config `"$ConfigPath`""
    
    # Try using the bridge's built-in installer first
    Write-Info "Attempting bridge built-in installer..."
    try {
        $installResult = & $ExePath install --config $ConfigPath 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Service created using bridge installer"
            $serviceCreated = $true
        } else {
            Write-Info "Bridge installer failed, trying manual creation..."
            $serviceCreated = $false
        }
    } catch {
        Write-Info "Bridge installer not available, using manual creation..."
        $serviceCreated = $false
    }
    
    # Fallback to manual service creation
    if (-not $serviceCreated) {
        try {
            Write-Info "Creating service with sc.exe..."
            $result = & sc.exe create $ServiceName binpath= $serviceBinPath start= auto displayname= $DisplayName 2>&1
            
            if ($LASTEXITCODE -eq 0) {
                Write-Success "Service created successfully with sc.exe"
                $serviceCreated = $true
            } else {
                Write-Warning "sc.exe failed: $result"
            }
        } catch {
            Write-Warning "sc.exe creation failed: $($_.Exception.Message)"
        }
        
        # Final fallback to PowerShell
        if (-not $serviceCreated) {
            try {
                Write-Info "Creating service with PowerShell..."
                New-Service -Name $ServiceName -BinaryPathName $serviceBinPath -DisplayName $DisplayName -StartupType Automatic -ErrorAction Stop
                Write-Success "Service created successfully with PowerShell"
                $serviceCreated = $true
            } catch {
                Write-Error "All service creation methods failed: $($_.Exception.Message)"
                throw "Unable to create Windows service"
            }
        }
    }

    Write-Step "6/6" "Starting and configuring service..."
    
    if ($serviceCreated) {
        # Set service to restart on failure
        try {
            & sc.exe failure $ServiceName reset= 86400 actions= restart/5000/restart/10000/restart/30000 | Out-Null
            Write-Info "Configured service recovery options"
        } catch {
            Write-Warning "Could not configure service recovery options"
        }
        
        # Start the service
        Start-Sleep -Seconds 2
        try {
            Write-Info "Starting service..."
            Start-Service -Name $ServiceName -ErrorAction Stop
            Write-Success "Service started successfully"
            
            # Verify service is running
            Start-Sleep -Seconds 5
            $service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
            if ($service -and $service.Status -eq "Running") {
                Write-Success "Service is running properly"
                
                # Test API endpoint if available
                try {
                    Start-Sleep -Seconds 5
                    $response = Invoke-WebRequest -Uri "http://localhost:8081/health" -UseBasicParsing -TimeoutSec 10 -ErrorAction SilentlyContinue
                    if ($response.StatusCode -eq 200) {
                        Write-Success "Bridge API is responding on port 8081"
                    }
                } catch {
                    Write-Info "Bridge API not yet available (normal during startup)"
                }
                
            } else {
                Write-Warning "Service may not be running correctly"
            }
            
        } catch {
            Write-Error "Failed to start service: $($_.Exception.Message)"
            Write-Info "Check Windows Event Log for service startup errors"
            Write-Info "You can try starting manually: Start-Service -Name $ServiceName"
        }
    }

    # Final status report
    Write-Host ""
    Write-Host "=== SERVICE SETUP COMPLETE ===" -ForegroundColor Green
    Write-Host ""
    
    $service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($service) {
        Write-Success "Service '$DisplayName' installed successfully"
        Write-Info "Service Name: $ServiceName"
        Write-Info "Service Status: $($service.Status)"
        Write-Info "Startup Type: $($service.StartType)"
        Write-Info "Executable: $ExePath"
        Write-Info "Config File: $ConfigPath"
        
        if ($service.Status -eq "Running") {
            Write-Host ""
            Write-Success "Your RepSet Bridge is now running as a Windows service!"
            Write-Success "It will start automatically when Windows boots"
            Write-Info "Check your RepSet admin dashboard - bridge should appear as 'Active'"
        } else {
            Write-Host ""
            Write-Warning "Service is installed but not running"
            Write-Info "Try: Start-Service -Name $ServiceName"
        }
    } else {
        Write-Error "Service installation may have failed"
    }

    Write-Host ""
    Write-Info "Useful commands:"
    Write-Info "Start service: Start-Service -Name $ServiceName"
    Write-Info "Stop service: Stop-Service -Name $ServiceName"
    Write-Info "Service status: Get-Service -Name $ServiceName"
    Write-Info "Bridge status: & '$ExePath' status --config '$ConfigPath'"
    Write-Info "View logs: Get-Content '$($ConfigPath.Replace('config.yaml', 'bridge.log'))'"
    
} catch {
    Write-Host ""
    Write-Host "=== SERVICE SETUP FAILED ===" -ForegroundColor Red
    Write-Error "Error: $($_.Exception.Message)"
    Write-Host ""
    Write-Info "Troubleshooting:"
    Write-Info "1. Make sure you're running PowerShell as Administrator"
    Write-Info "2. Verify bridge is installed: Test-Path '$ExePath'"
    Write-Info "3. Verify config exists: Test-Path '$ConfigPath'"
    Write-Info "4. Check if bridge is paired: & '$ExePath' status --config '$ConfigPath'"
    Write-Info "5. Try manual service creation:"
    Write-Info "   sc.exe create $ServiceName binpath= \"`\"$ExePath`\" --config \"`\"$ConfigPath`\"\"\" start= auto"
    Write-Host ""
}

Write-Host ""
Write-Host "Press Enter to exit..." -ForegroundColor Cyan
Read-Host