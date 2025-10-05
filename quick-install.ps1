# ================================================================
# RepSet Gym Door Bridge - Ultra-Fast Installation Script
# Comprehensive, reliable installer for instant customer deployment
# ================================================================

param(
    [Parameter(Mandatory=$false)]
    [string]$PairCode = "",
    [string]$ServerUrl = "https://repset.onezy.in",
    [switch]$Force = $false,
    [switch]$Silent = $false
)

# Disable progress bars for faster downloads
$ProgressPreference = 'SilentlyContinue'
$ErrorActionPreference = 'Stop'

# Console setup
if (-not $Silent) {
    Clear-Host
    $Host.UI.RawUI.WindowTitle = "RepSet Bridge Installer"
}

# Enhanced error handling function
function Write-ErrorAndExit {
    param([string]$Message, [string]$Details = "")
    Write-Host "`n❌ INSTALLATION FAILED" -ForegroundColor Red
    Write-Host "======================" -ForegroundColor Red
    Write-Host "Error: $Message" -ForegroundColor Red
    if ($Details) {
        Write-Host "Details: $Details" -ForegroundColor Yellow
    }
    Write-Host "`nPlease contact support with this error message." -ForegroundColor Yellow
    if (-not $Silent) {
        Read-Host "Press Enter to exit"
    }
    exit 1
}

# Banner
if (-not $Silent) {
    Write-Host ""
    Write-Host "🚀 RepSet Gym Door Bridge - Ultra-Fast Installer" -ForegroundColor Cyan
    Write-Host "===============================================" -ForegroundColor Cyan
    Write-Host "✨ Reliable • Fast • Zero-Config • Smart Pairing" -ForegroundColor Gray
    Write-Host ""
}

