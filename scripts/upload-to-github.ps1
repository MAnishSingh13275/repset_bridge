# GitHub Release Upload Script
# This script will help you upload release assets to GitHub releases

Write-Host "GitHub Release Upload Guide" -ForegroundColor Green
Write-Host "================================" -ForegroundColor Green
Write-Host ""

# Check if GitHub CLI is installed
$ghInstalled = Get-Command gh -ErrorAction SilentlyContinue

if (-not $ghInstalled) {
    Write-Host "Step 1: Install GitHub CLI" -ForegroundColor Yellow
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
    Write-Host "GitHub CLI is installed!" -ForegroundColor Green
    Write-Host ""
}

# Check authentication
Write-Host "Step 2: Check GitHub Authentication" -ForegroundColor Yellow
try {
    $authStatus = gh auth status 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "GitHub authentication verified" -ForegroundColor Green
        Write-Host ""
    } else {
        Write-Host "Not authenticated with GitHub" -ForegroundColor Red
        Write-Host "Please run: gh auth login --web" -ForegroundColor Cyan
        Write-Host ""
        exit 1
    }
} catch {
    Write-Host "Not authenticated with GitHub" -ForegroundColor Red
    Write-Host "Please run: gh auth login --web" -ForegroundColor Cyan
    Write-Host ""
    exit 1
}

# Parameters
param(
    [Parameter(Mandatory=$true)]
    [string]$Version,
    
    [Parameter(Mandatory=$true)]
    [string]$ZipFile,
    
    [string]$RepoOwner = "MAnishSingh13275",
    [string]$RepoName = "repset_bridge",
    [string]$ReleaseNotes = $null
)

# Check if files exist
if (-not (Test-Path $ZipFile)) {
    Write-Host "ZIP file not found: $ZipFile" -ForegroundColor Red
    Write-Host "Please ensure the ZIP file exists" -ForegroundColor Yellow
    exit 1
}

if ($ReleaseNotes -and (-not (Test-Path $ReleaseNotes))) {
    Write-Host "Release notes file not found: $ReleaseNotes" -ForegroundColor Yellow
    Write-Host "Creating release without detailed notes..." -ForegroundColor Gray
    $ReleaseNotes = $null
}

Write-Host "Step 3: Create GitHub Release" -ForegroundColor Yellow
Write-Host "Repository: $RepoOwner/$RepoName" -ForegroundColor White
Write-Host "Version: $Version" -ForegroundColor White
Write-Host "Asset: $ZipFile" -ForegroundColor White
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
    
    if ($ReleaseNotes) {
        # With release notes
        Write-Host "Using release notes file: $ReleaseNotes" -ForegroundColor Gray
        $result = gh release create $Version $ZipFile --repo "$RepoOwner/$RepoName" --title "Gym Door Bridge $Version" --notes-file $ReleaseNotes
    } else {
        # Without release notes  
        $result = gh release create $Version $ZipFile --repo "$RepoOwner/$RepoName" --title "Gym Door Bridge $Version" --notes "Production-ready release with updated configuration."
    }
    
    Write-Host ""
    Write-Host "SUCCESS! Release created successfully!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Release URL: https://github.com/$RepoOwner/$RepoName/releases/tag/$Version" -ForegroundColor Cyan
    Write-Host "Download URL: https://github.com/$RepoOwner/$RepoName/releases/download/$Version/$([System.IO.Path]::GetFileName($ZipFile))" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Release is now available for download!" -ForegroundColor Green
    
} catch {
    Write-Host ""
    Write-Host "Error creating release: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host ""
    Write-Host "Manual steps:" -ForegroundColor Yellow
    Write-Host "1. Go to: https://github.com/$RepoOwner/$RepoName/releases/new" -ForegroundColor Gray
    Write-Host "2. Tag: $Version" -ForegroundColor Gray  
    Write-Host "3. Title: Gym Door Bridge $Version" -ForegroundColor Gray
    Write-Host "4. Upload: $ZipFile" -ForegroundColor Gray
    Write-Host "5. Publish release" -ForegroundColor Gray
    
    exit 1
}

Write-Host ""
Write-Host "Next Steps:" -ForegroundColor Cyan
Write-Host "1. Test the installer from your admin dashboard" -ForegroundColor White
Write-Host "2. Verify it downloads the correct version" -ForegroundColor White  
Write-Host "3. Deploy to target locations" -ForegroundColor White
Write-Host ""
Write-Host "Done!" -ForegroundColor Green