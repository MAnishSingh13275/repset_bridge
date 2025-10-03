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
    Write-Host "Press any key to continue..." -ForegroundColor Yellow
    $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
    return
}

Write-Host "🚀 Gym Door Bridge Installation" -ForegroundColor Cyan
Write-Host "================================" -ForegroundColor Cyan

# Check if service already exists
$existingService = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
if ($existingService) {
    Write-Host "⚠️  Gym Door Bridge is already installed!" -ForegroundColor Yellow
    Write-Host "Service Status: $($existingService.Status)" -ForegroundColor White
    
    # If pair code is provided, automatically reinstall and re-pair
    if ($PairCode) {
        Write-Host "🔄 Pair code provided - will reinstall and re-pair automatically..." -ForegroundColor Green
        $Force = $true
    } elseif (-not $Force) {
        Write-Host "Use -Force parameter to reinstall or run 'gym-door-bridge status' to check status." -ForegroundColor Yellow
        Write-Host "Press any key to continue..." -ForegroundColor Yellow
        $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
        return
    }
}

try {
    # Download latest release with multiple fallback methods
    Write-Host "📥 Downloading latest Gym Door Bridge..." -ForegroundColor Green
    $downloadUrl = "https://github.com/MAnishSingh13275/repset_bridge/releases/latest/download/gym-door-bridge-windows.zip"
    $tempZip = "$env:TEMP\gym-door-bridge.zip"
    $tempExtract = "$env:TEMP\gym-door-bridge"
    
    # Create temp directory
    if (Test-Path $tempExtract) {
        Remove-Item $tempExtract -Recurse -Force
    }
    New-Item -ItemType Directory -Path $tempExtract -Force | Out-Null
    
    # Try multiple download methods
    $downloadSuccess = $false
    
    # Method 1: Invoke-WebRequest
    try {
        Write-Host "Trying download method 1..." -ForegroundColor Yellow
        Invoke-WebRequest -Uri $downloadUrl -OutFile $tempZip -UseBasicParsing
        $downloadSuccess = $true
        Write-Host "✅ Download method 1 successful" -ForegroundColor Green
    } catch {
        Write-Host "⚠️  Download method 1 failed: $($_.Exception.Message)" -ForegroundColor Yellow
    }
    
    # Method 2: WebClient (fallback)
    if (-not $downloadSuccess) {
        try {
            Write-Host "Trying download method 2..." -ForegroundColor Yellow
            $webClient = New-Object System.Net.WebClient
            $webClient.DownloadFile($downloadUrl, $tempZip)
            $downloadSuccess = $true
            Write-Host "✅ Download method 2 successful" -ForegroundColor Green
        } catch {
            Write-Host "⚠️  Download method 2 failed: $($_.Exception.Message)" -ForegroundColor Yellow
        }
    }
    
    # Method 3: BITS Transfer (fallback)
    if (-not $downloadSuccess) {
        try {
            Write-Host "Trying download method 3..." -ForegroundColor Yellow
            Import-Module BitsTransfer -ErrorAction SilentlyContinue
            Start-BitsTransfer -Source $downloadUrl -Destination $tempZip
            $downloadSuccess = $true
            Write-Host "✅ Download method 3 successful" -ForegroundColor Green
        } catch {
            Write-Host "⚠️  Download method 3 failed: $($_.Exception.Message)" -ForegroundColor Yellow
        }
    }
    
    if (-not $downloadSuccess) {
        throw "All download methods failed. Please check your internet connection and try again."
    }
    
    # Extract with error handling
    Write-Host "📦 Extracting files..." -ForegroundColor Green
    try {
        Expand-Archive -Path $tempZip -DestinationPath $tempExtract -Force
    } catch {
        # Try alternative extraction method
        Write-Host "⚠️  Standard extraction failed, trying alternative method..." -ForegroundColor Yellow
        $shell = New-Object -ComObject Shell.Application
        $zip = $shell.NameSpace($tempZip)
        $destination = $shell.NameSpace($tempExtract)
        $destination.CopyHere($zip.Items(), 4)
    }
    
    # Find executable with multiple search patterns
    $exePath = $null
    $searchPaths = @(
        "gym-door-bridge.exe",
        "*/gym-door-bridge.exe",
        "build/gym-door-bridge.exe"
    )
    
    foreach ($pattern in $searchPaths) {
        $found = Get-ChildItem -Path $tempExtract -Name $pattern -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
        if ($found) {
            $exePath = $found
            break
        }
    }
    
    if (-not $exePath) {
        # List contents for debugging
        Write-Host "⚠️  Executable not found. Package contents:" -ForegroundColor Yellow
        Get-ChildItem -Path $tempExtract -Recurse | ForEach-Object { Write-Host "  $($_.FullName)" -ForegroundColor White }
        throw "gym-door-bridge.exe not found in downloaded package"
    }
    
    $fullExePath = $exePath.FullName
    Write-Host "✅ Found executable: $fullExePath" -ForegroundColor Green
    
    # Stop existing service if running
    if ($existingService) {
        Write-Host "🛑 Stopping existing service..." -ForegroundColor Yellow
        try {
            Stop-Service -Name "GymDoorBridge" -Force -ErrorAction SilentlyContinue
            Start-Sleep -Seconds 2
            & "$fullExePath" uninstall -ErrorAction SilentlyContinue
            Write-Host "✅ Existing service stopped and uninstalled" -ForegroundColor Green
        } catch {
            Write-Host "⚠️  Warning: Could not fully uninstall existing service: $($_.Exception.Message)" -ForegroundColor Yellow
        }
    }
    
    # Run installation with detailed error handling
    Write-Host "⚙️  Installing Gym Door Bridge..." -ForegroundColor Green
    try {
        $installProcess = Start-Process -FilePath $fullExePath -ArgumentList "install" -Wait -PassThru -NoNewWindow -RedirectStandardOutput "$env:TEMP\install-output.log" -RedirectStandardError "$env:TEMP\install-error.log"
        
        if ($installProcess.ExitCode -ne 0) {
            $errorOutput = ""
            if (Test-Path "$env:TEMP\install-error.log") {
                $errorOutput = Get-Content "$env:TEMP\install-error.log" -Raw
            }
            throw "Installation failed with exit code $($installProcess.ExitCode). Error: $errorOutput"
        }
        Write-Host "✅ Service installation completed" -ForegroundColor Green
    } catch {
        Write-Host "❌ Installation error: $($_.Exception.Message)" -ForegroundColor Red
        Write-Host "Attempting alternative installation method..." -ForegroundColor Yellow
        
        # Try copying files manually and installing
        try {
            # Copy executable to Program Files
            $targetPath = "$InstallPath\gym-door-bridge.exe"
            New-Item -ItemType Directory -Path $InstallPath -Force | Out-Null
            Copy-Item -Path $fullExePath -Destination $targetPath -Force
            
            # Try installation again
            $installProcess2 = Start-Process -FilePath $targetPath -ArgumentList "install" -Wait -PassThru -NoNewWindow
            if ($installProcess2.ExitCode -eq 0) {
                Write-Host "✅ Alternative installation method successful" -ForegroundColor Green
            } else {
                throw "Alternative installation also failed with exit code $($installProcess2.ExitCode)"
            }
        } catch {
            throw "Both installation methods failed: $($_.Exception.Message)"
        }
    }
    
    Write-Host "✅ Installation completed successfully!" -ForegroundColor Green
    
    # Pair device if pair code provided
    if ($PairCode) {
        Write-Host "🔗 Pairing device with platform..." -ForegroundColor Green
        
        # First, try to unpair if already paired (for re-pairing scenarios)
        try {
            $pairExePath = "$InstallPath\gym-door-bridge.exe"
            if (-not (Test-Path $pairExePath)) {
                $pairExePath = $fullExePath
            }
            
            # Check if already paired by trying to get status
            $statusProcess = Start-Process -FilePath $pairExePath -ArgumentList "status" -Wait -PassThru -NoNewWindow -RedirectStandardOutput "$env:TEMP\status-output.log" -RedirectStandardError "$env:TEMP\status-error.log"
            
            if ($statusProcess.ExitCode -eq 0) {
                $statusOutput = Get-Content "$env:TEMP\status-output.log" -Raw -ErrorAction SilentlyContinue
                if ($statusOutput -and $statusOutput -match "PAIRED") {
                    Write-Host "🔄 Device is already paired - unpairing first..." -ForegroundColor Yellow
                    $unpairProcess = Start-Process -FilePath $pairExePath -ArgumentList "unpair" -Wait -PassThru -NoNewWindow
                    if ($unpairProcess.ExitCode -eq 0) {
                        Write-Host "✅ Successfully unpaired existing device" -ForegroundColor Green
                    }
                }
            }
        } catch {
            Write-Host "⚠️  Could not check existing pairing status: $($_.Exception.Message)" -ForegroundColor Yellow
        }
        
        try {
            $pairExePath = "$InstallPath\gym-door-bridge.exe"
            if (-not (Test-Path $pairExePath)) {
                # Try to find the executable in the temp location
                $pairExePath = $fullExePath
            }
            
            $pairProcess = Start-Process -FilePath $pairExePath -ArgumentList "pair", $PairCode -Wait -PassThru -NoNewWindow -RedirectStandardOutput "$env:TEMP\pair-output.log" -RedirectStandardError "$env:TEMP\pair-error.log"
            
            if ($pairProcess.ExitCode -eq 0) {
                Write-Host "✅ Device paired successfully!" -ForegroundColor Green
                
                # Verify pairing by checking config
                $configPath = "$InstallPath\config.yaml"
                if (Test-Path $configPath) {
                    $configContent = Get-Content $configPath -Raw
                    if ($configContent -match 'device_id:\s*"([^"]+)"') {
                        Write-Host "Device ID: $($matches[1])" -ForegroundColor White
                    }
                }
            } else {
                $errorOutput = ""
                if (Test-Path "$env:TEMP\pair-error.log") {
                    $errorOutput = Get-Content "$env:TEMP\pair-error.log" -Raw
                }
                Write-Host "⚠️  Pairing failed with exit code $($pairProcess.ExitCode)" -ForegroundColor Yellow
                if ($errorOutput) {
                    Write-Host "Error details: $errorOutput" -ForegroundColor Yellow
                }
                Write-Host "You can pair manually later using:" -ForegroundColor Yellow
                Write-Host "   gym-door-bridge pair $PairCode" -ForegroundColor White
            }
        } catch {
            Write-Host "⚠️  Pairing error: $($_.Exception.Message)" -ForegroundColor Yellow
            Write-Host "You can pair manually later using:" -ForegroundColor Yellow
            Write-Host "   gym-door-bridge pair $PairCode" -ForegroundColor White
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
    Write-Host "Press any key to continue..." -ForegroundColor Yellow
    $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
    return
} finally {
    # Cleanup
    if (Test-Path $tempZip) {
        Remove-Item $tempZip -Force -ErrorAction SilentlyContinue
    }
    if (Test-Path $tempExtract) {
        Remove-Item $tempExtract -Recurse -Force -ErrorAction SilentlyContinue
    }
    
    # Prevent PowerShell from closing when run from command line
    if ($Host.Name -eq "ConsoleHost") {
        Write-Host "`nInstallation script completed. Press any key to continue..." -ForegroundColor Yellow
        $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
    }
}