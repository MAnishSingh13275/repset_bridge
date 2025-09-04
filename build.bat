@echo off
REM Build script for Gym Door Bridge

echo.
echo === Building Gym Door Bridge ===
echo.

REM Check if Go is installed
go version >nul 2>&1
if %errorLevel% neq 0 (
    echo ERROR: Go is not installed or not in PATH.
    echo Please install Go from https://golang.org/dl/
    pause
    exit /b 1
)

echo Go version:
go version
echo.

REM Clean previous builds
if exist "gym-door-bridge.exe" (
    echo Removing previous build...
    del "gym-door-bridge.exe"
)

echo Building for Windows...
echo.

REM Build the executable
go build -ldflags "-s -w" -o gym-door-bridge.exe ./cmd

if %errorLevel% equ 0 (
    echo.
    echo === Build Successful! ===
    echo.
    echo Executable created: gym-door-bridge.exe
    echo.
    echo Next steps:
    echo 1. Run 'install.bat' as Administrator to install as service
    echo 2. Or run 'gym-door-bridge.exe --help' to see options
    echo.
) else (
    echo.
    echo ERROR: Build failed.
    echo Please check the error messages above.
    echo.
)

pause