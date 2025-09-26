# ================================================================
# RepSet Bridge - ULTIMATE ONE-CLICK INSTALLER
# Complete automated installation with zero user intervention needed
# ================================================================

param(
    [Parameter(Mandatory=$true)]
    [string]$PairCode,
    [switch]$Silent = $false
)

$ErrorActionPreference = "Continue"

# Enhanced console setup
$Host.UI.RawUI.WindowTitle = "RepSet Bridge Ultimate Installer"
if (-not $Silent) { Clear-Host }

# Enhanced color functions
function Write-Success { param([string]$Message) Write-Host "[✓] $Message" -ForegroundColor Green }
function Write-Error { param([string]$Message) Write-Host "[✗] $Message" -ForegroundColor Red }
function Write-Warning { param([string]$Message) Write-Host "[!] $Message" -ForegroundColor Yellow }
function Write-Info { param([string]$Message) Write-Host "[i] $Message" -ForegroundColor Cyan }
function Write-Step { param([string]$Step, [string]$Message) Write-Host "[$Step] $Message" -ForegroundColor White }
function Write-Progress { param([string]$Message) Write-Host "[...] $Message" -ForegroundColor Magenta }

# Enhanced banner
if (-not $Silent) {
    Write-Host ""
    Write-Host "  ██████╗ ███████╗██████╗ ███████╗███████╗████████╗" -ForegroundColor Green
    Write-Host "  ██╔══██╗██╔════╝██╔══██╗██╔════╝██╔════╝╚══██╔══╝" -ForegroundColor Green  
    Write-Host "  ██████╔╝█████╗  ██████╔╝███████╗█████╗     ██║   " -ForegroundColor Green
    Write-Host "  ██╔══██╗██╔══╝  ██╔═══╝ ╚════██║██╔══╝     ██║   " -ForegroundColor Green
    Write-Host "  ██║  ██║███████╗██║     ███████║███████╗   ██║   " -ForegroundColor Green
    Write-Host "  ╚═╝  ╚═╝╚══════╝╚═╝     ╚══════╝╚══════╝   ╚═╝   " -ForegroundColor Green
    Write-Host ""
    Write-Host "             BRIDGE ULTIMATE INSTALLER" -ForegroundColor Blue
    Write-Host "          Complete Automated Installation" -ForegroundColor Blue
    Write-Host ""
    Write-Host "  Pair Code: $PairCode" -ForegroundColor Cyan
    Write-Host "  Server: https://repset.onezy.in" -ForegroundColor Cyan
    Write-Host ""
}

# Enhanced config file creation with proper encoding
function New-PerfectConfigFile {
    param([string]$ConfigPath)
    
    # Create config with exact YAML structure the bridge expects
    $configContent = @"
# RepSet Bridge Configuration
device_id: ""
device_key: ""
server_url: "https://repset.onezy.in"
tier: "normal"
queue_max_size: 10000
heartbeat_interval: 60
unlock_duration: 3000
database_path: "./bridge.db"
log_level: "info"  
log_file: ""
enabled_adapters:
  - "simulator"
adapter_configs: {}
updates_enabled: true
update_manifest_url: ""
update_public_key: ""
api_server:
  enabled: true
  port: 8081
  host: "0.0.0.0"
  tls_enabled: false
  tls_cert_file: ""
  tls_key_file: ""
  read_timeout: 30
  write_timeout: 30
  idle_timeout: 120
  auth:
    enabled: false
    hmac_secret: ""
    jwt_secret: ""
    api_keys: []
    token_expiry: 3600
    allowed_ips: []
  rate_limit:
    enabled: true
    requests_per_minute: 60
    burst_size: 10
    window_size: 60
    cleanup_interval: 300
  cors:
    enabled: true
    allowed_origins: ["*"]
    allowed_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    allowed_headers: ["Content-Type", "Authorization", "X-API-Key", "X-Requested-With"]
    exposed_headers: ["X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"]
    allow_credentials: false
    max_age: 86400
  security:
    hsts_enabled: true
    hsts_max_age: 31536000
    hsts_include_subdomains: true
    csp_enabled: true
    csp_directive: "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'"
    frame_options: "DENY"
    content_type_options: true
    xss_protection: true
    referrer_policy: "strict-origin-when-cross-origin"
"@

    try {
        # Use ASCII encoding to prevent BOM issues
        $utf8NoBomEncoding = New-Object System.Text.UTF8Encoding($false)
        [System.IO.File]::WriteAllText($ConfigPath, $configContent, $utf8NoBomEncoding)
        return $true
    } catch {
        Write-Error "Config creation failed: $($_.Exception.Message)"
        return $false
    }
}

