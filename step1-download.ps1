# ================================================================
# RepSet Bridge - Step 1: Download Bridge Files
# Simple and reliable download script
# ================================================================

param(
    [switch]$Silent = $false
)

$ProgressPreference = 'SilentlyContinue'
$ErrorActionPreference = 'Stop'

function Write-Step {
    param([string]$Message, [string]$Level = "Info")
    $timestamp = Get-Date -Format "HH:mm:ss"
    $color = switch ($Level) {
        "Error" { "Red" }
        "Success" { "Green" }
        "Warning" { "Yellow" }
        default { "Cyan" }
    }
    if (-not $Silent) {
        Write-Host "[$timestamp] $Message" -ForegroundColor $color
    }
}

try {
    if (-not $Silent) {
        Clear-Host
        Write-Host ""
        Write-Host "🚀 RepSet Bridge - Step 1: Download" -ForegroundColor Cyan
        Write-Host "=================================" -ForegroundColor Cyan
        Write-Host ""
    }

    # Check admin privileges
    Write-Step "Checking administrator privileges..."
    if (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
        Write-Step "ERROR: Administrator privileges required" "Error"
        Write-Host ""
        Write-Host "Please:" -ForegroundColor Yellow
        Write-Host "1. Right-click PowerShell" -ForegroundColor Gray
        Write-Host "2. Select 'Run as Administrator'" -ForegroundColor Gray
        Write-Host "3. Run this script again" -ForegroundColor Gray
        if (-not $Silent) { Read-Host "Press Enter to exit" }
        exit 1
    }
    Write-Step "✅ Administrator privileges confirmed" "Success"

    # Setup paths
    $TempDir = "$env:TEMP\RepSetBridge"
    $DownloadZip = "$TempDir\gym-door-bridge.zip"
    $ExtractDir = "$TempDir\extracted"
    $DownloadUrl = "https://github.com/MAnishSingh13275/repset_bridge/releases/latest/download/gym-door-bridge-windows.zip"

    # Create temp directory
    Write-Step "Setting up temporary directory..."
    if (Test-Path $TempDir) {
        Remove-Item $TempDir -Recurse -Force
    }
    New-Item -ItemType Directory -Path $TempDir -Force | Out-Null
    New-Item -ItemType Directory -Path $ExtractDir -Force | Out-Null
    Write-Step "✅ Temporary directory created: $TempDir" "Success"

    # Download with multiple fallback methods
    Write-Step "Downloading RepSet Bridge (latest version)..."
    $downloadSuccess = $false

    # Method 1: Invoke-WebRequest
    try {
        Write-Step "Trying download method 1..."
        Invoke-WebRequest -Uri $DownloadUrl -OutFile $DownloadZip -UseBasicParsing -TimeoutSec 120
        if (Test-Path $DownloadZip) {
            $fileSize = (Get-Item $DownloadZip).Length
            if ($fileSize -gt 1MB) {
                $downloadSuccess = $true
                $sizeMB = [math]::Round($fileSize / 1MB, 2)
                Write-Step "✅ Download successful ($sizeMB MB)" "Success"
            }
        }
    } catch {
        Write-Step "Method 1 failed, trying alternative..." "Warning"
    }

    # Method 2: .NET WebClient
    if (-not $downloadSuccess) {
        try {
            Write-Step "Trying download method 2..."
            $webClient = New-Object System.Net.WebClient
            $webClient.Headers.Add("User-Agent", "RepSet-Bridge-Installer")
            $webClient.DownloadFile($DownloadUrl, $DownloadZip)
            if (Test-Path $DownloadZip) {
                $fileSize = (Get-Item $DownloadZip).Length
                if ($fileSize -gt 1MB) {
                    $downloadSuccess = $true
                    $sizeMB = [math]::Round($fileSize / 1MB, 2)
                    Write-Step "✅ Download successful ($sizeMB MB)" "Success"
                }
            }
        } catch {
            Write-Step "Method 2 failed, trying BITS..." "Warning"
        }
    }

    # Method 3: BITS Transfer
    if (-not $downloadSuccess) {
        try {
            Write-Step "Trying download method 3..."
            Import-Module BitsTransfer -ErrorAction Stop
            Start-BitsTransfer -Source $DownloadUrl -Destination $DownloadZip -TransferType Download
            if (Test-Path $DownloadZip) {
                $fileSize = (Get-Item $DownloadZip).Length
                if ($fileSize -gt 1MB) {
                    $downloadSuccess = $true
                    $sizeMB = [math]::Round($fileSize / 1MB, 2)
                    Write-Step "✅ Download successful ($sizeMB MB)" "Success"
                }
            }
        } catch {
            Write-Step "All download methods failed" "Error"
        }
    }

    if (-not $downloadSuccess) {
        Write-Step "DOWNLOAD FAILED" "Error"
        Write-Host ""
        Write-Host "Possible solutions:" -ForegroundColor Yellow
        Write-Host "1. Check internet connection" -ForegroundColor Gray
        Write-Host "2. Disable antivirus temporarily" -ForegroundColor Gray
        Write-Host "3. Check Windows Firewall settings" -ForegroundColor Gray
        Write-Host "4. Try running from different network" -ForegroundColor Gray
        if (-not $Silent) { Read-Host "Press Enter to exit" }
        exit 1
    }

    # Extract files
    Write-Step "Extracting bridge files..."
    try {
        # Use .NET extraction for reliability
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        [System.IO.Compression.ZipFile]::ExtractToDirectory($DownloadZip, $ExtractDir)
        Write-Step "✅ Files extracted successfully" "Success"
    } catch {
        try {
            # Fallback to PowerShell method
            Expand-Archive -Path $DownloadZip -DestinationPath $ExtractDir -Force
            Write-Step "✅ Files extracted successfully" "Success"
        } catch {
            Write-Step "EXTRACTION FAILED: $($_.Exception.Message)" "Error"
            exit 1
        }
    }

    # Verify extraction
    $extractedFiles = Get-ChildItem -Path $ExtractDir -Recurse -File
    Write-Step "Verifying extracted files..."
    
    $bridgeExe = $extractedFiles | Where-Object { $_.Name -match "gym-door-bridge.*\.exe|bridge.*\.exe" } | Select-Object -First 1
    if (-not $bridgeExe) {
        Write-Step "ERROR: Bridge executable not found in download" "Error"
        exit 1
    }
    
    Write-Step "✅ Found bridge executable: $($bridgeExe.Name)" "Success"
    Write-Step "✅ Extracted $($extractedFiles.Count) files" "Success"

    # Create info file for next step
    $infoFile = "$TempDir\download-info.json"
    $downloadInfo = @{
        downloadTime = (Get-Date).ToString()
        tempDir = $TempDir
        extractDir = $ExtractDir
        bridgeExe = $bridgeExe.FullName
        bridgeExeName = $bridgeExe.Name
        totalFiles = $extractedFiles.Count
        downloadSize = $sizeMB
    }
    $downloadInfo | ConvertTo-Json | Set-Content $infoFile

    if (-not $Silent) {
        Write-Host ""
        Write-Host "🎉 STEP 1 COMPLETED SUCCESSFULLY!" -ForegroundColor Green
        Write-Host "================================" -ForegroundColor Green
        Write-Host ""
        Write-Host "📋 Download Summary:" -ForegroundColor Cyan
        Write-Host "   📁 Files location: $ExtractDir" -ForegroundColor Gray
        Write-Host "   💾 Download size: $sizeMB MB" -ForegroundColor Gray
        Write-Host "   📦 Files extracted: $($extractedFiles.Count)" -ForegroundColor Gray
        Write-Host "   🔧 Bridge executable: $($bridgeExe.Name)" -ForegroundColor Gray
        Write-Host ""
        Write-Host "✅ Ready for Step 2: Installation" -ForegroundColor Green
        Write-Host ""
        Write-Host "Next: Run the installation script:" -ForegroundColor Yellow
        Write-Host "   .\step2-install.ps1" -ForegroundColor Gray
        Write-Host ""
        
        Read-Host "Press Enter to continue"
    }

} catch {
    Write-Step "UNEXPECTED ERROR: $($_.Exception.Message)" "Error"
    if (-not $Silent) { Read-Host "Press Enter to exit" }
    exit 1
}