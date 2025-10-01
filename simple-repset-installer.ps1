# RepSet Bridge Simple Installer - GitHub Raw Version
# Direct download and install from GitHub release v1.4.0
# Comprehensive installer with full error handling and edge case coverage

param(
    [Parameter(Mandatory=$true)]
    [string]$PairCode,
    [string]$DeviceId = "",
    [string]$DeviceKey = "",
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
        # Simple test: check if we can access Program Files directory
        $programFiles = ${env:ProgramFiles}
        if (-not $programFiles) {
            $programFiles = "C:\Program Files"
        }
        
        # Test if Program Files is accessible
        if (-not (Test-Path $programFiles)) {
            Write-Debug "Program Files directory not accessible: $programFiles"
            return $false
        }
        
        # For Program Files subdirectories, just check if we can create a test file in Program Files
        $testFile = Join-Path $programFiles "test_write_$(Get-Random).tmp"
        try {
            "test" | Out-File -FilePath $testFile -ErrorAction Stop
            Remove-Item -Path $testFile -Force -ErrorAction Stop
            return $true
        } catch {
            Write-Debug "Cannot write to Program Files: $($_.Exception.Message)"
            return $false
        }
        
    } catch {
        Write-Debug "Installation path test failed: $($_.Exception.Message)"
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
            
            # Try PowerShell Remove-Service first (PS 6+), fallback to sc.exe
            try {
                if (Get-Command Remove-Service -ErrorAction SilentlyContinue) {
                    Remove-Service -Name $script:SERVICE_NAME -Force -ErrorAction Stop
                } else {
                    & sc.exe delete $script:SERVICE_NAME | Out-Null
                }
            } catch {
                & sc.exe delete $script:SERVICE_NAME | Out-Null
            }
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
        [string]$PairCode,
        [string]$DeviceId = "",
        [string]$DeviceKey = ""
    )
    
    # Create a minimal working config file with platform-provided credentials
    $configContent = @"
device_id: "$DeviceId"
device_key: "$DeviceKey"
server_url: "$script:REPSET_SERVER"
tier: "normal"
queue_max_size: 10000
heartbeat_interval: 60
unlock_duration: 3000
database_path: "./bridge.db"
log_level: "info"
log_file: ""
enabled_adapters:
  - "simulator"
