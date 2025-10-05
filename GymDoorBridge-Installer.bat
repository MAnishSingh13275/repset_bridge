@echo off
title Gym Door Bridge - Installer

echo.
echo ============================================================
echo   GYM DOOR BRIDGE - INSTALLER  
echo ============================================================
echo.
echo Welcome! Installing your gym door access system...
echo Perfect for gym owners - completely automatic! 💪
echo.

REM Check administrator privileges
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ ERROR: Administrator privileges required!
    echo.
    echo HOW TO FIX:
    echo 1. Right-click on this file
    echo 2. Select "Run as administrator"
    echo 3. Click "Yes" when Windows asks
    echo.
    pause
    exit /b 1
)

echo ✓ Running as Administrator - Good!
echo.

REM Check PowerShell installer exists
if not exist "%~dp0GymDoorBridge-Installer.ps1" (
    echo ❌ ERROR: PowerShell installer not found!
    echo.
    echo Please download both files:
    echo • GymDoorBridge-Installer.bat (this file)
    echo • GymDoorBridge-Installer.ps1
    echo.
    echo Download from: https://github.com/MAnishSingh13275/repset_bridge/releases
    echo.
    pause
    exit /b 1
)

echo ✓ Installation files found
echo.

echo 🛡️  WINDOWS SECURITY NOTICE:
echo If Windows shows security warnings:
echo • Click "More info" → "Run anyway"
echo • OR click "Allow" for Smart App Control  
echo • This software is safe and legitimate
echo.

echo 🚀 Starting installation...
echo.

REM Run PowerShell installer
powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -WindowStyle Normal -File "%~dp0GymDoorBridge-Installer.ps1"

if %errorlevel% == 0 (
    echo.
    echo ✅ Installation completed successfully!
    echo Your gym door system is ready to use!
) else (
    echo.
    echo ⚠️ Installation issue (code: %errorlevel%)
    echo.
    echo TRY THIS: Right-click "GymDoorBridge-Installer.ps1" → "Run with PowerShell"
    echo.
    echo Contact support: support@repset.onezy.in
)

echo.
echo Press any key to close...
pause >nul