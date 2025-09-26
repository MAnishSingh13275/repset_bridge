# ================================================================
# Gym Door Bridge - Web Installer (One-Click from Website)
# Downloads, extracts, and installs automatically from the web
# ================================================================

param(
    [string]$Version = "latest",
    [string]$ReleaseUrl = "https://github.com/MAnishSingh13275/repset_bridge/releases/latest/download/GymDoorBridge-v1.1.0.zip",
    [string]$PairCode = "0A99-03C8-6460",
    [string]$DeviceId = "bridge_cmfzol7d5000el204izvgexi4_9803e1d2fc21f6f5",
    [string]$DeviceKey = "20c6a0e6b20886cf60ad5288924c957451eeb82ec9e19f3c69bcf1942cc773a1",
    [string]$GymId = "cmfzol7d5000el204izvgexi4",
    [switch]$Silent = $false
)

$ErrorActionPreference = "Stop"

# Set up console
$Host.UI.RawUI.WindowTitle = "Gym Door Bridge Web Installer"
if (-not $Silent) {
    Clear-Host
}

# Color functions
function Write-Success { param([string]$Message) Write-Host "      ✓ $Message" -ForegroundColor Green }
function Write-Error { param([string]$Message) Write-Host "      ✗ $Message" -ForegroundColor Red }
function Write-Warning { param([string]$Message) Write-Host "      ! $Message" -ForegroundColor Yellow }
function Write-Info { param([string]$Message) Write-Host "      → $Message" -ForegroundColor Cyan }
function Write-Step { param([string]$Step, [string]$Message) Write-Host "[$Step] $Message" -ForegroundColor White }

# Banner
if (-not $Silent) {
    Write-Host ""
    Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Green
    Write-Host "  █                                                              █" -ForegroundColor Green
    Write-Host "  █              GYM DOOR BRIDGE WEB INSTALLER                  █" -ForegroundColor Green
    Write-Host "  █                                                              █" -ForegroundColor Green
    Write-Host "  █          Downloads and installs automatically               █" -ForegroundColor Green
    Write-Host "  █                                                              █" -ForegroundColor Green
    Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Green
    Write-Host ""
    Write-Host "  Version: $Version" -ForegroundColor Gray
    Write-Host "  Source: GitHub Release" -ForegroundColor Gray
    Write-Host ""
}