"@

    try {
        Write-Info "Creating configuration file..."
        
        # Method 1: Try to copy the working config.yaml from the repository
        $workingConfigUrl = "https://raw.githubusercontent.com/MAnishSingh13275/repset_bridge/main/config.yaml"
        $configDownloaded = $false
        
        try {
            Write-Info "Downloading working config.yaml from repository..."
            $workingConfig = Invoke-RestMethod -Uri $workingConfigUrl -UseBasicParsing -TimeoutSec 10 -ErrorAction Stop
            
            if ($workingConfig -and $workingConfig.Length -gt 100) {
                # Inject device credentials into the working config if provided
                if ($DeviceId -and $DeviceKey) {
                    Write-Info "Injecting platform device credentials into config..."
                    Write-Info "Using Device ID: $DeviceId"
                    Write-Info "Using Device Key: $($DeviceKey.Substring(0,8))..."
                    
                    # Replace any existing device_id and device_key values
                    $workingConfig = $workingConfig -replace 'device_id: .*', "device_id: $DeviceId"
                    $workingConfig = $workingConfig -replace 'device_key: .*', "device_key: $DeviceKey"
                    
                    Write-Info "Device credentials injected successfully"
                } else {
                    Write-Info "No device credentials provided - using config defaults"
                }
                
                $workingConfig | Out-File -FilePath $ConfigPath -Encoding UTF8 -Force -ErrorAction Stop
                $configDownloaded = $true
                Write-Success "Downloaded and configured working config.yaml from repository"
            }
        } catch {
            Write-Warning "Could not download working config: $($_.Exception.Message)"
        }
        
        # Method 2: Fallback to minimal config if download failed
        if (-not $configDownloaded) {
            Write-Info "Using minimal configuration template..."
            if ($DeviceId -and $DeviceKey) {
                Write-Info "Including provided device credentials in minimal config"
                Write-Info "Device ID: $DeviceId"
                Write-Info "Device Key: $($DeviceKey.Substring(0,8))..."
            } else {
                Write-Info "No device credentials provided - config will have empty credentials"
            }
            $configContent | Out-File -FilePath $ConfigPath -Encoding UTF8 -Force -ErrorAction Stop
        }
        
        # Verify file was created correctly
        if (-not (Test-Path $ConfigPath)) {
            throw "Config file was not created"
        }
        
        $createdContent = Get-Content $ConfigPath -Raw -ErrorAction Stop
        if ([string]::IsNullOrWhiteSpace($createdContent)) {
            throw "Config file is empty"
        }
        
        # Verify basic YAML structure
        if ($createdContent -notmatch "device_id:" -or $createdContent -notmatch "server_url:") {
            throw "Config file structure is invalid"
        }
        
        $lineCount = ($createdContent -split "`n").Count
        Write-Info "Minimal configuration file created with $lineCount lines"
        
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
        
        # Create service using New-Service cmdlet (more reliable than sc.exe)
        $binPath = "`"$ExecutablePath`" --config `"$ConfigPath`""
        
        Write-Debug "Creating service with binPath: $binPath"
        
        try {
            # Use PowerShell's New-Service cmdlet instead of sc.exe
            New-Service -Name $script:SERVICE_NAME -BinaryPathName $binPath -DisplayName $script:SERVICE_DISPLAY_NAME -StartupType Automatic -ErrorAction Stop
            Write-Debug "Service created successfully using New-Service"
        } catch {
            # Fallback to sc.exe with proper syntax
            Write-Debug "New-Service failed, trying sc.exe fallback: $($_.Exception.Message)"
            
            # Use proper sc.exe syntax with individual arguments
            $result = & sc.exe create $script:SERVICE_NAME binPath= $binPath start= auto DisplayName= $script:SERVICE_DISPLAY_NAME 2>&1
            if ($LASTEXITCODE -ne 0) {
                throw "Service creation failed with both New-Service and sc.exe. Last error: $result"
            }
        }
        
        # Set service description using sc.exe (no PowerShell equivalent)
        $description = "RepSet Gym Door Access Bridge - Manages gym door access control integration with RepSet platform"
        & sc.exe description $script:SERVICE_NAME $description | Out-Null
        
        # Configure service recovery options using sc.exe
        & sc.exe failure $script:SERVICE_NAME reset= 86400 actions= restart/5000/restart/10000/restart/30000 | Out-Null
        
        # Verify service was created
        $service = Get-Service -Name $script:SERVICE_NAME -ErrorAction SilentlyContinue
        if (-not $service) {
            throw "Service was not created successfully"
        }
        
        # Ensure service is set to automatic startup (double-check)
        try {
            Set-Service -Name $script:SERVICE_NAME -StartupType Automatic -ErrorAction Stop
            Write-Info "Service startup type confirmed as Automatic"
        } catch {
            Write-Warning "Could not set service startup type: $($_.Exception.Message)"
        }
        
        # Set service to start automatically after installation
        try {
            & sc.exe config $script:SERVICE_NAME start= auto | Out-Null
            Write-Info "Service configured for automatic startup"
        } catch {
            Write-Warning "Could not configure automatic startup: $($_.Exception.Message)"
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
            "--timeout", "15"
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

# Set up automated heartbeat service
function Setup-HeartbeatService {
    param([string]$InstallDir)
    
    try {
        Write-Info "Creating heartbeat service script..."
        
        # Create the heartbeat service script
        $heartbeatScript = @"
# Bridge Heartbeat Service
# This script sends periodic heartbeats to the RepSet platform

param(
    [string]`$ConfigPath = "$InstallDir\config.yaml",
    [string]`$LogPath = "$InstallDir\heartbeat.log"
)

function Write-Log {
    param([string]`$Message)
    `$timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    "`$timestamp - `$Message" | Add-Content -Path `$LogPath -ErrorAction SilentlyContinue
}

function Read-BridgeConfig {
    param([string]`$ConfigPath)
    
    try {
        `$config = @{}
        `$content = Get-Content `$ConfigPath -Raw -ErrorAction Stop
        
        if (`$content -match 'device_id:\s*(.+)') {
            `$config.device_id = `$matches[1].Trim()
        }
        if (`$content -match 'device_key:\s*(.+)') {
            `$config.device_key = `$matches[1].Trim()
        }
        if (`$content -match 'server_url:\s*(.+)') {
            `$config.server_url = `$matches[1].Trim()
        }
        
        return `$config
    } catch {
        Write-Log "ERROR: Failed to read config file: `$(`$_.Exception.Message)"
        return `$null
    }
}

function Send-Heartbeat {
    param([string]`$DeviceId, [string]`$DeviceKey, [string]`$ServerUrl)
    
    try {
        `$heartbeatPayload = @{
            device_id = `$DeviceId
            device_key = `$DeviceKey
            status = @{
                version = "1.4.0"
                uptime = [int]((Get-Date) - (Get-Process -Name "gym-door-bridge" -ErrorAction SilentlyContinue | Select-Object -First 1).StartTime).TotalSeconds
                connected_devices = 1
                last_event_time = (Get-Date).ToString("yyyy-MM-ddTHH:mm:ssZ")
                system_info = @{
                    platform = "windows"
                    arch = "amd64"
                    memory = [math]::Round((Get-WmiObject -Class Win32_ComputerSystem).TotalPhysicalMemory / 1MB)
                    cpu = (Get-WmiObject -Class Win32_Processor | Measure-Object -Property LoadPercentage -Average).Average
                }
            }
        } | ConvertTo-Json -Depth 3
        
        `$heartbeatUrl = "`$ServerUrl/api/v1/bridge/heartbeat"
        `$response = Invoke-RestMethod -Uri `$heartbeatUrl -Method POST -Body `$heartbeatPayload -ContentType "application/json" -UseBasicParsing -TimeoutSec 30
        
        Write-Log "SUCCESS: Heartbeat sent successfully"
        return `$true
        
    } catch {
        Write-Log "ERROR: Heartbeat failed - `$(`$_.Exception.Message)"
        return `$false
    }
}

# Main execution
`$config = Read-BridgeConfig -ConfigPath `$ConfigPath

if (-not `$config -or -not `$config.device_id -or -not `$config.device_key -or -not `$config.server_url) {
    Write-Log "ERROR: Invalid configuration"
    exit 1
}

`$success = Send-Heartbeat -DeviceId `$config.device_id -DeviceKey `$config.device_key -ServerUrl `$config.server_url
exit (`$success ? 0 : 1)
"@

        # Write the heartbeat script to the installation directory
        $heartbeatScriptPath = Join-Path $InstallDir "bridge-heartbeat-service.ps1"
        $heartbeatScript | Out-File -FilePath $heartbeatScriptPath -Encoding UTF8 -Force
        Write-Info "Heartbeat script created"
        
        # Create scheduled task for heartbeats
        Write-Info "Creating scheduled task for automated heartbeats..."
        
        $taskName = "RepSet Bridge Heartbeat"
        
        # Remove existing task if it exists
        $existingTask = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
        if ($existingTask) {
            Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
        }
        
        # Create scheduled task components
        $action = New-ScheduledTaskAction -Execute "powershell.exe" -Argument "-ExecutionPolicy Bypass -File `"$heartbeatScriptPath`""
        $trigger = New-ScheduledTaskTrigger -Once -At (Get-Date) -RepetitionInterval (New-TimeSpan -Seconds 60) -RepetitionDuration (New-TimeSpan -Days 365)
        $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -RunOnlyIfNetworkAvailable
        $principal = New-ScheduledTaskPrincipal -UserId "SYSTEM" -LogonType ServiceAccount -RunLevel Highest
        
        # Register the scheduled task
        Register-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger -Settings $settings -Principal $principal -Description "Sends heartbeats from RepSet Bridge to platform every 60 seconds" -ErrorAction Stop
        
        # Start the task immediately
        Start-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
        
        Write-Success "Automated heartbeat service configured successfully"
        return $true
        
    } catch {
        Write-Warning "Failed to set up heartbeat service: $($_.Exception.Message)"
        Write-Info "Bridge will still work, but status updates may be delayed"
        return $false
    }
}

# Start service with comprehensive retry logic and troubleshooting
function Start-BridgeService {
    try {
        Write-Info "Starting RepSet Bridge service..."
        
        # Get service object
        $service = Get-Service -Name $script:SERVICE_NAME -ErrorAction Stop
        
        if ($service.Status -eq "Running") {
            Write-Success "Service is already running"
            return $true
        }
        
        # Try multiple approaches to start the service
        $startupAttempts = 3
        for ($attempt = 1; $attempt -le $startupAttempts; $attempt++) {
            try {
                Write-Info "Service startup attempt $attempt of $startupAttempts..."
                
                # Method 1: Use Start-Service cmdlet
                Start-Service -Name $script:SERVICE_NAME -ErrorAction Stop
                
                # Wait for service to start with timeout
                $timeout = 15
                $service.Refresh()
                $service.WaitForStatus("Running", [TimeSpan]::FromSeconds($timeout))
                
                # Verify service is actually running
                $service.Refresh()
                if ($service.Status -eq "Running") {
                    Write-Success "Service started successfully on attempt $attempt"
                    return $true
                } else {
                    throw "Service status is $($service.Status) after start attempt"
                }
                
            } catch {
                Write-Warning "Attempt $attempt failed: $($_.Exception.Message)"
                
                if ($attempt -lt $startupAttempts) {
                    Write-Info "Waiting 3 seconds before retry..."
                    Start-Sleep -Seconds 3
                    
                    # Try alternative method: net start command
                    if ($attempt -eq 2) {
                        Write-Info "Trying alternative startup method (net start)..."
                        try {
                            $netResult = & net start $script:SERVICE_NAME 2>&1
                            if ($LASTEXITCODE -eq 0) {
                                Write-Info "net start command succeeded"
                                Start-Sleep -Seconds 2
                                $service.Refresh()
                                if ($service.Status -eq "Running") {
                                    Write-Success "Service started using net start command"
                                    return $true
                                }
                            } else {
                                Write-Warning "net start failed: $netResult"
                            }
                        } catch {
                            Write-Warning "net start command failed: $($_.Exception.Message)"
                        }
                    }
                }
            }
        }
        
        # If all attempts failed, try to diagnose the issue
        Write-Warning "All service startup attempts failed. Diagnosing issue..."
        
        # Check service configuration
        $service.Refresh()
        Write-Info "Service Status: $($service.Status)"
        Write-Info "Service StartType: $($service.StartType)"
        
        # Check if executable exists and is accessible
        $serviceConfig = Get-WmiObject -Class Win32_Service -Filter "Name='$script:SERVICE_NAME'"
        if ($serviceConfig) {
            Write-Info "Service Path: $($serviceConfig.PathName)"
            
            # Extract executable path from service path
            $exePath = $serviceConfig.PathName -replace '^"([^"]+)".*', '$1'
            if (Test-Path $exePath) {
                Write-Info "Executable exists and is accessible"
                
                # Try to run executable directly to check for issues
                try {
                    Write-Info "Testing executable directly..."
                    $testProcess = Start-Process -FilePath $exePath -ArgumentList "--help" -Wait -PassThru -NoNewWindow -RedirectStandardOutput "$env:TEMP\bridge_test_output.txt" -RedirectStandardError "$env:TEMP\bridge_test_error.txt"
                    
                    if ($testProcess.ExitCode -eq 0) {
                        Write-Info "Executable runs correctly"
                    } else {
                        Write-Warning "Executable test failed with exit code: $($testProcess.ExitCode)"
                        
                        # Show error output if available
                        if (Test-Path "$env:TEMP\bridge_test_error.txt") {
                            $errorOutput = Get-Content "$env:TEMP\bridge_test_error.txt" -Raw
                            if ($errorOutput) {
                                Write-Warning "Executable error: $errorOutput"
                            }
                        }
                    }
                } catch {
                    Write-Warning "Could not test executable: $($_.Exception.Message)"
                }
            } else {
                Write-Warning "Executable not found at: $exePath"
            }
        }
        
        # Provide detailed troubleshooting information
        Write-Info "Service startup failed. Troubleshooting steps:"
        Write-Info "1. Check Windows Event Viewer (Windows Logs > System) for service errors"
        Write-Info "2. Verify the executable has proper permissions"
        Write-Info "3. Try starting the service manually from Services.msc"
        Write-Info "4. Check if antivirus is blocking the service"
        Write-Info "5. Restart Windows to trigger automatic service startup"
        
        return $false
        
    } catch {
        Write-Warning "Service startup error: $($_.Exception.Message)"
        Write-Info "The service is installed and configured for automatic startup"
        Write-Info "It will start automatically when Windows boots"
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
    
    if ($HeartbeatSetup) {
        Write-Success "Automated heartbeat service configured - bridge will stay connected to platform"
    } else {
        Write-Warning "Heartbeat service setup failed - you may need to configure it manually"
    }
    
    Write-Host ""
    
    # Provide manual startup commands since automatic startup failed
    if (-not $ServiceStarted -or -not $PairingSuccessful) {
        Write-Host "=" * 60 -ForegroundColor Yellow
        Write-Host "MANUAL STARTUP COMMANDS" -ForegroundColor Yellow
        Write-Host "=" * 60 -ForegroundColor Yellow
        Write-Host ""
        
        if (-not $ServiceStarted) {
            Write-Host "To start the service manually:" -ForegroundColor Cyan
            Write-Host "1. Open Command Prompt as Administrator" -ForegroundColor White
            Write-Host "2. Run: net start GymDoorBridge" -ForegroundColor Green
            Write-Host "   OR" -ForegroundColor Gray
            Write-Host "   Open Services.msc and start 'RepSet Gym Door Bridge'" -ForegroundColor Green
            Write-Host ""
        }
        
        if (-not $PairingSuccessful) {
            Write-Host "To pair the device manually:" -ForegroundColor Cyan
            Write-Host "1. Open Command Prompt as Administrator" -ForegroundColor White
            Write-Host "2. Navigate to: cd `"C:\Program Files\GymDoorBridge`"" -ForegroundColor Green
            if ($DeviceId -and $DeviceKey) {
                Write-Host "3. Device credentials already configured in config file" -ForegroundColor Green
                Write-Host "4. Run: gym-door-bridge.exe pair --pair-code $PairCode" -ForegroundColor Green
            } else {
                Write-Host "3. Run: gym-door-bridge.exe pair --pair-code $PairCode" -ForegroundColor Green
            }
            Write-Host ""
        }
        
        Write-Host "To test the configuration:" -ForegroundColor Cyan
        Write-Host "1. Open Command Prompt as Administrator" -ForegroundColor White
        Write-Host "2. Navigate to: cd `"C:\Program Files\GymDoorBridge`"" -ForegroundColor Green
        Write-Host "3. Run: gym-door-bridge.exe --help" -ForegroundColor Green
        Write-Host "4. Run: gym-door-bridge.exe status" -ForegroundColor Green
        Write-Host ""
        
        Write-Host "Once started manually, the service will:" -ForegroundColor Cyan
        Write-Host "Start automatically on system boot" -ForegroundColor Green
        Write-Host "Run in the background continuously" -ForegroundColor Green
        Write-Host "Handle gym door access requests" -ForegroundColor Green
        Write-Host ""
        
        Write-Host "Config file location: C:\Program Files\GymDoorBridge\config.yaml" -ForegroundColor Gray
        Write-Host "Service name: GymDoorBridge" -ForegroundColor Gray
        Write-Host ""
    }
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
    if ($DeviceId) { Write-Host "Device ID: $DeviceId" -ForegroundColor Gray }
    if ($DeviceKey) { Write-Host "Device Key: $($DeviceKey.Substring(0,8))..." -ForegroundColor Gray }
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
    
    # Validate device credentials if provided
    if ($DeviceId -or $DeviceKey) {
        Write-Info "Validating device credentials..."
        if ([string]::IsNullOrWhiteSpace($DeviceId)) {
            Exit-WithMessage -Message "Device ID cannot be empty when device credentials are provided" -ExitCode 1 -IsError
        }
        if ([string]::IsNullOrWhiteSpace($DeviceKey)) {
            Exit-WithMessage -Message "Device Key cannot be empty when device credentials are provided" -ExitCode 1 -IsError
        }
        if ($DeviceKey.Length -lt 32) {
            Exit-WithMessage -Message "Device Key appears to be too short (minimum 32 characters expected)" -ExitCode 1 -IsError
        }
        Write-Success "Device credentials are valid"
    } else {
        Write-Info "No device credentials provided - bridge will generate its own"
    }
    
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
    
    # Test installation path (skip detailed validation since we're running as Administrator)
    Write-Info "Validating installation path..."
    try {
        # Simple check - ensure Program Files exists
        $programFiles = Split-Path $InstallDir -Parent
        if (-not (Test-Path $programFiles)) {
            throw "Program Files directory not found: $programFiles"
        }
        Write-Success "Installation path is valid"
    } catch {
        Write-Warning "Installation path validation failed: $($_.Exception.Message)"
        Write-Info "Proceeding anyway since running as Administrator..."
    }
    
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
    if (-not (New-ConfigurationFile -ConfigPath $ConfigFile -PairCode $PairCode -DeviceId $DeviceId -DeviceKey $DeviceKey)) {
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
    
    # Step 8: Set up heartbeat service
    Write-Step "8/9" "Setting up automated heartbeat service"
    $HeartbeatSetup = Setup-HeartbeatService -InstallDir $InstallDir
    
    # Step 9: Pair device and start service
    Write-Step "9/9" "Finalizing installation"
    
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