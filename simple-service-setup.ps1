# Simple RepSet Bridge Service Setup
# No fancy characters, just basic PowerShell

param(
    [string]$InstallPath = ""
)

Write-Host "RepSet Bridge Service Setup" -ForegroundColor Blue
Write-Host "=============================" -ForegroundColor Blue
Write-Host ""

# Check admin rights
$currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
$principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Host "[ERROR] Administrator privileges required" -ForegroundColor Red
    Write-Host "Please run PowerShell as Administrator" -ForegroundColor Yellow
    exit 1
}
Write-Host "[OK] Running as Administrator" -ForegroundColor Green

# Find bridge installation
if ($InstallPath -eq "") {
    $searchPaths = @(
        "$env:ProgramFiles\GymDoorBridge",
        "$env:USERPROFILE\RepSetBridge",
        "$env:ProgramFiles\RepSetBridge"
    )
    
    foreach ($path in $searchPaths) {
        $exe = "$path\gym-door-bridge.exe"
        $config = "$path\config.yaml"
        if ((Test-Path $exe) -and (Test-Path $config)) {
            $InstallPath = $path
            break
        }
    }
}

if ($InstallPath -eq "" -or -not (Test-Path "$InstallPath\gym-door-bridge.exe")) {
    Write-Host "[ERROR] Bridge installation not found" -ForegroundColor Red
    Write-Host "Searched locations:" -ForegroundColor Yellow
    foreach ($path in $searchPaths) {
        Write-Host "  $path" -ForegroundColor Gray
    }
    exit 1
}

$exePath = "$InstallPath\gym-door-bridge.exe"
$configPath = "$InstallPath\config.yaml"

Write-Host "[OK] Found bridge at: $InstallPath" -ForegroundColor Green
Write-Host "[INFO] Executable: $exePath" -ForegroundColor Cyan
Write-Host "[INFO] Config: $configPath" -ForegroundColor Cyan

# Remove existing service
$existing = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
if ($existing) {
    Write-Host "[INFO] Stopping existing service..." -ForegroundColor Cyan
    Stop-Service -Name "GymDoorBridge" -Force -ErrorAction SilentlyContinue
    & sc.exe delete "GymDoorBridge" | Out-Null
    Start-Sleep -Seconds 2
}

# Create service
Write-Host "[INFO] Creating Windows service..." -ForegroundColor Cyan
$servicePath = "`"$exePath`" --config `"$configPath`""

try {
    $result = & sc.exe create "GymDoorBridge" binPath= $servicePath start= auto DisplayName= "RepSet Gym Door Bridge"
    if ($LASTEXITCODE -eq 0) {
        Write-Host "[OK] Service created successfully" -ForegroundColor Green
    } else {
        Write-Host "[ERROR] Service creation failed" -ForegroundColor Red
        Write-Host "Output: $result" -ForegroundColor Gray
        exit 1
    }
} catch {
    Write-Host "[ERROR] Service creation failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Start service
Write-Host "[INFO] Starting service..." -ForegroundColor Cyan
try {
    Start-Service -Name "GymDoorBridge"
    Write-Host "[OK] Service started successfully" -ForegroundColor Green
} catch {
    Write-Host "[WARNING] Service created but failed to start: $($_.Exception.Message)" -ForegroundColor Yellow
    Write-Host "[INFO] You can start it manually from Services.msc" -ForegroundColor Cyan
}

# Verify
$service = Get-Service -Name "GymDoorBridge" -ErrorAction SilentlyContinue
if ($service) {
    Write-Host ""
    Write-Host "Service Status:" -ForegroundColor Blue
    Write-Host "  Name: $($service.Name)" -ForegroundColor Gray
    Write-Host "  Status: $($service.Status)" -ForegroundColor Gray
    Write-Host "  Start Type: $($service.StartType)" -ForegroundColor Gray
    Write-Host ""
    Write-Host "[OK] RepSet Bridge will now start automatically with Windows!" -ForegroundColor Green
} else {
    Write-Host "[ERROR] Service verification failed" -ForegroundColor Red
}

Write-Host ""
Write-Host "Press Enter to exit..."
Read-Host