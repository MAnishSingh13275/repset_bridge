@echo off
REM Simple deployment script for Gym Door Bridge Windows Service
REM Run as Administrator

echo === Gym Door Bridge Service Deployment ===
echo.

REM Check admin privileges
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo ERROR: Administrator privileges required.
    echo Right-click and select "Run as administrator"
    pause
    exit /b 1
)

REM Build if needed
if not exist "gym-door-bridge.exe" (
    echo Building executable...
    go build -o gym-door-bridge.exe ./cmd
    if %errorLevel% neq 0 (
        echo ERROR: Build failed
        pause
        exit /b 1
    )
    echo Build completed.
    echo.
)

REM Install service with auto-discovery
echo Installing service with device auto-discovery...
echo This may take a minute while scanning for devices...
echo.

gym-door-bridge.exe install

if %errorLevel% equ 0 (
    echo.
    echo === SERVICE INSTALLED SUCCESSFULLY! ===
    echo.
    echo The service will automatically:
    echo - Start on Windows boot
    echo - Discover biometric devices
    echo - Process member check-ins
    echo.
    echo Next steps:
    echo 1. Pair with platform: gym-door-bridge pair --pair-code YOUR_CODE
    echo 2. Check status: gym-door-bridge service status
    echo 3. Manage via Services.msc
    echo.
) else (
    echo ERROR: Installation failed
    echo Check error messages above
)

pause