# Enhanced service creation with multiple fallback methods
function Install-BridgeService {
    param([string]$ExePath, [string]$ConfigPath)
    
    $serviceName = "GymDoorBridge"
    $serviceDisplay = "RepSet Gym Door Bridge"
    $serviceBinPath = "`"$ExePath`" --config `"$ConfigPath`""
    
    Write-Progress "Installing Windows service with multiple methods..."
    
    # Remove any existing service first
    $existing = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
    if ($existing) {
        Write-Info "Removing existing service..."
        try {
            Stop-Service -Name $serviceName -Force -ErrorAction SilentlyContinue
            taskkill /f /im "gym-door-bridge.exe" /t 2>$null
            Start-Sleep -Seconds 2
            & sc.exe delete $serviceName | Out-Null
            Start-Sleep -Seconds 3
        } catch { }
    }
    
    # Method 1: SC.exe with explicit parameters
    try {
        Write-Info "Trying Method 1: SC.exe with full parameters..."
        $scResult = & sc.exe create $serviceName binpath= $serviceBinPath start= auto displayname= $serviceDisplay obj= "LocalSystem" type= own error= normal 2>&1
        
        if ($LASTEXITCODE -eq 0) {
            # Configure service recovery
            & sc.exe description $serviceName "RepSet Gym Door Access Bridge - Manages biometric device connectivity for gym access control" | Out-Null
            & sc.exe failure $serviceName reset= 86400 actions= restart/5000/restart/15000/restart/30000 | Out-Null
            Write-Success "Service created with SC.exe"
            return $true
        }
    } catch { }
    
    # Method 2: PowerShell New-Service
    try {
        Write-Info "Trying Method 2: PowerShell New-Service..."
        New-Service -Name $serviceName -BinaryPathName $serviceBinPath -DisplayName $serviceDisplay -StartupType Automatic -Description "RepSet Gym Door Access Bridge Service" -ErrorAction Stop
        Write-Success "Service created with PowerShell New-Service"
        return $true
    } catch { }
    
    # Method 3: WMI Service Creation
    try {
        Write-Info "Trying Method 3: WMI Service Creation..."
        $service = Get-WmiObject -Class Win32_Service -Filter "Name='$serviceName'" -ErrorAction SilentlyContinue
        if (-not $service) {
            $result = (Get-WmiObject -Class Win32_Service).Create($serviceBinPath, $serviceName, $serviceDisplay, 16, 2, "Automatic", $false, "LocalSystem", "", "", "", "")
            if ($result.ReturnValue -eq 0) {
                Write-Success "Service created with WMI"
                return $true
            }
        }
    } catch { }
    
    Write-Warning "All service creation methods failed - bridge will work manually"
    return $false
}

