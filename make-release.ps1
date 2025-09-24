# Simple Release Builder for GymDoorBridge
param(
    [Parameter(Mandatory=$true)]
    [string]$Version
)

$ErrorActionPreference = "Stop"

Write-Host "Building release v$Version..." -ForegroundColor Green

# Create release directory
$releaseDir = ".\release"
if (Test-Path $releaseDir) {
    Remove-Item $releaseDir -Recurse -Force
}
New-Item -ItemType Directory -Path $releaseDir -Force | Out-Null

# Copy files
$files = @(
    "gym-door-bridge.exe",
    "GymDoorBridge-Installer.ps1", 
    "GymDoorBridge-Installer.bat",
    "config.yaml.example",
    "README.md",
    "LICENSE"
)

foreach ($file in $files) {
    if (Test-Path $file) {
        Copy-Item $file $releaseDir -Force
        Write-Host "Copied: $file" -ForegroundColor Cyan
    }
}

# Create ZIP
$zipName = "GymDoorBridge-v$Version.zip"
if (Test-Path $zipName) {
    Remove-Item $zipName -Force
}

Add-Type -AssemblyName System.IO.Compression.FileSystem
$fullReleasePath = (Resolve-Path $releaseDir).Path
$fullZipPath = Join-Path (Get-Location) $zipName
[System.IO.Compression.ZipFile]::CreateFromDirectory($fullReleasePath, $fullZipPath)

$zipInfo = Get-Item $zipName
$sizeMB = [math]::Round($zipInfo.Length / 1MB, 2)

Write-Host "`nRelease created: $zipName ($sizeMB MB)" -ForegroundColor Green
Write-Host "`nNext steps:" -ForegroundColor Yellow
Write-Host "1. Make repository public on GitHub" -ForegroundColor Gray
Write-Host "2. Create release v$Version" -ForegroundColor Gray  
Write-Host "3. Upload $zipName as asset" -ForegroundColor Gray
Write-Host "4. Test web installer" -ForegroundColor Gray