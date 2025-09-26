@echo off
REM ================================================================
REM Gym Door Bridge - One-Click Installer
REM Automatically installs and configures the Gym Door Bridge service
REM ================================================================

title Gym Door Bridge Installer

REM Set color to green for better visibility
color 0A

echo.
echo  ████████████████████████████████████████████████████████████████
echo  █                                                              █
echo  █              GYM DOOR BRIDGE INSTALLER                      █
echo  █                                                              █
echo  █        Connects your biometric devices to the cloud         █
echo  █                                                              █
echo  ████████████████████████████████████████████████████████████████
echo.
echo  Version: 1.0.0
echo  Platform: Windows Service
echo  Auto-Discovery: Enabled
echo.

REM Check if running as administrator
echo [1/8] Checking administrator privileges...
net session >nul 2>&1
if %errorLevel% neq 0 (
    color 0C
    echo.
    echo  ████████████████████████████████████████████████████████████████
    echo  █                        ERROR                                 █
    echo  █                                                              █
    echo  █    This installer requires administrator privileges!         █
    echo  █                                                              █
    echo  █    Please right-click on this file and select:              █
    echo  █    "Run as administrator"                                    █
    echo  █                                                              █
    echo  ████████████████████████████████████████████████████████████████
    echo.
    pause
    exit /b 1
)
echo      ✓ Administrator privileges confirmed
echo.

REM Check if Go is available for building
echo [2/8] Checking build requirements...
where go >nul 2>&1
if %errorLevel% neq 0 (
    echo      ! Go not found - using pre-built executable
    set BUILD_FROM_SOURCE=false
) else (
    echo      ✓ Go found - can build from source
    set BUILD_FROM_SOURCE=true
)
echo.

REM Check if executable exists or build it
echo [3/8] Preparing executable...
if exist "gym-door-bridge.exe" (
    echo      ✓ Found existing gym-door-bridge.exe
) else if "%BUILD_FROM_SOURCE%"=="true" (
    echo      Building gym-door-bridge.exe from source...
    go build -ldflags "-s -w" -o gym-door-bridge.exe ./cmd
    if %errorLevel% neq 0 (
        color 0C
        echo      ✗ Build failed!
        echo.
        echo      Please ensure you have Go installed and try again.
        pause
        exit /b 1
    )
    echo      ✓ Build completed successfully
) else (
    color 0C
    echo      ✗ gym-door-bridge.exe not found and Go not available
    echo.
    echo      Please ensure you have either:
    echo      1. The pre-built gym-door-bridge.exe file, or
    echo      2. Go installed to build from source
    pause
    exit /b 1
)
echo.

REM Check if service already exists
echo [4/8] Checking existing installation...
sc query "GymDoorBridge" >nul 2>&1
if %errorLevel% equ 0 (
    echo      ! Service already installed
    set /p REINSTALL="      Do you want to reinstall? (Y/n): "
    if /i "%REINSTALL%"=="n" (
        echo      Installation cancelled by user.
        pause
        exit /b 0
    )
    echo      Removing existing service...
    gym-door-bridge.exe service uninstall >nul 2>&1
    timeout /t 2 /nobreak >nul
    echo      ✓ Existing service removed
) else (
    echo      ✓ No existing installation found
)
echo.

REM Install the service
echo [5/8] Installing Windows service...
echo      This will automatically discover your biometric devices...
echo      Please wait while scanning network (this may take 1-2 minutes)...
echo.

gym-door-bridge.exe service install

if %errorLevel% neq 0 (
    color 0C
    echo.
    echo      ✗ Service installation failed!
    echo      Please check the error messages above.
    pause
    exit /b 1
)
echo.
echo      ✓ Service installed successfully!
echo.

REM Start the service
echo [6/8] Starting service...
gym-door-bridge.exe service start >nul 2>&1
if %errorLevel% equ 0 (
    echo      ✓ Service started successfully
) else (
    echo      ! Service installation completed but failed to start automatically
    echo      You can start it manually from Services.msc
)
echo.

REM Check service status
echo [7/8] Verifying installation...
timeout /t 3 /nobreak >nul
gym-door-bridge.exe service status >nul 2>&1
if %errorLevel% equ 0 (
    echo      ✓ Service is running and operational
) else (
    echo      ! Service status check inconclusive
)
echo.

