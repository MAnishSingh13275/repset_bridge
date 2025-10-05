# Complete Release Creation Script for Gym Door Bridge
# This script handles: commit, tag, build, and GitHub release creation

param(
    [string]$Version = "",
    [string]$Message = "",
    [switch]$DryRun = $false,
    [switch]$SkipBuild = $false,
    [switch]$SkipSigning = $false
)

# Configuration
$ProjectName = "Gym Door Bridge"
$RepoOwner = "MAnishSingh13275"
$RepoName = "repset_bridge"

# Helper functions
function Write-Step {
    param([string]$Message)
    Write-Host ""
    Write-Host "🔄 $Message" -ForegroundColor Cyan
}

function Write-Success {
    param([string]$Message)
    Write-Host "✅ $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "⚠️  $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "❌ $Message" -ForegroundColor Red
    exit 1
}

# Validate prerequisites
function Test-Prerequisites {
    Write-Step "Checking prerequisites..."
    
    # Check Git
    if (-not (Get-Command "git.exe" -ErrorAction SilentlyContinue)) {
        Write-Error "Git is not installed or not in PATH"
    }
    
    # Check GitHub CLI
    if (-not (Get-Command "gh.exe" -ErrorAction SilentlyContinue)) {
        Write-Error "GitHub CLI (gh) is not installed or not in PATH"
    }
    
    # Check Go (if not skipping build)
    if (-not $SkipBuild -and -not (Get-Command "go.exe" -ErrorAction SilentlyContinue)) {
        Write-Error "Go is not installed or not in PATH"
    }
    
    # Check if logged into GitHub CLI
    $ghAuth = & gh auth status 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Not logged into GitHub CLI. Run: gh auth login"
    }
    
    Write-Success "All prerequisites met"
}

# Get next version
function Get-NextVersion {
    if ($Version) {
        if (-not $Version.StartsWith("v")) {
            $Version = "v$Version"
        }
        return $Version
    }
    
    Write-Step "Determining next version..."
    
    try {
        $latestTag = git tag --sort=-version:refname | Select-Object -First 1
        if (-not $latestTag) {
            return "v1.0.0"
        }
        
        # Parse version
        if ($latestTag -match 'v(\d+)\.(\d+)\.(\d+)') {
            $major = [int]$matches[1]
            $minor = [int]$matches[2]
            $patch = [int]$matches[3]
            
            # Increment minor version for this major fix release
            $minor++
            $patch = 0
            
            return "v$major.$minor.$patch"
        }
    } catch {
        Write-Warning "Could not determine version from tags, using v1.0.0"
        return "v1.0.0"
    }
    
    return "v1.0.0"
}

