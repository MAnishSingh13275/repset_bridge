@echo off
REM Gym Door Bridge Installer Batch Script
REM This script provides a simple interface to install the Gym Door Bridge

echo.
echo === Gym Door Bridge Installer ===
echo.

REM Check if running as administrator
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo ERROR: This installer requires administrator privileges.
    echo Please run this batch file as Administrator.
    echo.
    echo Right-click on install.bat and select "Run as administrator"
    pause
    exit /b 1
)

echo Administrator privileges confirmed.
echo.

REM Check if executable exists
if not exist "gym-door-bridge.exe" (
    echo ERROR: gym-door-bridge.exe not found in current directory.
    echo.
    echo Please ensure you have:
    echo 1. Built the executable: go build -o gym-door-bridge.exe ./cmd
    echo 2. Or downloaded the release binary
    echo.
    pause
    exit /b 1
)

echo Found gym-door-bridge.exe
echo.

REM Check if service already exists
sc query "GymDoorBridge" >nul 2>&1
if %errorLevel% equ 0 (
    echo WARNING: Gym Door Bridge service already exists.
    echo.
    set /p choice="Do you want to reinstall? (y/N): "
    if /i not "%choice%"=="y" (
        echo Installation cancelled.
        pause
        exit /b 0
    )
    echo.
    echo Uninstalling existing service...
    gym-door-bridge.exe uninstall
    echo.
)

echo Starting installation with automatic device discovery...
echo This may take a few minutes while scanning for biometric devices...
echo.

REM Run the installer
gym-door-bridge.exe install

if %errorLevel% equ 0 (
    echo.
    echo === Installation Completed Successfully! ===
    echo.
    echo The Gym Door Bridge has been installed as a Windows service.
    echo.
    echo Next Steps:
    echo 1. Pair with your platform: gym-door-bridge pair
    echo 2. Check service status in Services.msc
    echo 3. View logs in the installation directory
    echo.
    echo The service will automatically discover and configure
    echo biometric devices on your network.
    echo.
) else (
    echo.
    echo ERROR: Installation failed.
    echo Please check the error messages above.
    echo.
)

echo.
echo Installation process completed.
pause