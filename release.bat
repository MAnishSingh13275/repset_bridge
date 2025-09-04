@echo off
REM Release packaging script for Gym Door Bridge

echo.
echo === Creating Release Package ===
echo.

REM Set version (you can pass this as parameter)
set VERSION=v1.0.0
if not "%1"=="" set VERSION=%1

echo Building version: %VERSION%
echo.

REM Clean previous builds
if exist "release" rmdir /s /q "release"
mkdir "release"

REM Build the executable
echo Building executable...
go build -ldflags "-s -w -X main.version=%VERSION%" -o release\gym-door-bridge.exe ./cmd

if %errorLevel% neq 0 (
    echo ERROR: Build failed
    exit /b 1
)

REM Copy installation files
echo Copying installation files...
copy install.bat release\
copy install.ps1 release\
copy build.bat release\
copy README.md release\
copy INSTALLATION.md release\
copy config.yaml.example release\

REM Create release package
echo Creating release package...
cd release
powershell -Command "Compress-Archive -Path * -DestinationPath ..\gym-door-bridge-%VERSION%-windows.zip"
cd ..

echo.
echo === Release Package Created ===
echo.
echo Package: gym-door-bridge-%VERSION%-windows.zip
echo Contents:
echo   - gym-door-bridge.exe (main executable)
echo   - install.bat (simple installer)
echo   - install.ps1 (PowerShell installer)
echo   - README.md (quick start guide)
echo   - INSTALLATION.md (detailed guide)
echo.
echo Ready for distribution!
pause