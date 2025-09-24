param(
    [string]$Version = "1.0.0",
    [string]$Token = ""
)

if ([string]::IsNullOrEmpty($Token)) {
    Write-Host "Error: GitHub token required" -ForegroundColor Red
    Write-Host "Usage: .\simple-release.ps1 -Version '1.0.0' -Token 'your_token'" -ForegroundColor Yellow
    exit 1
}

$repo = "MAnishSingh13275/repset_bridge"
$tagName = "v$Version"
$zipFile = "GymDoorBridge-v$Version.zip"

Write-Host "Creating release v$Version..." -ForegroundColor Green

$releaseNotes = "# GymDoorBridge v$Version

## Installation

### Web Installer (Recommended)
Run as Administrator:
```powershell
iex ((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/MAnishSingh13275/repset_bridge/main/web-install.ps1'))
```

### Manual Installation
1. Download the ZIP file below
2. Extract and run GymDoorBridge-Installer.ps1 as Administrator

## System Requirements
- Windows 10/11 or Windows Server 2019+
- Administrator privileges
- Network access for device discovery"

$headers = @{
    'Authorization' = "Bearer $Token"
    'Accept' = 'application/vnd.github.v3+json'
    'User-Agent' = 'PowerShell'
}

$releaseData = @{
    tag_name = $tagName
    target_commitish = "main"
    name = "GymDoorBridge v$Version"
    body = $releaseNotes
    draft = $false
    prerelease = $false
} | ConvertTo-Json -Depth 3

try {
    Write-Host "Creating release..." -ForegroundColor Cyan
    $releaseResponse = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases" -Method POST -Headers $headers -Body $releaseData
    
    Write-Host "Release created: $($releaseResponse.html_url)" -ForegroundColor Green
    
    if (Test-Path $zipFile) {
        Write-Host "Uploading $zipFile..." -ForegroundColor Cyan
        
        $uploadUrl = $releaseResponse.upload_url -replace '\{\?.*\}', "?name=$zipFile"
        $fileBytes = [System.IO.File]::ReadAllBytes((Resolve-Path $zipFile).Path)
        
        $uploadHeaders = $headers.Clone()
        $uploadHeaders['Content-Type'] = 'application/zip'
        
        $assetResponse = Invoke-RestMethod -Uri $uploadUrl -Method POST -Headers $uploadHeaders -Body $fileBytes
        
        Write-Host "Asset uploaded: $($assetResponse.browser_download_url)" -ForegroundColor Green
    }
    
    Write-Host ""
    Write-Host "SUCCESS! Release v$Version is live!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Global installer command:" -ForegroundColor Yellow
    Write-Host "iex ((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/MAnishSingh13275/repset_bridge/main/web-install.ps1'))" -ForegroundColor Cyan
    
} catch {
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.Exception.Message -match "422") {
        Write-Host "Release v$Version may already exist" -ForegroundColor Yellow
    }
}