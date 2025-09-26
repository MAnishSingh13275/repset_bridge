# ================================================================
# RepSet Gym Door Bridge - WEB INSTALLER v3.0
# Downloads the final fixed installer and executes it
# ================================================================

param(
    [Parameter(Mandatory=$true)]
    [string]$PairCode
)

$ErrorActionPreference = "Stop"

# Clear screen and show banner
Clear-Host
Write-Host "" 
Write-Host "  ================================================================" -ForegroundColor Green
Write-Host "  |                                                              |" -ForegroundColor Green  
Write-Host "  |              REPSET GYM DOOR BRIDGE INSTALLER               |" -ForegroundColor Green
Write-Host "  |                                                              |" -ForegroundColor Green
Write-Host "  |                    Web Installer v3.0                      |" -ForegroundColor Green
Write-Host "  |                                                              |" -ForegroundColor Green
Write-Host "  ================================================================" -ForegroundColor Green
Write-Host "" 
Write-Host "  Pairing Code: $PairCode" -ForegroundColor Gray
Write-Host "  RepSet Server: https://repset.onezy.in" -ForegroundColor Gray
Write-Host "" 

# Status functions
function Write-Success { param([string]$Message) Write-Host "  [OK] $Message" -ForegroundColor Green }
function Write-Error { param([string]$Message) Write-Host "  [ERROR] $Message" -ForegroundColor Red }
function Write-Warning { param([string]$Message) Write-Host "  [WARNING] $Message" -ForegroundColor Yellow }
function Write-Info { param([string]$Message) Write-Host "  [INFO] $Message" -ForegroundColor Cyan }
function Write-Step { param([string]$Step, [string]$Message) Write-Host "[$Step] $Message" -ForegroundColor White }

try {
    # Check administrator privileges
    Write-Step "1/3" "Checking administrator privileges..."
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    
    if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        throw "This installer requires administrator privileges. Please run PowerShell as Administrator."
    }
    Write-Success "Administrator privileges confirmed"

    # Create temp directory  
    Write-Step "2/3" "Downloading RepSet installer..."
    $tempDir = "$env:TEMP\RepSetWebInstall-$(Get-Random)"
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
    
    # Download the final fixed installer
    $installerUrl = "https://raw.githubusercontent.com/MAnishSingh13275/repset_bridge/main/repset-final-installer.ps1"
    $installerPath = "$tempDir\repset-installer.ps1"
    
    try {
        Invoke-WebRequest -Uri $installerUrl -OutFile $installerPath -UseBasicParsing
        Write-Success "Downloaded RepSet installer"
    } catch {
        throw "Failed to download installer: $($_.Exception.Message)"
    }

    # Execute the main installer with pairing code
    Write-Step "3/3" "Executing RepSet installation..."
    Write-Info "Starting comprehensive installation process..."
    
    try {
        # Run directly in current session to maintain visibility
        & $installerPath -PairCode $PairCode
        
        Write-Success "RepSet installation completed successfully!"
        
    } catch {
        throw "Failed to execute installer: $($_.Exception.Message)"
    }

    # Success message
    Write-Host ""
    Write-Host "  ================================================================" -ForegroundColor Green
    Write-Host "  |                                                              |" -ForegroundColor Green
    Write-Host "  |         REPSET WEB INSTALLATION COMPLETED!                  |" -ForegroundColor Green
    Write-Host "  |                                                              |" -ForegroundColor Green
    Write-Host "  ================================================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "Your gym is now connected to RepSet!" -ForegroundColor Cyan
    Write-Host "Check your RepSet dashboard to verify the connection." -ForegroundColor Gray

} catch {
    Write-Host ""
    Write-Host "  ================================================================" -ForegroundColor Red
    Write-Host "  |                        ERROR                                 |" -ForegroundColor Red
    Write-Host "  |                                                              |" -ForegroundColor Red
    Write-Host "  |             RepSet Web Install Failed                       |" -ForegroundColor Red
    Write-Host "  |                                                              |" -ForegroundColor Red
    Write-Host "  ================================================================" -ForegroundColor Red
    Write-Host ""
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host ""
    Write-Host "Please contact RepSet support with this error message." -ForegroundColor Yellow
    Write-Host "Support available through your RepSet admin dashboard." -ForegroundColor Gray
    Write-Host ""
    Read-Host "Press Enter to exit"
    exit 1
} finally {
    # Cleanup
    if ($tempDir -and (Test-Path $tempDir)) {
        try {
            Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        } catch {}
    }
}

Read-Host "Press Enter to exit"