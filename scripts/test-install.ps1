# Test Installation Script
param(
    [string]$PairCode = "AEE5-3378-D2E"
)

Write-Host "🔍 Testing Gym Door Bridge Installation" -ForegroundColor Cyan
Write-Host "=======================================" -ForegroundColor Cyan

try {
    # Check admin privileges
    Write-Host "1. Checking administrator privileges..." -ForegroundColor Yellow
    if (-NOT ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
        Write-Host "❌ Not running as Administrator!" -ForegroundColor Red
        Write-Host "Please run PowerShell as Administrator and try again." -ForegroundColor Yellow
        Read-Host "Press Enter to exit"
        exit 1
    }
    Write-Host "✅ Running as Administrator" -ForegroundColor Green

    # Test internet connectivity
    Write-Host "2. Testing internet connectivity..." -ForegroundColor Yellow
    try {
        $testConnection = Test-NetConnection -ComputerName "github.com" -Port 443 -InformationLevel Quiet
        if ($testConnection) {
            Write-Host "✅ Internet connection OK" -ForegroundColor Green
        } else {
            throw "No internet connection"
        }
    } catch {
        Write-Host "❌ Internet connection failed: $($_.Exception.Message)" -ForegroundColor Red
        Read-Host "Press Enter to exit"
        exit 1
    }

    # Test script download
    Write-Host "3. Testing script download..." -ForegroundColor Yellow
    try {
        $scriptUrl = "https://raw.githubusercontent.com/MAnishSingh13275/repset_bridge/main/scripts/install-bridge.ps1"
        $script = Invoke-WebRequest -Uri $scriptUrl -UseBasicParsing
        
        if ($script.Content.Length -gt 0) {
            Write-Host "✅ Script downloaded successfully: $($script.Content.Length) characters" -ForegroundColor Green
        } else {
            throw "Script is empty"
        }
    } catch {
        Write-Host "❌ Script download failed: $($_.Exception.Message)" -ForegroundColor Red
        Write-Host "URL: $scriptUrl" -ForegroundColor Yellow
        Read-Host "Press Enter to exit"
        exit 1
    }

    # Test GitHub release access
    Write-Host "4. Testing GitHub release access..." -ForegroundColor Yellow
    try {
        $releaseUrl = "https://github.com/MAnishSingh13275/repset_bridge/releases/latest"
        $releaseTest = Invoke-WebRequest -Uri $releaseUrl -UseBasicParsing -Method Head
        Write-Host "✅ GitHub releases accessible" -ForegroundColor Green
    } catch {
        Write-Host "⚠️  GitHub releases may not be accessible: $($_.Exception.Message)" -ForegroundColor Yellow
    }

    # Check if service already exists
    Write-Host "5. Checking existing service..." -ForegroundColor Yellow
    $existingService = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
    if ($existingService) {
        Write-Host "⚠️  Service already exists: $($existingService.Status)" -ForegroundColor Yellow
    } else {
        Write-Host "✅ No existing service found" -ForegroundColor Green
    }

    # Test execution policy
    Write-Host "6. Checking PowerShell execution policy..." -ForegroundColor Yellow
    $executionPolicy = Get-ExecutionPolicy
    Write-Host "Current execution policy: $executionPolicy" -ForegroundColor White
    
    if ($executionPolicy -eq "Restricted") {
        Write-Host "⚠️  Execution policy is Restricted, this may cause issues" -ForegroundColor Yellow
        Write-Host "Consider running: Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser" -ForegroundColor Yellow
    } else {
        Write-Host "✅ Execution policy allows script execution" -ForegroundColor Green
    }

    Write-Host "`n🎉 All tests passed! Ready for installation." -ForegroundColor Green
    Write-Host "`n📋 To proceed with installation:" -ForegroundColor Cyan
    Write-Host "Invoke-Expression `"& { `$(`$script.Content) } -PairCode '$PairCode'`"" -ForegroundColor White
    
    $proceed = Read-Host "`nDo you want to proceed with installation? (y/N)"
    if ($proceed -eq "y" -or $proceed -eq "Y") {
        Write-Host "`n🚀 Starting installation..." -ForegroundColor Green
        Invoke-Expression "& { $($script.Content) } -PairCode '$PairCode'"
    } else {
        Write-Host "Installation cancelled by user." -ForegroundColor Yellow
    }

} catch {
    Write-Host "❌ Test failed: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "Stack trace: $($_.ScriptStackTrace)" -ForegroundColor Red
    Read-Host "Press Enter to exit"
    exit 1
}

Read-Host "Press Enter to exit"