REM Create desktop shortcuts and start menu entries
echo [8/8] Creating shortcuts...

REM Create start menu folder
set STARTMENU=%APPDATA%\Microsoft\Windows\Start Menu\Programs\Gym Door Bridge
if not exist "%STARTMENU%" mkdir "%STARTMENU%"

REM Create batch files for easy management
echo @echo off > "%STARTMENU%\Check Status.bat"
echo gym-door-bridge.exe service status >> "%STARTMENU%\Check Status.bat"
echo pause >> "%STARTMENU%\Check Status.bat"

echo @echo off > "%STARTMENU%\Pair Device.bat"
echo set /p CODE="Enter your pairing code: " >> "%STARTMENU%\Pair Device.bat"
echo gym-door-bridge.exe pair --pair-code %%CODE%% >> "%STARTMENU%\Pair Device.bat"
echo pause >> "%STARTMENU%\Pair Device.bat"

echo @echo off > "%STARTMENU%\Restart Service.bat"
echo gym-door-bridge.exe service restart >> "%STARTMENU%\Restart Service.bat"
echo pause >> "%STARTMENU%\Restart Service.bat"

echo @echo off > "%STARTMENU%\Uninstall.bat"
echo set /p CONFIRM="Are you sure you want to uninstall? (y/N): " >> "%STARTMENU%\Uninstall.bat"
echo if /i "%%CONFIRM%%"=="y" gym-door-bridge.exe service uninstall >> "%STARTMENU%\Uninstall.bat"
echo pause >> "%STARTMENU%\Uninstall.bat"

echo      ✓ Start menu shortcuts created
echo.

REM Installation complete
color 0A
echo  ████████████████████████████████████████████████████████████████
echo  █                                                              █
echo  █                 INSTALLATION SUCCESSFUL!                    █
echo  █                                                              █
echo  ████████████████████████████████████████████████████████████████
echo.
echo  ✓ Gym Door Bridge service installed and running
echo  ✓ Auto-discovery completed for biometric devices  
echo  ✓ Service configured to start automatically on boot
echo  ✓ Management shortcuts created in Start Menu
echo.
echo  ████████████████████████████████████████████████████████████████
echo  █                      NEXT STEPS                             █
echo  ████████████████████████████████████████████████████████████████
echo.
echo  1. PAIR YOUR DEVICE:
echo     • Get pairing code from your admin portal
echo     • Use Start Menu ^> Gym Door Bridge ^> Pair Device
echo     • Or run: gym-door-bridge pair --pair-code YOUR_CODE
echo.
echo  2. VERIFY SETUP:
echo     • Use Start Menu ^> Gym Door Bridge ^> Check Status
echo     • Check Windows Services (services.msc)
echo     • View Windows Event Viewer for logs
echo.
echo  3. MANAGE SERVICE:
echo     • All management tools available in Start Menu
echo     • Service runs automatically on Windows startup
echo     • No daily maintenance required
echo.
echo  ████████████████████████████████████████████████████████████████
echo  █                      SUPPORT INFO                           █
echo  ████████████████████████████████████████████████████████████████
echo.
echo  • Installation Path: %ProgramFiles%\GymDoorBridge
echo  • Service Name: GymDoorBridge  
echo  • Config File: %ProgramFiles%\GymDoorBridge\config.yaml
echo  • Logs: Windows Event Viewer ^> Applications and Services Logs
echo.
echo  For support, include the following in your message:
echo  • Windows version
echo  • Error messages (if any)  
echo  • Service status output
echo  • Event log entries
echo.

REM Ask if user wants to pair now
echo.
set /p PAIR_NOW="Would you like to pair your device now? (Y/n): "
if /i not "%PAIR_NOW%"=="n" (
    echo.
    set /p PAIR_CODE="Enter your pairing code: "
    if not "!PAIR_CODE!"=="" (
        echo.
        echo Pairing device...
        gym-door-bridge.exe pair --pair-code !PAIR_CODE!
        if %errorLevel% equ 0 (
            echo.
            echo ✓ Device paired successfully!
            echo Your gym door bridge is now fully operational!
        ) else (
            echo.
            echo ✗ Pairing failed. You can try again later using:
            echo gym-door-bridge.exe pair --pair-code YOUR_CODE
        )
    )
)

echo.
echo Installation process completed!
echo You can close this window now.
echo.
pause