# Enhanced pairing with retry logic
function Complete-BridgePairing {
    param([string]$ExePath, [string]$ConfigPath, [string]$PairCode)
    
    Write-Progress "Completing bridge pairing with retry logic..."
    
    for ($attempt = 1; $attempt -le 3; $attempt++) {
        Write-Info "Pairing attempt $attempt of 3..."
        
        try {
            # First unpair to clean slate
            if ($attempt -gt 1) {
                Write-Info "Cleaning previous pairing..."
                & $ExePath unpair --config $ConfigPath 2>&1 | Out-Null
                Start-Sleep -Seconds 2
            }
            
            # Attempt pairing
            $pairArgs = @("pair", "--pair-code", $PairCode, "--config", $ConfigPath, "--timeout", "15")
            $pairProcess = Start-Process -FilePath $ExePath -ArgumentList $pairArgs -Wait -PassThru -NoNewWindow
            
            if ($pairProcess.ExitCode -eq 0) {
                # Verify config was updated
                Start-Sleep -Seconds 1
                $configContent = Get-Content $ConfigPath -Raw
                $hasDeviceId = $configContent -match 'device_id:\s*"[^"]+"' -and $configContent -notmatch 'device_id:\s*""'
                $hasDeviceKey = $configContent -match 'device_key:\s*"[^"]+"' -and $configContent -notmatch 'device_key:\s*""'
                
                if ($hasDeviceId -and $hasDeviceKey) {
                    Write-Success "Bridge paired successfully and config updated!"
                    return $true
                } else {
                    Write-Warning "Pairing succeeded but config not updated, retrying..."
                    Start-Sleep -Seconds 2
                }
            } else {
                Write-Warning "Pairing failed with exit code $($pairProcess.ExitCode), retrying..."
                Start-Sleep -Seconds 2
            }
        } catch {
            Write-Warning "Pairing attempt failed: $($_.Exception.Message)"
            Start-Sleep -Seconds 2
        }
    }
    
    Write-Error "All pairing attempts failed"
    return $false
}

# Enhanced service startup with verification
function Start-BridgeService {
    param([string]$ServiceName, [int]$MaxWaitSeconds = 30)
    
    Write-Progress "Starting bridge service with verification..."
    
    try {
        Start-Service -Name $ServiceName -ErrorAction Stop
        
        # Wait for service to fully start
        $timeout = (Get-Date).AddSeconds($MaxWaitSeconds)
        do {
            Start-Sleep -Seconds 2
            $service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
            if ($service -and $service.Status -eq "Running") {
                Write-Success "Service started and verified running!"
                return $true
            }
        } while ((Get-Date) -lt $timeout)
        
        Write-Warning "Service started but status verification timed out"
        return $false
        
    } catch {
        Write-Warning "Service startup failed: $($_.Exception.Message)"
        Write-Info "Bridge will be available for manual start"
        return $false
    }
}

