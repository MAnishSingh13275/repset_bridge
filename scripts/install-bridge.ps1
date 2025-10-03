# Gym Door Bridge - One-Click Installation Script
# This script installs and configures the Gym Door Bridge as a Windows service

param(
    [string]$PairCode = "",
    [string]$ServerUrl = "https://repset.onezy.in",
    [string]$InstallPath = "$env:ProgramFiles\GymDoorBridge",
    [switch]$Force = $false
)

# Ensure running as Administrator
if (-NOT ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    Write-Host "❌ This script requires Administrator privileges!" -ForegroundColor Red
    Write-Host "Please run PowerShell as Administrator and try again." -ForegroundColor Yellow
    exit 1
}

Write-Host "🚀 Gym Door Bridge Installation" -ForegroundColor Cyan
Write-Host "================================" -ForegroundColor Cyan

# Check if service already exists
$existingService = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
if ($existingService -and -not $Force) {
    Write-Host "⚠️  Gym Door Bridge is already installed!" -ForegroundColor Yellow
    Write-Host "Use -Force parameter to reinstall or run 'gym-door-bridge status' to check status." -ForegroundColor Yellow
    exit 1
}

try {
    # Download latest release
    Write-Host "📥 Downloading latest Gym Door Bridge..." -ForegroundColor Green
    $downloadUrl = "https://github.com/your-org/gym-door-bridge/releases/latest/download/gym-door-bridge-windows.zip"
    $tempZip = "$env:TEMP\gym-door-bridge.zip"
    $tempExtract = "$env:TEMP\gym-door-bridge"
    
    # Create temp directory
    if (Test-Path $tempExtract) {
        Remove-Item $tempExtract -Recurse -Force
    }
    New-Item -ItemType Directory -Path $tempExtract -Force | Out-Null
    
    # Download with progress
    $webClient = New-Object System.Net.WebClient
    $webClient.DownloadFile($downloadUrl, $tempZip)
    
    # Extract
    Write-Host "📦 Extracting files..." -ForegroundColor Green
    Expand-Archive -Path $tempZip -DestinationPath $tempExtract -Force
    
    # Find executable
    $exePath = Get-ChildItem -Path $tempExtract -Name "gym-door-bridge.exe" -Recurse | Select-Object -First 1
    if (-not $exePath) {
        throw "gym-door-bridge.exe not found in downloaded package"
    }
    $fullExePath = Join-Path $tempExtract $exePath.FullName
    
    # Stop existing service if running
    if ($existingService) {
        Write-Host "🛑 Stopping existing service..." -ForegroundColor Yellow
        Stop-Service -Name "GymDoorBridge" -Force -ErrorAction SilentlyContinue
        & "$fullExePath" uninstall
    }
    
    # Run installation
    Write-Host "⚙️  Installing Gym Door Bridge..." -ForegroundColor Green
    $installProcess = Start-Process -FilePath $fullExePath -ArgumentList "install" -Wait -PassThru -NoNewWindow
    
    if ($installProcess.ExitCode -ne 0) {
        throw "Installation failed with exit code $($installProcess.ExitCode)"
    }
    
    Write-Host "✅ Installation completed successfully!" -ForegroundColor Green
    
    # Pair device if pair code provided
    if ($PairCode) {
        Write-Host "🔗 Pairing device with platform..." -ForegroundColor Green
        $pairProcess = Start-Process -FilePath "$InstallPath\gym-door-bridge.exe" -ArgumentList "pair", $PairCode -Wait -PassThru -NoNewWindow
        
        if ($pairProcess.ExitCode -eq 0) {
            Write-Host "✅ Device paired successfully!" -ForegroundColor Green
        } else {
            Write-Host "⚠️  Pairing failed. You can pair manually later using:" -ForegroundColor Yellow
            Write-Host "   gym-door-bridge pair YOUR_PAIR_CODE" -ForegroundColor White
        }
    }
    
    # Check service status
    Start-Sleep -Seconds 3
    $service = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
    if ($service -and $service.Status -eq "Running") {
        Write-Host "✅ Service is running successfully!" -ForegroundColor Green
    } else {
        Write-Host "⚠️  Service installation completed but not running. Starting now..." -ForegroundColor Yellow
        Start-Service -Name "GymDoorBridge"
        Write-Host "✅ Service started!" -ForegroundColor Green
    }
    
    # Show status
    Write-Host "`n📊 Installation Summary:" -ForegroundColor Cyan
    Write-Host "========================" -ForegroundColor Cyan
    Write-Host "Installation Path: $InstallPath" -ForegroundColor White
    Write-Host "Service Name: GymDoorBridge" -ForegroundColor White
    Write-Host "API Endpoint: http://localhost:8081" -ForegroundColor White
    Write-Host "Server URL: $ServerUrl" -ForegroundColor White
    
    if ($PairCode) {
        Write-Host "Pair Code Used: $PairCode" -ForegroundColor White
    } else {
        Write-Host "`n🔗 To pair with your platform:" -ForegroundColor Yellow
        Write-Host "   gym-door-bridge pair YOUR_PAIR_CODE" -ForegroundColor White
    }
    
    Write-Host "`n📋 Useful Commands:" -ForegroundColor Cyan
    Write-Host "   gym-door-bridge status    - Check bridge status" -ForegroundColor White
    Write-Host "   gym-door-bridge pair CODE - Pair with platform" -ForegroundColor White
    Write-Host "   gym-door-bridge unpair    - Unpair from platform" -ForegroundColor White
    Write-Host "   net stop GymDoorBridge    - Stop service" -ForegroundColor White
    Write-Host "   net start GymDoorBridge   - Start service" -ForegroundColor White
    
    Write-Host "`n🎉 Gym Door Bridge is now installed and running!" -ForegroundColor Green
    
} catch {
    Write-Host "❌ Installation failed: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "Please check the error and try again, or contact support." -ForegroundColor Yellow
    exit 1
} finally {
    # Cleanup
    if (Test-Path $tempZip) {
        Remove-Item $tempZip -Force -ErrorAction SilentlyContinue
    }
    if (Test-Path $tempExtract) {
        Remove-Item $tempExtract -Recurse -Force -ErrorAction SilentlyContinue
    }
}