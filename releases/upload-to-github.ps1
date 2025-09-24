# GitHub Release Upload Script
# This script will help you upload GymDoorBridge-v1.1.0.zip to GitHub releases

Write-Host "🚀 GitHub Release Upload Guide" -ForegroundColor Green
Write-Host "================================" -ForegroundColor Green
Write-Host ""

# Check if GitHub CLI is installed
$ghInstalled = Get-Command gh -ErrorAction SilentlyContinue

if (-not $ghInstalled) {
    Write-Host "📥 Step 1: Install GitHub CLI" -ForegroundColor Yellow
    Write-Host "GitHub CLI is not installed. Please install it first:" -ForegroundColor White
    Write-Host ""
    Write-Host "Option A - Using winget (Windows 11/10):" -ForegroundColor Cyan
    Write-Host "  winget install --id GitHub.cli" -ForegroundColor Gray
    Write-Host ""
    Write-Host "Option B - Using Chocolatey:" -ForegroundColor Cyan  
    Write-Host "  choco install gh" -ForegroundColor Gray
    Write-Host ""
    Write-Host "Option C - Manual download:" -ForegroundColor Cyan
    Write-Host "  Download from: https://github.com/cli/cli/releases" -ForegroundColor Gray
    Write-Host ""
    Write-Host "After installation, restart PowerShell and run this script again." -ForegroundColor Yellow
    Write-Host ""
    exit 1
} else {
    Write-Host "✅ GitHub CLI is installed!" -ForegroundColor Green
    Write-Host ""
}

# Check authentication
Write-Host "🔐 Step 2: Check GitHub Authentication" -ForegroundColor Yellow
try {
    $authStatus = gh auth status 2>&1
    Write-Host "✅ GitHub authentication verified" -ForegroundColor Green
    Write-Host ""
} catch {
    Write-Host "❌ Not authenticated with GitHub" -ForegroundColor Red
    Write-Host "Please run: gh auth login" -ForegroundColor Cyan
    Write-Host ""
    exit 1
}

# Set variables
$repoOwner = "MAnishSingh13275"
$repoName = "repset_bridge"
$version = "v1.1.0"
$zipFile = "..\GymDoorBridge-v1.1.0.zip"
$releaseNotes = "..\RELEASE_NOTES_v1.1.0.md"

# Check if files exist
if (-not (Test-Path $zipFile)) {
    Write-Host "❌ ZIP file not found: $zipFile" -ForegroundColor Red
    Write-Host "Please ensure GymDoorBridge-v1.1.0.zip exists in the parent directory" -ForegroundColor Yellow
    exit 1
}

if (-not (Test-Path $releaseNotes)) {
    Write-Host "⚠️ Release notes not found: $releaseNotes" -ForegroundColor Yellow
    Write-Host "Creating release without detailed notes..." -ForegroundColor Gray
    $releaseNotes = $null
}

Write-Host "📦 Step 3: Create GitHub Release" -ForegroundColor Yellow
Write-Host "Repository: $repoOwner/$repoName" -ForegroundColor White
Write-Host "Version: $version" -ForegroundColor White
Write-Host "Asset: $zipFile" -ForegroundColor White
Write-Host ""

# Confirm before proceeding
$confirm = Read-Host "Do you want to create the release? (Y/n)"
if ($confirm.ToLower() -eq "n") {
    Write-Host "Operation cancelled." -ForegroundColor Yellow
    exit 0
}

try {
    # Create the release
    Write-Host "Creating release..." -ForegroundColor Cyan
    
    if ($releaseNotes) {
        # With release notes
        $result = gh release create $version $zipFile --repo "$repoOwner/$repoName" --title "Gym Door Bridge $version" --notes-file $releaseNotes
    } else {
        # Without release notes  
        $result = gh release create $version $zipFile --repo "$repoOwner/$repoName" --title "Gym Door Bridge $version" --notes "Production-ready release with updated configuration for repset.onezy.in platform."
    }
    
    Write-Host ""
    Write-Host "🎉 SUCCESS! Release created successfully!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Release URL: https://github.com/$repoOwner/$repoName/releases/tag/$version" -ForegroundColor Cyan
    Write-Host "Download URL: https://github.com/$repoOwner/$repoName/releases/download/$version/GymDoorBridge-v1.1.0.zip" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "✅ Your admin dashboard installer will now download v1.1.0 automatically!" -ForegroundColor Green
    
} catch {
    Write-Host ""
    Write-Host "❌ Error creating release: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host ""
    Write-Host "Manual steps:" -ForegroundColor Yellow
    Write-Host "1. Go to: https://github.com/$repoOwner/$repoName/releases/new" -ForegroundColor Gray
    Write-Host "2. Tag: $version" -ForegroundColor Gray  
    Write-Host "3. Title: Gym Door Bridge $version" -ForegroundColor Gray
    Write-Host "4. Upload: $zipFile" -ForegroundColor Gray
    Write-Host "5. Publish release" -ForegroundColor Gray
    
    exit 1
}

Write-Host ""
Write-Host "🔄 Next Steps:" -ForegroundColor Cyan
Write-Host "1. Test the installer from your admin dashboard" -ForegroundColor White
Write-Host "2. Verify it downloads v1.1.0 from the new release" -ForegroundColor White  
Write-Host "3. Deploy to gym locations" -ForegroundColor White
Write-Host ""
Write-Host "Done! 🎉" -ForegroundColor Green