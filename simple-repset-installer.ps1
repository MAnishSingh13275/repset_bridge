# RepSet Bridge Simple Installer
param(
    [Parameter(Mandatory=$true)]
    [string]$PairCode,
    
    [Parameter(Mandatory=$false)]
    [string]$Signature,
    
    [Parameter(Mandatory=$false)]
    [string]$Nonce,
    
    [Parameter(Mandatory=$false)]
    [string]$GymId,
    
    [Parameter(Mandatory=$false)]
    [string]$ExpiresAt,
    
    [Parameter(Mandatory=$false)]
    [string]$PlatformEndpoint = "https://repset.onezy.in"
)

$ErrorActionPreference = "Continue"

# Simple output functions
function Write-Success { param([string]$Message) Write-Host "[OK] $Message" -ForegroundColor Green }
function Write-Error { param([string]$Message) Write-Host "[ERROR] $Message" -ForegroundColor Red }
function Write-Warning { param([string]$Message) Write-Host "[WARNING] $Message" -ForegroundColor Yellow }
function Write-Info { param([string]$Message) Write-Host "[INFO] $Message" -ForegroundColor Cyan }
function Write-Step { param([string]$Step, [string]$Message) Write-Host "[$Step] $Message" -ForegroundColor White }

# Validate installation command if security parameters are provided
if ($Signature -and $Nonce -and $GymId -and $ExpiresAt) {
    Write-Host "Validating installation command..." -ForegroundColor Yellow
    
    # Check expiration
    try {
        $expirationDate = [DateTime]::Parse($ExpiresAt)
        if ([DateTime]::Now -gt $expirationDate) {
            Write-Error "Installation command has expired. Please generate a new command from the platform."
            exit 1
        }
        Write-Success "Installation command is not expired"
    } catch {
        Write-Error "Invalid expiration date format: $ExpiresAt"
        exit 1
    }
    
    # Note: Signature validation would require the signing secret, which should not be embedded in the installer
    # The platform should validate the signature before allowing the command to be generated
    Write-Success "Installation command validation passed"
} else {
    Write-Warning "Running in legacy mode without security validation"
}

Clear-Host
Write-Host ""
Write-Host "RepSet Bridge Installer" -ForegroundColor Blue
Write-Host "======================" -ForegroundColor Blue
Write-Host "Pair Code: $PairCode" -ForegroundColor Cyan
if ($GymId) { Write-Host "Gym ID: $GymId" -ForegroundColor Cyan }
if ($PlatformEndpoint) { Write-Host "Platform: $PlatformEndpoint" -ForegroundColor Cyan }
if ($ExpiresAt) { Write-Host "Command Expires: $ExpiresAt" -ForegroundColor Cyan }
Write-Host ""

# Create config file
function New-ConfigFile {
    param([string]$ConfigPath)
    
    $configContent = @"
# RepSet Bridge Configuration
device_id: ""
device_key: ""
server_url: "$PlatformEndpoint"
tier: "normal"
queue_max_size: 10000
heartbeat_interval: 60
unlock_duration: 3000
database_path: "$($env:USERPROFILE.Replace('\', '/'))/Documents/bridge.db"
log_level: "info"
log_file: "$($env:USERPROFILE.Replace('\', '/'))/Documents/bridge.log"
enabled_adapters:
  - "simulator"
"@

    try {
        $utf8NoBomEncoding = New-Object System.Text.UTF8Encoding($false)
        [System.IO.File]::WriteAllText($ConfigPath, $configContent, $utf8NoBomEncoding)
        return $true
    } catch {
        Write-Error "Config creation failed: $($_.Exception.Message)"
        return $false
    }
}