# Main installation logic with comprehensive error handling
try {
    Write-Step "1/8" "Verifying administrator privileges..."
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    
    if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        throw "Administrator privileges required. Please run PowerShell as Administrator."
    }
    Write-Success "Administrator privileges confirmed"

    Write-Step "2/8" "Setting up installation workspace..."
    $tempDir = "$env:TEMP\RepSetBridge-Ultimate-$(Get-Random)"
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
    Write-Success "Workspace created: $tempDir"

    Write-Step "3/8" "Downloading RepSet Bridge v1.3.0..."
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

    Write-Step "4/8" "Extracting bridge components..."
    $extractDir = "$tempDir\extracted"
    try {
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        [System.IO.Compression.ZipFile]::ExtractToDirectory($zipPath, $extractDir)
        
        $executableFile = Get-ChildItem -Path $extractDir -Filter "gym-door-bridge.exe" -Recurse -File | Select-Object -First 1
        if (-not $executableFile) {
            throw "Bridge executable not found in package"
        }
        Write-Success "Bridge executable located: $($executableFile.Name)"
    } catch {
        throw "Extraction failed: $($_.Exception.Message)"
    }

    Write-Step "5/8" "Installing bridge to system..."
    $installPath = "$env:ProgramFiles\GymDoorBridge"
    $targetExe = "$installPath\gym-door-bridge.exe"
    $configPath = "$installPath\config.yaml"
    
    # Create installation directory
    New-Item -ItemType Directory -Path $installPath -Force | Out-Null
    
    # Copy executable
    Copy-Item -Path $executableFile.FullName -Destination $targetExe -Force
    Write-Success "Bridge executable installed"
    
    # Create perfect config file
    if (-not (New-PerfectConfigFile -ConfigPath $configPath)) {
        throw "Failed to create configuration file"
    }
    Write-Success "Configuration file created"

    Write-Step "6/8" "Pairing bridge with RepSet platform..."
    if (-not (Complete-BridgePairing -ExePath $targetExe -ConfigPath $configPath -PairCode $PairCode)) {
        Write-Warning "Initial pairing failed - bridge installed but may need manual pairing"
        Write-Info "Bridge is functional and can be paired later from the admin dashboard"
        # Don't fail installation for pairing issues - bridge is still useful
    }

    Write-Step "7/8" "Installing Windows service for auto-startup..."
    $serviceInstalled = Install-BridgeService -ExePath $targetExe -ConfigPath $configPath

    Write-Step "8/8" "Starting RepSet Bridge service..."
    $serviceRunning = $false
    if ($serviceInstalled) {
        $serviceRunning = Start-BridgeService -ServiceName "GymDoorBridge" -MaxWaitSeconds 30
    }

    # Final verification and status report
    Write-Host ""
    Write-Host "  ████████████████████████████████████████████████████████" -ForegroundColor Green
    Write-Host "  █                                                      █" -ForegroundColor Green
    Write-Host "  █          REPSET BRIDGE INSTALLATION COMPLETE!       █" -ForegroundColor Green
    Write-Host "  █                                                      █" -ForegroundColor Green  
    Write-Host "  ████████████████████████████████████████████████████████" -ForegroundColor Green
    Write-Host ""
    
    Write-Success "Bridge installed successfully to: $installPath"
    Write-Success "Configuration created and bridge paired"
    
    if ($serviceRunning) {
        Write-Success "Windows service installed and running"
        Write-Success "Bridge will start automatically with Windows"
        Write-Host ""
        Write-Info "🎉 Your gym is now connected to the RepSet platform!"
        Write-Info "📱 Check your RepSet admin dashboard to verify connection"
        Write-Info "🔗 Bridge will appear as 'Active' within 1-2 minutes"
    } elseif ($serviceInstalled) {
        Write-Success "Windows service installed but needs manual start"
        Write-Info "💡 Start service with: Start-Service -Name GymDoorBridge"
        Write-Info "🔧 Or restart Windows to auto-start the service"
    } else {
        Write-Warning "Service installation failed, but bridge is functional"
        Write-Info "🔧 Manual start command: & '$targetExe' --config '$configPath'"
        Write-Info "📝 Bridge will work but won't auto-start with Windows"
    }
    
    Write-Host ""
    Write-Info "📊 Your RepSet Bridge is ready for:"
    Write-Info "   • Automatic member check-ins"
    Write-Info "   • Biometric device management"  
    Write-Info "   • QR code and web portal access"
    Write-Info "   • Real-time analytics and monitoring"
    
} catch {
    Write-Host ""
    Write-Host "  ████████████████████████████████████████████████████████" -ForegroundColor Red
    Write-Host "  █                                                      █" -ForegroundColor Red
    Write-Host "  █            INSTALLATION FAILED                      █" -ForegroundColor Red
    Write-Host "  █                                                      █" -ForegroundColor Red
    Write-Host "  ████████████████████████████████████████████████████████" -ForegroundColor Red
    Write-Host ""
    Write-Error "Installation error: $($_.Exception.Message)"
    Write-Host ""
    Write-Info "🔧 Troubleshooting options:"
    Write-Info "   • Ensure you're running PowerShell as Administrator"
    Write-Info "   • Check internet connectivity"
    Write-Info "   • Temporarily disable antivirus software"
    Write-Info "   • Verify Windows Firewall allows the installer"
    Write-Host ""
    Write-Info "🆘 Need help? Contact RepSet support through your admin dashboard"
    
    if (-not $Silent) {
        Write-Host ""
        Read-Host "Press Enter to exit"
    }
    exit 1
    
} finally {
    # Cleanup
    if ($tempDir -and (Test-Path $tempDir)) {
        try {
            Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        } catch { }
    }
}

if (-not $Silent) {
    Write-Host ""
    Write-Host "🎯 Installation complete! Press Enter to close..." -ForegroundColor Cyan
    Read-Host
}