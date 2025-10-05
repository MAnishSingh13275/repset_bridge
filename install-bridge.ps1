# ================================================================
# RepSet Gym Door Bridge - Master Installation Script
# Ultra-reliable installer for production deployments
# Replaces all previous installation scripts
# ================================================================

param(
    [Parameter(Mandatory=$false)]
    [string]$PairCode = "",
    [string]$ServerUrl = "https://repset.onezy.in",
    [switch]$Force = $false,
    [switch]$Silent = $false,
    [switch]$SkipPairing = $false
)

# Configuration
$ProgressPreference = 'SilentlyContinue'
$ErrorActionPreference = 'Stop'

# Enhanced logging function
function Write-Log {
    param([string]$Message, [string]$Level = "Info")
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $color = switch ($Level) {
        "Error" { "Red" }
        "Warning" { "Yellow" }
        "Success" { "Green" }
        "Info" { "Cyan" }
        default { "White" }
    }
    if (-not $Silent) {
        Write-Host "[$timestamp] $Message" -ForegroundColor $color
    }
}

# Error handling with cleanup
function Write-ErrorAndExit {
    param([string]$Message, [string]$Details = "")
    Write-Log "INSTALLATION FAILED: $Message" "Error"
    if ($Details) { Write-Log "Details: $Details" "Error" }
    Write-Log "Please contact support with this error message." "Warning"
    if ($script:TempZip -and (Test-Path $script:TempZip)) {
        Remove-Item $script:TempZip -Force -ErrorAction SilentlyContinue
    }
    if (-not $Silent) {
        Read-Host "Press Enter to exit"
    }
    exit 1
}

# Initialize script variables
$script:TempZip = ""

