# Safe Installation Test Script
param(
    [string]$PairCode = "AEE5-3378-D2E"
)

Write-Host "🔍 Safe Installation Test" -ForegroundColor Cyan
Write-Host "=========================" -ForegroundColor Cyan

try {
    # Check admin privileges first
    Write-Host "Checking administrator privileges..." -ForegroundColor Yellow
    if (-NOT ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
        Write-Host "❌ Not running as Administrator!" -ForegroundColor Red
        Write-Host "Please run PowerShell as Administrator and try again." -ForegroundColor Yellow
        Read-Host "Press Enter to exit"
        exit 1
    }
    Write-Host "✅ Running as Administrator" -ForegroundColor Green

    # Download the script
    Write-Host "Downloading installation script..." -ForegroundColor Yellow
    $scriptUrl = "https://raw.githubusercontent.com/MAnishSingh13275/repset_bridge/main/scripts/install-bridge.ps1"
    $script = Invoke-WebRequest -Uri $scriptUrl -UseBasicParsing
    Write-Host "✅ Script downloaded: $($script.Content.Length) characters" -ForegroundColor Green

    # Save script to temp file for debugging
    $tempScript = "$env:TEMP\gym-door-bridge-install.ps1"
    $script.Content | Out-File -FilePath $tempScript -Encoding UTF8
    Write-Host "Script saved to: $tempScript" -ForegroundColor White

    # Try to execute with error handling
    Write-Host "Executing installation script with PairCode: $PairCode" -ForegroundColor Yellow
    
    # Method 1: Direct execution with error capture
    Write-Host "Attempting Method 1: Direct execution..." -ForegroundColor Yellow
    try {
        $result = & {
            param($Code)
            # Execute the script content with the pair code
            Invoke-Expression "& { $($script.Content) } -PairCode '$Code'"
        } -Code $PairCode
        
        Write-Host "✅ Method 1 completed successfully" -ForegroundColor Green
    }
    catch {
        Write-Host "❌ Method 1 failed: $($_.Exception.Message)" -ForegroundColor Red
        Write-Host "Error details: $($_.ScriptStackTrace)" -ForegroundColor Yellow
        
        # Method 2: Execute from temp file
        Write-Host "Attempting Method 2: File execution..." -ForegroundColor Yellow
        try {
            & $tempScript -PairCode $PairCode
            Write-Host "✅ Method 2 completed successfully" -ForegroundColor Green
        }
        catch {
            Write-Host "❌ Method 2 failed: $($_.Exception.Message)" -ForegroundColor Red
            Write-Host "Error details: $($_.ScriptStackTrace)" -ForegroundColor Yellow
            
            # Method 3: PowerShell subprocess
            Write-Host "Attempting Method 3: Subprocess execution..." -ForegroundColor Yellow
            try {
                $processArgs = "-ExecutionPolicy Bypass -File `"$tempScript`" -PairCode `"$PairCode`""
                $process = Start-Process -FilePath "powershell.exe" -ArgumentList $processArgs -Wait -PassThru -NoNewWindow
                
                if ($process.ExitCode -eq 0) {
                    Write-Host "✅ Method 3 completed successfully" -ForegroundColor Green
                } else {
                    Write-Host "❌ Method 3 failed with exit code: $($process.ExitCode)" -ForegroundColor Red
                }
            }
            catch {
                Write-Host "❌ Method 3 failed: $($_.Exception.Message)" -ForegroundColor Red
            }
        }
    }

} catch {
    Write-Host "❌ Critical error: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "Stack trace: $($_.ScriptStackTrace)" -ForegroundColor Red
} finally {
    # Cleanup
    if (Test-Path $tempScript) {
        Write-Host "Cleaning up temporary file..." -ForegroundColor Yellow
        Remove-Item $tempScript -Force -ErrorAction SilentlyContinue
    }
}

Write-Host "`n🏁 Test completed. Check the output above for any errors." -ForegroundColor Cyan
Read-Host "Press Enter to exit"