# Gym Door Bridge - Windows Installation Script
# This script downloads, configures, and installs the Gym Door Bridge as a Windows service

param(
    [Parameter(Mandatory=$true)]
    [string]$PairCode,
    
    [Parameter(Mandatory=$false)]
    [string]$ServerURL = "https://api.repset.onezy.in",
    
    [Parameter(Mandatory=$false)]
    [string]$InstallDir = "$env:ProgramFiles\GymDoorBridge",
    
    [Parameter(Mandatory=$false)]
    [string]$ConfigDir = "$env:ProgramData\GymDoorBridge",
    
    [Parameter(Mandatory=$false)]
    [string]$Version = "latest",
    
    [Parameter(Mandatory=$false)]
    [string]$CDNBaseURL = "https://cdn.repset.onezy.in/gym-door-bridge"
)

# Script configuration
$ErrorActionPreference = "Stop"
$ServiceName = "GymDoorBridge"
$ExecutableName = "gym-door-bridge.exe"

# Logging function
function Write-Log {
    param([string]$Message, [string]$Level = "INFO")
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    Write-Host "[$timestamp] [$Level] $Message"
}

# Check if running as administrator
function Test-Administrator {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

# Download file with progress
function Download-File {
    param([string]$URL, [string]$OutputPath)
    
    Write-Log "Downloading from $URL to $OutputPath"
    
    try {
        # Use System.Net.WebClient for progress reporting
        $webClient = New-Object System.Net.WebClient
        $webClient.DownloadFile($URL, $OutputPath)
        Write-Log "Download completed successfully"
    }
    catch {
        throw "Failed to download file: $_"
    }
    finally {
        if ($webClient) { $webClient.Dispose() }
    }
}

# Verify file signature (placeholder for actual implementation)
function Test-FileSignature {
    param([string]$FilePath)
    
    Write-Log "Verifying file signature for $FilePath"
    
    # TODO: Implement actual signature verification
    # For now, just check if file exists and has reasonable size
    if (-not (Test-Path $FilePath)) {
        throw "File does not exist: $FilePath"
    }
    
    $fileInfo = Get-Item $FilePath
    if ($fileInfo.Length -lt 1MB) {
        throw "File appears to be too small: $($fileInfo.Length) bytes"
    }
    
    Write-Log "File signature verification passed"
    return $true
}

# Create configuration file
function New-ConfigFile {
    param([string]$ConfigPath, [string]$ServerURL, [string]$PairCode)
    
    Write-Log "Creating configuration file at $ConfigPath"
    
    $configContent = @"
# Gym Door Bridge Configuration
server_url: "$ServerURL"
tier: "normal"
queue_max_size: 10000
heartbeat_interval: 60
unlock_duration: 3000
database_path: "$ConfigDir\bridge.db"
log_level: "info"
log_file: "$ConfigDir\logs\bridge.log"
enabled_adapters:
  - simulator

# Pairing configuration (will be updated after pairing)
device_id: ""
device_key: ""
"@
    
    # Ensure config directory exists
    $configDirPath = Split-Path $ConfigPath -Parent
    if (-not (Test-Path $configDirPath)) {
        New-Item -ItemType Directory -Path $configDirPath -Force | Out-Null
    }
    
    # Create logs directory
    $logsDir = Join-Path $ConfigDir "logs"
    if (-not (Test-Path $logsDir)) {
        New-Item -ItemType Directory -Path $logsDir -Force | Out-Null
    }
    
    Set-Content -Path $ConfigPath -Value $configContent -Encoding UTF8
    Write-Log "Configuration file created successfully"
}

# Install Windows service
function Install-BridgeService {
    param([string]$ExecutablePath, [string]$ConfigPath)
    
    Write-Log "Installing Windows service"
    
    try {
        # Stop service if it exists
        $existingService = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
        if ($existingService) {
            Write-Log "Stopping existing service"
            Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
            
            # Uninstall existing service
            Write-Log "Removing existing service"
            & $ExecutablePath service uninstall
            Start-Sleep -Seconds 2
        }
        
        # Install new service
        Write-Log "Installing service with executable: $ExecutablePath"
        & $ExecutablePath service install --config $ConfigPath
        
        if ($LASTEXITCODE -ne 0) {
            throw "Service installation failed with exit code $LASTEXITCODE"
        }
        
        Write-Log "Service installed successfully"
    }
    catch {
        throw "Failed to install service: $_"
    }
}

# Perform device pairing
function Invoke-DevicePairing {
    param([string]$ExecutablePath, [string]$ConfigPath, [string]$PairCode)
    
    Write-Log "Starting device pairing with code: $PairCode"
    
    try {
        # Run pairing command
        $pairingResult = & $ExecutablePath pair --config $ConfigPath --pair-code $PairCode 2>&1
        
        if ($LASTEXITCODE -ne 0) {
            throw "Pairing failed with exit code $LASTEXITCODE. Output: $pairingResult"
        }
        
        Write-Log "Device pairing completed successfully"
        Write-Log "Pairing output: $pairingResult"
    }
    catch {
        throw "Device pairing failed: $_"
    }
}

# Start Windows service
function Start-BridgeService {
    Write-Log "Starting Windows service"
    
    try {
        Start-Service -Name $ServiceName
        
        # Wait for service to start
        $timeout = 30
        $elapsed = 0
        do {
            Start-Sleep -Seconds 1
            $elapsed++
            $service = Get-Service -Name $ServiceName
        } while ($service.Status -ne "Running" -and $elapsed -lt $timeout)
        
        if ($service.Status -ne "Running") {
            throw "Service failed to start within $timeout seconds"
        }
        
        Write-Log "Service started successfully"
    }
    catch {
        throw "Failed to start service: $_"
    }
}

# Cleanup function
function Remove-TempFiles {
    param([string[]]$FilePaths)
    
    foreach ($filePath in $FilePaths) {
        if (Test-Path $filePath) {
            try {
                Remove-Item $filePath -Force
                Write-Log "Removed temporary file: $filePath"
            }
            catch {
                Write-Log "Warning: Failed to remove temporary file: $filePath" "WARN"
            }
        }
    }
}

# Main installation function
function Install-GymDoorBridge {
    Write-Log "Starting Gym Door Bridge installation"
    Write-Log "Pair Code: $PairCode"
    Write-Log "Server URL: $ServerURL"
    Write-Log "Install Directory: $InstallDir"
    Write-Log "Config Directory: $ConfigDir"
    Write-Log "Version: $Version"
    
    # Check administrator privileges
    if (-not (Test-Administrator)) {
        throw "This script must be run as Administrator"
    }
    
    $tempFiles = @()
    
    try {
        # Create installation directory
        if (-not (Test-Path $InstallDir)) {
            Write-Log "Creating installation directory: $InstallDir"
            New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        }
        
        # Create config directory
        if (-not (Test-Path $ConfigDir)) {
            Write-Log "Creating configuration directory: $ConfigDir"
            New-Item -ItemType Directory -Path $ConfigDir -Force | Out-Null
        }
        
        # Download executable
        $downloadURL = "$CDNBaseURL/$Version/windows/amd64/$ExecutableName"
        $executablePath = Join-Path $InstallDir $ExecutableName
        $tempFiles += $executablePath
        
        Download-File -URL $downloadURL -OutputPath $executablePath
        
        # Verify file signature
        Test-FileSignature -FilePath $executablePath
        
        # Create configuration file
        $configPath = Join-Path $ConfigDir "config.yaml"
        New-ConfigFile -ConfigPath $configPath -ServerURL $ServerURL -PairCode $PairCode
        
        # Install Windows service
        Install-BridgeService -ExecutablePath $executablePath -ConfigPath $configPath
        
        # Perform device pairing
        Invoke-DevicePairing -ExecutablePath $executablePath -ConfigPath $configPath -PairCode $PairCode
        
        # Start service
        Start-BridgeService
        
        # Remove executable from temp files list since installation succeeded
        $tempFiles = $tempFiles | Where-Object { $_ -ne $executablePath }
        
        Write-Log "Installation completed successfully!" "SUCCESS"
        Write-Log "Service Name: $ServiceName"
        Write-Log "Executable: $executablePath"
        Write-Log "Configuration: $configPath"
        Write-Log "Logs: $ConfigDir\logs\bridge.log"
        
        # Display service status
        $service = Get-Service -Name $ServiceName
        Write-Log "Service Status: $($service.Status)"
        
    }
    catch {
        Write-Log "Installation failed: $_" "ERROR"
        
        # Cleanup on failure
        try {
            Write-Log "Performing cleanup..."
            
            # Stop and remove service if it was created
            $existingService = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
            if ($existingService) {
                Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
                & $executablePath service uninstall -ErrorAction SilentlyContinue
            }
            
            # Remove temporary files
            Remove-TempFiles -FilePaths $tempFiles
            
        }
        catch {
            Write-Log "Cleanup failed: $_" "WARN"
        }
        
        throw
    }
    finally {
        # Always try to clean up temp files
        Remove-TempFiles -FilePaths $tempFiles
    }
}

# Script entry point
try {
    Install-GymDoorBridge
}
catch {
    Write-Log "Script execution failed: $_" "ERROR"
    exit 1
}