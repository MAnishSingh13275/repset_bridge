# File Cleanup Automation Script (PowerShell Wrapper)
# This script provides a PowerShell interface to the Go-based cleanup automation

param(
    [switch]$DryRun,
    [switch]$Verbose,
    [string]$BackupDir = "",
    [switch]$Help
)

function Show-Usage {
    Write-Host "File Cleanup Automation Script" -ForegroundColor Green
    Write-Host "Usage: .\cleanup-automation.ps1 [options]" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Options:" -ForegroundColor Cyan
    Write-Host "  -DryRun       Show what would be done without making changes"
    Write-Host "  -Verbose      Enable verbose output"
    Write-Host "  -BackupDir    Specify backup directory (default: cleanup-backup-TIMESTAMP)"
    Write-Host "  -Help         Show this help message"
    Write-Host ""
    Write-Host "Examples:" -ForegroundColor Cyan
    Write-Host "  .\cleanup-automation.ps1 -DryRun -Verbose"
    Write-Host "  .\cleanup-automation.ps1 -BackupDir 'my-backup'"
}

if ($Help) {
    Show-Usage
    exit 0
}

# Check if Go is installed
try {
    $goVersion = go version 2>$null
    if (-not $goVersion) {
        throw "Go not found"
    }
    Write-Host "Using Go: $goVersion" -ForegroundColor Green
} catch {
    Write-Error "Go is not installed or not in PATH. Please install Go to run this script."
    Write-Host "Download Go from: https://golang.org/dl/" -ForegroundColor Yellow
    exit 1
}

# Build arguments for the Go script
$goArgs = @()

if ($DryRun) {
    $goArgs += "--dry-run"
}

if ($Verbose) {
    $goArgs += "--verbose"
}

if ($BackupDir -ne "") {
    $goArgs += "--backup-dir"
    $goArgs += $BackupDir
}

# Get the script directory
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$goScript = Join-Path $scriptDir "cleanup-automation.go"

# Check if the Go script exists
if (-not (Test-Path $goScript)) {
    Write-Error "Go script not found at: $goScript"
    exit 1
}

Write-Host "Starting file cleanup automation..." -ForegroundColor Green
Write-Host "Script location: $goScript" -ForegroundColor Gray

try {
    # Change to the project root directory (parent of scripts)
    $projectRoot = Split-Path -Parent $scriptDir
    Push-Location $projectRoot
    
    # Run the Go script
    if ($goArgs.Count -gt 0) {
        & go run $goScript $goArgs
    } else {
        & go run $goScript
    }
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Cleanup completed successfully!" -ForegroundColor Green
    } else {
        Write-Error "Cleanup failed with exit code: $LASTEXITCODE"
    }
} catch {
    Write-Error "Failed to run cleanup script: $_"
} finally {
    Pop-Location
}