# ================================================================
# RepSet Bridge - Step 2: Install Bridge Files
# Sets up directories, files, and configuration
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
        Write-Host "🚀 RepSet Bridge - Step 2: Installation" -ForegroundColor Cyan
        Write-Host "======================================" -ForegroundColor Cyan
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

    # Check if Step 1 was completed
    $TempDir = "$env:TEMP\RepSetBridge"
    $infoFile = "$TempDir\download-info.json"
    
    Write-Step "Checking Step 1 completion..."
    if (-not (Test-Path $infoFile)) {
        Write-Step "ERROR: Step 1 (Download) must be completed first" "Error"
        Write-Host ""
        Write-Host "Please run Step 1 first:" -ForegroundColor Yellow
        Write-Host "   .\step1-download.ps1" -ForegroundColor Gray
        if (-not $Silent) { Read-Host "Press Enter to exit" }
        exit 1
    }

    # Load download info
    $downloadInfo = Get-Content $infoFile | ConvertFrom-Json
    if (-not (Test-Path $downloadInfo.bridgeExe)) {
        Write-Step "ERROR: Bridge executable not found. Please re-run Step 1" "Error"
        exit 1
    }
    Write-Step "✅ Step 1 completion verified" "Success"

    # Define installation paths
    $InstallPath = "$env:ProgramFiles\GymDoorBridge"
    $DataDir = "$env:ProgramData\GymDoorBridge"
    $ExePath = "$InstallPath\gym-door-bridge.exe"
    $ConfigPath = "$InstallPath\config.yaml"

    # Stop any existing service
    Write-Step "Checking for existing installation..."
    $existingService = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
    if ($existingService) {
        Write-Step "Found existing installation - stopping service..." "Warning"
        try {
            Stop-Service -Name "GymDoorBridge" -Force -ErrorAction SilentlyContinue
            Get-Process -Name "gym-door-bridge" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
            Start-Sleep 3
            Write-Step "✅ Existing service stopped" "Success"
        } catch {
            Write-Step "⚠️ Could not stop existing service cleanly" "Warning"
        }
    }

    # Create directories with proper permissions
    Write-Step "Creating installation directories..."
    try {
        if (Test-Path $InstallPath) {
            Remove-Item $InstallPath -Recurse -Force -ErrorAction SilentlyContinue
        }
        New-Item -ItemType Directory -Path $InstallPath -Force | Out-Null
        New-Item -ItemType Directory -Path $DataDir -Force | Out-Null
        
        # Set directory permissions
        $acl = Get-Acl $DataDir
        $accessRule = New-Object System.Security.AccessControl.FileSystemAccessRule("Users","FullControl","ContainerInherit,ObjectInherit","None","Allow")
        $acl.SetAccessRule($accessRule)
        Set-Acl $DataDir $acl
        
        Write-Step "✅ Directories created successfully" "Success"
    } catch {
        Write-Step "ERROR: Failed to create directories: $($_.Exception.Message)" "Error"
        exit 1
    }

    # Copy bridge executable
    Write-Step "Installing bridge executable..."
    try {
        Copy-Item $downloadInfo.bridgeExe $ExePath -Force
        
        # Verify executable
        if (-not (Test-Path $ExePath)) {
            throw "Executable not copied successfully"
        }
        
        # Test executable
        $version = & $ExePath --version 2>&1 | Select-Object -First 1
        Write-Step "✅ Bridge executable installed: $version" "Success"
    } catch {
        Write-Step "ERROR: Failed to install executable: $($_.Exception.Message)" "Error"
        exit 1
    }

    # Create configuration file
    Write-Step "Creating bridge configuration..."
    try {
        $configContent = @"
# RepSet Bridge Configuration
# Auto-generated by Step 2 installer

server_url: "https://repset.onezy.in"
tier: "normal"
queue_max_size: 10000
heartbeat_interval: 60
unlock_duration: 3000

# Device credentials (will be filled during pairing)
device_id: ""
device_key: ""

# Storage paths  
database_path: "$($DataDir.Replace('\','/'))/bridge.db"
log_level: "info"
log_file: "$($DataDir.Replace('\','/'))/bridge.log"

# Hardware adapters (auto-discovery enabled)
enabled_adapters:
  - "simulator"  # For testing
  - "zkteco"     # ZKTeco fingerprint devices  
  - "essl"       # ESSL biometric devices
  - "realtime"   # Realtime devices

# API server configuration
api_server:
  enabled: true
  port: 8081
  host: "0.0.0.0"

# Connection settings
retry_interval: 30
max_retry_attempts: 3
connection_timeout: 30

# Auto-discovery settings
discovery:
  enabled: true
  scan_interval: 300
  port_ranges:
    - "4370"      # ZKTeco default
    - "80,8080"   # ESSL defaults  
    - "5005,9999" # Realtime defaults
"@

        Set-Content -Path $ConfigPath -Value $configContent -Encoding UTF8
        Write-Step "✅ Configuration file created" "Success"
    } catch {
        Write-Step "ERROR: Failed to create configuration: $($_.Exception.Message)" "Error"
        exit 1
    }

    # Set Windows Defender exclusions
    Write-Step "Setting up Windows Defender exclusions..."
    try {
        $exclusionPaths = @($InstallPath, $DataDir)
        foreach ($path in $exclusionPaths) {
            Add-MpPreference -ExclusionPath $path -ErrorAction SilentlyContinue
        }
        Add-MpPreference -ExclusionProcess "$ExePath" -ErrorAction SilentlyContinue
        Write-Step "✅ Windows Defender exclusions added" "Success"
    } catch {
        Write-Step "⚠️ Could not add Windows Defender exclusions (non-critical)" "Warning"
    }

    # Create installation info file
    Write-Step "Creating installation record..."
    try {
        $installInfo = @{
            installTime = (Get-Date).ToString()
            installPath = $InstallPath
            dataDir = $DataDir
            exePath = $ExePath
            configPath = $ConfigPath
            version = $version
            installStep = 2
        }
        $installInfo | ConvertTo-Json | Set-Content "$TempDir\install-info.json"
        Write-Step "✅ Installation record created" "Success"
    } catch {
        Write-Step "⚠️ Could not create installation record (non-critical)" "Warning"
    }

    # Verify installation
    Write-Step "Verifying installation..."
    $verificationPassed = $true
    $issues = @()

    if (-not (Test-Path $ExePath)) {
        $verificationPassed = $false
        $issues += "Bridge executable missing"
    }

    if (-not (Test-Path $ConfigPath)) {
        $verificationPassed = $false
        $issues += "Configuration file missing"
    }

    if (-not (Test-Path $DataDir)) {
        $verificationPassed = $false
        $issues += "Data directory missing"
    }

    # Test bridge executable
    try {
        $testResult = & $ExePath --help 2>&1
        if ($LASTEXITCODE -ne 0) {
            $verificationPassed = $false
            $issues += "Bridge executable test failed"
        }
    } catch {
        $verificationPassed = $false
        $issues += "Bridge executable cannot run"
    }

    if (-not $Silent) {
        Write-Host ""
        if ($verificationPassed) {
            Write-Host "🎉 STEP 2 COMPLETED SUCCESSFULLY!" -ForegroundColor Green
            Write-Host "=================================" -ForegroundColor Green
        } else {
            Write-Host "⚠️ STEP 2 COMPLETED WITH ISSUES" -ForegroundColor Yellow
            Write-Host "===============================" -ForegroundColor Yellow
            foreach ($issue in $issues) {
                Write-Host "   ❗ $issue" -ForegroundColor Red
            }
        }

        Write-Host ""
        Write-Host "📋 Installation Summary:" -ForegroundColor Cyan
        Write-Host "   📁 Install Path: $InstallPath" -ForegroundColor Gray
        Write-Host "   🔧 Executable: $ExePath" -ForegroundColor Gray
        Write-Host "   ⚙️ Config File: $ConfigPath" -ForegroundColor Gray
        Write-Host "   💾 Data Directory: $DataDir" -ForegroundColor Gray
        Write-Host "   🔍 Version: $version" -ForegroundColor Gray
        Write-Host ""

        if ($verificationPassed) {
            Write-Host "✅ Ready for Step 3: Service Setup" -ForegroundColor Green
            Write-Host ""
            Write-Host "Next: Run the service setup script:" -ForegroundColor Yellow
            Write-Host "   .\step3-service.ps1" -ForegroundColor Gray
        } else {
            Write-Host "❌ Please resolve the issues above before continuing" -ForegroundColor Red
        }
        Write-Host ""
        
        Read-Host "Press Enter to continue"
    }

} catch {
    Write-Step "UNEXPECTED ERROR: $($_.Exception.Message)" "Error"
    if (-not $Silent) { Read-Host "Press Enter to exit" }
    exit 1
}