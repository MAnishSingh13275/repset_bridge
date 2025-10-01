# RepSet Bridge Simple Installer - GitHub Raw Version
# Direct download and install from GitHub release v1.4.0
# Comprehensive installer with full error handling and edge case coverage

param(
    [Parameter(Mandatory=$true)]
    [string]$PairCode,
    [switch]$Silent = $false,
    [switch]$Force = $false,
    [string]$InstallPath = "",
    [switch]$SkipPairing = $false
)

# Global configuration
$script:INSTALLER_VERSION = "1.4.0"
$script:BRIDGE_VERSION = "v1.4.0"
$script:GITHUB_RELEASE_URL = "https://github.com/MAnishSingh13275/repset_bridge/releases/download/v1.4.0/gym-door-bridge.exe"
$script:SERVICE_NAME = "GymDoorBridge"
$script:SERVICE_DISPLAY_NAME = "RepSet Gym Door Bridge"
$script:REPSET_SERVER = "https://repset.onezy.in"

# Set error handling
$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

# Enhanced logging and color functions
function Write-Success { 
    param([string]$Message) 
    Write-Host "[✓] $Message" -ForegroundColor Green 
}

function Write-Error { 
    param([string]$Message) 
    Write-Host "[✗] $Message" -ForegroundColor Red 
}

function Write-Warning { 
    param([string]$Message) 
    Write-Host "[!] $Message" -ForegroundColor Yellow 
}

function Write-Info { 
    param([string]$Message) 
    Write-Host "[i] $Message" -ForegroundColor Cyan 
}

function Write-Step { 
    param([string]$Step, [string]$Message) 
    Write-Host "[$Step] $Message" -ForegroundColor White 
}

function Write-Debug { 
    param([string]$Message) 
    if ($VerbosePreference -eq "Continue") {
        Write-Host "[DEBUG] $Message" -ForegroundColor DarkGray 
    }
}

# Safe exit function that always waits for user input on error
function Exit-WithMessage {
    param(
        [string]$Message = "",
        [int]$ExitCode = 0,
        [switch]$IsError = $false
    )
    
    if ($Message) {
        if ($IsError) {
            Write-Error $Message
        } else {
            Write-Info $Message
        }
    }
    
    if (-not $Silent -and ($IsError -or $ExitCode -ne 0)) {
        Write-Host ""
        Write-Host "Press any key to exit..." -ForegroundColor Gray
        $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
    }
    
    exit $ExitCode
}

# Validate pair code format
function Test-PairCodeFormat {
    param([string]$Code)
    
    if ([string]::IsNullOrWhiteSpace($Code)) {
        return $false
    }
    
    # Expected format: XXXX-XXXX-XXXX (alphanumeric)
    $pattern = '^[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}$'
    return $Code -match $pattern
}

# Test internet connectivity
function Test-InternetConnection {
    try {
        $response = Invoke-WebRequest -Uri "https://www.google.com" -Method Head -TimeoutSec 5 -UseBasicParsing -ErrorAction Stop
        return $true
    } catch {
        try {
            # Fallback test
            $response = Invoke-WebRequest -Uri "https://github.com" -Method Head -TimeoutSec 5 -UseBasicParsing -ErrorAction Stop
            return $true
        } catch {
            return $false
        }
    }
}

