# Web-based installer for Gym Door Bridge
# This script downloads and installs the latest version

param(
    [string]$Version = "latest",
    [string]$InstallPath = "$env:ProgramFiles\GymDoorBridge"
)

# GitHub repository details
$RepoOwner = "your-org"
$RepoName = "gym-door-bridge"
$BaseUrl = "https://github.com/$RepoOwner/$RepoName"

function Test-Administrator {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Get-LatestVersion {
    try {
        $apiUrl = "https://api.github.com/repos/$RepoOwner/$RepoName/releases/latest"
        $response = Invoke-RestMethod -Uri $apiUrl -ErrorAction Stop
        return $response.tag_name
    }
    catch {
        Write-Host "Warning: Could not fetch latest version, using 'v1.0.0'" -ForegroundColor Yellow
        return "v1.0.0"
    }
}

function Download-File {
    param(
        [string]$Url,
        [string]$OutputPath
    )
    
    try {
        Write-Host "Downloading: $Url" -ForegroundColor Green
        Invoke-WebRequest -Uri $Url -OutFile $OutputPath -ErrorAction Stop
        return $true
    }
    catch {
        Write-Host "Download failed: $($_.Exception.Message)" -ForegroundColor Red
        return $false
    }
}

# Main installation
Write-Host "=== Gym Door Bridge Web Installer ===" -ForegroundColor Cyan
Write-Host ""

# Check administrator privileges
if (-not (Test-Administrator)) {
    Write-Host "ERROR: This installer requires administrator privileges." -ForegroundColor Red
    Write-Host "Please run PowerShell as Administrator and try again." -ForegroundColor Yellow
    exit 1
}

# Get version to install
if ($Version -eq "latest") {
    $Version = Get-LatestVersion
    Write-Host "Latest version: $Version" -ForegroundColor Green
}

# Create temp directory
$tempDir = Join-Path $env:TEMP "GymDoorBridge-WebInstall"
if (Test-Path $tempDir) {
    Remove-Item $tempDir -Recurse -Force
}
New-Item -ItemType Directory -Path $tempDir -Force | Out-Null

try {
    # Download executable
    $downloadUrl = "$BaseUrl/releases/download/$Version/gym-door-bridge.exe"
    $exePath = Join-Path $tempDir "gym-door-bridge.exe"
    
    Write-Host "Downloading Gym Door Bridge $Version..." -ForegroundColor Yellow
    
    if (-not (Download-File -Url $downloadUrl -OutputPath $exePath)) {
        throw "Failed to download executable"
    }
    
    # Verify download
    if (-not (Test-Path $exePath)) {
        throw "Downloaded file not found"
    }
    
    Write-Host "Download completed successfully!" -ForegroundColor Green
    
    # Run installer
    Write-Host "Running installation..." -ForegroundColor Yellow
    
    $process = Start-Process -FilePath $exePath -ArgumentList "install" -Wait -PassThru -NoNewWindow
    
    if ($process.ExitCode -eq 0) {
        Write-Host ""
        Write-Host "=== Installation Completed Successfully! ===" -ForegroundColor Green
        Write-Host ""
        Write-Host "Gym Door Bridge has been installed and is running." -ForegroundColor White
        Write-Host ""
        Write-Host "Next steps:" -ForegroundColor Yellow
        Write-Host "1. Pair with platform: gym-door-bridge pair" -ForegroundColor White
        Write-Host "2. Check status: gym-door-bridge status" -ForegroundColor White
        Write-Host ""
    }
    else {
        throw "Installation failed with exit code $($process.ExitCode)"
    }
}
catch {
    Write-Host "ERROR: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host ""
    Write-Host "Manual installation options:" -ForegroundColor Yellow
    Write-Host "1. Download from: $BaseUrl/releases" -ForegroundColor White
    Write-Host "2. Run install.bat as Administrator" -ForegroundColor White
    exit 1
}
finally {
    # Cleanup
    if (Test-Path $tempDir) {
        Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Write-Host "Installation completed!" -ForegroundColor Green