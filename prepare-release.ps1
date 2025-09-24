# ================================================================
# Release Preparation Script for GymDoorBridge
# Builds, packages, and prepares files for GitHub release
# ================================================================

param(
    [Parameter(Mandatory=$true)]
    [string]$Version,
    [string]$OutputDir = ".\release",
    [switch]$SkipBuild = $false,
    [switch]$CreateGitHubRelease = $false
)

$ErrorActionPreference = "Stop"

# Color functions
function Write-Success { param([string]$Message) Write-Host "✓ $Message" -ForegroundColor Green }
function Write-Error { param([string]$Message) Write-Host "✗ $Message" -ForegroundColor Red }
function Write-Warning { param([string]$Message) Write-Host "! $Message" -ForegroundColor Yellow }
function Write-Info { param([string]$Message) Write-Host "→ $Message" -ForegroundColor Cyan }
function Write-Step { param([string]$Step, [string]$Message) Write-Host "[$Step] $Message" -ForegroundColor White }

Clear-Host
Write-Host "══════════════════════════════════════════════════════════════════" -ForegroundColor Green
Write-Host "              GYMDOORBRIDGE RELEASE PREPARATION" -ForegroundColor Green
Write-Host "══════════════════════════════════════════════════════════════════" -ForegroundColor Green
Write-Host ""
Write-Host "Version: $Version" -ForegroundColor Cyan
Write-Host "Output: $OutputDir" -ForegroundColor Cyan
Write-Host ""

