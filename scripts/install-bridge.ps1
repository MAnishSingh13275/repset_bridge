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
    
    # Find executable with better search logic
    Write-Host "🔍 Searching for executable..." -ForegroundColor Yellow
    $exePath = $null
    
    # First, list all contents for debugging
    Write-Host "Package contents:" -ForegroundColor Yellow
    $allFiles = Get-ChildItem -Path $tempExtract -Recurse -File
    foreach ($file in $allFiles) {
        Write-Host "  $($file.FullName)" -ForegroundColor White
    }
    
    # Search for the executable
    $exePath = Get-ChildItem -Path $tempExtract -Filter "gym-door-bridge.exe" -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
    
    if (-not $exePath) {
        # Try alternative search methods
        Write-Host "⚠️  Direct search failed, trying alternative methods..." -ForegroundColor Yellow
        
        # Method 1: Search by name pattern
        $exePath = $allFiles | Where-Object { $_.Name -eq "gym-door-bridge.exe" } | Select-Object -First 1
        
        # Method 2: Search for any .exe file
        if (-not $exePath) {
            $exeFiles = $allFiles | Where-Object { $_.Extension -eq ".exe" }
            if ($exeFiles.Count -gt 0) {
                Write-Host "Found .exe files:" -ForegroundColor Yellow
                foreach ($exe in $exeFiles) {
                    Write-Host "  $($exe.FullName)" -ForegroundColor White
                }
                # Use the first .exe file found
                $exePath = $exeFiles[0]
                Write-Host "Using: $($exePath.FullName)" -ForegroundColor Yellow
            }
        }
    }
    
    if (-not $exePath) {
        throw "No executable found in downloaded package"
    }
    
    $fullExePath = $exePath.FullName
    Write-Host "✅ Found executable: $fullExePath" -ForegroundColor Green
    
    # Stop existing service if running
    if ($existingService) {
        Write-Host "🛑 Stopping existing service..." -ForegroundColor Yellow
        try {
            # Force stop the service with timeout
            $stopTimeout = 30 # seconds
            $stopWatch = [System.Diagnostics.Stopwatch]::StartNew()
            
            Stop-Service -Name "GymDoorBridge" -Force -ErrorAction SilentlyContinue
            
            # Wait for service to stop with timeout
            do {
                Start-Sleep -Seconds 1
                $service = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
            } while ($service.Status -eq "Running" -and $stopWatch.Elapsed.TotalSeconds -lt $stopTimeout)
            
            $stopWatch.Stop()
            
            if ($service.Status -eq "Stopped") {
                Write-Host "✅ Service stopped successfully" -ForegroundColor Green
            } else {
                Write-Host "⚠️  Service did not stop within timeout, forcing termination..." -ForegroundColor Yellow
                # Try to kill the process
                Get-Process -Name "gym-door-bridge" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
                Start-Sleep -Seconds 2
            }
            
            # Kill any remaining processes that might be using the executable
            Get-Process -Name "gym-door-bridge*" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
            Start-Sleep -Seconds 2
            
            # Try to uninstall the service
            try {
                $existingExePath = "$InstallPath\gym-door-bridge.exe"
                if (Test-Path $existingExePath) {
                    $uninstallProcess = Start-Process -FilePath $existingExePath -ArgumentList "uninstall" -Wait -PassThru -NoNewWindow
                    if ($uninstallProcess.ExitCode -eq 0) {
                        Write-Host "✅ Existing service uninstalled" -ForegroundColor Green
                    } else {
                        Write-Host "⚠️  Service uninstall returned exit code $($uninstallProcess.ExitCode)" -ForegroundColor Yellow
                    }
                    
                    # Wait a moment for the service to be fully removed
                    Start-Sleep -Seconds 3
                }
            } catch {
                Write-Host "⚠️  Could not uninstall existing service, continuing..." -ForegroundColor Yellow
            }
            
        } catch {
            Write-Host "⚠️  Warning: Could not fully stop existing service: $($_.Exception.Message)" -ForegroundColor Yellow
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
            # Copy executable to Program Files with retry logic
            $targetPath = "$InstallPath\gym-door-bridge.exe"
            New-Item -ItemType Directory -Path $InstallPath -Force | Out-Null
            
            # If target file exists and is in use, try to replace it
            if (Test-Path $targetPath) {
                Write-Host "⚠️  Target file exists, attempting to replace..." -ForegroundColor Yellow
                
                # Try to remove the existing file
                for ($i = 0; $i -lt 5; $i++) {
                    try {
                        Remove-Item $targetPath -Force
                        break
                    } catch {
                        Write-Host "⚠️  File in use, waiting... (attempt $($i+1)/5)" -ForegroundColor Yellow
                        Start-Sleep -Seconds 2
                    }
                }
            }
            
            # Copy the new file
            Copy-Item -Path $fullExePath -Destination $targetPath -Force
            Write-Host "✅ Executable copied to $targetPath" -ForegroundColor Green
            
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
    
    # Verify service installation
    Start-Sleep -Seconds 2
    $newService = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
    if ($newService) {
        Write-Host "✅ Service verification: $($newService.Status)" -ForegroundColor Green
    } else {
        Write-Host "⚠️  Service not found after installation" -ForegroundColor Yellow
    }
    
    # Pair device if pair code provided
    if ($PairCode) {
        Write-Host "🔗 Pairing device with platform..." -ForegroundColor Green
        
        # First, try to unpair if already paired (for re-pairing scenarios)
        $pairExePath = "$InstallPath\gym-door-bridge.exe"
        if (-not (Test-Path $pairExePath)) {
            $pairExePath = $fullExePath
        }
        
        # Always try to unpair first to ensure clean pairing
        Write-Host "🔄 Ensuring clean pairing state..." -ForegroundColor Yellow
        try {
            $unpairProcess = Start-Process -FilePath $pairExePath -ArgumentList "unpair" -Wait -PassThru -NoNewWindow -RedirectStandardOutput "$env:TEMP\unpair-output.log" -RedirectStandardError "$env:TEMP\unpair-error.log"
            if ($unpairProcess.ExitCode -eq 0) {
                Write-Host "✅ Successfully cleared existing pairing" -ForegroundColor Green
            } else {
                Write-Host "ℹ️  No existing pairing to clear" -ForegroundColor White
            }
        } catch {
            Write-Host "ℹ️  No existing pairing to clear" -ForegroundColor White
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
                
                # Check if it's an "already paired" error and try to handle it
                if ($errorOutput -and $errorOutput -match "already paired") {
                    Write-Host "🔄 Device reports already paired, attempting force re-pair..." -ForegroundColor Yellow
                    
                    # Try unpair again with more force
                    $forceUnpairProcess = Start-Process -FilePath $pairExePath -ArgumentList "unpair" -Wait -PassThru -NoNewWindow
                    Start-Sleep -Seconds 2
                    
                    # Try pairing again
                    $retryPairProcess = Start-Process -FilePath $pairExePath -ArgumentList "pair", $PairCode -Wait -PassThru -NoNewWindow
                    if ($retryPairProcess.ExitCode -eq 0) {
                        Write-Host "✅ Device paired successfully on retry!" -ForegroundColor Green
                    } else {
                        Write-Host "⚠️  Retry pairing also failed" -ForegroundColor Yellow
                        Write-Host "You can pair manually later using:" -ForegroundColor Yellow
                        Write-Host "   gym-door-bridge unpair && gym-door-bridge pair $PairCode" -ForegroundColor White
                    }
                } else {
                    Write-Host "⚠️  Pairing failed with exit code $($pairProcess.ExitCode)" -ForegroundColor Yellow
                    if ($errorOutput) {
                        Write-Host "Error details: $errorOutput" -ForegroundColor Yellow
                    }
                    Write-Host "You can pair manually later using:" -ForegroundColor Yellow
                    Write-Host "   gym-door-bridge pair $PairCode" -ForegroundColor White
                }
            }
        } catch {
            Write-Host "⚠️  Pairing error: $($_.Exception.Message)" -ForegroundColor Yellow
            Write-Host "You can pair manually later using:" -ForegroundColor Yellow
            Write-Host "   gym-door-bridge pair $PairCode" -ForegroundColor White
        }
    }
    
    # Check service status with better verification
    Write-Host "🔍 Verifying service status..." -ForegroundColor Yellow
    Start-Sleep -Seconds 3
    
    $service = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
    if ($service) {
        Write-Host "Service Status: $($service.Status)" -ForegroundColor White
        
        if ($service.Status -eq "Running") {
            Write-Host "✅ Service is running successfully!" -ForegroundColor Green
            
            # Test API endpoint
            try {
                Start-Sleep -Seconds 2
                $apiResponse = Invoke-WebRequest -Uri "http://localhost:8081/api/v1/health" -UseBasicParsing -TimeoutSec 10
                Write-Host "✅ API is responding: HTTP $($apiResponse.StatusCode)" -ForegroundColor Green
            } catch {
                Write-Host "⚠️  API not responding yet (this is normal, may take a moment to start)" -ForegroundColor Yellow
            }
        } else {
            Write-Host "⚠️  Service installed but not running. Starting now..." -ForegroundColor Yellow
            try {
                Start-Service -Name "GymDoorBridge"
                Start-Sleep -Seconds 3
                $service = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
                if ($service.Status -eq "Running") {
                    Write-Host "✅ Service started successfully!" -ForegroundColor Green
                } else {
                    Write-Host "⚠️  Service failed to start. Status: $($service.Status)" -ForegroundColor Yellow
                }
            } catch {
                Write-Host "❌ Failed to start service: $($_.Exception.Message)" -ForegroundColor Red
            }
        }
    } else {
        Write-Host "❌ Service not found after installation" -ForegroundColor Red
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