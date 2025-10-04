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
                    Write-Host "Uninstalling existing service..." -ForegroundColor White
                    $uninstallProcess = Start-Process -FilePath $existingExePath -ArgumentList "uninstall" -Wait -PassThru -NoNewWindow -RedirectStandardOutput "$env:TEMP\uninstall-output.log" -RedirectStandardError "$env:TEMP\uninstall-error.log"
                    
                    if ($uninstallProcess.ExitCode -eq 0) {
                        Write-Host "✅ Existing service uninstalled" -ForegroundColor Green
                    } else {
                        Write-Host "⚠️  Service uninstall returned exit code $($uninstallProcess.ExitCode)" -ForegroundColor Yellow
                        
                        # Show uninstall errors if any
                        if (Test-Path "$env:TEMP\uninstall-error.log") {
                            $uninstallError = Get-Content "$env:TEMP\uninstall-error.log" -Raw
                            if ($uninstallError) {
                                Write-Host "Uninstall details: $uninstallError" -ForegroundColor Yellow
                            }
                        }
                    }
                    
                    # Wait longer for the service to be fully removed and files to be unlocked
                    Write-Host "Waiting for service cleanup..." -ForegroundColor White
                    Start-Sleep -Seconds 5
                    
                    # Ensure all processes are stopped
                    Get-Process -Name "gym-door-bridge*" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
                    Start-Sleep -Seconds 2
                }
            } catch {
                Write-Host "⚠️  Could not uninstall existing service: $($_.Exception.Message)" -ForegroundColor Yellow
                Write-Host "Continuing with installation..." -ForegroundColor White
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
                
                # Stop any running processes that might be using the file
                Get-Process -Name "gym-door-bridge*" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
                Start-Sleep -Seconds 3
                
                # Try to remove the existing file with retry logic
                $removed = $false
                for ($i = 0; $i -lt 10; $i++) {
                    try {
                        Remove-Item $targetPath -Force -ErrorAction Stop
                        $removed = $true
                        break
                    } catch {
                        Write-Host "⚠️  File in use, waiting... (attempt $($i+1)/10)" -ForegroundColor Yellow
                        Start-Sleep -Seconds 2
                    }
                }
                
                if (-not $removed) {
                    Write-Host "⚠️  Could not remove existing file, trying alternative approach..." -ForegroundColor Yellow
                    # Try to rename the old file instead of deleting it
                    try {
                        $backupPath = "$targetPath.backup.$(Get-Date -Format 'yyyyMMdd-HHmmss')"
                        Move-Item $targetPath $backupPath -Force
                        Write-Host "✅ Old file backed up to: $backupPath" -ForegroundColor Green
                    } catch {
                        throw "Cannot replace existing executable. Please stop the service manually and try again."
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
        
        # First, test server connectivity
        Write-Host "🔍 Testing server connectivity..." -ForegroundColor Yellow
        try {
            $healthResponse = Invoke-WebRequest -Uri "$ServerUrl/api/v1/health" -UseBasicParsing -TimeoutSec 10 -ErrorAction Stop
            Write-Host "✅ Server is reachable (HTTP $($healthResponse.StatusCode))" -ForegroundColor Green
        } catch {
            Write-Host "⚠️  Server connectivity issue: $($_.Exception.Message)" -ForegroundColor Yellow
            Write-Host "Pairing may fail. Check your internet connection and try again later." -ForegroundColor Yellow
        }
        
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
            
            Write-Host "Attempting to pair with code: $PairCode" -ForegroundColor White
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
                
                # Provide detailed error analysis
                Write-Host "⚠️  Pairing failed with exit code $($pairProcess.ExitCode)" -ForegroundColor Yellow
                
                if ($errorOutput) {
                    Write-Host "Error details: $errorOutput" -ForegroundColor Yellow
                    
                    # Check for specific error patterns
                    if ($errorOutput -match "HTTP error 500") {
                        Write-Host "🔍 Server Error (500) - This indicates a server-side issue:" -ForegroundColor Red
                        Write-Host "   • The pair code may be invalid or expired" -ForegroundColor White
                        Write-Host "   • The server may be experiencing issues" -ForegroundColor White
                        Write-Host "   • Try generating a new pair code from the admin portal" -ForegroundColor White
                    } elseif ($errorOutput -match "already paired") {
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
                        }
                    } elseif ($errorOutput -match "network\|connection\|timeout") {
                        Write-Host "🌐 Network connectivity issue detected" -ForegroundColor Red
                        Write-Host "   • Check your internet connection" -ForegroundColor White
                        Write-Host "   • Verify firewall settings allow outbound HTTPS" -ForegroundColor White
                        Write-Host "   • Try again in a few minutes" -ForegroundColor White
                    }
                }
                
                Write-Host "`nYou can pair manually later using:" -ForegroundColor Yellow
                Write-Host "gym-door-bridge pair $PairCode" -ForegroundColor White
            }
        } catch {
            Write-Host "⚠️  Pairing error: $($_.Exception.Message)" -ForegroundColor Yellow
            Write-Host "You can pair manually later using:" -ForegroundColor Yellow
            Write-Host "gym-door-bridge pair $PairCode" -ForegroundColor White
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
                Start-Sleep -Seconds 5
                $service = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
                if ($service.Status -eq "Running") {
                    Write-Host "✅ Service started successfully!" -ForegroundColor Green
                } else {
                    Write-Host "⚠️  Service failed to start. Status: $($service.Status)" -ForegroundColor Yellow
                    
                    # Provide detailed troubleshooting
                    Write-Host "🔍 Troubleshooting service startup..." -ForegroundColor Yellow
                    
                    # Check if executable exists and is accessible
                    $serviceExe = "$InstallPath\gym-door-bridge.exe"
                    if (-not (Test-Path $serviceExe)) {
                        Write-Host "❌ Service executable not found: $serviceExe" -ForegroundColor Red
                    } else {
                        Write-Host "✅ Service executable exists" -ForegroundColor Green
                        
                        # Try to run executable manually to check for errors
                        try {
                            Write-Host "Testing executable manually..." -ForegroundColor White
                            $testProcess = Start-Process -FilePath $serviceExe -ArgumentList "--help" -Wait -PassThru -NoNewWindow -RedirectStandardOutput "$env:TEMP\bridge-test.log" -RedirectStandardError "$env:TEMP\bridge-test-error.log"
                            
                            if ($testProcess.ExitCode -eq 0) {
                                Write-Host "✅ Executable runs correctly" -ForegroundColor Green
                            } else {
                                Write-Host "❌ Executable has issues (exit code: $($testProcess.ExitCode))" -ForegroundColor Red
                                if (Test-Path "$env:TEMP\bridge-test-error.log") {
                                    $testError = Get-Content "$env:TEMP\bridge-test-error.log" -Raw
                                    if ($testError) {
                                        Write-Host "Error details: $testError" -ForegroundColor Red
                                    }
                                }
                            }
                        } catch {
                            Write-Host "❌ Cannot run executable: $($_.Exception.Message)" -ForegroundColor Red
                        }
                    }
                    
                    # Check Windows Event Log for service errors
                    try {
                        $recentErrors = Get-WinEvent -FilterHashtable @{LogName='System'; Level=2; StartTime=(Get-Date).AddMinutes(-5)} -ErrorAction SilentlyContinue |
                                       Where-Object { $_.Message -like "*GymDoorBridge*" } |
                                       Select-Object -First 3
                        
                        if ($recentErrors) {
                            Write-Host "Recent service errors from Event Log:" -ForegroundColor Red
                            foreach ($error in $recentErrors) {
                                Write-Host "  [$($error.TimeCreated)] $($error.Message)" -ForegroundColor Red
                            }
                        }
                    } catch {
                        Write-Host "Could not check Event Log for errors" -ForegroundColor Yellow
                    }
                    
                    Write-Host "`n📋 Service Troubleshooting Steps:" -ForegroundColor Cyan
                    Write-Host "1. Check Windows Event Viewer > System for detailed error messages" -ForegroundColor White
                    Write-Host "2. Ensure the device is paired: gym-door-bridge pair $PairCode" -ForegroundColor White
                    Write-Host "3. Try starting manually: net start GymDoorBridge" -ForegroundColor White
                    Write-Host "4. Check config file: $InstallPath\config.yaml" -ForegroundColor White
                }
            } catch {
                Write-Host "❌ Failed to start service: $($_.Exception.Message)" -ForegroundColor Red
                
                # Check for specific error patterns
                $errorMsg = $_.Exception.Message
                if ($errorMsg -match "access.*denied|privilege") {
                    Write-Host "🔒 Permission issue detected. Try running as Administrator." -ForegroundColor Yellow
                } elseif ($errorMsg -match "service.*not.*found") {
                    Write-Host "🔍 Service not properly installed. Try reinstalling." -ForegroundColor Yellow
                } else {
                    Write-Host "🔍 Check Windows Event Viewer for detailed error information." -ForegroundColor Yellow
                }
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
    
    # Final status check
    $finalService = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
    if ($finalService -and $finalService.Status -eq "Running") {
        Write-Host "`n🎉 Gym Door Bridge is now installed and running!" -ForegroundColor Green
        
        # Check if paired
        $configPath = "$InstallPath\config.yaml"
        if (Test-Path $configPath) {
            $configContent = Get-Content $configPath -Raw
            if ($configContent -match 'device_id:\s*"([^"]+)"' -and $matches[1] -ne "") {
                Write-Host "✅ Device is paired and ready!" -ForegroundColor Green
            } else {
                Write-Host "⚠️  Device is not paired yet. Use: gym-door-bridge pair YOUR_CODE" -ForegroundColor Yellow
            }
        }
    } else {
        Write-Host "`n⚠️  Installation completed but service needs attention" -ForegroundColor Yellow
        Write-Host "Check the troubleshooting steps above or run 'net start GymDoorBridge'" -ForegroundColor White
    }
    
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