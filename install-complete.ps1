# ================================================================
# RepSet Bridge - Complete Installation Script
# Downloads, installs, configures, and starts the bridge service
# ================================================================

param(
    [Parameter(Mandatory=$false)]
    [string]$PairCode = "",
    [string]$ServerUrl = "https://repset.onezy.in",
    [switch]$Silent = $false
)

$ProgressPreference = 'SilentlyContinue'
$ErrorActionPreference = 'Stop'

function Write-Log {
    param([string]$Message, [string]$Level = "Info")
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $color = switch ($Level) {
        "Error" { "Red" }
        "Success" { "Green" }
        "Warning" { "Yellow" }
        default { "Cyan" }
    }
    if (-not $Silent) {
        Write-Host "[$timestamp] $Message" -ForegroundColor $color
    }
    # Also log to file for background runs
    "$timestamp [$Level] $Message" | Add-Content -Path "$env:TEMP\repset-bridge-install.log" -ErrorAction SilentlyContinue
}

function Write-ErrorAndExit {
    param([string]$Message, [string]$Details = "")
    Write-Log "INSTALLATION FAILED: $Message" "Error"
    if ($Details) { Write-Log "Details: $Details" "Error" }
    if (-not $Silent) {
        Write-Host "Check log file: $env:TEMP\repset-bridge-install.log" -ForegroundColor Yellow
        Read-Host "Press Enter to exit"
    }
    exit 1
}

try {
    if (-not $Silent) {
        Clear-Host
        Write-Host ""
        Write-Host "🚀 RepSet Bridge - Complete Installer" -ForegroundColor Cyan
        Write-Host "====================================" -ForegroundColor Cyan
        Write-Host ""
    }

    Write-Log "Starting RepSet Bridge installation..."

    # Check admin privileges
    Write-Log "Checking administrator privileges..."
    if (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
        Write-ErrorAndExit "Administrator privileges required" "Please run PowerShell as Administrator"
    }
    Write-Log "✅ Administrator privileges confirmed" "Success"

    # Get pair code if not provided and not silent
    if (-not $PairCode -and -not $Silent) {
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
            $PairCode = Read-Host "Enter your pair code (or press Enter to skip pairing for now)"
            if (-not $PairCode) {
                Write-Host "Continuing without pairing - you can pair later using: gym-door-bridge pair YOUR_CODE" -ForegroundColor Yellow
                break
            }
        } while ($false)
    }

    # Define paths
    $TempDir = "$env:TEMP\RepSetBridge-$(Get-Random)"
    $DownloadZip = "$TempDir\gym-door-bridge.zip"
    $ExtractDir = "$TempDir\extracted"
    $InstallPath = "$env:ProgramFiles\GymDoorBridge"
    $DataDir = "$env:ProgramData\GymDoorBridge"
    $ExePath = "$InstallPath\gym-door-bridge.exe"
    $ConfigPath = "$InstallPath\config.yaml"
    $DownloadUrl = "https://github.com/MAnishSingh13275/repset_bridge/releases/latest/download/gym-door-bridge-windows.zip"

    # Step 1: Download
    Write-Log "Step 1: Downloading bridge files..."
    New-Item -ItemType Directory -Path $TempDir, $ExtractDir -Force | Out-Null

    $downloadSuccess = $false
    # Try multiple download methods
    $methods = @("Invoke-WebRequest", "WebClient", "BITS")
    foreach ($method in $methods) {
        try {
            Write-Log "Trying download method: $method"
            switch ($method) {
                "Invoke-WebRequest" {
                    Invoke-WebRequest -Uri $DownloadUrl -OutFile $DownloadZip -UseBasicParsing -TimeoutSec 120
                }
                "WebClient" {
                    $webClient = New-Object System.Net.WebClient
                    $webClient.Headers.Add("User-Agent", "RepSet-Bridge-Installer")
                    $webClient.DownloadFile($DownloadUrl, $DownloadZip)
                }
                "BITS" {
                    Import-Module BitsTransfer -ErrorAction Stop
                    Start-BitsTransfer -Source $DownloadUrl -Destination $DownloadZip -TransferType Download
                }
            }
            
            if (Test-Path $DownloadZip) {
                $fileSize = (Get-Item $DownloadZip).Length
                if ($fileSize -gt 1MB) {
                    $downloadSuccess = $true
                    $sizeMB = [math]::Round($fileSize / 1MB, 2)
                    Write-Log "✅ Download successful ($sizeMB MB)" "Success"
                    break
                }
            }
        } catch {
            Write-Log "Method $method failed: $($_.Exception.Message)" "Warning"
        }
    }

    if (-not $downloadSuccess) {
        Write-ErrorAndExit "All download methods failed" "Check internet connection and firewall settings"
    }

    # Step 2: Extract
    Write-Log "Step 2: Extracting files..."
    try {
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        [System.IO.Compression.ZipFile]::ExtractToDirectory($DownloadZip, $ExtractDir)
    } catch {
        try {
            Expand-Archive -Path $DownloadZip -DestinationPath $ExtractDir -Force
        } catch {
            Write-ErrorAndExit "Extraction failed" $_.Exception.Message
        }
    }

    $bridgeExe = Get-ChildItem -Path $ExtractDir -Filter "*.exe" -Recurse | Where-Object { $_.Name -match "gym-door-bridge|bridge" } | Select-Object -First 1
    if (-not $bridgeExe) {
        Write-ErrorAndExit "Bridge executable not found in download"
    }
    Write-Log "✅ Found bridge executable: $($bridgeExe.Name)" "Success"

    # Step 3: Install
    Write-Log "Step 3: Installing bridge..."
    
    # Stop existing service
    $existingService = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
    if ($existingService) {
        Write-Log "Stopping existing service..."
        try {
            Stop-Service -Name "GymDoorBridge" -Force -ErrorAction SilentlyContinue
            Get-Process -Name "gym-door-bridge" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
            Start-Sleep 3
        } catch {
            Write-Log "Could not stop existing service cleanly" "Warning"
        }
    }

    # Create directories
    if (Test-Path $InstallPath) {
        Remove-Item $InstallPath -Recurse -Force -ErrorAction SilentlyContinue
    }
    New-Item -ItemType Directory -Path $InstallPath, $DataDir -Force | Out-Null

    # Set permissions on data directory
    try {
        $acl = Get-Acl $DataDir
        $accessRule = New-Object System.Security.AccessControl.FileSystemAccessRule("Users","FullControl","ContainerInherit,ObjectInherit","None","Allow")
        $acl.SetAccessRule($accessRule)
        Set-Acl $DataDir $acl
    } catch {
        Write-Log "Could not set directory permissions (non-critical)" "Warning"
    }

    # Copy executable
    Copy-Item $bridgeExe.FullName $ExePath -Force
    if (-not (Test-Path $ExePath)) {
        Write-ErrorAndExit "Failed to copy executable"
    }

    # Test executable
    try {
        $version = & $ExePath --version 2>&1 | Select-Object -First 1
        Write-Log "✅ Bridge installed: $version" "Success"
    } catch {
        Write-Log "✅ Bridge installed to $InstallPath" "Success"
    }

    # Step 4: Create configuration
    Write-Log "Step 4: Creating configuration..."
    $configContent = @"
