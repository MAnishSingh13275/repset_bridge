# ================================================================
# Gym Door Bridge - Enhanced PowerShell Installer with Auto-Unpair  
# Automatically installs and configures the Gym Door Bridge service
# Features smart pairing with automatic unpair/re-pair capability
# ================================================================

param(
    [string]$PairCode = "",
    [switch]$Silent = $false,
    [switch]$NoStart = $false
)

# Set up console
$Host.UI.RawUI.WindowTitle = "Gym Door Bridge Enhanced Installer"
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
    Write-Host "  █         GYM DOOR BRIDGE ENHANCED INSTALLER                  █" -ForegroundColor Green
    Write-Host "  █                                                              █" -ForegroundColor Green
    Write-Host "  █     Connects your biometric devices to the cloud with       █" -ForegroundColor Green
    Write-Host "  █           automatic unpair/re-pair capability               █" -ForegroundColor Green
    Write-Host "  █                                                              █" -ForegroundColor Green
    Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Green
    Write-Host ""
    Write-Host "  Version: 1.0.1-Enhanced" -ForegroundColor Gray
    Write-Host "  Platform: Windows Service" -ForegroundColor Gray  
    Write-Host "  Auto-Discovery: Enabled" -ForegroundColor Gray
    Write-Host "  Smart Pairing: Enabled" -ForegroundColor Gray
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

    # Check build requirements  
    Write-Step "2/8" "Checking build requirements..."
    $goPath = Get-Command go -ErrorAction SilentlyContinue
    if ($goPath) {
        Write-Success "Go found - can build from source"
        $canBuild = $true
    } else {
        Write-Warning "Go not found - using pre-built executable"  
        $canBuild = $false
    }
    Write-Host ""

    # Check/build executable
    Write-Step "3/8" "Preparing executable..."
    if (Test-Path "gym-door-bridge.exe") {
        Write-Success "Found existing gym-door-bridge.exe"
    } elseif ($canBuild) {
        Write-Info "Building gym-door-bridge.exe from source..."
        $buildResult = & go build -ldflags "-s -w" -o gym-door-bridge.exe ./cmd 2>&1
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Build failed!"
            Write-Host "Build output: $buildResult" -ForegroundColor Red
            throw "Build process failed"
        }
        Write-Success "Build completed successfully"
    } else {
        Write-Error "gym-door-bridge.exe not found and Go not available"
        Write-Host ""
        Write-Host "Please ensure you have either:" -ForegroundColor Yellow
        Write-Host "1. The pre-built gym-door-bridge.exe file, or" -ForegroundColor Yellow
        Write-Host "2. Go installed to build from source" -ForegroundColor Yellow
        throw "Required executable not available"
    }
    Write-Host ""

    # Check existing installation
    Write-Step "4/8" "Checking existing installation..."
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
        & ".\gym-door-bridge.exe" service uninstall | Out-Null
        Start-Sleep -Seconds 2
        Write-Success "Existing service removed"
    } else {
        Write-Success "No existing installation found"
    }
    Write-Host ""

    # Install service
    Write-Step "5/8" "Installing Windows service..."
    Write-Info "This will automatically discover your biometric devices..."
    Write-Info "Please wait while scanning network (this may take 1-2 minutes)..."
    Write-Host ""
    
    $installOutput = & ".\gym-door-bridge.exe" service install 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Service installation failed!"
        Write-Host "Installation output:" -ForegroundColor Red
        Write-Host $installOutput -ForegroundColor Red
        throw "Service installation failed"
    }
    Write-Host ""
    Write-Success "Service installed successfully!"
    Write-Host ""

    # Start service
    if (-not $NoStart) {
        Write-Step "6/8" "Starting service..."
        $startResult = & ".\gym-door-bridge.exe" service start 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Service started successfully"
        } else {
            Write-Warning "Service installation completed but failed to start automatically"
            Write-Info "You can start it manually from Services.msc"
        }
        Write-Host ""

        # Verify installation
        Write-Step "7/8" "Verifying installation..."
        Start-Sleep -Seconds 3
        $statusResult = & ".\gym-door-bridge.exe" service status 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Service is running and operational"
        } else {
            Write-Warning "Service status check inconclusive"
        }
    } else {
        Write-Step "6/8" "Skipping service start (--NoStart specified)"
        Write-Step "7/8" "Skipping verification (--NoStart specified)"  
    }
    Write-Host ""

    # Create shortcuts
    Write-Step "8/8" "Creating shortcuts..."
    
    # Create start menu folder
    $startMenuPath = "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Gym Door Bridge"
    if (-not (Test-Path $startMenuPath)) {
        New-Item -ItemType Directory -Path $startMenuPath -Force | Out-Null
    }

    # Create management shortcuts with enhanced pairing
    $shortcuts = @(
        @{
            Name = "Check Status"
            Content = "@echo off`ngym-door-bridge.exe service status`npause"
        },
        @{
            Name = "Smart Pair Device"  
            Content = "@echo off`necho This uses the enhanced pairing with auto-unpair capability`nset /p CODE=`"Enter your pairing code: `"`necho.`necho Attempting smart pairing (will auto-unpair if needed)...`ngym-door-bridge.exe pair --pair-code %CODE%`nif %ERRORLEVEL% neq 0 (`n    echo.`n    echo Pairing failed. If device was already paired, trying unpair and re-pair...`n    gym-door-bridge.exe unpair --force`n    if %ERRORLEVEL% equ 0 (`n        echo Device unpaired successfully. Retrying pairing...`n        gym-door-bridge.exe pair --pair-code %CODE%`n        if %ERRORLEVEL% equ 0 (`n            echo Device re-paired successfully!`n        ) else (`n            echo Re-pairing failed. Please check your pairing code.`n        )`n    ) else (`n        echo Unpair failed. You may need to unpair manually from the admin portal.`n    )`n)`npause"
        },
        @{
            Name = "Restart Service"
            Content = "@echo off`ngym-door-bridge.exe service restart`npause"
        },
        @{
            Name = "Uninstall"
            Content = "@echo off`nset /p CONFIRM=`"Are you sure you want to uninstall? (y/N): `"`nif /i `"%CONFIRM%`"==`"y`" gym-door-bridge.exe service uninstall`npause"
        }
    )

    foreach ($shortcut in $shortcuts) {
        $filePath = "$startMenuPath\$($shortcut.Name).bat"
        $shortcut.Content | Out-File -FilePath $filePath -Encoding ASCII -Force
    }

    Write-Success "Enhanced start menu shortcuts created"
    Write-Host ""

    # Installation complete
    Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Green
    Write-Host "  █                                                              █" -ForegroundColor Green
    Write-Host "  █                 INSTALLATION SUCCESSFUL!                    █" -ForegroundColor Green  
    Write-Host "  █                                                              █" -ForegroundColor Green
    Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Green
    Write-Host ""
    Write-Host "  ✓ Gym Door Bridge service installed and running" -ForegroundColor Green
    Write-Host "  ✓ Auto-discovery completed for biometric devices" -ForegroundColor Green
    Write-Host "  ✓ Service configured to start automatically on boot" -ForegroundColor Green  
    Write-Host "  ✓ Enhanced management shortcuts created in Start Menu" -ForegroundColor Green
    Write-Host "  ✓ Smart pairing with auto-unpair capability enabled" -ForegroundColor Green
    Write-Host ""

    if (-not $Silent) {
        Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Cyan
        Write-Host "  █                      ENHANCED FEATURES                      █" -ForegroundColor Cyan
        Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "  ★ SMART PAIRING:" -ForegroundColor Yellow
        Write-Host "     • Automatically detects if device is already paired" -ForegroundColor Gray
        Write-Host "     • Runs unpair --force and re-pairs with current code" -ForegroundColor Gray
        Write-Host "     • Eliminates manual intervention for paired devices" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  1. PAIR YOUR DEVICE:" -ForegroundColor White
        Write-Host "     • Get pairing code from your admin portal" -ForegroundColor Gray
        Write-Host "     • Use Start Menu > Gym Door Bridge > Smart Pair Device" -ForegroundColor Gray
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
        Write-Host "     • No daily maintenance required" -ForegroundColor Gray
        Write-Host ""
    }

    # Enhanced pairing function with auto-unpair capability
    function Invoke-SmartPairing {
        param([string]$Code, [string]$ExePath)
        
        Write-Host "🔄 Smart Pairing: Attempting to pair device with code: $Code" -ForegroundColor Yellow
        $pairResult = & $ExePath pair --pair-code $Code 2>&1
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host ""
            Write-Success "Device paired successfully!"
            Write-Host "Your gym door bridge is now fully operational!" -ForegroundColor Green
            return $true
        } elseif ($pairResult -match "already paired|device is already paired") {
            Write-Host ""
            Write-Warning "Device is already paired - initiating smart re-pairing process..."
            
            # Attempt to unpair with force flag
            Write-Info "🔧 Running unpair command with --force flag..."
            $unpairResult = & $ExePath unpair --force 2>&1
            
            if ($LASTEXITCODE -eq 0) {
                Write-Success "Device unpaired successfully"
                Write-Info "🔄 Retrying pairing with current code..."
                
                # Retry pairing
                $retryResult = & $ExePath pair --pair-code $Code 2>&1
                if ($LASTEXITCODE -eq 0) {
                    Write-Host ""
                    Write-Success "🎉 Device re-paired successfully!"
                    Write-Host "Your gym door bridge is now fully operational with the latest pairing code!" -ForegroundColor Green
                    return $true
                } else {
                    Write-Host ""
                    Write-Error "Re-pairing failed after successful unpair"
                    Write-Host "Retry output: $retryResult" -ForegroundColor Red
                    Write-Info "The device was unpaired but re-pairing failed. Please check your pairing code."
                    return $false
                }
            } else {
                Write-Host ""
                Write-Error "Failed to unpair existing device"
                Write-Host "Unpair output: $unpairResult" -ForegroundColor Red
                Write-Info "You may need to unpair manually from the admin portal"
                Write-Info "Or contact support if the issue persists"
                return $false
            }
        } else {
            Write-Host ""
            Write-Error "Pairing failed for unknown reason"
            Write-Host "Pairing output: $pairResult" -ForegroundColor Red
            Write-Info "Please verify your pairing code and network connectivity"
            return $false
        }
    }

    # Handle pairing with enhanced auto-unpair logic
    if ($PairCode) {
        Write-Host ""
        Write-Host "🚀 Initiating smart pairing with provided code..." -ForegroundColor Cyan
        $pairSuccess = Invoke-SmartPairing -Code $PairCode -ExePath ".\gym-door-bridge.exe"
        if (-not $pairSuccess) {
            Write-Warning "Smart pairing unsuccessful. You can try again later using:"
            Write-Host "gym-door-bridge.exe pair --pair-code YOUR_CODE" -ForegroundColor Cyan
            Write-Host "Or use the enhanced 'Smart Pair Device' shortcut from the Start Menu" -ForegroundColor Cyan
        }
    } elseif (-not $Silent) {
        Write-Host ""
        $pairNow = Read-Host "🔗 Would you like to pair your device now using smart pairing? (Y/n)"
        if ($pairNow.ToLower() -ne "n") {
            $inputCode = Read-Host "Enter your pairing code"
            if ($inputCode.Trim()) {
                Write-Host ""
                $pairSuccess = Invoke-SmartPairing -Code $inputCode.Trim() -ExePath ".\gym-door-bridge.exe"
                if (-not $pairSuccess) {
                    Write-Warning "Smart pairing unsuccessful. You can try again later using:"
                    Write-Host "Start Menu > Gym Door Bridge > Smart Pair Device" -ForegroundColor Cyan
                }
            }
        }
    }

    Write-Host ""
    Write-Host "✅ Enhanced installation process completed!" -ForegroundColor Green
    if (-not $Silent) {
        Write-Host "Your gym door bridge now includes smart pairing capabilities." -ForegroundColor Gray
        Write-Host "You can close this window now." -ForegroundColor Gray
        Write-Host ""
        Read-Host "Press Enter to exit"
    }

} catch {
    Write-Host ""
    Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Red
    Write-Host "  █                        ERROR                                 █" -ForegroundColor Red
    Write-Host "  █                                                              █" -ForegroundColor Red
    Write-Host "  █                Installation Failed!                         █" -ForegroundColor Red
    Write-Host "  █                                                              █" -ForegroundColor Red
    Write-Host "  ████████████████████████████████████████████████████████████████" -ForegroundColor Red
    Write-Host ""
    Write-Host "Error details: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host ""
    Write-Host "Please try the following:" -ForegroundColor Yellow
    Write-Host "1. Run PowerShell as Administrator" -ForegroundColor Gray
    Write-Host "2. Ensure gym-door-bridge.exe is present" -ForegroundColor Gray  
    Write-Host "3. Check Windows Event Viewer for more details" -ForegroundColor Gray
    Write-Host "4. Contact support with the error message above" -ForegroundColor Gray
    Write-Host ""
    if (-not $Silent) {
        Read-Host "Press Enter to exit"
    }
    exit 1
}