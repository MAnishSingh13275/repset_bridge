# Gym Door Bridge Installer Script
# This script downloads and installs the Gym Door Bridge as a Windows service

param(
    [string]$Version = "latest",
    [string]$InstallPath = "$env:ProgramFiles\GymDoorBridge",
    [switch]$Force = $false
)

# Check if running as administrator
function Test-Administrator {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

# Download file with progress
function Download-File {
    param(
        [string]$Url,
        [string]$OutputPath
    )
    
    Write-Host "Downloading from: $Url" -ForegroundColor Green
    
    try {
        $webClient = New-Object System.Net.WebClient
        $webClient.DownloadFile($Url, $OutputPath)
        Write-Host "Download completed: $OutputPath" -ForegroundColor Green
        return $true
    }
    catch {
        Write-Host "Download failed: $($_.Exception.Message)" -ForegroundColor Red
        return $false
    }
}

# Main installation function
function Install-GymDoorBridge {
    Write-Host "=== Gym Door Bridge Installer ===" -ForegroundColor Cyan
    Write-Host ""
    
    # Check administrator privileges
    if (-not (Test-Administrator)) {
        Write-Host "ERROR: This installer requires administrator privileges." -ForegroundColor Red
        Write-Host "Please run PowerShell as Administrator and try again." -ForegroundColor Yellow
        exit 1
    }
    
    Write-Host "Administrator privileges confirmed." -ForegroundColor Green
    
    # Check if service already exists
    $service = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
    if ($service -and -not $Force) {
        Write-Host "ERROR: Gym Door Bridge service already exists." -ForegroundColor Red
        Write-Host "Use -Force parameter to reinstall." -ForegroundColor Yellow
        exit 1
    }
    
    # Create temporary directory
    $tempDir = Join-Path $env:TEMP "GymDoorBridge-Install"
    if (Test-Path $tempDir) {
        Remove-Item $tempDir -Recurse -Force
    }
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
    
    try {
        # Download the latest release
        Write-Host "Downloading Gym Door Bridge..." -ForegroundColor Yellow
        
        # For now, we'll assume the executable is built and available
        # In production, this would download from GitHub releases
        $downloadUrl = "https://github.com/MAnishSingh13275/repset_bridge/releases/download/$Version/gym-door-bridge-windows-amd64.exe"
        $exePath = Join-Path $tempDir "gym-door-bridge.exe"
        
        # For development, copy from current directory if available
        $localExe = ".\gym-door-bridge.exe"
        if (Test-Path $localExe) {
            Write-Host "Using local executable for installation..." -ForegroundColor Yellow
            Copy-Item $localExe $exePath
        }
        else {
            Write-Host "Local executable not found. In production, this would download from GitHub releases." -ForegroundColor Yellow
            Write-Host "Please build the executable first: go build -o gym-door-bridge.exe ./cmd" -ForegroundColor Red
            exit 1
        }
        
        # Verify executable
        if (-not (Test-Path $exePath)) {
            Write-Host "ERROR: Failed to obtain executable." -ForegroundColor Red
            exit 1
        }
        
        Write-Host "Executable ready for installation." -ForegroundColor Green
        
        # Run the installer
        Write-Host "Running installation with device discovery..." -ForegroundColor Yellow
        
        $installArgs = @("install")
        if ($InstallPath -ne "$env:ProgramFiles\GymDoorBridge") {
            $installArgs += "--install-path", $InstallPath
        }
        
        $process = Start-Process -FilePath $exePath -ArgumentList $installArgs -Wait -PassThru -NoNewWindow
        
        if ($process.ExitCode -eq 0) {
            Write-Host ""
            Write-Host "=== Installation Completed Successfully! ===" -ForegroundColor Green
            Write-Host ""
            Write-Host "The Gym Door Bridge has been installed as a Windows service." -ForegroundColor White
            Write-Host "Installation Path: $InstallPath" -ForegroundColor White
            Write-Host ""
            Write-Host "Next Steps:" -ForegroundColor Yellow
            Write-Host "1. Pair with your platform: gym-door-bridge pair" -ForegroundColor White
            Write-Host "2. Check service status: Get-Service GymDoorBridge" -ForegroundColor White
            Write-Host "3. View logs in: $InstallPath\logs\" -ForegroundColor White
            Write-Host ""
            Write-Host "The service will automatically discover and configure biometric devices on your network." -ForegroundColor Cyan
        }
        else {
            Write-Host "ERROR: Installation failed with exit code $($process.ExitCode)" -ForegroundColor Red
            exit 1
        }
    }
    catch {
        Write-Host "ERROR: Installation failed: $($_.Exception.Message)" -ForegroundColor Red
        exit 1
    }
    finally {
        # Cleanup
        if (Test-Path $tempDir) {
            Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

# Uninstall function
function Uninstall-GymDoorBridge {
    Write-Host "=== Gym Door Bridge Uninstaller ===" -ForegroundColor Cyan
    Write-Host ""
    
    # Check administrator privileges
    if (-not (Test-Administrator)) {
        Write-Host "ERROR: This uninstaller requires administrator privileges." -ForegroundColor Red
        Write-Host "Please run PowerShell as Administrator and try again." -ForegroundColor Yellow
        exit 1
    }
    
    # Find installation
    $installPath = $null
    
    # Check registry for installation path
    try {
        $regKey = Get-ItemProperty -Path "HKLM:\SOFTWARE\GymDoorBridge" -ErrorAction SilentlyContinue
        if ($regKey) {
            $installPath = $regKey.InstallPath
        }
    }
    catch {
        # Registry key not found
    }
    
    # Fallback to default path
    if (-not $installPath -or -not (Test-Path $installPath)) {
        $installPath = "$env:ProgramFiles\GymDoorBridge"
    }
    
    if (-not (Test-Path $installPath)) {
        Write-Host "ERROR: Gym Door Bridge installation not found." -ForegroundColor Red
        exit 1
    }
    
    $exePath = Join-Path $installPath "gym-door-bridge.exe"
    
    if (Test-Path $exePath) {
        Write-Host "Running uninstaller..." -ForegroundColor Yellow
        
        $process = Start-Process -FilePath $exePath -ArgumentList "uninstall" -Wait -PassThru -NoNewWindow
        
        if ($process.ExitCode -eq 0) {
            Write-Host "Uninstallation completed successfully!" -ForegroundColor Green
        }
        else {
            Write-Host "ERROR: Uninstallation failed with exit code $($process.ExitCode)" -ForegroundColor Red
            exit 1
        }
    }
    else {
        Write-Host "ERROR: Uninstaller executable not found at: $exePath" -ForegroundColor Red
        exit 1
    }
}

# Show help
function Show-Help {
    Write-Host "Gym Door Bridge Installer" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "USAGE:" -ForegroundColor Yellow
    Write-Host "  .\install.ps1                    # Install with default settings"
    Write-Host "  .\install.ps1 -Force             # Force reinstall"
    Write-Host "  .\install.ps1 -InstallPath C:\MyPath  # Custom install path"
    Write-Host "  .\install.ps1 -Uninstall         # Uninstall"
    Write-Host ""
    Write-Host "PARAMETERS:" -ForegroundColor Yellow
    Write-Host "  -Version      Version to install (default: latest)"
    Write-Host "  -InstallPath  Installation directory"
    Write-Host "  -Force        Force reinstall if already installed"
    Write-Host "  -Uninstall    Uninstall the service"
    Write-Host "  -Help         Show this help message"
    Write-Host ""
    Write-Host "EXAMPLES:" -ForegroundColor Yellow
    Write-Host "  .\install.ps1"
    Write-Host "  .\install.ps1 -Force -InstallPath 'C:\GymBridge'"
    Write-Host "  .\install.ps1 -Uninstall"
}

# Main script logic
if ($args -contains "-Help" -or $args -contains "--help" -or $args -contains "-h") {
    Show-Help
    exit 0
}

if ($args -contains "-Uninstall") {
    Uninstall-GymDoorBridge
}
else {
    Install-GymDoorBridge
}