# PowerShell build script for cross-platform binary compilation
# Supports Windows, macOS, and Linux builds with proper signing

param(
    [string]$Action = "build",
    [string]$Version = "",
    [switch]$SkipSigning
)

# Configuration
$ProjectName = "gym-door-bridge"
$BuildDir = "build"
$DistDir = "dist"

# Get version from git or use provided version
if (-not $Version) {
    try {
        $Version = git describe --tags --always --dirty 2>$null
        if (-not $Version) { $Version = "dev" }
    } catch {
        $Version = "dev"
    }
}

# Build flags
$BuildTime = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$BuildFlags = @(
    "-trimpath"
    "-ldflags=-s -w -X main.version=$Version -X main.buildTime=$BuildTime"
)

# Logging functions
function Write-Log {
    param([string]$Message)
    Write-Host "[BUILD] $Message" -ForegroundColor Green
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[WARN] $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
    exit 1
}

# Clean previous builds
function Clean-Build {
    Write-Log "Cleaning previous builds..."
    if (Test-Path $BuildDir) { Remove-Item -Recurse -Force $BuildDir }
    if (Test-Path $DistDir) { Remove-Item -Recurse -Force $DistDir }
    New-Item -ItemType Directory -Path $BuildDir -Force | Out-Null
    New-Item -ItemType Directory -Path $DistDir -Force | Out-Null
}

# Build for specific platform
function Build-Platform {
    param(
        [string]$GOOS,
        [string]$GOARCH,
        [string]$Extension = ""
    )
    
    $OutputName = "$ProjectName-$GOOS-$GOARCH$Extension"
    Write-Log "Building $OutputName..."
    
    $env:GOOS = $GOOS
    $env:GOARCH = $GOARCH
    $env:CGO_ENABLED = "1"
    
    $BuildArgs = $BuildFlags + @("-o", "$BuildDir\$OutputName", ".\cmd")
    
    try {
        & go build @BuildArgs
        if ($LASTEXITCODE -eq 0) {
            Write-Log "✓ Built $OutputName"
        } else {
            Write-Error "✗ Failed to build $OutputName"
        }
    } catch {
        Write-Error "✗ Failed to build $OutputName`: $_"
    } finally {
        Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
        Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
        Remove-Item Env:\CGO_ENABLED -ErrorAction SilentlyContinue
    }
}

# Sign Windows binary
function Sign-WindowsBinary {
    param([string]$BinaryPath)
    
    if (-not $env:WINDOWS_CERT_PATH -or -not $env:WINDOWS_CERT_PASSWORD) {
        Write-Warn "Windows signing certificate not configured, skipping signing"
        return
    }
    
    # Check if signtool is available
    $SignTool = Get-Command "signtool.exe" -ErrorAction SilentlyContinue
    if (-not $SignTool) {
        Write-Warn "signtool.exe not found, skipping Windows binary signing"
        return
    }
    
    Write-Log "Signing Windows binary: $BinaryPath"
    
    try {
        & signtool.exe sign `
            /f $env:WINDOWS_CERT_PATH `
            /p $env:WINDOWS_CERT_PASSWORD `
            /n $ProjectName `
            /d "Gym Door Access Bridge" `
            /du "https://repset.onezy.in" `
            /t "http://timestamp.digicert.com" `
            $BinaryPath
        
        if ($LASTEXITCODE -eq 0) {
            Write-Log "✓ Windows binary signed"
        } else {
            Write-Warn "Failed to sign Windows binary"
        }
    } catch {
        Write-Warn "Failed to sign Windows binary: $_"
    }
}

# Create checksums
function Create-Checksums {
    Write-Log "Creating checksums..."
    
    $ChecksumFile = Join-Path $BuildDir "checksums.txt"
    Get-ChildItem -Path $BuildDir -File | ForEach-Object {
        if ($_.Name -ne "checksums.txt") {
            $Hash = Get-FileHash -Path $_.FullName -Algorithm SHA256
            "$($Hash.Hash.ToLower())  $($_.Name)" | Add-Content -Path $ChecksumFile
        }
    }
    
    Write-Log "✓ Checksums created"
}

# Package binaries
function Package-Binaries {
    Write-Log "Packaging binaries..."
    
    Get-ChildItem -Path $BuildDir -File | Where-Object { 
        $_.Name -like "$ProjectName-*" -and $_.Extension -ne ".zip" -and $_.Extension -ne ".gz"
    } | ForEach-Object {
        $ArchiveName = $_.Name
        $ChecksumFile = Join-Path $BuildDir "checksums.txt"
        
        if ($_.Name -like "*windows*") {
            # Create ZIP for Windows
            $ZipPath = Join-Path $DistDir "$ArchiveName.zip"
            Compress-Archive -Path $_.FullName, $ChecksumFile -DestinationPath $ZipPath -Force
        } else {
            # Create tar.gz for other platforms
            $TarPath = Join-Path $DistDir "$ArchiveName.tar.gz"
            
            # Use tar if available, otherwise create zip
            $Tar = Get-Command "tar.exe" -ErrorAction SilentlyContinue
            if ($Tar) {
                Push-Location $BuildDir
                try {
                    & tar.exe -czf "..\$TarPath" $_.Name "checksums.txt"
                } finally {
                    Pop-Location
                }
            } else {
                # Fallback to ZIP
                $ZipPath = Join-Path $DistDir "$ArchiveName.zip"
                Compress-Archive -Path $_.FullName, $ChecksumFile -DestinationPath $ZipPath -Force
            }
        }
    }
    
    Write-Log "✓ Binaries packaged"
}

# Main build process
function Start-Build {
    Write-Log "Starting build process for $ProjectName v$Version"
    
    # Check Go installation
    if (-not (Get-Command "go.exe" -ErrorAction SilentlyContinue)) {
        Write-Error "Go is not installed or not in PATH"
    }
    
    # Clean previous builds
    Clean-Build
    
    # Build for different platforms
    Write-Log "Building binaries..."
    
    # Windows builds
    Build-Platform -GOOS "windows" -GOARCH "amd64" -Extension ".exe"
    Build-Platform -GOOS "windows" -GOARCH "386" -Extension ".exe"
    
    # macOS builds
    Build-Platform -GOOS "darwin" -GOARCH "amd64"
    Build-Platform -GOOS "darwin" -GOARCH "arm64"
    
    # Linux builds (for Docker)
    Build-Platform -GOOS "linux" -GOARCH "amd64"
    Build-Platform -GOOS "linux" -GOARCH "arm64"
    
    # Sign binaries
    if (-not $SkipSigning) {
        Write-Log "Signing binaries..."
        
        # Sign Windows binaries
        Get-ChildItem -Path $BuildDir -Filter "$ProjectName-windows-*.exe" | ForEach-Object {
            Sign-WindowsBinary -BinaryPath $_.FullName
        }
        
        # Note: macOS signing requires macOS environment
        if ($env:OS -eq "Windows_NT") {
            Write-Warn "macOS binary signing requires macOS environment"
        }
    }
    
    # Create checksums and package
    Create-Checksums
    Package-Binaries
    
    Write-Log "Build completed successfully!"
    Write-Log "Binaries available in: $DistDir\"
    Get-ChildItem -Path $DistDir | Format-Table Name, Length, LastWriteTime
}

# Handle command line arguments
switch ($Action.ToLower()) {
    "clean" {
        Clean-Build
    }
    "build" {
        Start-Build
    }
    default {
        Write-Error "Unknown action: $Action. Use 'build' or 'clean'"
    }
}