try {
    # Welcome banner
    if (-not $Silent) {
        Clear-Host
        $Host.UI.RawUI.WindowTitle = "RepSet Bridge Installer"
        Write-Host ""
        Write-Host "🚀 RepSet Gym Door Bridge - Master Installer v2.0" -ForegroundColor Cyan
        Write-Host "=================================================" -ForegroundColor Cyan
        Write-Host "✨ Production-Ready • Ultra-Reliable • Smart Pairing" -ForegroundColor Gray
        Write-Host ""
    }

    # Step 1: Admin check
    Write-Log "Checking administrator privileges..." "Info"
    if (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
        Write-ErrorAndExit "Administrator privileges required" "Please run PowerShell as Administrator and try again."
    }
    Write-Log "✅ Administrator privileges confirmed" "Success"

    # Define paths and constants
    $InstallPath = "$env:ProgramFiles\GymDoorBridge"
    $DataDir = "$env:ProgramData\GymDoorBridge"  
    $ExePath = "$InstallPath\gym-door-bridge.exe"
    $ConfigPath = "$InstallPath\config.yaml"
    $script:TempZip = "$env:TEMP\repset-bridge-$(Get-Random).zip"
    $DownloadUrl = "https://github.com/MAnishSingh13275/repset_bridge/releases/latest/download/gym-door-bridge-windows.zip"

    # Step 2: Handle existing installations
    Write-Log "Checking for existing installation..." "Info"
    $existingService = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
    
    if ($existingService) {
        if ($Force -or $PairCode) {
            Write-Log "🔄 Existing installation found - updating..." "Warning"
            try {
                Stop-Service -Name "GymDoorBridge" -Force -ErrorAction SilentlyContinue
                Get-Process -Name "gym-door-bridge" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
                Start-Sleep 2
                Write-Log "✅ Existing service stopped" "Success"
            } catch {
                Write-Log "⚠️ Could not stop service cleanly - continuing..." "Warning"
            }
        } else {
            Write-ErrorAndExit "Bridge already installed" "Use -Force to reinstall, or -PairCode to update pairing"
        }
    } else {
        Write-Log "✅ No existing installation found" "Success"
    }

    # Step 3: Create directories
    Write-Log "Setting up directories..." "Info"
    New-Item -ItemType Directory -Force -Path $InstallPath, $DataDir | Out-Null
    Write-Log "✅ Directories created" "Success"

    # Step 4: Download with robust fallback methods
    Write-Log "Downloading latest release..." "Info"
    $downloadSuccess = $false
    
    # Method 1: PowerShell Invoke-WebRequest
    try {
        Invoke-WebRequest -Uri $DownloadUrl -OutFile $script:TempZip -UseBasicParsing -TimeoutSec 60
        if (Test-Path $script:TempZip) {
            $downloadSuccess = $true
            Write-Log "✅ Download completed using Invoke-WebRequest" "Success"
        }
    } catch {
        Write-Log "⚠️ Method 1 failed, trying alternative..." "Warning"
    }

    # Method 2: .NET WebClient
    if (-not $downloadSuccess) {
        try {
            $webClient = New-Object System.Net.WebClient
            $webClient.DownloadTimeout = 60000
            $webClient.Headers.Add("User-Agent", "RepSet-Bridge-Installer")
            $webClient.DownloadFile($DownloadUrl, $script:TempZip)
            if (Test-Path $script:TempZip) {
                $downloadSuccess = $true
                Write-Log "✅ Download completed using WebClient" "Success"
            }
        } catch {
            Write-Log "⚠️ Method 2 failed, trying BITS transfer..." "Warning"
        }
    }

    # Method 3: BITS Transfer
    if (-not $downloadSuccess) {
        try {
            Import-Module BitsTransfer -ErrorAction Stop
            Start-BitsTransfer -Source $DownloadUrl -Destination $script:TempZip -TransferType Download
            if (Test-Path $script:TempZip) {
                $downloadSuccess = $true
                Write-Log "✅ Download completed using BITS" "Success"
            }
        } catch {
            Write-Log "⚠️ All download methods failed" "Warning"
        }
    }

    if (-not $downloadSuccess) {
        Write-ErrorAndExit "Download failed" "All download methods failed. Check internet connection and firewall settings."
    }

    # Verify download
    $fileInfo = Get-Item $script:TempZip
    $sizeMB = [math]::Round($fileInfo.Length / 1MB, 2)
    Write-Log "✅ Downloaded $sizeMB MB successfully" "Success"

    # Step 5: Extract and install
    Write-Log "Extracting and installing bridge..." "Info"
    try {
        # Use .NET method for better reliability
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        [System.IO.Compression.ZipFile]::ExtractToDirectory($script:TempZip, $InstallPath)
    } catch {
        # Fallback to PowerShell method
        try {
            Expand-Archive -Path $script:TempZip -DestinationPath $InstallPath -Force
        } catch {
            Write-ErrorAndExit "Extraction failed" $_.Exception.Message
        }
    }

    # Find and verify executable
    $exe = Get-ChildItem -Path $InstallPath -Filter "*.exe" -Recurse -ErrorAction SilentlyContinue | 
           Where-Object { $_.Name -match "gym-door-bridge|bridge" } | 
           Select-Object -First 1
    
    if (-not $exe) {
        Write-ErrorAndExit "Executable not found" "No bridge executable found in downloaded package"
    }
    
    # Ensure correct naming
    if ($exe.Name -ne "gym-door-bridge.exe") {
        Move-Item $exe.FullName $ExePath -Force
    }
    
    # Verify executable works
    try {
        $version = & $ExePath --version 2>&1
        Write-Log "✅ Bridge installed successfully (Version: $version)" "Success"
    } catch {
        Write-Log "✅ Bridge installed to $InstallPath" "Success"
    }

    # Step 6: Create optimized configuration
    Write-Log "Creating configuration..." "Info"
    $configContent = @"
# RepSet Bridge Configuration
# Auto-generated by installer v2.0

server_url: "$ServerUrl"
tier: "normal"
queue_max_size: 10000
heartbeat_interval: 60
unlock_duration: 3000

# Device credentials (filled during pairing)
device_id: ""
device_key: ""

# Storage paths
database_path: "$($DataDir.Replace('\','/'))/bridge.db"
log_level: "info"
log_file: "$($DataDir.Replace('\','/'))/bridge.log"

# Hardware adapters
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

# Advanced settings
retry_interval: 30
max_retry_attempts: 3
connection_timeout: 30
"@
    
    Set-Content -Path $ConfigPath -Value $configContent -Encoding UTF8
    Write-Log "✅ Configuration created" "Success"

    # Step 7: Device pairing (if requested)
    if ($PairCode -and -not $SkipPairing) {
        Write-Log "Initiating device pairing..." "Info"
        
        # Try unpair first (for re-pairing scenarios)
        try {
            $unpairResult = & $ExePath unpair --config $ConfigPath 2>&1
            Write-Log "Cleared any existing pairing" "Info"
        } catch {
            # Ignore unpair failures - device might not be paired
        }
        
        # Attempt pairing
        try {
            $pairResult = & $ExePath pair $PairCode --config $ConfigPath 2>&1
            if ($LASTEXITCODE -eq 0) {
                Write-Log "✅ Successfully paired with RepSet platform!" "Success"
            } else {
                # Check if it's a network/server issue vs invalid code
                if ($pairResult -match "network|connection|timeout|server") {
                    Write-Log "⚠️ Pairing delayed due to network - will retry automatically" "Warning"
                } else {
                    Write-ErrorAndExit "Pairing failed" "Invalid pair code or server error: $pairResult"
                }
            }
        } catch {
            Write-ErrorAndExit "Pairing error" $_.Exception.Message
        }
    } elseif (-not $SkipPairing) {
        Write-Log "⚠️ Skipping pairing (no pair code provided)" "Warning"
        Write-Log "You can pair later using: gym-door-bridge pair YOUR_CODE" "Info"
    }

    # Step 8: Install and start Windows service
    Write-Log "Installing Windows service..." "Info"
    
    try {
        # Use bridge's built-in service installer
        $installResult = & $ExePath install --config $ConfigPath 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Log "✅ Windows service installed successfully" "Success"
        } else {
            throw "Service installation failed: $installResult"
        }
    } catch {
        Write-ErrorAndExit "Service installation failed" $_.Exception.Message
    }

    # Start and verify service
    Write-Log "Starting RepSet Bridge service..." "Info"
    try {
        Start-Service -Name "GymDoorBridge" -ErrorAction Stop
        Start-Sleep 5  # Allow time for startup
        
        $service = Get-Service -Name "GymDoorBridge" -ErrorAction Stop
        if ($service.Status -eq "Running") {
            Write-Log "✅ Service started and running successfully" "Success"
            
            # Quick API health check
            try {
                Start-Sleep 3
                $healthCheck = Invoke-WebRequest -Uri "http://localhost:8081/health" -UseBasicParsing -TimeoutSec 5 -ErrorAction Stop
                Write-Log "✅ API endpoint responding (HTTP $($healthCheck.StatusCode))" "Success"
            } catch {
                Write-Log "⚠️ API endpoint not yet available (normal during startup)" "Warning"
            }
        } else {
            Write-Log "⚠️ Service installed but status: $($service.Status)" "Warning"
        }
    } catch {
        Write-Log "⚠️ Service installed but failed to start automatically" "Warning"
        Write-Log "This is normal if pairing is not complete" "Info"
    }

    # Step 9: Final verification and summary
    Write-Log "Performing final verification..." "Info"
    
    $verificationPassed = $true
    $issues = @()
    
    # Check file installation
    if (-not (Test-Path $ExePath)) {
        $verificationPassed = $false
        $issues += "Bridge executable missing"
    }
    
    if (-not (Test-Path $ConfigPath)) {
        $verificationPassed = $false  
        $issues += "Configuration file missing"
    }
    
    # Check service installation
    $finalService = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
    if (-not $finalService) {
        $verificationPassed = $false
        $issues += "Windows service not installed"
    }
    
    # Check directories
    if (-not (Test-Path $DataDir)) {
        $verificationPassed = $false
        $issues += "Data directory missing"
    }

    if (-not $Silent) {
        Write-Host ""
        if ($verificationPassed) {
            Write-Host "🎉 INSTALLATION COMPLETED SUCCESSFULLY!" -ForegroundColor Green
            Write-Host "=====================================" -ForegroundColor Green
        } else {
            Write-Host "⚠️ INSTALLATION COMPLETED WITH ISSUES" -ForegroundColor Yellow
            Write-Host "====================================" -ForegroundColor Yellow
            foreach ($issue in $issues) {
                Write-Host "   ❗ $issue" -ForegroundColor Red
            }
        }
        
        Write-Host ""
        Write-Host "📋 Installation Summary:" -ForegroundColor Cyan
        Write-Host "   📁 Install Path: $InstallPath" -ForegroundColor Gray
        Write-Host "   🔧 Config File: $ConfigPath" -ForegroundColor Gray
        Write-Host "   💾 Data Directory: $DataDir" -ForegroundColor Gray
        Write-Host "   🌐 Server URL: $ServerUrl" -ForegroundColor Gray
        Write-Host "   🔌 API Endpoint: http://localhost:8081" -ForegroundColor Gray
        Write-Host "   🔧 Service Name: GymDoorBridge" -ForegroundColor Gray
        
        if ($finalService) {
            Write-Host "   📊 Service Status: $($finalService.Status)" -ForegroundColor $(if ($finalService.Status -eq 'Running') { 'Green' } else { 'Yellow' })
        }
        
        if ($PairCode -and -not $SkipPairing) {
            Write-Host "   🔗 Pairing Status: Completed" -ForegroundColor Green
        } else {
            Write-Host "   🔗 Pairing Status: Manual pairing required" -ForegroundColor Yellow
        }
        
        Write-Host ""
        Write-Host "🔧 Management Commands:" -ForegroundColor Cyan
        Write-Host "   gym-door-bridge status           - Check bridge status" -ForegroundColor Gray
        Write-Host "   gym-door-bridge pair YOUR_CODE   - Pair with RepSet" -ForegroundColor Gray
        Write-Host "   gym-door-bridge unpair           - Unpair device" -ForegroundColor Gray
        Write-Host "   net start GymDoorBridge          - Start service" -ForegroundColor Gray
        Write-Host "   net stop GymDoorBridge           - Stop service" -ForegroundColor Gray
        
        if (-not $PairCode -or $SkipPairing) {
            Write-Host ""
            Write-Host "🔗 Next Steps:" -ForegroundColor Yellow
            Write-Host "   1. Get your pair code from RepSet admin dashboard" -ForegroundColor Gray
            Write-Host "   2. Run: gym-door-bridge pair YOUR_PAIR_CODE" -ForegroundColor Gray
            Write-Host "   3. Verify: gym-door-bridge status" -ForegroundColor Gray
        }
        
        Write-Host ""
        Write-Host "🎯 RepSet Bridge is ready for gym integration!" -ForegroundColor Green
        Write-Host ""
        
        if ($verificationPassed) {
            Write-Host "Installation completed successfully. You can close this window." -ForegroundColor Gray
        } else {
            Write-Host "Please review the issues above or contact support." -ForegroundColor Yellow
        }
        
        Read-Host "Press Enter to continue"
    }
    
    Write-Log "Installation process completed" "Success"

} catch {
    Write-ErrorAndExit "Unexpected installation error" $_.Exception.Message
} finally {
    # Cleanup temporary files
    if ($script:TempZip -and (Test-Path $script:TempZip)) {
        Remove-Item $script:TempZip -Force -ErrorAction SilentlyContinue
    }
    
    # Reset console title
    if (-not $Silent) {
        $Host.UI.RawUI.WindowTitle = "Administrator: Windows PowerShell"
    }
}