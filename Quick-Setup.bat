@echo off
:: RepSet Bridge Quick Setup
:: Double-click this file to start the bridge setup process

title RepSet Bridge Setup

echo.
echo ===============================================
echo         REPSET BRIDGE QUICK SETUP
echo ===============================================
echo.
echo This will help you set up the RepSet Bridge for
echo your gym's biometric access control system.
echo.

:: Check if PowerShell script exists
if not exist "bridge-admin-tools.ps1" (
    echo ERROR: bridge-admin-tools.ps1 not found!
    echo.
    echo Please ensure all files are in the same folder:
    echo - gym-door-bridge.exe
    echo - config.yaml  
    echo - bridge-admin-tools.ps1
    echo - Quick-Setup.bat (this file)
    echo.
    pause
    exit /b 1
)

:: Check if bridge executable exists
if not exist "gym-door-bridge.exe" (
    echo ERROR: gym-door-bridge.exe not found!
    echo.
    echo Please download and extract the bridge software
    echo to this folder before running setup.
    echo.
    pause
    exit /b 1
)

echo Files found successfully!
echo.
echo NEXT STEPS:
echo 1. Get your pair code from the admin dashboard
echo 2. This will open PowerShell to complete the setup
echo.
echo Press any key to continue...
pause >nul

:: Launch PowerShell with the admin tools
echo.
echo Opening PowerShell for bridge setup...
echo.
echo You can use these commands:
echo   .\bridge-admin-tools.ps1 pair
echo   .\bridge-admin-tools.ps1 install  
echo   .\bridge-admin-tools.ps1 status
echo.

:: Start PowerShell in current directory
powershell.exe -NoExit -ExecutionPolicy Bypass -Command "& { Clear-Host; Write-Host 'RepSet Bridge Setup - PowerShell Interface' -ForegroundColor Cyan; Write-Host '=============================================' -ForegroundColor Cyan; Write-Host ''; Write-Host 'Quick Commands:' -ForegroundColor Yellow; Write-Host '  .\bridge-admin-tools.ps1 help     # Show all commands' -ForegroundColor White; Write-Host '  .\bridge-admin-tools.ps1 pair     # Pair with your gym' -ForegroundColor White; Write-Host '  .\bridge-admin-tools.ps1 install  # Install and start bridge' -ForegroundColor White; Write-Host '  .\bridge-admin-tools.ps1 status   # Check bridge status' -ForegroundColor White; Write-Host ''; Write-Host 'Get your pair code from: https://repset.onezy.in/{gymId}/admin/dashboard' -ForegroundColor Green; Write-Host ''; }"