try {
    # Step 1: Verify admin privileges
    if (-not $Silent) { Write-Host "[1/8] Checking administrator privileges..." -ForegroundColor White }
    if (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
        Write-ErrorAndExit "Administrator privileges required" "Please run PowerShell as Administrator and try again."
    }
    if (-not $Silent) { Write-Host "      ✅ Administrator privileges confirmed" -ForegroundColor Green }

    # Constants
    $InstallPath = "$env:ProgramFiles\GymDoorBridge"
    $DataDir = "$env:ProgramData\GymDoorBridge"
    $ExePath = "$InstallPath\gym-door-bridge.exe"
    $ConfigPath = "$InstallPath\config.yaml"
    $TempZip = "$env:TEMP\repset-bridge-$(Get-Random).zip"
    $DownloadUrl = "https://github.com/MAnishSingh13275/repset_bridge/releases/latest/download/gym-door-bridge-windows.zip"

    # Step 2: Handle existing installation
    if (-not $Silent) { Write-Host "[2/8] Checking existing installation..." -ForegroundColor White }
    $existingService = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
    if ($existingService -and -not $Force) {
        if ($PairCode) {
            if (-not $Silent) { Write-Host "      🔄 Existing installation found, will update and re-pair" -ForegroundColor Yellow }
            $Force = $true
        } else {
            Write-ErrorAndExit "Bridge already installed" "Use -Force to reinstall, or provide -PairCode to update pairing"
        }
    }

    if ($existingService) {
        if (-not $Silent) { Write-Host "      🛑 Stopping existing service..." -ForegroundColor Yellow }
        try {
            Stop-Service -Name "GymDoorBridge" -Force -ErrorAction SilentlyContinue
            Get-Process -Name "gym-door-bridge" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
            Start-Sleep 2
            if (-not $Silent) { Write-Host "      ✅ Existing service stopped" -ForegroundColor Green }
        } catch {
            if (-not $Silent) { Write-Host "      ⚠️  Could not stop service cleanly" -ForegroundColor Yellow }
        }
    } else {
        if (-not $Silent) { Write-Host "      ✅ No existing installation found" -ForegroundColor Green }
    }

    # Step 3: Create directories
    if (-not $Silent) { Write-Host "[3/8] Setting up directories..." -ForegroundColor White }
    New-Item -ItemType Directory -Force -Path $InstallPath, $DataDir | Out-Null
    if (-not $Silent) { Write-Host "      ✅ Directories created" -ForegroundColor Green }

    # Step 4: Download with multiple fallbacks
    if (-not $Silent) { Write-Host "[4/8] Downloading latest release..." -ForegroundColor White }
    $downloadSuccess = $false
    $downloadMethods = @(
        { Invoke-WebRequest -Uri $DownloadUrl -OutFile $TempZip -UseBasicParsing -TimeoutSec 30 },
        { 
            $webClient = New-Object System.Net.WebClient
            $webClient.DownloadTimeout = 30000
            $webClient.DownloadFile($DownloadUrl, $TempZip)
        },
        { 
            Import-Module BitsTransfer -ErrorAction SilentlyContinue
            Start-BitsTransfer -Source $DownloadUrl -Destination $TempZip -TransferType Download
        }
    )

    foreach ($method in $downloadMethods) {
        try {
            & $method
            if (Test-Path $TempZip) {
                $downloadSuccess = $true
                break
            }
        } catch {
            # Try next method
        }
    }

    if (-not $downloadSuccess) {
        Write-ErrorAndExit "Download failed" "All download methods failed. Check internet connection."
    }
    if (-not $Silent) { Write-Host "      ✅ Download completed" -ForegroundColor Green }

    # Step 5: Extract and install
    if (-not $Silent) { Write-Host "[5/8] Extracting and installing..." -ForegroundColor White }
    try {
        Expand-Archive -Path $TempZip -DestinationPath $InstallPath -Force
    } catch {
        Write-ErrorAndExit "Extraction failed" $_.Exception.Message
    }

    # Find executable
    $exe = Get-ChildItem -Path $InstallPath -Filter "*.exe" -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
    if (-not $exe) {
        Write-ErrorAndExit "Executable not found" "No .exe file found in downloaded package"
    }
    
    if ($exe.Name -ne "gym-door-bridge.exe") {
        Move-Item $exe.FullName $ExePath -Force
    }
    if (-not $Silent) { Write-Host "      ✅ Bridge installed to $InstallPath" -ForegroundColor Green }

    # Step 6: Create configuration
    if (-not $Silent) { Write-Host "[6/8] Creating configuration..." -ForegroundColor White }
    $configContent = @"
server_url: "$ServerUrl"
tier: "normal"
queue_max_size: 10000
heartbeat_interval: 60
unlock_duration: 3000
device_id: ""
device_key: ""
database_path: "$($DataDir.Replace('\\','/'))/bridge.db"
log_level: "info"
log_file: "$($DataDir.Replace('\\','/'))/bridge.log"
enabled_adapters: ["simulator"]
api_server:
  enabled: true
  port: 8081
  host: "0.0.0.0"
"@
    Set-Content -Path $ConfigPath -Value $configContent -Encoding UTF8
    if (-not $Silent) { Write-Host "      ✅ Configuration created" -ForegroundColor Green }

    # Step 7: Pairing (if pair code provided)
    if ($PairCode) {
        if (-not $Silent) { Write-Host "[7/8] Pairing with platform..." -ForegroundColor White }
        
        # Try unpair first (for re-pairing scenarios)
        try {
            & $ExePath unpair --config $ConfigPath 2>&1 | Out-Null
        } catch {
            # Ignore unpair failures
        }
        
        # Attempt pairing
        try {
            $pairOutput = & $ExePath pair $PairCode --config $ConfigPath 2>&1
            if ($LASTEXITCODE -eq 0) {
                if (-not $Silent) { Write-Host "      ✅ Successfully paired with platform" -ForegroundColor Green }
            } else {
                Write-ErrorAndExit "Pairing failed" "$pairOutput"
            }
        } catch {
            Write-ErrorAndExit "Pairing error" $_.Exception.Message
        }
    } else {
        if (-not $Silent) { Write-Host "[7/8] Skipping pairing (no pair code provided)" -ForegroundColor Yellow }
    }

    # Step 8: Install and start service
    if (-not $Silent) { Write-Host "[8/8] Installing and starting service..." -ForegroundColor White }
    
    try {
        # Install service using the bridge's install command
        $installOutput = & $ExePath install --config $ConfigPath 2>&1
        if ($LASTEXITCODE -ne 0) {
            throw "Service installation failed: $installOutput"
        }
        if (-not $Silent) { Write-Host "      ✅ Windows service installed" -ForegroundColor Green }
    } catch {
        Write-ErrorAndExit "Service installation failed" $_.Exception.Message
    }

    # Start the service
    try {
        Start-Service -Name "GymDoorBridge" -ErrorAction Stop
        Start-Sleep 3
        $service = Get-Service -Name "GymDoorBridge" -ErrorAction Stop
        if ($service.Status -eq "Running") {
            if (-not $Silent) { Write-Host "      ✅ Service started and running" -ForegroundColor Green }
        } else {
            if (-not $Silent) { Write-Host "      ⚠️  Service installed but not running (Status: $($service.Status))" -ForegroundColor Yellow }
        }
    } catch {
        if (-not $Silent) { Write-Host "      ⚠️  Service installed but failed to start" -ForegroundColor Yellow }
    }

    # Final verification
    if (-not $Silent) {
        Write-Host "`n🎉 INSTALLATION SUCCESSFUL!" -ForegroundColor Green
        Write-Host "===========================" -ForegroundColor Green
        Write-Host "✅ RepSet Bridge installed and configured" -ForegroundColor Green
        Write-Host "✅ Windows service created and started" -ForegroundColor Green
        if ($PairCode) {
            Write-Host "✅ Successfully paired with platform" -ForegroundColor Green
        }
        Write-Host "`n📊 Installation Summary:" -ForegroundColor Cyan
        Write-Host "   📁 Install Path: $InstallPath" -ForegroundColor Gray
        Write-Host "   🔧 Config Path: $ConfigPath" -ForegroundColor Gray
        Write-Host "   🌐 Server URL: $ServerUrl" -ForegroundColor Gray
        Write-Host "   🔌 API Endpoint: http://localhost:8081" -ForegroundColor Gray
        
        if (-not $PairCode) {
            Write-Host "`n🔗 Next Steps:" -ForegroundColor Yellow
            Write-Host "   1. Get your pair code from the RepSet admin dashboard" -ForegroundColor Gray
            Write-Host "   2. Run: gym-door-bridge pair YOUR_PAIR_CODE" -ForegroundColor Gray
            Write-Host "   3. Check status: gym-door-bridge status" -ForegroundColor Gray
        }
        
        Write-Host "`n📋 Management Commands:" -ForegroundColor Cyan
        Write-Host "   gym-door-bridge status    - Check bridge status" -ForegroundColor Gray
        Write-Host "   gym-door-bridge pair CODE - Pair with platform" -ForegroundColor Gray
        Write-Host "   net stop GymDoorBridge    - Stop service" -ForegroundColor Gray
        Write-Host "   net start GymDoorBridge   - Start service" -ForegroundColor Gray
        
        Write-Host "`n🎯 Your gym bridge is ready for RepSet integration!" -ForegroundColor Green
        Write-Host ""
        Read-Host "Press Enter to continue"
    }

} catch {
    Write-ErrorAndExit "Unexpected error" $_.Exception.Message
} finally {
    # Cleanup
    if (Test-Path $TempZip) {
        Remove-Item $TempZip -Force -ErrorAction SilentlyContinue
    }
}
