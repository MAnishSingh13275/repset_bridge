# Deploy Gym Door Bridge as Windows Service
# Run this script as Administrator

param(
    [string]$ConfigFile = "",
    [string]$InstallPath = "$env:ProgramFiles\GymDoorBridge",
    [switch]$Reinstall = $false,
    [switch]$StartService = $true
)

# Check administrator privileges
$currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
if (-not $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Host "ERROR: This script requires administrator privileges." -ForegroundColor Red
    Write-Host "Please run PowerShell as Administrator and try again." -ForegroundColor Yellow
    exit 1
}

Write-Host "=== Gym Door Bridge Service Deployment ===" -ForegroundColor Green
Write-Host ""

# Build the executable if it doesn't exist
if (-not (Test-Path "gym-door-bridge.exe")) {
    Write-Host "Building gym-door-bridge.exe..." -ForegroundColor Yellow
    go build -o gym-door-bridge.exe ./cmd
    
    if (-not $?) {
        Write-Host "ERROR: Failed to build executable" -ForegroundColor Red
        exit 1
    }
    Write-Host "✓ Build completed successfully" -ForegroundColor Green
}

# Check if service already exists
$serviceExists = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue

if ($serviceExists -and -not $Reinstall) {
    Write-Host "WARNING: Gym Door Bridge service already exists." -ForegroundColor Yellow
    $choice = Read-Host "Do you want to reinstall? (y/N)"
    if ($choice -ne "y" -and $choice -ne "Y") {
        Write-Host "Deployment cancelled." -ForegroundColor Yellow
        exit 0
    }
    $Reinstall = $true
}

# Uninstall existing service if reinstalling
if ($Reinstall -and $serviceExists) {
    Write-Host "Uninstalling existing service..." -ForegroundColor Yellow
    & ".\gym-door-bridge.exe" service uninstall
    Start-Sleep -Seconds 2
}

# Install the service
Write-Host "Installing Gym Door Bridge service..." -ForegroundColor Yellow
Write-Host "This will scan for biometric devices on your network..." -ForegroundColor Cyan

$installArgs = @("install")
if ($ConfigFile) {
    $installArgs += "--config", $ConfigFile
}

& ".\gym-door-bridge.exe" @installArgs

if (-not $?) {
    Write-Host "ERROR: Service installation failed" -ForegroundColor Red
    exit 1
}

# Start service if requested
if ($StartService) {
    Write-Host "Starting service..." -ForegroundColor Yellow
    & ".\gym-door-bridge.exe" service start
    
    if ($?) {
        Write-Host "✓ Service started successfully" -ForegroundColor Green
    } else {
        Write-Host "WARNING: Failed to start service automatically" -ForegroundColor Yellow
        Write-Host "You can start it manually from Services.msc or run: gym-door-bridge.exe service start" -ForegroundColor Cyan
    }
}

Write-Host ""
Write-Host "=== Deployment Completed Successfully! ===" -ForegroundColor Green
Write-Host ""
Write-Host "Service Details:" -ForegroundColor Cyan
Write-Host "  Name: GymDoorBridge"
Write-Host "  Display Name: Gym Door Access Bridge"
Write-Host "  Status: Check with 'gym-door-bridge.exe service status'"
Write-Host ""
Write-Host "Next Steps:" -ForegroundColor Yellow
Write-Host "1. Pair device: gym-door-bridge.exe pair --pair-code YOUR_PAIR_CODE"
Write-Host "2. Check status: gym-door-bridge.exe service status"
Write-Host "3. View logs: Check Windows Event Viewer or service log files"
Write-Host "4. Manage service: Use Services.msc or service commands"
Write-Host ""
Write-Host "Service Commands:" -ForegroundColor Cyan
Write-Host "  gym-door-bridge.exe service start    - Start service"
Write-Host "  gym-door-bridge.exe service stop     - Stop service" 
Write-Host "  gym-door-bridge.exe service restart  - Restart service"
Write-Host "  gym-door-bridge.exe service status   - Show status"
Write-Host "  gym-door-bridge.exe service uninstall - Remove service"