try {
    # Step 1: Clean and prepare output directory
    Write-Step "1/6" "Preparing release directory..."
    if (Test-Path $OutputDir) {
        Remove-Item $OutputDir -Recurse -Force
    }
    New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
    Write-Success "Release directory prepared: $OutputDir"
    Write-Host ""

    # Step 2: Build executable (if not skipping)
    if (-not $SkipBuild) {
        Write-Step "2/6" "Building executable..."
        
        # Check if Go is available
        $goPath = Get-Command go -ErrorAction SilentlyContinue
        if (-not $goPath) {
            Write-Warning "Go not found - using existing executable"
            if (-not (Test-Path "gym-door-bridge.exe")) {
                throw "No executable found and Go not available for building"
            }
        } else {
            Write-Info "Building gym-door-bridge.exe..."
            $buildResult = & go build -ldflags "-s -w" -o gym-door-bridge.exe ./cmd 2>&1
            if ($LASTEXITCODE -ne 0) {
                throw "Build failed: $buildResult"
            }
            Write-Success "Build completed successfully"
        }
    } else {
        Write-Step "2/6" "Skipping build (using existing executable)..."
        if (-not (Test-Path "gym-door-bridge.exe")) {
            throw "gym-door-bridge.exe not found"
        }
    }
    Write-Host ""

    # Step 3: Update version in installer scripts
    Write-Step "3/6" "Updating version in installer scripts..."
    
    # Update PowerShell installer
    $installerContent = Get-Content "GymDoorBridge-Installer.ps1" -Raw
    $installerContent = $installerContent -replace 'Version: \d+\.\d+\.\d+', "Version: $Version"
    Set-Content "GymDoorBridge-Installer.ps1" -Value $installerContent -Encoding UTF8
    
    # Update web installer  
    $webInstallerContent = Get-Content "web-install.ps1" -Raw
    $webInstallerContent = $webInstallerContent -replace 'GymDoorBridge-v\d+\.\d+\.\d+\.zip', "GymDoorBridge-v$Version.zip"
    Set-Content "web-install.ps1" -Value $webInstallerContent -Encoding UTF8
    
    Write-Success "Version updated to $Version in installer scripts"
    Write-Host ""

    # Step 4: Copy files to release directory
    Write-Step "4/6" "Copying release files..."
    
    $filesToInclude = @(
        "gym-door-bridge.exe",
        "GymDoorBridge-Installer.ps1", 
        "GymDoorBridge-Installer.bat",
        "config.yaml.example",
        "README.md",
        "LICENSE",
        "CHANGELOG.md"
    )
    
    foreach ($file in $filesToInclude) {
        if (Test-Path $file) {
            Copy-Item $file $OutputDir -Force
            Write-Info "Copied: $file"
        } else {
            Write-Warning "File not found, skipping: $file"
        }
    }
    Write-Success "Release files copied"
    Write-Host ""

    # Step 5: Create ZIP package
    Write-Step "5/6" "Creating ZIP package..."
    $zipName = "GymDoorBridge-v$Version.zip"
    $zipPath = Join-Path (Get-Location) $zipName
    
    if (Test-Path $zipPath) {
        Remove-Item $zipPath -Force
    }
    
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    [System.IO.Compression.ZipFile]::CreateFromDirectory($OutputDir, $zipPath)
    
    $zipInfo = Get-Item $zipPath
    $sizeMB = [math]::Round($zipInfo.Length / 1MB, 2)
    Write-Success "ZIP created: $zipName ($sizeMB MB)"
    Write-Host ""

    # Step 6: Generate release notes
    Write-Step "6/6" "Generating release information..."
    
    $releaseNotes = @'
# GymDoorBridge v{0}

## Installation Options

### Option 1: Web Installer (Recommended)
```powershell
# Run as Administrator
iex ((New-Object System.Net.WebClient).DownloadString("https://raw.githubusercontent.com/MAnishSingh13275/repset_bridge/main/web-install.ps1"))
```

### Option 2: Manual Installation
1. Download the ZIP file below
2. Extract to a folder
3. Run GymDoorBridge-Installer.ps1 as Administrator

## Whats Included
* gym-door-bridge.exe (Windows Service)
* Automated installer scripts
* Configuration examples
* Documentation

## System Requirements
* Windows 10/11 or Windows Server 2019+
* Administrator privileges for installation
* Network access for device discovery

## Quick Start
1. Install using one of the methods above
2. Get your pairing code from the admin portal
3. Run pairing: gym-door-bridge.exe pair --pair-code YOUR_CODE
4. Service will start automatically and discover devices

## Support
* Check Windows Event Viewer for logs
* Use included management shortcuts in Start Menu
* Service runs automatically on system startup
'@ -f $Version
"@

    $releaseNotesPath = Join-Path $OutputDir "RELEASE_NOTES.md"
    Set-Content -Path $releaseNotesPath -Value $releaseNotes -Encoding UTF8
    
    Write-Success "Release notes generated"
    Write-Host ""

    # Summary
    Write-Host "══════════════════════════════════════════════════════════════════" -ForegroundColor Green
    Write-Host "                     RELEASE PREPARED!" -ForegroundColor Green
    Write-Host "══════════════════════════════════════════════════════════════════" -ForegroundColor Green
    Write-Host ""
    Write-Host "Release package: $zipName" -ForegroundColor Cyan
    Write-Host "Size: $sizeMB MB" -ForegroundColor Cyan
    Write-Host "Files included: $($filesToInclude.Count)" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Next steps:" -ForegroundColor Yellow
    Write-Host "1. Make repository public on GitHub" -ForegroundColor Gray
    Write-Host "2. Create release v$Version on GitHub" -ForegroundColor Gray
    Write-Host "3. Upload $zipName as release asset" -ForegroundColor Gray
    Write-Host "4. Test web installer" -ForegroundColor Gray
    Write-Host ""
    
    if ($CreateGitHubRelease) {
        Write-Host "GitHub CLI commands to create release:" -ForegroundColor Yellow
        Write-Host "gh release create v$Version $zipPath --title `"GymDoorBridge v$Version`" --notes-file `"$releaseNotesPath`"" -ForegroundColor Cyan
        Write-Host ""
    }

} catch {
    Write-Host ""
    Write-Host "══════════════════════════════════════════════════════════════════" -ForegroundColor Red
    Write-Host "                      RELEASE FAILED!" -ForegroundColor Red  
    Write-Host "══════════════════════════════════════════════════════════════════" -ForegroundColor Red
    Write-Host ""
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host ""
    exit 1
}