# Get available disk space
function Get-AvailableDiskSpace {
    param([string]$Path)
    
    try {
        $drive = [System.IO.Path]::GetPathRoot($Path)
        $driveInfo = Get-WmiObject -Class Win32_LogicalDisk | Where-Object { $_.DeviceID -eq $drive.TrimEnd('\') }
        return [math]::Round($driveInfo.FreeSpace / 1GB, 2)
    } catch {
        return -1
    }
}

# Validate installation path
function Test-InstallationPath {
    param([string]$Path)
    
    try {
        # Test if we can create the directory
        if (-not (Test-Path $Path)) {
            New-Item -ItemType Directory -Path $Path -Force -ErrorAction Stop | Out-Null
            Remove-Item -Path $Path -Force -ErrorAction Stop
        }
        
        # Test write permissions
        $testFile = Join-Path $Path "test_write_$(Get-Random).tmp"
        "test" | Out-File -FilePath $testFile -ErrorAction Stop
        Remove-Item -Path $testFile -Force -ErrorAction Stop
        
        return $true
    } catch {
        return $false
    }
}

# Clean up any existing installation
function Remove-ExistingInstallation {
    param([string]$InstallDir)
    
    Write-Info "Checking for existing installation..."
    
    # Stop and remove service
    $service = Get-Service -Name $script:SERVICE_NAME -ErrorAction SilentlyContinue
    if ($service) {
        Write-Info "Removing existing service..."
        try {
            if ($service.Status -eq "Running") {
                Stop-Service -Name $script:SERVICE_NAME -Force -TimeoutSec 30 -ErrorAction Stop
                Write-Success "Service stopped"
            }
            
            & sc.exe delete $script:SERVICE_NAME | Out-Null
            if ($LASTEXITCODE -eq 0) {
                Write-Success "Service removed"
            }
            
            # Wait for service to be fully removed
            $timeout = 10
            while ((Get-Service -Name $script:SERVICE_NAME -ErrorAction SilentlyContinue) -and $timeout -gt 0) {
                Start-Sleep -Seconds 1
                $timeout--
            }
        } catch {
            Write-Warning "Could not cleanly remove existing service: $($_.Exception.Message)"
        }
    }
    
    # Remove installation directory if it exists
    if (Test-Path $InstallDir) {
        Write-Info "Removing existing installation directory..."
        try {
            # Kill any running processes that might lock files
            Get-Process | Where-Object { $_.Path -like "$InstallDir*" } | Stop-Process -Force -ErrorAction SilentlyContinue
            
            Start-Sleep -Seconds 2
            Remove-Item -Path $InstallDir -Recurse -Force -ErrorAction Stop
            Write-Success "Previous installation removed"
        } catch {
            Write-Warning "Could not remove existing installation: $($_.Exception.Message)"
            if (-not $Force) {
                throw "Cannot proceed with existing installation present. Use -Force to override."
            }
        }
    }
}

# Download file with progress and retry logic
function Download-FileWithRetry {
    param(
        [string]$Url,
        [string]$OutputPath,
        [int]$MaxRetries = 3,
        [int]$TimeoutSec = 30
    )
    
    for ($attempt = 1; $attempt -le $MaxRetries; $attempt++) {
        try {
            Write-Info "Download attempt $attempt of $MaxRetries..."
            
            # Use WebClient for better progress reporting
            $webClient = New-Object System.Net.WebClient
            $webClient.Headers.Add("User-Agent", "RepSet-Bridge-Installer/$script:INSTALLER_VERSION")
            
            # Download with timeout
            $task = $webClient.DownloadFileTaskAsync($Url, $OutputPath)
            $timeoutTask = [System.Threading.Tasks.Task]::Delay([TimeSpan]::FromSeconds($TimeoutSec))
            
            $completedTask = [System.Threading.Tasks.Task]::WaitAny(@($task, $timeoutTask))
            
            if ($completedTask -eq 0) {
                # Download completed
                $webClient.Dispose()
                
                # Verify file was downloaded and has content
                if ((Test-Path $OutputPath) -and (Get-Item $OutputPath).Length -gt 0) {
                    return $true
                } else {
                    throw "Downloaded file is empty or invalid"
                }
            } else {
                # Timeout occurred
                $webClient.CancelAsync()
                $webClient.Dispose()
                throw "Download timeout after $TimeoutSec seconds"
            }
        } catch {
            Write-Warning "Download attempt $attempt failed: $($_.Exception.Message)"
            
            if ($attempt -lt $MaxRetries) {
                $waitTime = $attempt * 2
                Write-Info "Waiting $waitTime seconds before retry..."
                Start-Sleep -Seconds $waitTime
            } else {
                throw "All download attempts failed. Last error: $($_.Exception.Message)"
            }
        }
    }
    
    return $false
}

# Verify executable integrity
function Test-ExecutableIntegrity {
    param([string]$FilePath)
    
    try {
        # Check file exists and has reasonable size (should be > 10MB for Go binary)
        $fileInfo = Get-Item $FilePath -ErrorAction Stop
        if ($fileInfo.Length -lt (10 * 1024 * 1024)) {
            return $false
        }
        
        # Try to get file version info (basic integrity check)
        $versionInfo = [System.Diagnostics.FileVersionInfo]::GetVersionInfo($FilePath)
        
        # Check if it's a valid PE file
        $bytes = [System.IO.File]::ReadAllBytes($FilePath)
        if ($bytes.Length -lt 64 -or $bytes[0] -ne 0x4D -or $bytes[1] -ne 0x5A) {
            return $false
        }
        
        return $true
    } catch {
        return $false
    }
}

# Create configuration file with validation
function New-ConfigurationFile {
    param(
        [string]$ConfigPath,
        [string]$PairCode
    )
    
    $configContent = @"
# RepSet Gym Door Bridge Configuration
# Generated by installer v$script:INSTALLER_VERSION on $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')

# Device identification (will be populated during pairing)
device_id: ""
device_key: ""

# Server configuration
server_url: "$script:REPSET_SERVER"
tier: "normal"

# Performance settings
queue_max_size: 10000
heartbeat_interval: 60
unlock_duration: 3000

# Storage
database_path: "./bridge.db"

# Logging
log_level: "info"
log_file: ""

# Hardware adapters
enabled_adapters:
  - "simulator"

# Installation metadata
installer_version: "$script:INSTALLER_VERSION"
bridge_version: "$script:BRIDGE_VERSION"
install_date: "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"
pair_code_used: "$PairCode"
"@

    try {
        $configContent | Out-File -FilePath $ConfigPath -Encoding UTF8 -Force -ErrorAction Stop
        
        # Verify file was created correctly
        if (-not (Test-Path $ConfigPath)) {
            throw "Config file was not created"
        }
        
        $createdContent = Get-Content $ConfigPath -Raw -ErrorAction Stop
        if ([string]::IsNullOrWhiteSpace($createdContent)) {
            throw "Config file is empty"
        }
        
        return $true
    } catch {
        throw "Failed to create configuration file: $($_.Exception.Message)"
    }
}

# Install Windows service with comprehensive error handling
function Install-WindowsService {
    param(
        [string]$ExecutablePath,
        [string]$ConfigPath
    )
    
    try {
        # Validate paths exist
        if (-not (Test-Path $ExecutablePath)) {
            throw "Executable not found: $ExecutablePath"
        }
        
        if (-not (Test-Path $ConfigPath)) {
            throw "Config file not found: $ConfigPath"
        }
        
        # Create service
        $binPath = "`"$ExecutablePath`" --config `"$ConfigPath`""
        $createArgs = @(
            "create",
            $script:SERVICE_NAME,
            "binPath=",
            $binPath,
            "start=",
            "auto",
            "DisplayName=",
            $script:SERVICE_DISPLAY_NAME,
            "depend=",
            "Tcpip"
        )
        
        Write-Debug "Creating service with command: sc.exe $($createArgs -join ' ')"
        
        $result = & sc.exe @createArgs 2>&1
        if ($LASTEXITCODE -ne 0) {
            throw "Service creation failed with exit code $LASTEXITCODE`: $result"
        }
        
        # Set service description
        & sc.exe description $script:SERVICE_NAME "RepSet Gym Door Access Bridge - Manages gym door access control integration with RepSet platform" | Out-Null
        
        # Configure service recovery options
        & sc.exe failure $script:SERVICE_NAME reset= 86400 actions= restart/5000/restart/10000/restart/30000 | Out-Null
        
        # Verify service was created
        $service = Get-Service -Name $script:SERVICE_NAME -ErrorAction SilentlyContinue
        if (-not $service) {
            throw "Service was not created successfully"
        }
        
        Write-Success "Windows service installed successfully"
        return $true
        
    } catch {
        throw "Service installation failed: $($_.Exception.Message)"
    }
}

# Pair device with comprehensive error handling and retry logic
function Invoke-DevicePairing {
    param(
        [string]$ExecutablePath,
        [string]$ConfigPath,
        [string]$PairCode,
        [string]$TempDir
    )
    
    if ($SkipPairing) {
        Write-Warning "Pairing skipped as requested"
        return $true
    }
    
    try {
        Write-Info "Initiating device pairing..."
        Write-Info "Pair Code: $PairCode"
        Write-Info "Server: $script:REPSET_SERVER"
        
        # Prepare pairing arguments
        $pairArgs = @(
            "pair",
            "--pair-code", $PairCode,
            "--config", $ConfigPath,
            "--timeout", "15",
            "--verbose"
        )
        
        $outputFile = Join-Path $TempDir "pair_output.txt"
        $errorFile = Join-Path $TempDir "pair_error.txt"
        
        Write-Debug "Pairing command: `"$ExecutablePath`" $($pairArgs -join ' ')"
        
        # Execute pairing with timeout
        $pairProcess = Start-Process -FilePath $ExecutablePath -ArgumentList $pairArgs -Wait -PassThru -NoNewWindow -RedirectStandardOutput $outputFile -RedirectStandardError $errorFile -ErrorAction Stop
        
        # Read output files
        $pairOutput = ""
        $pairError = ""
        
        if (Test-Path $outputFile) {
            $pairOutput = Get-Content $outputFile -Raw -ErrorAction SilentlyContinue
        }
        
        if (Test-Path $errorFile) {
            $pairError = Get-Content $errorFile -Raw -ErrorAction SilentlyContinue
        }
        
        Write-Debug "Pairing exit code: $($pairProcess.ExitCode)"
        Write-Debug "Pairing output: $pairOutput"
        Write-Debug "Pairing error: $pairError"
        
        if ($pairProcess.ExitCode -eq 0) {
            Write-Success "Device paired successfully!"
            return $true
        } else {
            # Check if device is already paired
            Write-Info "Checking current pairing status..."
            
            $statusArgs = @("status", "--config", $ConfigPath)
            $statusOutputFile = Join-Path $TempDir "status_output.txt"
            $statusErrorFile = Join-Path $TempDir "status_error.txt"
            
            $statusProcess = Start-Process -FilePath $ExecutablePath -ArgumentList $statusArgs -Wait -PassThru -NoNewWindow -RedirectStandardOutput $statusOutputFile -RedirectStandardError $statusErrorFile -ErrorAction SilentlyContinue
            
            if ($statusProcess.ExitCode -eq 0 -and (Test-Path $statusOutputFile)) {
                $statusOutput = Get-Content $statusOutputFile -Raw -ErrorAction SilentlyContinue
                if ($statusOutput -and $statusOutput -notmatch "not paired|unpaired") {
                    Write-Success "Device is already paired with RepSet!"
                    return $true
                }
            }
            
            # Pairing failed
            $errorMessage = "Pairing failed with exit code $($pairProcess.ExitCode)"
            if ($pairError) {
                $errorMessage += ": $pairError"
            }
            
            Write-Warning $errorMessage
            Write-Info "You can pair the device manually later using the RepSet dashboard"
            return $false
        }
        
    } catch {
        Write-Warning "Pairing process error: $($_.Exception.Message)"
        Write-Info "You can pair the device manually later using the RepSet dashboard"
        return $false
    }
}

# Start service with retry logic
function Start-BridgeService {
    try {
        Write-Info "Starting RepSet Bridge service..."
        
        # Start service with timeout
        $service = Get-Service -Name $script:SERVICE_NAME -ErrorAction Stop
        
        if ($service.Status -eq "Running") {
            Write-Success "Service is already running"
            return $true
        }
        
        Start-Service -Name $script:SERVICE_NAME -ErrorAction Stop
        
        # Wait for service to start with timeout
        $timeout = 30
        $service.WaitForStatus("Running", [TimeSpan]::FromSeconds($timeout))
        
        Write-Success "Service started successfully"
        return $true
        
    } catch {
        Write-Warning "Service could not be started automatically: $($_.Exception.Message)"
        Write-Info "The service is installed and will start automatically on next boot"
        Write-Info "You can start it manually from Services.msc or by restarting Windows"
        return $false
    }
}

# Display installation summary
function Show-InstallationSummary {
    param(
        [string]$InstallDir,
        [string]$PairCode,
        [bool]$ServiceStarted,
        [bool]$PairingSuccessful
    )
    
    Write-Host ""
    Write-Host "=" * 50 -ForegroundColor Green
    Write-Host "INSTALLATION COMPLETED SUCCESSFULLY" -ForegroundColor Green
    Write-Host "=" * 50 -ForegroundColor Green
    Write-Host ""
    
    Write-Success "RepSet Gym Door Bridge v$script:BRIDGE_VERSION installed"
    Write-Success "Installation directory: $InstallDir"
    Write-Success "Windows service configured for automatic startup"
    
    if ($PairingSuccessful) {
        Write-Success "Device paired successfully with code: $($PairCode.Substring(0,4))****"
    } else {
        Write-Warning "Device pairing needs to be completed manually"
    }
    
    if ($ServiceStarted) {
        Write-Success "Service is running and ready"
    } else {
        Write-Warning "Service needs to be started manually"
    }
    
    Write-Host ""
    Write-Info "Next steps:"
    if (-not $PairingSuccessful) {
        Write-Info "1. Complete device pairing in the RepSet dashboard"
    }
    if (-not $ServiceStarted) {
        Write-Info "2. Start the service from Services.msc or restart Windows"
    }
    Write-Info "3. Test door access from the RepSet platform"
    
    Write-Host ""
    Write-Success "RepSet Gym Door Bridge is ready for gym door access management!"
    Write-Host ""
}

# Initialize variables
$TempDir = $null
$InstallDir = $null
$PairingSuccessful = $false
$ServiceStarted = $false

# Display header
if (-not $Silent) {
    Clear-Host
    Write-Host ""
    Write-Host "RepSet Bridge Installer v$script:INSTALLER_VERSION" -ForegroundColor Cyan
    Write-Host "=" * 50 -ForegroundColor Cyan
    Write-Host "Pair Code: $PairCode" -ForegroundColor Gray
    Write-Host "Platform: $script:REPSET_SERVER" -ForegroundColor Gray
    Write-Host "Bridge Version: $script:BRIDGE_VERSION" -ForegroundColor Gray
    Write-Host "PowerShell Version: $($PSVersionTable.PSVersion)" -ForegroundColor Gray
    Write-Host "Execution Policy: $(Get-ExecutionPolicy)" -ForegroundColor Gray
    Write-Host "Date: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Gray
    Write-Host ""
}

try {
    # Step 1: Pre-flight checks
    Write-Step "1/8" "Running pre-flight checks"
    
    # Check administrator privileges
    Write-Info "Checking administrator privileges..."
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    
    if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        Write-Host ""
        Write-Host "=" * 60 -ForegroundColor Red
        Write-Host "ADMINISTRATOR PRIVILEGES REQUIRED" -ForegroundColor Red
        Write-Host "=" * 60 -ForegroundColor Red
        Write-Error "This installer must be run as Administrator!"
        Write-Host ""
        Write-Info "To fix this:"
        Write-Info "1. Right-click on PowerShell and select 'Run as Administrator'"
        Write-Info "2. Run the command again:"
        Write-Host ""
        Write-Host "Invoke-Expression `"& {`$(Invoke-RestMethod 'https://raw.githubusercontent.com/MAnishSingh13275/repset_bridge/main/simple-repset-installer.ps1')} -PairCode '$PairCode'`"" -ForegroundColor Yellow
        Write-Host ""
        Exit-WithMessage -ExitCode 1 -IsError
    }
    Write-Success "Administrator privileges confirmed"
    
    # Validate pair code format
    Write-Info "Validating pair code format..."
    if (-not (Test-PairCodeFormat -Code $PairCode)) {
        Exit-WithMessage -Message "Invalid pair code format. Expected format: XXXX-XXXX-XXXX (e.g., A1B2-C3D4-E5F6)" -ExitCode 1 -IsError
    }
    Write-Success "Pair code format is valid"
    
    # Test internet connectivity
    Write-Info "Testing internet connectivity..."
    if (-not (Test-InternetConnection)) {
        Exit-WithMessage -Message "No internet connection detected. Please check your network connection and try again." -ExitCode 1 -IsError
    }
    Write-Success "Internet connectivity confirmed"
    
    # Check PowerShell version
    Write-Info "Checking PowerShell version..."
    if ($PSVersionTable.PSVersion.Major -lt 3) {
        Exit-WithMessage -Message "PowerShell 3.0 or higher is required. Current version: $($PSVersionTable.PSVersion)" -ExitCode 1 -IsError
    }
    Write-Success "PowerShell version is compatible"
    
    # Step 2: Set up workspace
    Write-Step "2/8" "Setting up workspace"
    
    $TempDir = Join-Path ([System.IO.Path]::GetTempPath()) "RepSetBridge-$(Get-Random)"
    New-Item -ItemType Directory -Path $TempDir -Force | Out-Null
    Write-Success "Temporary workspace created: $TempDir"
    
    # Determine installation directory
    if ([string]::IsNullOrWhiteSpace($InstallPath)) {
        $ProgramFilesPath = ${env:ProgramFiles}
        if (-not $ProgramFilesPath -or $ProgramFilesPath.Trim() -eq "") {
            $ProgramFilesPath = "C:\Program Files"
        }
        $InstallDir = Join-Path $ProgramFilesPath "GymDoorBridge"
    } else {
        $InstallDir = $InstallPath
    }
    
    Write-Info "Installation directory: $InstallDir"
    
    # Check available disk space
    $availableSpace = Get-AvailableDiskSpace -Path $InstallDir
    if ($availableSpace -ne -1 -and $availableSpace -lt 0.1) {
        Exit-WithMessage -Message "Insufficient disk space. At least 100MB required, only $($availableSpace)GB available." -ExitCode 1 -IsError
    }
    Write-Success "Sufficient disk space available"
    
    # Test installation path
    Write-Info "Validating installation path..."
    if (-not (Test-InstallationPath -Path $InstallDir)) {
        Exit-WithMessage -Message "Cannot write to installation directory: $InstallDir. Please check permissions or choose a different path." -ExitCode 1 -IsError
    }
    Write-Success "Installation path is valid"
    
    # Step 3: Download RepSet Bridge
    Write-Step "3/8" "Downloading RepSet Bridge $script:BRIDGE_VERSION"
    
    $exePath = Join-Path $TempDir "gym-door-bridge.exe"
    Write-Info "Downloading from: $script:GITHUB_RELEASE_URL"
    
    if (-not (Download-FileWithRetry -Url $script:GITHUB_RELEASE_URL -OutputPath $exePath -MaxRetries 3 -TimeoutSec 60)) {
        Exit-WithMessage -Message "Failed to download RepSet Bridge after multiple attempts. Please check your internet connection and try again." -ExitCode 1 -IsError
    }
    
    $exeInfo = Get-Item $exePath
    $sizeMB = [math]::Round($exeInfo.Length / 1MB, 1)
    Write-Success "Downloaded RepSet Bridge $script:BRIDGE_VERSION ($sizeMB MB)"
    
    # Step 4: Verify download integrity
    Write-Step "4/8" "Verifying download integrity"
    
    Write-Info "Checking executable integrity..."
    if (-not (Test-ExecutableIntegrity -FilePath $exePath)) {
        Exit-WithMessage -Message "Downloaded file failed integrity check. The file may be corrupted." -ExitCode 1 -IsError
    }
    Write-Success "Download integrity verified"
    
    # Step 5: Clean up existing installation
    Write-Step "5/8" "Preparing installation environment"
    
    Remove-ExistingInstallation -InstallDir $InstallDir
    Write-Success "Installation environment prepared"
    
    # Step 6: Install RepSet Bridge
    Write-Step "6/8" "Installing RepSet Bridge"
    
    # Create installation directory
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Write-Info "Created installation directory"
    
    # Copy executable
    $TargetExe = Join-Path $InstallDir "gym-door-bridge.exe"
    Copy-Item -Path $exePath -Destination $TargetExe -Force
    Write-Info "Executable installed"
    
    # Create configuration file
    $ConfigFile = Join-Path $InstallDir "config.yaml"
    if (-not (New-ConfigurationFile -ConfigPath $ConfigFile -PairCode $PairCode)) {
        Exit-WithMessage -Message "Failed to create configuration file" -ExitCode 1 -IsError
    }
    Write-Info "Configuration file created"
    
    # Set appropriate permissions
    try {
        $acl = Get-Acl $InstallDir
        $accessRule = New-Object System.Security.AccessControl.FileSystemAccessRule("SYSTEM", "FullControl", "ContainerInherit,ObjectInherit", "None", "Allow")
        $acl.SetAccessRule($accessRule)
        Set-Acl -Path $InstallDir -AclObject $acl
        Write-Info "Permissions configured"
    } catch {
        Write-Warning "Could not set optimal permissions: $($_.Exception.Message)"
    }
    
    Write-Success "RepSet Bridge installed successfully"
    
    # Step 7: Install Windows service
    Write-Step "7/8" "Installing Windows service"
    
    if (-not (Install-WindowsService -ExecutablePath $TargetExe -ConfigPath $ConfigFile)) {
        Exit-WithMessage -Message "Failed to install Windows service" -ExitCode 1 -IsError
    }
    
    # Step 8: Pair device and start service
    Write-Step "8/8" "Finalizing installation"
    
    # Pair the device
    Write-Info "Pairing device with RepSet platform..."
    $PairingSuccessful = Invoke-DevicePairing -ExecutablePath $TargetExe -ConfigPath $ConfigFile -PairCode $PairCode -TempDir $TempDir
    
    # Start the service
    Write-Info "Starting RepSet Bridge service..."
    $ServiceStarted = Start-BridgeService
    
    # Display installation summary
    Show-InstallationSummary -InstallDir $InstallDir -PairCode $PairCode -ServiceStarted $ServiceStarted -PairingSuccessful $PairingSuccessful

} catch {
    Write-Host ""
    Write-Host "=" * 60 -ForegroundColor Red
    Write-Host "INSTALLATION FAILED" -ForegroundColor Red
    Write-Host "=" * 60 -ForegroundColor Red
    Write-Error "Installation error: $($_.Exception.Message)"
    Write-Host ""
    
    Write-Info "Troubleshooting steps:"
    Write-Info "1. Ensure PowerShell is running as Administrator"
    Write-Info "2. Check internet connectivity"
    Write-Info "3. Temporarily disable antivirus software"
    Write-Info "4. Verify the pair code is correct and not expired"
    Write-Info "5. Check Windows Event Viewer for additional error details"
    Write-Host ""
    
    Write-Info "For support, contact RepSet support with this error message"
    Write-Host ""
    
    Exit-WithMessage -ExitCode 1 -IsError
    
} finally {
    # Cleanup temp directory
    if ($TempDir -and (Test-Path $TempDir)) {
        try {
            Remove-Item $TempDir -Recurse -Force -ErrorAction SilentlyContinue
            Write-Debug "Cleaned up temporary directory: $TempDir"
        } catch {
            Write-Debug "Could not clean up temporary directory: $($_.Exception.Message)"
        }
    }
}

# Final success message
if (-not $Silent) {
    Write-Host ""
    Write-Host "Installation completed. Press any key to exit..." -ForegroundColor Gray
    $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
}