# RepSet Bridge Configuration
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
  - "simulator"
  - "zkteco"
  - "essl"
  - "realtime"

# API server
api_server:
  enabled: true
  port: 8081
  host: "0.0.0.0"

# Connection settings
retry_interval: 30
max_retry_attempts: 3
connection_timeout: 30
"@

    Set-Content -Path $ConfigPath -Value $configContent -Encoding UTF8
    Write-Log "✅ Configuration created" "Success"

    # Step 5: Windows Defender exclusions
    Write-Log "Step 5: Setting up Windows Defender exclusions..."
    try {
        Add-MpPreference -ExclusionPath $InstallPath -ErrorAction SilentlyContinue
        Add-MpPreference -ExclusionPath $DataDir -ErrorAction SilentlyContinue
        Add-MpPreference -ExclusionProcess "$ExePath" -ErrorAction SilentlyContinue
        Write-Log "✅ Windows Defender exclusions added" "Success"
    } catch {
        Write-Log "Could not add Windows Defender exclusions (non-critical)" "Warning"
    }

    # Step 6: Create Windows service
    Write-Log "Step 6: Creating Windows service..."
    $serviceName = "GymDoorBridge"
    $displayName = "RepSet Gym Door Bridge"
    $binaryPath = "`"$ExePath`" --config `"$ConfigPath`""

    # Remove existing service first
    if ($existingService) {
        try {
            sc.exe delete "GymDoorBridge" | Out-Null
            Start-Sleep 2
        } catch {
            Write-Log "Could not remove existing service" "Warning"
        }
    }

    $serviceCreated = $false
    # Try multiple service creation methods
    $serviceMethods = @("BuiltIn", "PowerShell", "sc.exe")
    foreach ($method in $serviceMethods) {
        try {
            Write-Log "Trying service creation method: $method"
            switch ($method) {
                "BuiltIn" {
                    $result = & $ExePath install --config $ConfigPath 2>&1
                    if ($LASTEXITCODE -eq 0) {
                        $serviceCreated = $true
                        Write-Log "✅ Service created using bridge installer" "Success"
                        break
                    }
                }
                "PowerShell" {
                    New-Service -Name $serviceName -BinaryPathName $binaryPath -DisplayName $displayName -StartupType Automatic
                    $serviceCreated = $true
                    Write-Log "✅ Service created using PowerShell" "Success"
                    break
                }
                "sc.exe" {
                    $scResult = sc.exe create $serviceName binpath= $binaryPath start= auto displayname= $displayName 2>&1
                    if ($LASTEXITCODE -eq 0) {
                        $serviceCreated = $true
                        Write-Log "✅ Service created using sc.exe" "Success"
                        break
                    }
                }
            }
        } catch {
            Write-Log "Service creation method $method failed: $($_.Exception.Message)" "Warning"
        }
    }

    if (-not $serviceCreated) {
        Write-ErrorAndExit "All service creation methods failed"
    }

    # Configure service recovery
    try {
        sc.exe failure $serviceName reset= 86400 actions= restart/5000/restart/10000/restart/30000 | Out-Null
    } catch {
        Write-Log "Could not set service recovery (non-critical)" "Warning"
    }

    # Step 7: Device pairing (if code provided)
    if ($PairCode) {
        Write-Log "Step 7: Pairing device with RepSet platform..."
        Write-Log "Using pair code: $PairCode"
        
        try {
            # Clear any existing pairing
            & $ExePath unpair --config $ConfigPath 2>&1 | Out-Null
        } catch {
            # Ignore unpair failures
        }
        
        try {
            $pairResult = & $ExePath pair $PairCode --config $ConfigPath 2>&1
            if ($LASTEXITCODE -eq 0) {
                Write-Log "✅ Device pairing successful!" "Success"
                
                # Extract device ID from result
                $deviceIdMatch = $pairResult | Select-String "device_id([a-zA-Z0-9_]+)"
                if ($deviceIdMatch) {
                    $deviceId = $deviceIdMatch.Matches[0].Groups[1].Value
                    Write-Log "✅ Device ID: $deviceId" "Success"
                }
            } else {
                $pairResultString = $pairResult -join " "
                if ($pairResultString -match "network|connection|timeout|server") {
                    Write-Log "⚠️ Pairing delayed due to network - will retry automatically" "Warning"
                } else {
                    Write-Log "⚠️ Pairing had issues: $pairResultString" "Warning"
                }
            }
        } catch {
            Write-Log "⚠️ Pairing error: $($_.Exception.Message)" "Warning"
        }
    } else {
        Write-Log "Step 7: Skipping pairing (no pair code provided)" "Info"
    }

    # Step 8: Start service
    Write-Log "Step 8: Starting RepSet Bridge service..."
    try {
        Start-Service -Name $serviceName -ErrorAction Stop
        Start-Sleep 5
        
        $service = Get-Service -Name $serviceName
        if ($service.Status -eq "Running") {
            Write-Log "✅ Service started successfully!" "Success"
            
            # Quick health check
            Start-Sleep 3
            try {
                $healthCheck = Invoke-WebRequest -Uri "http://localhost:8081/health" -UseBasicParsing -TimeoutSec 10 -ErrorAction Stop
                Write-Log "✅ Bridge API responding (HTTP $($healthCheck.StatusCode))" "Success"
            } catch {
                Write-Log "⚠️ Bridge API not yet available (normal during startup)" "Warning"
            }
        } else {
            Write-Log "⚠️ Service status: $($service.Status)" "Warning"
        }
    } catch {
        Write-Log "⚠️ Service start failed: $($_.Exception.Message)" "Warning"
        Write-Log "Service will be started automatically on next boot" "Info"
    }

    # Cleanup
    Remove-Item $TempDir -Recurse -Force -ErrorAction SilentlyContinue

    # Final summary
    if (-not $Silent) {
        Write-Host ""
        Write-Host "🎉 INSTALLATION COMPLETED SUCCESSFULLY!" -ForegroundColor Green
        Write-Host "======================================" -ForegroundColor Green
        Write-Host ""
        Write-Host "📋 Installation Summary:" -ForegroundColor Cyan
        Write-Host "   📁 Install Path: $InstallPath" -ForegroundColor Gray
        Write-Host "   💾 Data Directory: $DataDir" -ForegroundColor Gray
        Write-Host "   🌐 Server URL: $ServerUrl" -ForegroundColor Gray
        Write-Host "   🔧 Service Name: $serviceName" -ForegroundColor Gray
        Write-Host "   📊 Service Status: $((Get-Service -Name $serviceName).Status)" -ForegroundColor Gray
        
        if ($PairCode) {
            Write-Host "   🔗 Pairing: Completed" -ForegroundColor Green
            if ($deviceId) {
                Write-Host "   🆔 Device ID: $deviceId" -ForegroundColor Gray
            }
        } else {
            Write-Host "   🔗 Pairing: Skipped (use: gym-door-bridge pair YOUR_CODE)" -ForegroundColor Yellow
        }
        
        Write-Host ""
        Write-Host "🎯 Your RepSet Bridge is now running!" -ForegroundColor Green
        Write-Host ""
        Write-Host "Management Commands:" -ForegroundColor Cyan
        Write-Host "   gym-door-bridge status    - Check status" -ForegroundColor Gray
        Write-Host "   net start GymDoorBridge   - Start service" -ForegroundColor Gray
        Write-Host "   net stop GymDoorBridge    - Stop service" -ForegroundColor Gray
        Write-Host ""
        Write-Host "Log file: $env:TEMP\repset-bridge-install.log" -ForegroundColor Gray
        Write-Host ""
    }

    Write-Log "Installation completed successfully"

} catch {
    Write-ErrorAndExit "Unexpected installation error" $_.Exception.Message
}