# Generate release notes
function Generate-ReleaseNotes {
    param([string]$Version)
    
    $releaseNotes = @"
# 🚀 Gym Door Bridge $Version - Installation & Service Reliability Fixes

This release includes major improvements to installation reliability and service startup on Windows systems.

## 🔧 Major Fixes

### Installation & Service Startup Issues
- **Fixed critical service startup failure** - Resolved "deviceId is required in processor configuration" error
- **Enhanced credential management** - Improved handling of device credentials for Windows services  
- **Machine-wide credential storage** - Credentials now stored in PROGRAMDATA for service accessibility
- **Robust service startup** - Extended retry logic with comprehensive health monitoring

### Enhanced Installation Script
- **One-click installation** - Fully automated installation with better error handling
- **Configuration validation** - Automatic validation of bridge setup after pairing
- **Comprehensive diagnostics** - Enhanced error reporting with Event Log analysis
- **Service health monitoring** - Real-time health checks with API endpoint testing
- **Better permission management** - Proper permissions for LocalService account

## ✨ Improvements

### User Experience
- Clear visual feedback during installation process
- Automatic error recovery and retry mechanisms  
- Actionable troubleshooting guidance when issues occur
- Enhanced validation to catch configuration problems early

### Developer Experience  
- Improved error logging and debug information
- Better separation of concerns in codebase
- Comprehensive installation status reporting
- Machine-wide credential storage compatible with Windows services

## 🐛 Bug Fixes

- Fixed bridge manager failing to initialize event processor due to missing device ID
- Resolved credential access issues for Windows services running as LocalService
- Fixed installation script pairing command argument handling
- Enhanced service startup reliability with proper timeout handling
- Fixed permission issues with credential storage directory

## 📋 Installation

Download and run the installer with administrator privileges:

```powershell
.\install-bridge.ps1 -PairCode "YOUR-PAIR-CODE" -ServerUrl "https://repset.onezy.in" -Force
```

The installation script now handles all edge cases and provides comprehensive feedback.

## 🔍 Verification

After installation, verify the service is running:

```powershell
Get-Service -Name "GymDoorBridge"
gym-door-bridge status
```

## 📖 Documentation

See `INSTALLATION_FIX_SUMMARY.md` for detailed technical information about the fixes applied.

---

**Full Changelog**: [View Changes](https://github.com/$RepoOwner/$RepoName/compare/$((git tag --sort=-version:refname | Select-Object -First 1))...$Version)
"@

    return $releaseNotes
}

# Main release process
function Start-Release {
    param([string]$Version, [string]$ReleaseNotes)
    
    if ($DryRun) {
        Write-Warning "DRY RUN MODE - No actual changes will be made"
    }
    
    Write-Step "Starting release process for $Version"
    
    # 1. Commit and push changes
    Write-Step "Committing and pushing changes..."
    
    if (-not $DryRun) {
        $commitMessage = if ($Message) { $Message } else { "Release $Version - Installation & Service Reliability Fixes" }
        
        git add .
        git commit -m $commitMessage
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Failed to commit changes"
        }
        
        git push origin main
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Failed to push changes"
        }
    }
    
    Write-Success "Changes committed and pushed"
    
    # 2. Create and push tag
    Write-Step "Creating and pushing tag $Version..."
    
    if (-not $DryRun) {
        git tag -a $Version -m "Release $Version"
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Failed to create tag"
        }
        
        git push origin $Version
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Failed to push tag"
        }
    }
    
    Write-Success "Tag $Version created and pushed"
    
    # 3. Build binaries
    if (-not $SkipBuild) {
        Write-Step "Building release binaries..."
        
        if (-not $DryRun) {
            $buildParams = @("-Action", "build", "-Version", $Version.TrimStart('v'))
            if ($SkipSigning) {
                $buildParams += "-SkipSigning"
            }
            
            & .\scripts\build.ps1 @buildParams
            if ($LASTEXITCODE -ne 0) {
                Write-Error "Build failed"
            }
        }
        
        Write-Success "Binaries built successfully"
    } else {
        Write-Warning "Skipping build (SkipBuild flag set)"
    }
    
    # 4. Create GitHub release
    Write-Step "Creating GitHub release..."
    
    if (-not $DryRun) {
        # Save release notes to temp file
        $releaseNotesFile = Join-Path $env:TEMP "release-notes-$Version.md"
        $ReleaseNotes | Set-Content -Path $releaseNotesFile -Encoding UTF8
        
        try {
            # Create release
            $releaseArgs = @(
                "release", "create", $Version
                "--title", "$ProjectName $Version"
                "--notes-file", $releaseNotesFile
            )
            
            # Add assets if build was not skipped
            if (-not $SkipBuild -and (Test-Path "dist")) {
                $assets = Get-ChildItem -Path "dist" -File | ForEach-Object { $_.FullName }
                $releaseArgs += $assets
            }
            
            & gh @releaseArgs
            if ($LASTEXITCODE -ne 0) {
                Write-Error "Failed to create GitHub release"
            }
            
        } finally {
            # Clean up temp file
            if (Test-Path $releaseNotesFile) {
                Remove-Item $releaseNotesFile -Force
            }
        }
    }
    
    Write-Success "GitHub release created successfully"
    
    # 5. Final summary
    Write-Step "Release Summary"
    Write-Host ""
    Write-Host "🎉 Release $Version completed successfully!" -ForegroundColor Green
    Write-Host ""
    Write-Host "📋 What was done:" -ForegroundColor Cyan
    Write-Host "   • Code committed and pushed to main branch" -ForegroundColor White
    Write-Host "   • Tag $Version created and pushed" -ForegroundColor White
    if (-not $SkipBuild) {
        Write-Host "   • Release binaries built and packaged" -ForegroundColor White
    }
    Write-Host "   • GitHub release created with assets" -ForegroundColor White
    Write-Host ""
    Write-Host "🔗 Links:" -ForegroundColor Cyan
    Write-Host "   • Release: https://github.com/$RepoOwner/$RepoName/releases/tag/$Version" -ForegroundColor White
    Write-Host "   • Repository: https://github.com/$RepoOwner/$RepoName" -ForegroundColor White
    Write-Host ""
}

# Main execution
try {
    Write-Host "🚀 Gym Door Bridge Release Creator" -ForegroundColor Magenta
    Write-Host "====================================" -ForegroundColor Magenta
    
    Test-Prerequisites
    
    $nextVersion = Get-NextVersion
    $releaseNotes = Generate-ReleaseNotes -Version $nextVersion
    
    Write-Host ""
    Write-Host "📋 Release Plan:" -ForegroundColor Yellow
    Write-Host "   Version: $nextVersion" -ForegroundColor White
    Write-Host "   Repository: https://github.com/$RepoOwner/$RepoName" -ForegroundColor White
    Write-Host "   Build: $(if ($SkipBuild) { 'Skipped' } else { 'Included' })" -ForegroundColor White
    Write-Host "   Signing: $(if ($SkipSigning) { 'Skipped' } else { 'Included' })" -ForegroundColor White
    Write-Host ""
    
    if ($DryRun) {
        Write-Warning "DRY RUN MODE - Review the plan above"
        Write-Host ""
        Write-Host "Release Notes Preview:" -ForegroundColor Cyan
        Write-Host $releaseNotes -ForegroundColor Gray
        Write-Host ""
        Write-Host "To execute the release, run without -DryRun flag" -ForegroundColor Yellow
    } else {
        $confirm = Read-Host "Proceed with release? (y/N)"
        if ($confirm -eq 'y' -or $confirm -eq 'Y') {
            Start-Release -Version $nextVersion -ReleaseNotes $releaseNotes
        } else {
            Write-Warning "Release cancelled by user"
        }
    }
    
} catch {
    Write-Error "Release process failed: $_"
}