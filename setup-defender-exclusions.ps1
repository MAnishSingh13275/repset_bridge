# Windows Defender Exclusions Setup for Gym Door Bridge
# Run this script as Administrator BEFORE installing the bridge

param(
    [switch]$Remove = $false
)

# Check admin privileges
if (-NOT ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    Write-Host "ERROR: This script requires Administrator privileges!" -ForegroundColor Red
    Write-Host "Please run PowerShell as Administrator and try again." -ForegroundColor Yellow
    exit 1
}

Write-Host "Windows Defender Exclusions Setup for Gym Door Bridge" -ForegroundColor Cyan
Write-Host "====================================================" -ForegroundColor Cyan

$InstallPath = "$env:ProgramFiles\GymDoorBridge"
$DataPath = "$env:ProgramData\GymDoorBridge"
$ProcessName = "gym-door-bridge.exe"

if ($Remove) {
    Write-Host "Removing Windows Defender exclusions..." -ForegroundColor Yellow
    
    try {
        Remove-MpPreference -ExclusionPath $InstallPath -ErrorAction SilentlyContinue
        Remove-MpPreference -ExclusionPath $DataPath -ErrorAction SilentlyContinue
        Remove-MpPreference -ExclusionProcess $ProcessName -ErrorAction SilentlyContinue
        
        Write-Host "Exclusions removed successfully!" -ForegroundColor Green
    } catch {
        Write-Host "Warning: Some exclusions may not have been removed: $($_.Exception.Message)" -ForegroundColor Yellow
    }
} else {
    Write-Host "Adding Windows Defender exclusions..." -ForegroundColor Green
    
    try {
        # Add path exclusions
        Add-MpPreference -ExclusionPath $InstallPath
        Write-Host "Added exclusion: $InstallPath" -ForegroundColor Green
        
        Add-MpPreference -ExclusionPath $DataPath
        Write-Host "Added exclusion: $DataPath" -ForegroundColor Green
        
        # Add process exclusion
        Add-MpPreference -ExclusionProcess $ProcessName
        Write-Host "Added exclusion: $ProcessName" -ForegroundColor Green
        
        Write-Host ""
        Write-Host "Windows Defender exclusions added successfully!" -ForegroundColor Green
        Write-Host "You can now safely install Gym Door Bridge without interference." -ForegroundColor Green
        
    } catch {
        Write-Host "ERROR: Failed to add exclusions: $($_.Exception.Message)" -ForegroundColor Red
        Write-Host "You may need to add exclusions manually through Windows Security." -ForegroundColor Yellow
        exit 1
    }
}

Write-Host ""
Write-Host "Current exclusions:" -ForegroundColor Cyan
try {
    $pathExclusions = Get-MpPreference | Select-Object -ExpandProperty ExclusionPath | Where-Object { $_ -like "*GymDoorBridge*" }
    $processExclusions = Get-MpPreference | Select-Object -ExpandProperty ExclusionProcess | Where-Object { $_ -like "*gym-door-bridge*" }
    
    if ($pathExclusions) {
        Write-Host "Path exclusions:" -ForegroundColor White
        $pathExclusions | ForEach-Object { Write-Host "  - $_" -ForegroundColor Gray }
    }
    
    if ($processExclusions) {
        Write-Host "Process exclusions:" -ForegroundColor White
        $processExclusions | ForEach-Object { Write-Host "  - $_" -ForegroundColor Gray }
    }
    
    if (-not $pathExclusions -and -not $processExclusions) {
        Write-Host "No Gym Door Bridge exclusions found." -ForegroundColor Yellow
    }
} catch {
    Write-Host "Could not retrieve current exclusions." -ForegroundColor Yellow
}

Write-Host ""
if (-not $Remove) {
    Write-Host "Next steps:" -ForegroundColor Cyan
    Write-Host "1. Download and run the installation script" -ForegroundColor White
    Write-Host "2. The installation should now complete without Windows Defender interference" -ForegroundColor White
    Write-Host ""
    Write-Host "Installation command:" -ForegroundColor Yellow
    Write-Host 'Invoke-WebRequest -Uri "https://raw.githubusercontent.com/MAnishSingh13275/repset_bridge/main/install-bridge.ps1" -OutFile "install-bridge.ps1"' -ForegroundColor Gray
    Write-Host '.\install-bridge.ps1 -PairCode "YOUR_PAIR_CODE" -ServerUrl "https://repset.onezy.in" -Force' -ForegroundColor Gray
}