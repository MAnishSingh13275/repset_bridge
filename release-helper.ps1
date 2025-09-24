# Interactive GitHub Release Helper
Clear-Host

Write-Host "🚀 GymDoorBridge GitHub Release Creator" -ForegroundColor Green
Write-Host "=====================================" -ForegroundColor Green
Write-Host ""

# Check if ZIP exists
$zipFile = "GymDoorBridge-v1.0.0.zip"
if (-not (Test-Path $zipFile)) {
    Write-Host "❌ ZIP file not found: $zipFile" -ForegroundColor Red
    Write-Host "Run this first: .\make-release.ps1 -Version '1.0.0'" -ForegroundColor Yellow
    Read-Host "Press Enter to exit"
    exit 1
}

Write-Host "✅ Found release package: $zipFile" -ForegroundColor Green
$zipInfo = Get-Item $zipFile
$sizeMB = [math]::Round($zipInfo.Length / 1MB, 2)
Write-Host "   Size: $sizeMB MB" -ForegroundColor Cyan
Write-Host ""

Write-Host "To create the GitHub release, you need a Personal Access Token." -ForegroundColor White
Write-Host ""
Write-Host "📋 How to get a GitHub token:" -ForegroundColor Yellow
Write-Host "1. Go to: https://github.com/settings/tokens" -ForegroundColor Gray
Write-Host "2. Click 'Generate new token (classic)'" -ForegroundColor Gray
Write-Host "3. Select scopes: 'repo' and 'write:packages'" -ForegroundColor Gray
Write-Host "4. Click 'Generate token' and copy it" -ForegroundColor Gray
Write-Host ""

$token = Read-Host "Enter your GitHub Personal Access Token" -MaskInput

if ([string]::IsNullOrWhiteSpace($token)) {
    Write-Host "❌ No token provided. Exiting." -ForegroundColor Red
    Read-Host "Press Enter to exit"
    exit 1
}

Write-Host ""
Write-Host "🔄 Creating GitHub release..." -ForegroundColor Cyan
Write-Host ""

try {
    # Call the release creation script
    & .\create-github-release.ps1 -Version "1.0.0" -Token $token
    
    Write-Host ""
    Write-Host "✅ Success! Your global installer is ready:" -ForegroundColor Green
    Write-Host ""
    Write-Host "Global Admin Command:" -ForegroundColor Yellow
    Write-Host "iex ((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/MAnishSingh13275/repset_bridge/main/web-install.ps1'))" -ForegroundColor White -BackgroundColor DarkBlue
    Write-Host ""
    
} catch {
    Write-Host "❌ Failed to create release: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host ""
Read-Host "Press Enter to exit"