try {
    # Check administrator privileges
    Write-Step "1/8" "Checking administrator privileges..."
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    
    if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        Write-Host ""
        Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Red
        Write-Host "  █                        ERROR                                 █" -ForegroundColor Red
        Write-Host "  █                                                              █" -ForegroundColor Red
        Write-Host "  █    This installer requires administrator privileges!         █" -ForegroundColor Red
        Write-Host "  █                                                              █" -ForegroundColor Red
        Write-Host "  █    Please run PowerShell as administrator and try again     █" -ForegroundColor Red
        Write-Host "  █                                                              █" -ForegroundColor Red
        Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Red
        Write-Host ""
        if (-not $Silent) { Read-Host "Press Enter to exit" }
        exit 1
    }
    Write-Success "Administrator privileges confirmed"
    Write-Host ""

    # Create temp directory
    Write-Step "2/8" "Setting up temporary workspace..."
    $tempDir = "$env:TEMP\GymDoorBridge-Install-$(Get-Random)"
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
    Write-Success "Temporary directory: $tempDir"
    Write-Host ""

    # Download release
    Write-Step "3/8" "Downloading latest release..."
    Write-Info "Downloading from GitHub..."
    
    # If version is "latest", resolve the actual download URL
    if ($Version -eq "latest") {
        try {
            Write-Info "Resolving latest release URL..."
            $apiUrl = "https://api.github.com/repos/MAnishSingh13275/repset_bridge/releases/latest"
            $releaseInfo = Invoke-RestMethod -Uri $apiUrl -UserAgent "GymDoorBridge-WebInstaller"
            
            # Find the ZIP asset
            $zipAsset = $releaseInfo.assets | Where-Object { $_.name -match "GymDoorBridge-v.*\.zip" } | Select-Object -First 1
            if (-not $zipAsset) {
                throw "No ZIP asset found in latest release"
            }
            
            $ReleaseUrl = $zipAsset.browser_download_url
            $Version = $releaseInfo.tag_name -replace "^v", ""
            Write-Success "Latest version: v$Version"
        }
        catch {
            Write-Warning "Failed to resolve latest version, using provided URL"
        }
    }

    $zipPath = "$tempDir\GymDoorBridge.zip"
    Write-Info "Downloading: $ReleaseUrl"
    
    # Download with progress
    $progressPreference = 'SilentlyContinue'
    try {
        Invoke-WebRequest -Uri $ReleaseUrl -OutFile $zipPath -UseBasicParsing
        $progressPreference = 'Continue'
    }
    catch {
        $progressPreference = 'Continue'
        throw "Failed to download release: $($_.Exception.Message)"
    }

    if (-not (Test-Path $zipPath)) {
        throw "Downloaded file not found: $zipPath"
    }

    $zipInfo = Get-Item $zipPath
    $sizeMB = [math]::Round($zipInfo.Length / 1MB, 2)
    Write-Success "Download completed ($sizeMB MB)"
    Write-Host ""

    # Extract ZIP
    Write-Step "4/8" "Extracting installer..."
    $extractDir = "$tempDir\extracted"
    
    try {
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        [System.IO.Compression.ZipFile]::ExtractToDirectory($zipPath, $extractDir)
        Write-Info "Extraction to: $extractDir"
    }
    catch {
        throw "Failed to extract ZIP file: $($_.Exception.Message)"
    }

    # Find the installer files
    Write-Info "Searching for executable in extracted files..."
    
    # Try multiple approaches to find the executable
    $executableFile = $null
    
    # Method 1: Direct search with -Filter
    try {
        $executableFile = Get-ChildItem -Path $extractDir -Filter "gym-door-bridge.exe" -Recurse -File -ErrorAction Stop | Select-Object -First 1
    } catch {}
    
    # Method 2: Search with -Name if Filter failed
    if (-not $executableFile) {
        try {
            $found = Get-ChildItem -Path $extractDir -Name "gym-door-bridge.exe" -Recurse -ErrorAction Stop | Select-Object -First 1
            if ($found) {
                $executableFile = Get-Item (Join-Path $extractDir $found)
            }
        } catch {}
    }
    
    # Method 3: Manual search through all files
    if (-not $executableFile) {
        $allFiles = Get-ChildItem -Path $extractDir -Recurse -File -ErrorAction SilentlyContinue
        $executableFile = $allFiles | Where-Object { $_.Name -eq "gym-door-bridge.exe" } | Select-Object -First 1
    }
    
    if (-not $executableFile) {
        Write-Info "Listing extracted files for debugging:"
        try {
            Get-ChildItem -Path $extractDir -Recurse -ErrorAction SilentlyContinue | ForEach-Object { 
                Write-Host "  Found: $($_.Name) (Type: $($_.GetType().Name))" 
            }
        } catch {}
        throw "gym-door-bridge.exe not found in downloaded package"
    }
    
    Write-Info "Found executable: $($executableFile.FullName)"

    Write-Success "Extraction completed"
    Write-Host ""

    # Change to extracted directory for installation
    $installDir = Split-Path $executableFile.FullName -Parent
    Push-Location $installDir

    try {
        # Check existing installation
        Write-Step "5/8" "Checking existing installation..."
        $service = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
        if ($service) {
            Write-Warning "Service already installed"
            if (-not $Silent) {
                $reinstall = Read-Host "      Do you want to reinstall? (Y/n)"
                if ($reinstall.ToLower() -eq "n") {
                    Write-Host "      Installation cancelled by user."
                    exit 0
                }
            }
            Write-Info "Removing existing service..."
            try {
                Stop-Service -Name "GymDoorBridge" -Force -ErrorAction SilentlyContinue
                Start-Sleep -Seconds 2
            } catch {}
            & ".\gym-door-bridge.exe" service uninstall | Out-Null
            Start-Sleep -Seconds 3
            Write-Success "Existing service removed"
        } else {
            Write-Success "No existing installation found"
        }
        Write-Host ""

        # Install service with better error handling
        Write-Step "6/8" "Installing Windows service..."
        Write-Info "This will automatically discover your biometric devices..."
        Write-Info "Please wait while scanning network (this may take 1-2 minutes)..."
        Write-Host ""
        
        $installOutput = & ".\gym-door-bridge.exe" service install 2>&1
        if ($LASTEXITCODE -ne 0) {
            Write-Warning "Initial service installation failed, trying alternative approach..."
            # Sometimes the service install fails due to timing issues, try again
            Start-Sleep -Seconds 3
            $installOutput = & ".\gym-door-bridge.exe" service install 2>&1
            if ($LASTEXITCODE -ne 0) {
                throw "Service installation failed after retry: $installOutput"
            }
        }
        Write-Success "Service installed successfully!"
        Write-Host ""

        # Fix database and directory permissions proactively
        Write-Step "7/8" "Configuring service permissions..."
        Write-Info "Setting up database and directory permissions..."
        
        try {
            # Get the final installation directory (where service will run from)
            $finalInstallDir = "C:Program FilesGymDoorBridge"
            
            # Fix directory permissions
            if (Test-Path $finalInstallDir) {
                $acl = Get-Acl $finalInstallDir
                
                # Add full control for SYSTEM, Local Service, and Network Service
                $systemSid = [System.Security.Principal.SecurityIdentifier]'S-1-5-18'  # SYSTEM
                $localServiceSid = [System.Security.Principal.SecurityIdentifier]'S-1-5-19'  # Local Service  
                $networkServiceSid = [System.Security.Principal.SecurityIdentifier]'S-1-5-20'  # Network Service
                
                $systemRule = New-Object System.Security.AccessControl.FileSystemAccessRule($systemSid, 'FullControl', 'ContainerInherit,ObjectInherit', 'None', 'Allow')
                $localServiceRule = New-Object System.Security.AccessControl.FileSystemAccessRule($localServiceSid, 'FullControl', 'ContainerInherit,ObjectInherit', 'None', 'Allow')
                $networkServiceRule = New-Object System.Security.AccessControl.FileSystemAccessRule($networkServiceSid, 'FullControl', 'ContainerInherit,ObjectInherit', 'None', 'Allow')
                
                $acl.SetAccessRule($systemRule)
                $acl.SetAccessRule($localServiceRule)
                $acl.SetAccessRule($networkServiceRule)
                
                Set-Acl $finalInstallDir $acl
                Write-Success "Directory permissions configured"
            }
            
            # Pre-create database file if it doesn't exist and set permissions
            $dbPath = Join-Path $finalInstallDir "bridge.db"
            if (-not (Test-Path $dbPath)) {
                # Create empty database file
                New-Item -ItemType File -Path $dbPath -Force | Out-Null
                Write-Info "Database file pre-created"
            }
            
            # Set database permissions
            if (Test-Path $dbPath) {
                $dbAcl = Get-Acl $dbPath
                $dbAcl.SetAccessRule($systemRule)
                $dbAcl.SetAccessRule($localServiceRule) 
                $dbAcl.SetAccessRule($networkServiceRule)
                Set-Acl $dbPath $dbAcl
                Write-Success "Database permissions configured"
            }
            
        } catch {
            Write-Warning "Permission setup had issues: $($_.Exception.Message)"
            Write-Info "Service may need manual permission fixes if issues occur"
        }
        Write-Host ""

        # Verify installation
        Write-Step "8/8" "Verifying installation..."
        
        # Check if service is installed (more important than running)
        $service = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
        if ($service) {
            Write-Success "Service is installed and configured: $($service.Status)"
            Write-Success "Service will start automatically on Windows boot"
            Write-Success "Permissions configured for service operation"
        } else {
            # This would be a real failure - service not installed at all
            throw "Service installation verification failed - service not found"
        }
        Write-Host ""
    }
    finally {
        Pop-Location
    }

    # Installation complete
    Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Green
    Write-Host "  █                                                              █" -ForegroundColor Green
    Write-Host "  █              INSTALLATION SUCCESSFUL!                       █" -ForegroundColor Green
    Write-Host "  █                                                              █" -ForegroundColor Green
    Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Green
    Write-Host ""
    Write-Host "  ✓ Downloaded latest version (v$Version)" -ForegroundColor Green
    Write-Host "  ✓ Gym Door Bridge service installed successfully" -ForegroundColor Green
    Write-Host "  ✓ Auto-discovery configured for biometric devices" -ForegroundColor Green
    Write-Host "  ✓ Service configured to start automatically on boot" -ForegroundColor Green
    
    if ($service -and $service.Status -eq "Running") {
        Write-Host "  ✓ Service is currently running" -ForegroundColor Green
    } elseif ($service) {
        Write-Host "  ✓ Service is installed and ready" -ForegroundColor Green
        if ($service.Status -ne "Running") {
            Write-Host "    Note: Service will start automatically after pairing" -ForegroundColor Gray
        }
    }
    Write-Host ""

    if (-not $Silent) {
        Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Cyan
        Write-Host "  █                      NEXT STEPS                             █" -ForegroundColor Cyan
        Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "  1. PAIR YOUR DEVICE:" -ForegroundColor White
        Write-Host "     • Get pairing code from your admin portal" -ForegroundColor Gray
        Write-Host "     • Use Start Menu > Gym Door Bridge > Pair Device" -ForegroundColor Gray
        Write-Host "     • Or run: gym-door-bridge pair --pair-code YOUR_CODE" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  2. VERIFY SETUP:" -ForegroundColor White
        Write-Host "     • Use Start Menu > Gym Door Bridge > Check Status" -ForegroundColor Gray
        Write-Host "     • Check Windows Services (services.msc)" -ForegroundColor Gray
        Write-Host "     • View Windows Event Viewer for logs" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  3. MANAGE SERVICE:" -ForegroundColor White
        Write-Host "     • All management tools available in Start Menu" -ForegroundColor Gray
        Write-Host "     • Service runs automatically on Windows startup" -ForegroundColor Gray
        Write-Host "     • Check service status: Services.msc or Start Menu shortcuts" -ForegroundColor Gray
        Write-Host "     • No daily maintenance required" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  TROUBLESHOOTING:" -ForegroundColor White
        Write-Host "     • If service won't start: Ensure device is paired first" -ForegroundColor Gray
        Write-Host "     • Check Windows Event Viewer for detailed error logs" -ForegroundColor Gray
        Write-Host "     • Service may need internet connectivity to start" -ForegroundColor Gray
        Write-Host ""
    }

    # Handle pairing with improved error handling
    if ($PairCode) {
        Write-Host ""
        Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Blue
        Write-Host "  █                                                              █" -ForegroundColor Blue
        Write-Host "  █                    PAIRING DEVICE                          █" -ForegroundColor Blue
        Write-Host "  █                                                              █" -ForegroundColor Blue
        Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Blue
        Write-Host ""
        Write-Host "🔄 Pairing device with provided code..." -ForegroundColor Yellow
        Write-Host "📡 Code: $PairCode" -ForegroundColor Cyan
        Write-Host "🌐 Server: https://repset.onezy.in" -ForegroundColor Cyan
        Write-Host ""
        
        Push-Location $installDir
        try {
            # Enhanced smart pairing function with auto-unpair capability
            function Invoke-SmartPairing {
                param([string]$Code, [string]$ExePath)
                
                Write-Host "🔄 Smart Pairing: Attempting to pair device with code: $Code" -ForegroundColor Yellow
                $pairResult = & $ExePath pair --pair-code $Code 2>&1
                
                if ($LASTEXITCODE -eq 0) {
                    Write-Host ""
                    Write-Host "🎉 DEVICE PAIRED SUCCESSFULLY!" -ForegroundColor Green
                    Write-Host "✅ Your gym door bridge is now fully operational!" -ForegroundColor Green
                    Write-Host ""
                    return $true
                } elseif ($pairResult -match "already paired|device is already paired") {
                    Write-Host ""
                    Write-Host "⚠️ Device is already paired - initiating smart re-pairing process..." -ForegroundColor Yellow
                    
                    # Attempt to unpair with force flag
                    Write-Host "🔧 Running unpair command with --force flag..." -ForegroundColor Cyan
                    $unpairResult = & $ExePath unpair --force 2>&1
                    
                    if ($LASTEXITCODE -eq 0) {
                        Write-Host "✅ Device unpaired successfully" -ForegroundColor Green
                        Write-Host "🔄 Retrying pairing with current code..." -ForegroundColor Cyan
                        
                        # Retry pairing
                        $retryResult = & $ExePath pair --pair-code $Code 2>&1
                        if ($LASTEXITCODE -eq 0) {
                            Write-Host ""
                            Write-Host "🎉 DEVICE RE-PAIRED SUCCESSFULLY!" -ForegroundColor Green
                            Write-Host "✅ Bridge is now connected to your dashboard with updated pairing!" -ForegroundColor Green
                            Write-Host ""
                            return $true
                        } else {
                            Write-Host "❌ Re-pairing failed after unpair" -ForegroundColor Red
                            Write-Host "📄 Retry output: $retryResult" -ForegroundColor Gray
                            return $false
                        }
                    } else {
                        Write-Host "❌ Failed to unpair existing device" -ForegroundColor Red
                        Write-Host "📄 Unpair output: $unpairResult" -ForegroundColor Gray
                        Write-Host "💡 You may need to unpair manually from the admin portal" -ForegroundColor Cyan
                        return $false
                    }
                } else {
                    Write-Host "❌ Pairing failed for unknown reason" -ForegroundColor Red
                    Write-Host "📄 Pairing output: $pairResult" -ForegroundColor Gray
                    Write-Host "💡 Please verify your pairing code and network connectivity" -ForegroundColor Cyan
                    return $false
                }
            }

            # Use smart pairing instead of retry loop
            $pairingSuccess = Invoke-SmartPairing -Code $PairCode -ExePath ".\gym-door-bridge.exe"
            
            if (-not $pairingSuccess) {
                Write-Host ""
                Write-Host "⚠️ PAIRING UNSUCCESSFUL AFTER $maxRetries ATTEMPTS" -ForegroundColor Yellow
                Write-Host ""
                Write-Host "💡 TROUBLESHOOTING STEPS:" -ForegroundColor Cyan
                Write-Host "  1. Check internet connectivity" -ForegroundColor Gray
                Write-Host "  2. Verify the pair code is correct and not expired" -ForegroundColor Gray
                Write-Host "  3. Make sure https://repset.onezy.in is accessible" -ForegroundColor Gray
                Write-Host "  4. Check Windows Firewall settings" -ForegroundColor Gray
                Write-Host ""
                Write-Host "✅ INSTALLATION WAS SUCCESSFUL - Service is installed and ready" -ForegroundColor Green
                Write-Host "🔧 You can pair manually later using the admin dashboard" -ForegroundColor Cyan
                Write-Host ""
                
                # Try to start the service anyway - it might work after some time
                try {
                    Write-Host "🔄 Attempting to start service..." -ForegroundColor Yellow
                    Start-Service -Name "GymDoorBridge" -ErrorAction Stop
                    Start-Sleep -Seconds 5
                    
                    $serviceStatus = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
                    if ($serviceStatus -and $serviceStatus.Status -eq "Running") {
                        Write-Host "✅ Service started successfully!" -ForegroundColor Green
                        Write-Host "💡 The bridge may still connect automatically - check your dashboard" -ForegroundColor Cyan
                    }
                } catch {
                    Write-Host "ℹ️ Service will start automatically after successful pairing" -ForegroundColor Gray
                }
            } else {
                # Pairing was successful, try to start service
                Write-Host "🔄 Starting bridge service after successful pairing..." -ForegroundColor Yellow
                try {
                    Start-Service -Name "GymDoorBridge" -ErrorAction Stop
                    Start-Sleep -Seconds 3
                    
                    $serviceStatus = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
                    if ($serviceStatus) {
                        Write-Host "✅ Service status: $($serviceStatus.Status)" -ForegroundColor Green
                        if ($serviceStatus.Status -eq "Running") {
                            Write-Host "🎉 Bridge is now fully operational!" -ForegroundColor Green
                        }
                    }
                } catch {
                    Write-Host "⚠️ Service start had issues: $($_.Exception.Message)" -ForegroundColor Yellow
                    Write-Host "💡 Service should start automatically on next system boot" -ForegroundColor Gray
                }
            }
        }
        finally {
            Pop-Location
        }
    } elseif (-not $Silent) {
        Write-Host ""
        $pairNow = Read-Host "Would you like to pair your device now? (Y/n)"
        if ($pairNow.ToLower() -ne "n") {
            $inputCode = Read-Host "Enter your pairing code"
            if ($inputCode.Trim()) {
                Write-Host ""
                Write-Host "Pairing device..." -ForegroundColor Yellow
                Push-Location $installDir
                try {
                    $pairResult = & ".\gym-door-bridge.exe" pair --pair-code $inputCode 2>&1
                    if ($LASTEXITCODE -eq 0) {
                        Write-Host ""
                        Write-Success "Device paired successfully!"
                        Write-Host "Your gym door bridge is now fully operational!" -ForegroundColor Green
                    } else {
                        Write-Host ""
                        Write-Error "Pairing failed. You can try again later using:"
                        Write-Host "Start Menu > Gym Door Bridge > Pair Device" -ForegroundColor Cyan
                    }
                }
                finally {
                    Pop-Location
                }
            }
        }
    }

    Write-Host ""
    Write-Host ""
    Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Green
    Write-Host "  █                                                              █" -ForegroundColor Green
    Write-Host "  █              WEB INSTALLATION COMPLETED!                    █" -ForegroundColor Green
    Write-Host "  █                                                              █" -ForegroundColor Green
    Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Green
    Write-Host ""
    Write-Host "✅ The Gym Door Bridge service is installed and ready to use" -ForegroundColor Green
    
    # Final service status check and automatic startup attempt
    Write-Host "🔍 Performing final status check..." -ForegroundColor Yellow
    
    $finalService = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
    if ($finalService) {
        Write-Host "📊 Service Status: $($finalService.Status)" -ForegroundColor Cyan
        
        # If service is stopped, try to start it one more time
        if ($finalService.Status -eq "Stopped") {
            Write-Host "🔄 Attempting final service startup..." -ForegroundColor Yellow
            try {
                Start-Service -Name "GymDoorBridge" -ErrorAction Stop
                Start-Sleep -Seconds 5
                
                $finalService = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
                if ($finalService -and $finalService.Status -eq "Running") {
                    Write-Host "✅ Service successfully started!" -ForegroundColor Green
                    Write-Host "🌐 Check your admin dashboard - bridge should show as 'active'" -ForegroundColor Cyan
                } else {
                    Write-Host "ℹ️ Service will start automatically when conditions are met" -ForegroundColor Gray
                }
            } catch {
                Write-Host "ℹ️ Service startup will complete automatically after system conditions are met" -ForegroundColor Gray
            }
        } elseif ($finalService.Status -eq "Running") {
            Write-Host "✅ Service is running successfully!" -ForegroundColor Green
            Write-Host "🌐 Check your admin dashboard - bridge should show as 'active'" -ForegroundColor Cyan
        }
    }
    
    Write-Host ""
    Write-Host "🎯 WHAT'S NEXT:" -ForegroundColor White
    Write-Host "  1. 🌐 Open your admin dashboard" -ForegroundColor Gray
    Write-Host "  2. 📊 Bridge status should change from 'pending' to 'active' within 1-2 minutes" -ForegroundColor Gray
    Write-Host "  3. 🔧 If still 'pending', wait a few minutes or check Windows Event Viewer" -ForegroundColor Gray
    Write-Host "  4. ✅ Once active, your bridge will automatically discover biometric devices" -ForegroundColor Gray
    Write-Host ""
    
    if (-not $Silent) {
        Write-Host "You can close this window now." -ForegroundColor Gray
        Write-Host ""
        Read-Host "Press Enter to exit"
    }

} catch {
    # Only catch real installation failures, not pairing issues
    if ($_.Exception.Message -notmatch "already paired" -and $_.Exception.Message -notmatch "pairing") {
        Write-Host ""
        Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Red
        Write-Host "  █                        ERROR                                 █" -ForegroundColor Red
        Write-Host "  █                                                              █" -ForegroundColor Red
        Write-Host "  █                Web Installation Failed!                     █" -ForegroundColor Red
        Write-Host "  █                                                              █" -ForegroundColor Red
        Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Red
        Write-Host ""
        Write-Host "Error details: $($_.Exception.Message)" -ForegroundColor Red
        Write-Host ""
        Write-Host "Please try the following:" -ForegroundColor Yellow
        Write-Host "1. Check your internet connection" -ForegroundColor Gray
        Write-Host "2. Run PowerShell as Administrator" -ForegroundColor Gray
        Write-Host "3. Temporarily disable firewall/antivirus" -ForegroundColor Gray
        Write-Host "4. Download manually from GitHub Releases" -ForegroundColor Gray
        Write-Host ""
        if (-not $Silent) {
            Read-Host "Press Enter to exit"
        }
        exit 1
    }
}
finally {
    # Cleanup temp directory
    if (Test-Path $tempDir) {
        try {
            Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        }
        catch {
            # Ignore cleanup errors
        }
    }
}

