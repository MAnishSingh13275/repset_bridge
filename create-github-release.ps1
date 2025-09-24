# Create GitHub Release Script
param(
    [Parameter(Mandatory=$true)]
    [string]$Version,
    [Parameter(Mandatory=$true)]
    [string]$Token,
    [string]$ZipFile = "GymDoorBridge-v$Version.zip"
)

$ErrorActionPreference = "Stop"

$repo = "MAnishSingh13275/repset_bridge"
$tagName = "v$Version"

Write-Host "Creating GitHub release v$Version..." -ForegroundColor Green

# Release notes
$releaseNotes = @"
# GymDoorBridge v$Version

## Installation Options

### Option 1: Web Installer (Recommended)
``````powershell
# Run as Administrator
iex ((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/MAnishSingh13275/repset_bridge/main/web-install.ps1'))
``````

### Option 2: Manual Installation
1. Download the ZIP file below
2. Extract to a folder  
3. Run GymDoorBridge-Installer.ps1 as Administrator

## What's Included
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
"@

try {
    # Create release
    $headers = @{
        'Authorization' = "Bearer $Token"
        'Accept' = 'application/vnd.github.v3+json'
        'User-Agent' = 'PowerShell-GitHub-Release'
    }
    
    $releaseData = @{
        tag_name = $tagName
        target_commitish = "main"
        name = "GymDoorBridge v$Version"
        body = $releaseNotes
        draft = $false
        prerelease = $false
    }
    
    $jsonBody = $releaseData | ConvertTo-Json -Depth 3
    
    Write-Host "Creating release..." -ForegroundColor Cyan
    $releaseResponse = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases" -Method POST -Headers $headers -Body $jsonBody
    
    Write-Host "✓ Release created successfully!" -ForegroundColor Green
    Write-Host "Release ID: $($releaseResponse.id)" -ForegroundColor Cyan
    Write-Host "Release URL: $($releaseResponse.html_url)" -ForegroundColor Cyan
    
    # Upload asset if ZIP file exists
    if (Test-Path $ZipFile) {
        Write-Host "`nUploading asset: $ZipFile" -ForegroundColor Cyan
        
        $uploadUrl = $releaseResponse.upload_url -replace '\{\?.*\}', "?name=$ZipFile"
        $fileBytes = [System.IO.File]::ReadAllBytes((Resolve-Path $ZipFile).Path)
        
        $uploadHeaders = @{
            'Authorization' = "Bearer $Token"
            'Content-Type' = 'application/zip'
            'User-Agent' = 'PowerShell-GitHub-Release'
        }
        
        $assetResponse = Invoke-RestMethod -Uri $uploadUrl -Method POST -Headers $uploadHeaders -Body $fileBytes
        
        Write-Host "✓ Asset uploaded successfully!" -ForegroundColor Green
        Write-Host "Asset URL: $($assetResponse.browser_download_url)" -ForegroundColor Cyan
    } else {
        Write-Host "! ZIP file not found: $ZipFile" -ForegroundColor Yellow
    }
    
    Write-Host "`n🎉 Release v$Version is ready!" -ForegroundColor Green
    Write-Host "Global admins can now use the web installer:" -ForegroundColor Yellow
    Write-Host "iex ((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/MAnishSingh13275/repset_bridge/main/web-install.ps1'))" -ForegroundColor Cyan
    
} catch {
    Write-Host "✗ Failed to create release: $($_.Exception.Message)" -ForegroundColor Red
    
    if ($_.Exception.Message -match "401") {
        Write-Host "Authentication failed. Please check your token." -ForegroundColor Red
    } elseif ($_.Exception.Message -match "422") {
        Write-Host "Release may already exist. Check GitHub releases page." -ForegroundColor Yellow
    }
}