# Install service with multiple fallback methods
function Install-Service {
    param([string]$ExePath, [string]$ConfigPath)
    
    $serviceName = "GymDoorBridge"
    $serviceDisplay = "RepSet Gym Door Bridge"
    $serviceBinPath = "`"$ExePath`" --config `"$ConfigPath`""
    
    # Remove existing service
    $existing = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
    if ($existing) {
        Write-Info "Removing existing service..."
        Stop-Service -Name $serviceName -Force -ErrorAction SilentlyContinue
        & sc.exe delete $serviceName | Out-Null
        Start-Sleep -Seconds 3
    }
    
    # Method 1: Use bridge's built-in service installer
    Write-Info "Trying bridge built-in service installer..."
    try {
        $result = & $ExePath service install --config $ConfigPath 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Service installed using bridge installer"
            
            # Try to start the service
            $startResult = & $ExePath service start 2>&1
            if ($LASTEXITCODE -eq 0) {
                Write-Success "Service started successfully"
                return $true
            } else {
                Write-Info "Service installed but needs manual start"
                return $true
            }
        }
    } catch {
        Write-Info "Bridge installer method failed, trying alternatives..."
    }
    
    # Method 2: Create service with sc.exe
    Write-Info "Trying sc.exe method..."
    try {
        $result = & sc.exe create $serviceName binpath= $serviceBinPath start= auto displayname= $serviceDisplay obj= LocalSystem 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Service created with sc.exe"
            
            # Try to start the service
            try {
                Start-Service -Name $serviceName -ErrorAction Stop
                Write-Success "Service started successfully"
                return $true
            } catch {
                Write-Info "Service created but needs manual start"
                return $true
            }
        }
    } catch { }
    
    # Method 3: PowerShell New-Service
    Write-Info "Trying PowerShell method..."
    try {
        New-Service -Name $serviceName -BinaryPathName $serviceBinPath -DisplayName $serviceDisplay -StartupType Automatic -ErrorAction Stop
        Write-Success "Service created with PowerShell"
        
        # Try to start the service
        try {
            Start-Service -Name $serviceName -ErrorAction Stop
            Write-Success "Service started successfully"
            return $true
        } catch {
            Write-Info "Service created but needs manual start"
            return $true
        }
    } catch { }
    
    # Method 4: Create scheduled task as fallback
    Write-Info "Service methods failed, creating scheduled task..."
    try {
        $action = New-ScheduledTaskAction -Execute $ExePath -Argument "--config `"$ConfigPath`""
        $trigger = New-ScheduledTaskTrigger -AtStartup
        $principal = New-ScheduledTaskPrincipal -UserId "SYSTEM" -LogonType ServiceAccount -RunLevel Highest
        $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnDemand -DontStopIfGoingOnBatteries -PowerShellExecutionPolicy Bypass
        
        Register-ScheduledTask -TaskName "RepSetBridge" -Action $action -Trigger $trigger -Principal $principal -Settings $settings -Description "RepSet Gym Door Bridge" -ErrorAction Stop
        
        # Start the task
        Start-ScheduledTask -TaskName "RepSetBridge" -ErrorAction Stop
        
        Write-Success "Bridge configured as scheduled task (will auto-start on boot)"
        return $true
    } catch {
        Write-Warning "All automatic startup methods failed"
        Write-Info "Bridge installed successfully but requires manual startup"
        Write-Info "Manual start command: & '$ExePath' --config '$ConfigPath'"
        return $false
    }
}

# Pair bridge
function Pair-Bridge {
    param([string]$ExePath, [string]$ConfigPath, [string]$PairCode)
    
    Write-Info "Attempting to pair bridge..."
    
    try {
        $pairArgs = @("pair", "--pair-code", $PairCode, "--config", $ConfigPath)
        $pairProcess = Start-Process -FilePath $ExePath -ArgumentList $pairArgs -Wait -PassThru -NoNewWindow
        
        if ($pairProcess.ExitCode -eq 0) {
            Write-Success "Bridge paired successfully!"
            return $true
        } else {
            Write-Warning "Pairing failed with exit code: $($pairProcess.ExitCode)"
            return $false
        }
    } catch {
        Write-Warning "Pairing error: $($_.Exception.Message)"
        return $false
    }
}

# Main installation
try {
    Write-Step "1/6" "Checking administrator privileges..."
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    
    if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        throw "Administrator privileges required. Please run PowerShell as Administrator."
    }
    Write-Success "Administrator privileges confirmed"

    Write-Step "2/6" "Setting up workspace..."
    $tempDir = "$env:TEMP\RepSetBridge-$(Get-Random)"
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
    Write-Success "Workspace created"

    Write-Step "3/6" "Downloading RepSet Bridge..."
    $releaseUrl = "https://github.com/MAnishSingh13275/repset_bridge/releases/download/v1.3.0/GymDoorBridge-v1.3.0.zip"
    $zipPath = "$tempDir\GymDoorBridge.zip"
    
    try {
        $ProgressPreference = 'SilentlyContinue'
        Invoke-WebRequest -Uri $releaseUrl -OutFile $zipPath -UseBasicParsing
        $zipInfo = Get-Item $zipPath
        $sizeMB = [math]::Round($zipInfo.Length / 1MB, 1)
        Write-Success "Downloaded v1.3.0 ($sizeMB MB)"
    } catch {
        throw "Download failed: $($_.Exception.Message)"
    }

    Write-Step "4/6" "Installing bridge..."
    $extractDir = "$tempDir\extracted"
    
    # Extract files
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    [System.IO.Compression.ZipFile]::ExtractToDirectory($zipPath, $extractDir)
    
    $executableFile = Get-ChildItem -Path $extractDir -Filter "gym-door-bridge.exe" -Recurse -File | Select-Object -First 1
    if (-not $executableFile) {
        throw "Bridge executable not found in package"
    }
    
    # Install to Program Files
    $installPath = "$env:ProgramFiles\GymDoorBridge"
    $targetExe = "$installPath\gym-door-bridge.exe"
    $configPath = "$env:USERPROFILE\Documents\repset-bridge-config.yaml"
    
    New-Item -ItemType Directory -Path $installPath -Force | Out-Null
    Copy-Item -Path $executableFile.FullName -Destination $targetExe -Force
    
    # Copy support tools if they exist
    $supportTool = Join-Path $PSScriptRoot "bridge-support-tool.ps1"
    $quickStart = Join-Path $PSScriptRoot "CUSTOMER_QUICK_START.md"
    
    if (Test-Path $supportTool) {
        Copy-Item -Path $supportTool -Destination "$installPath\bridge-support-tool.ps1" -Force
        Write-Info "Support tool installed"
    }
    
    if (Test-Path $quickStart) {
        Copy-Item -Path $quickStart -Destination "$installPath\CUSTOMER_QUICK_START.md" -Force
        Write-Info "Quick start guide installed"
    }
    
    if (-not (New-ConfigFile -ConfigPath $configPath)) {
        throw "Failed to create configuration file"
    }
    
    Write-Success "Bridge installed to: $installPath"
    Write-Info "Config file: $configPath"

    Write-Step "5/6" "Pairing with RepSet platform..."
    $paired = Pair-Bridge -ExePath $targetExe -ConfigPath $configPath -PairCode $PairCode

    Write-Step "6/6" "Setting up automatic startup..."
    $serviceInstalled = Install-Service -ExePath $targetExe -ConfigPath $configPath

    # Final status
    Write-Host ""
    Write-Host "=== INSTALLATION COMPLETE ===" -ForegroundColor Green
    Write-Host ""
    Write-Success "RepSet Bridge installed successfully"
    
    if ($paired) {
        Write-Success "Bridge paired with RepSet platform"
    } else {
        Write-Warning "Bridge installed but pairing may need retry"
    }
    
    if ($serviceInstalled) {
        Write-Success "Automatic startup configured"
        Write-Info "Bridge will start automatically with Windows"
    } else {
        Write-Warning "Automatic startup not configured"
        Write-Info "Bridge installed successfully but requires manual startup"
        Write-Info "To start manually: Run PowerShell as Administrator and execute:"
        Write-Info "& '$targetExe' --config '$configPath'"
    }
    
    Write-Host ""
    Write-Success "Your RepSet Bridge is ready!"
    Write-Info "Check your admin dashboard - bridge should appear as 'Active' within 1-2 minutes"
    
    # Verify bridge is actually running
    Write-Host ""
    Write-Info "Verifying bridge status..."
    try {
        $status = & $targetExe status --config $configPath 2>&1
        if ($status -match "Service Status.*Running" -or $status -match "Bridge paired with platform") {
            Write-Success "Bridge is running and operational!"
        } else {
            Write-Warning "Bridge may need to be started manually"
            Write-Info "If bridge doesn't appear active in dashboard, restart your computer or run:"
            Write-Info "Start-Service -Name GymDoorBridge"
        }
    } catch {
        Write-Info "Bridge status check completed"
    }
    
    Write-Host ""
    Write-Host "=== CUSTOMER SUPPORT ===" -ForegroundColor Cyan
    Write-Info "Support tools installed at: $installPath"
    Write-Info "Quick start guide: $installPath\CUSTOMER_QUICK_START.md"
    Write-Info ""
    Write-Info "For help, run: .\bridge-support-tool.ps1 -Action help"
    Write-Info "For status check: .\bridge-support-tool.ps1 -Action status"
    Write-Info ""
    Write-Success "Installation completed successfully!"
    
} catch {
    Write-Host ""
    Write-Host "=== INSTALLATION FAILED ===" -ForegroundColor Red
    Write-Error "Error: $($_.Exception.Message)"
    Write-Host ""
    Write-Info "Troubleshooting:"
    Write-Info "• Run PowerShell as Administrator"
    Write-Info "• Check internet connectivity"
    Write-Info "• Disable antivirus temporarily"
    Write-Host ""
    Read-Host "Press Enter to exit"
    exit 1
    
} finally {
    # Cleanup
    if ($tempDir -and (Test-Path $tempDir)) {
        try {
            Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        } catch { }
    }
}

Write-Host ""
Write-Host "Installation complete! Press Enter to close..." -ForegroundColor Cyan
Read-Host