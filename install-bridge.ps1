# Gym Door Bridge - One-Click Installation Script (Fixed)
# - Stores mutable data in %ProgramData%\GymDoorBridge
# - Uses absolute paths inside config.yaml
# - Enforces correct service binPath with absolute --config
# - Robust download, extract, uninstall, reinstall, pairing, start, and verify

[CmdletBinding()]
param(
    [string]$PairCode = "",
    [string]$ServerUrl = "https://repset.onezy.in",
    [string]$InstallPath = "$env:ProgramFiles\GymDoorBridge",
    [switch]$Force = $false
)

function Install-GymDoorBridge {
    [CmdletBinding()]
    param(
        [string]$PairCode = $PairCode,
        [string]$ServerUrl = $ServerUrl,
        [string]$InstallPath = $InstallPath,
        [switch]$Force = $Force
    )

    # ---- Admin check ----
    if (-NOT ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()
        ).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
        Write-Host "❌ This script requires Administrator privileges!" -ForegroundColor Red
        Write-Host "Please run PowerShell as Administrator and try again." -ForegroundColor Yellow
        return
    }

    Write-Host "🚀 Gym Door Bridge Installation" -ForegroundColor Cyan
    Write-Host "================================" -ForegroundColor Cyan

    # ---- Constants / Paths ----
    $ServiceName  = "GymDoorBridge"
    $ServiceDisp  = "Gym Door Access Bridge"
    $DataDir      = Join-Path $env:ProgramData "GymDoorBridge"
    $TempZip      = Join-Path $env:TEMP "gym-door-bridge.zip"
    $TempExtract  = Join-Path $env:TEMP "gym-door-bridge"
    $DownloadUrl  = "https://github.com/MAnishSingh13275/repset_bridge/releases/latest/download/gym-door-bridge-windows.zip"

    # Ensure folders
    New-Item -ItemType Directory -Force -Path $DataDir     | Out-Null
    New-Item -ItemType Directory -Force -Path $InstallPath | Out-Null

    # Loosen ACLs for service accounts (SYSTEM / LocalService)
    try {
        $acl = Get-Acl $DataDir
        $rules = @(
            New-Object System.Security.AccessControl.FileSystemAccessRule("NT AUTHORITY\SYSTEM","FullControl","ContainerInherit, ObjectInherit","None","Allow"),
            New-Object System.Security.AccessControl.FileSystemAccessRule("NT AUTHORITY\LOCAL SERVICE","Modify","ContainerInherit, ObjectInherit","None","Allow")
        )
        foreach ($r in $rules) { $acl.SetAccessRule($r) }
        Set-Acl -Path $DataDir -AclObject $acl
    } catch {}

    # ---- Service existence ----
    $existingService = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($existingService) {
        Write-Host "⚠️  Gym Door Bridge is already installed!" -ForegroundColor Yellow
        Write-Host "Service Status: $($existingService.Status)" -ForegroundColor White
        if ($PairCode) {
            Write-Host "🔄 Pair code provided - will reinstall and re-pair automatically..." -ForegroundColor Green
            $Force = $true
        } elseif (-not $Force) {
            Write-Host "Use -Force to reinstall or run 'gym-door-bridge status' to check status." -ForegroundColor Yellow
            return
        }
    }

    try {
        # ---- Download ----
        Write-Host "📥 Downloading latest Gym Door Bridge..." -ForegroundColor Green

        if (Test-Path $TempExtract) { Remove-Item $TempExtract -Recurse -Force -ErrorAction SilentlyContinue }
        New-Item -ItemType Directory -Path $TempExtract -Force | Out-Null

        $downloadSuccess = $false

        try {
            Write-Host "Trying download method 1..." -ForegroundColor Yellow
            Invoke-WebRequest -Uri $DownloadUrl -OutFile $TempZip -UseBasicParsing -TimeoutSec 60
            $downloadSuccess = $true
            Write-Host "✅ Download method 1 successful" -ForegroundColor Green
        } catch {
            Write-Host "⚠️  Download method 1 failed: $($_.Exception.Message)" -ForegroundColor Yellow
        }

        if (-not $downloadSuccess) {
            try {
                Write-Host "Trying download method 2..." -ForegroundColor Yellow
                $wc = New-Object System.Net.WebClient
                $wc.DownloadFile($DownloadUrl, $TempZip)
                $downloadSuccess = $true
                Write-Host "✅ Download method 2 successful" -ForegroundColor Green
            } catch {
                Write-Host "⚠️  Download method 2 failed: $($_.Exception.Message)" -ForegroundColor Yellow
            }
        }

        if (-not $downloadSuccess) {
            try {
                Write-Host "Trying download method 3..." -ForegroundColor Yellow
                Import-Module BitsTransfer -ErrorAction SilentlyContinue
                Start-BitsTransfer -Source $DownloadUrl -Destination $TempZip
                $downloadSuccess = $true
                Write-Host "✅ Download method 3 successful" -ForegroundColor Green
            } catch {
                Write-Host "⚠️  Download method 3 failed: $($_.Exception.Message)" -ForegroundColor Yellow
            }
        }

        if (-not $downloadSuccess) { throw "All download methods failed." }

        # ---- Extract ----
        Write-Host "📦 Extracting files..." -ForegroundColor Green
        try {
            Expand-Archive -Path $TempZip -DestinationPath $TempExtract -Force
        } catch {
            Write-Host "⚠️  Standard extraction failed, trying Shell.Application..." -ForegroundColor Yellow
            $shell = New-Object -ComObject Shell.Application
            $zip   = $shell.NameSpace($TempZip)
            $dest  = $shell.NameSpace($TempExtract)
            $dest.CopyHere($zip.Items(), 4)
        }

        # ---- Locate EXE ----
        Write-Host "🔍 Searching for executable..." -ForegroundColor Yellow
        $allFiles = Get-ChildItem -Path $TempExtract -Recurse -File
        foreach ($f in $allFiles) { Write-Host "  $($f.FullName)" -ForegroundColor Gray }

        $exe = Get-ChildItem -Path $TempExtract -Filter "gym-door-bridge.exe" -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
        if (-not $exe) {
            $exe = $allFiles | Where-Object { $_.Name -eq "gym-door-bridge.exe" } | Select-Object -First 1
            if (-not $exe) {
                $alt = $allFiles | Where-Object { $_.Extension -eq ".exe" } | Select-Object -First 1
                if ($alt) { $exe = $alt }
            }
        }
        if (-not $exe) { throw "No executable found in package." }
        $fullExePath = $exe.FullName
        Write-Host "✅ Found executable: $fullExePath" -ForegroundColor Green

        # ---- If installed, stop & uninstall via app ----
        if ($existingService) {
            Write-Host "🛑 Stopping existing service..." -ForegroundColor Yellow
            try {
                Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
                $sw = [System.Diagnostics.Stopwatch]::StartNew()
                do {
                    Start-Sleep 1
                    $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
                } while ($svc.Status -eq "Running" -and $sw.Elapsed.TotalSeconds -lt 30)
                $sw.Stop()
                if ($svc.Status -eq "Stopped") {
                    Write-Host "✅ Service stopped successfully" -ForegroundColor Green
                } else {
                    Write-Host "⚠️  Forcing process termination..." -ForegroundColor Yellow
                    Get-Process -Name "gym-door-bridge*" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
                }

                $existingExe = Join-Path $InstallPath "gym-door-bridge.exe"
                if (Test-Path $existingExe) {
                    Start-Process -FilePath $existingExe -ArgumentList "uninstall" -Wait -NoNewWindow -ErrorAction SilentlyContinue | Out-Null
                    Start-Sleep 2
                }
            } catch {
                Write-Host "⚠️  Could not fully stop existing service: $($_.Exception.Message)" -ForegroundColor Yellow
            }
        }

        # ---- Copy EXE into InstallPath ----
        $targetExe = Join-Path $InstallPath "gym-door-bridge.exe"
        try {
            if (Test-Path $targetExe) {
                for ($i=0; $i -lt 5; $i++) {
                    try { Remove-Item $targetExe -Force; break } catch { Start-Sleep 2 }
                }
            }
            Copy-Item -Path $fullExePath -Destination $targetExe -Force
            Write-Host "✅ Executable copied to $targetExe" -ForegroundColor Green
        } catch {
            throw "Failed to place executable in $InstallPath: $($_.Exception.Message)"
        }

        # ---- Build absolute-path config.yaml ----
        $tpl = Get-ChildItem -Path $TempExtract -Filter "config.yaml.template" -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
        $configPath = Join-Path $InstallPath "config.yaml"
        if ($tpl) { $cfg = Get-Content $tpl.FullName -Raw } else {
            $cfg = @"
server_url: "$ServerUrl"
tier: "normal"
queue_max_size: 10000
heartbeat_interval: 60
unlock_duration: 3000
device_id: ""
device_key: ""
database_path: "./bridge.db"
log_level: "info"
log_file: ""
enabled_adapters:
  - "simulator"
"@
        }

        # Force absolute paths in config
        $absData = ($DataDir.Replace('\','/'))
        if ($cfg -notmatch '(?m)^\s*server_url:') {
            $cfg = "server_url: `"$ServerUrl`"`r`n" + $cfg
        }
        $cfg = $cfg -replace '(?m)^\s*server_url:\s*".*"$', ('server_url: "' + $ServerUrl + '"')
        $cfg = $cfg -replace '(?m)^\s*database_path:\s*".*"$', ('database_path: "' + $absData + '/bridge.db"')
        if ($cfg -match '(?m)^\s*log_file:') {
            $cfg = $cfg -replace '(?m)^\s*log_file:\s*".*"$', ('log_file: "' + $absData + '/bridge.log"')
        } else {
            $cfg += "`r`nlog_file: `"$absData/bridge.log`"`r`n"
        }

        $cfg | Set-Content -Path $configPath -Encoding UTF8
        Write-Host "✅ Config written: $configPath" -ForegroundColor Green

        # ---- Install service (app-managed first) ----
        Write-Host "⚙️  Installing Gym Door Bridge..." -ForegroundColor Green
        $install = Start-Process -FilePath $targetExe -ArgumentList "install" -Wait -PassThru -NoNewWindow -RedirectStandardOutput "$env:TEMP\install-output.log" -RedirectStandardError "$env:TEMP\install-error.log"
        if ($install.ExitCode -ne 0) {
            $err = (Test-Path "$env:TEMP\install-error.log") ? (Get-Content "$env:TEMP\install-error.log" -Raw) : ""
            Write-Host "❌ App-managed install failed ($($install.ExitCode)): $err" -ForegroundColor Yellow
            # Fallback: create service ourselves
            New-Service -Name $ServiceName `
                        -BinaryPathName "`"$targetExe`" --config `"$configPath`"" `
                        -DisplayName $ServiceDisp `
                        -StartupType Automatic `
                        -Description "Gym Door Access Bridge - integrates RepSet with door controllers" | Out-Null
        } else {
            Write-Host "✅ Service installation completed" -ForegroundColor Green
        }

        # ---- Enforce correct binPath and recovery ----
        $binPath = "`"$targetExe`" --config `"$configPath`""
        & sc.exe config $ServiceName binPath= $binPath | Out-Null
        & sc.exe failure $ServiceName reset= 86400 actions= restart/5000/restart/10000/restart/30000 | Out-Null
        # Optional run-as LocalService (uncomment if needed in locked-down envs)
        # & sc.exe config $ServiceName obj= "NT AUTHORITY\LocalService" | Out-Null

        # ---- Verify service existence ----
        Start-Sleep 2
        $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
        if (-not $svc) { throw "Service not found after installation." }
        Write-Host "✅ Service verification: $($svc.Status)" -ForegroundColor Green

        # ---- Pair after install (uses installed EXE so config paths match) ----
        if ($PairCode) {
            Write-Host "🔗 Pairing device with platform..." -ForegroundColor Green
            # Clean pairing (ignore failures)
            Start-Process -FilePath $targetExe -ArgumentList "unpair" -Wait -NoNewWindow -ErrorAction SilentlyContinue | Out-Null
            Start-Sleep 1
            # Pair
            $pair = Start-Process -FilePath $targetExe -ArgumentList @("pair", $PairCode) -Wait -PassThru -NoNewWindow -RedirectStandardOutput "$env:TEMP\pair-output.log" -RedirectStandardError "$env:TEMP\pair-error.log"
            if ($pair.ExitCode -eq 0) {
                Write-Host "✅ Device paired successfully!" -ForegroundColor Green
                # Sanity check: device_id present
                $cfgNow = Get-Content $configPath -Raw
                if ($cfgNow -match 'device_id:\s*"([^"]+)"' -and $matches[1]) {
                    Write-Host "✅ Device ID: $($matches[1])" -ForegroundColor Green
                } else {
                    Write-Host "⚠️  Pairing completed but device_id not found in config (check logs)" -ForegroundColor Yellow
                }
            } else {
                $perr = (Test-Path "$env:TEMP\pair-error.log") ? (Get-Content "$env:TEMP\pair-error.log" -Raw) : ""
                $pout = (Test-Path "$env:TEMP\pair-output.log") ? (Get-Content "$env:TEMP\pair-output.log" -Raw) : ""
                Write-Host "⚠️  Pairing failed (exit $($pair.ExitCode))" -ForegroundColor Yellow
                if ($perr) { Write-Host "Error: $perr" -ForegroundColor DarkYellow }
                if ($pout) { Write-Host "Output: $pout" -ForegroundColor DarkYellow }
                Write-Host "Manual: gym-door-bridge pair $PairCode" -ForegroundColor Yellow
            }
        }

        # ---- Start + Verify ----
        Write-Host "🔍 Verifying service status..." -ForegroundColor Yellow
        try {
            Start-Service -Name $ServiceName -ErrorAction SilentlyContinue
            Start-Sleep 8
            $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
            Write-Host "Service Status: $($svc.Status)" -ForegroundColor White
            if ($svc.Status -eq "Running") {
                Write-Host "✅ Service is running successfully!" -ForegroundColor Green
                try {
                    Start-Sleep 2
                    $resp = Invoke-WebRequest -Uri "http://localhost:8081/api/v1/health" -UseBasicParsing -TimeoutSec 8
                    Write-Host "✅ API is responding: HTTP $($resp.StatusCode)" -ForegroundColor Green
                } catch {
                    Write-Host "ℹ️  API may take a moment to start" -ForegroundColor Yellow
                }
            } else {
                Write-Host "⚠️  Service installed but not running. Attempting start again..." -ForegroundColor Yellow
                Start-Service -Name $ServiceName -ErrorAction SilentlyContinue
                Start-Sleep 5
                $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
                if ($svc.Status -ne "Running") {
                    Write-Host "❌ Failed to start service (Status: $($svc.Status)). Check logs at $DataDir\bridge.log" -ForegroundColor Red
                }
            }
        } catch {
            Write-Host "❌ Failed to start service: $($_.Exception.Message)" -ForegroundColor Red
            Write-Host "Check logs: $DataDir\bridge.log" -ForegroundColor Yellow
        }

        # ---- Summary ----
        Write-Host "`n📊 Installation Summary:" -ForegroundColor Cyan
        Write-Host "========================" -ForegroundColor Cyan
        Write-Host "Installation Path : $InstallPath" -ForegroundColor White
        Write-Host "Data Path         : $DataDir" -ForegroundColor White
        Write-Host "Service Name      : $ServiceName" -ForegroundColor White
        Write-Host "API Endpoint      : http://localhost:8081" -ForegroundColor White
        Write-Host "Server URL        : $ServerUrl" -ForegroundColor White
        if ($PairCode) { Write-Host "Pair Code Used    : $PairCode" -ForegroundColor White }

        Write-Host "`n📋 Useful Commands:" -ForegroundColor Cyan
        Write-Host "   gym-door-bridge status    - Check bridge status" -ForegroundColor White
        Write-Host "   gym-door-bridge pair CODE - Pair with platform" -ForegroundColor White
        Write-Host "   gym-door-bridge unpair    - Unpair from platform" -ForegroundColor White
        Write-Host "   sc query $ServiceName     - Windows service status" -ForegroundColor White
        Write-Host "   net start $ServiceName    - Start service" -ForegroundColor White
        Write-Host "   net stop $ServiceName     - Stop service" -ForegroundColor White

        Write-Host "`n🎉 Gym Door Bridge installation completed." -ForegroundColor Green
    }
    catch {
        Write-Host "❌ Installation failed: $($_.Exception.Message)" -ForegroundColor Red
        Write-Host "Please check logs at $DataDir\bridge.log and Windows Event Viewer." -ForegroundColor Yellow
        throw
    }
    finally {
        # Cleanup
        foreach ($p in @($TempZip,$TempExtract)) {
            if (Test-Path $p) { Remove-Item $p -Recurse -Force -ErrorAction SilentlyContinue }
        }
    }
}

# If user invoked the file directly, run the function immediately with provided params.
if ($MyInvocation.InvocationName -ne '') {
    Install-GymDoorBridge -PairCode $PairCode -ServerUrl $ServerUrl -InstallPath $InstallPath -Force:$Force
}
