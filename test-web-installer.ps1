# Test Web Installer - Simulates global admin experience
Write-Host "🧪 Testing GymDoorBridge Web Installer" -ForegroundColor Green
Write-Host "=======================================" -ForegroundColor Green
Write-Host ""

# Test the exact command global admins will use
$webInstallerUrl = "https://raw.githubusercontent.com/MAnishSingh13275/repset_bridge/main/web-install.ps1"

Write-Host "Testing global admin command:" -ForegroundColor Yellow
Write-Host "iex ((New-Object System.Net.WebClient).DownloadString('$webInstallerUrl'))" -ForegroundColor Cyan
Write-Host ""

Write-Host "Step 1: Downloading web installer script..." -ForegroundColor Cyan
try {
    $webClient = New-Object System.Net.WebClient
    $scriptContent = $webClient.DownloadString($webInstallerUrl)
    Write-Host "✓ Web installer script downloaded successfully!" -ForegroundColor Green
    Write-Host "  Size: $($scriptContent.Length) characters" -ForegroundColor Gray
} catch {
    Write-Host "✗ Failed to download web installer: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "Step 2: Checking script contains fix..." -ForegroundColor Cyan
if ($scriptContent -match "MAnishSingh13275/repset_bridge") {
    Write-Host "✓ Script contains correct repository URL!" -ForegroundColor Green
} else {
    Write-Host "✗ Script still contains placeholder URLs!" -ForegroundColor Red
}

Write-Host ""
Write-Host "Step 3: Testing GitHub release access..." -ForegroundColor Cyan
try {
    $headers = @{ 'User-Agent' = 'GymDoorBridge-WebInstaller' }
    $apiUrl = "https://api.github.com/repos/MAnishSingh13275/repset_bridge/releases/latest"
    $releaseInfo = Invoke-RestMethod -Uri $apiUrl -Headers $headers
    
    $zipAsset = $releaseInfo.assets | Where-Object { $_.name -match "GymDoorBridge-v.*\.zip" } | Select-Object -First 1
    
    if ($zipAsset) {
        Write-Host "✓ GitHub release and assets accessible!" -ForegroundColor Green
        Write-Host "  Version: $($releaseInfo.tag_name)" -ForegroundColor Gray
        Write-Host "  Asset: $($zipAsset.name)" -ForegroundColor Gray
        Write-Host "  Download URL: $($zipAsset.browser_download_url)" -ForegroundColor Gray
        
        # Test actual download (just headers)
        Write-Host ""
        Write-Host "Step 4: Testing asset download..." -ForegroundColor Cyan
        $assetResponse = Invoke-WebRequest -Uri $zipAsset.browser_download_url -Method HEAD -UseBasicParsing
        Write-Host "✓ Asset download accessible!" -ForegroundColor Green
        Write-Host "  Status: $($assetResponse.StatusCode)" -ForegroundColor Gray
        Write-Host "  Size: $([math]::Round($assetResponse.Headers['Content-Length'] / 1MB, 2)) MB" -ForegroundColor Gray
        
    } else {
        Write-Host "✗ No ZIP asset found in release!" -ForegroundColor Red
    }
    
} catch {
    Write-Host "✗ GitHub API access failed: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host ""
Write-Host "🎉 TEST COMPLETE!" -ForegroundColor Green
Write-Host ""
Write-Host "The web installer should now work for global admins." -ForegroundColor White
Write-Host "Updated command:" -ForegroundColor Yellow
Write-Host "iex ((New-Object System.Net.WebClient).DownloadString('$webInstallerUrl'))" -ForegroundColor White -BackgroundColor DarkBlue
Write-Host ""