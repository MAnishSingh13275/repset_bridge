# RepSet Bridge Installation Troubleshooting Guide

## Overview

This comprehensive guide helps resolve common issues encountered during RepSet Bridge installation. The guide is organized by error categories and provides step-by-step solutions with diagnostic commands.

## Quick Diagnostic Commands

Before diving into specific issues, run these commands to gather system information:

```powershell
# Check PowerShell version and execution policy
$PSVersionTable.PSVersion
Get-ExecutionPolicy -List

# Check administrator privileges
([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")

# Check .NET Framework version
Get-ChildItem "HKLM:SOFTWARE\Microsoft\NET Framework Setup\NDP\v4\Full\" | Get-ItemPropertyValue -Name Release | ForEach-Object { $_ -ge 461808 }

# Test network connectivity
Test-NetConnection -ComputerName github.com -Port 443
Test-NetConnection -ComputerName api.repset.com -Port 443
```

## Common Installation Issues

### 1. PowerShell Execution Policy Restrictions

**Error Messages:**
- "Execution of scripts is disabled on this system"
- "cannot be loaded because running scripts is disabled"
- "UnauthorizedAccess" errors

**Symptoms:**
- Installation script fails to run
- PowerShell blocks script execution
- Security warnings prevent execution

**Solutions:**

#### Option A: Temporary Bypass (Recommended)
```powershell
# Run installation with bypassed execution policy
Set-ExecutionPolicy -ExecutionPolicy Bypass -Scope Process -Force
# Then run your installation command
```

#### Option B: Set RemoteSigned Policy
```powershell
# Allow local scripts and signed remote scripts
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser -Force
```

#### Option C: Run with Bypass Parameter
```powershell
# Run PowerShell with bypass flag
powershell.exe -ExecutionPolicy Bypass -File "Install-RepSetBridge.ps1"
```

**Verification:**
```powershell
Get-ExecutionPolicy -List
```

### 2. Administrator Privileges Required

**Error Messages:**
- "Access to the path is denied"
- "Requested operation requires elevation"
- "Service installation failed: Access denied"

**Symptoms:**
- Cannot create directories in Program Files
- Service installation fails
- Registry access denied

**Solutions:**

#### Check Current Privileges
```powershell
# Verify administrator status
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")
Write-Host "Running as Administrator: $isAdmin"
```

#### Run PowerShell as Administrator
1. Press `Win + X`
2. Select "Windows PowerShell (Admin)" or "Terminal (Admin)"
3. Click "Yes" on UAC prompt
4. Re-run installation command

#### Alternative: Use RunAs
```cmd
runas /user:Administrator powershell.exe
```

### 3. Network Connectivity Issues

**Error Messages:**
- "Unable to connect to the remote server"
- "The request timed out"
- "SSL/TLS secure channel could not be established"
- "Name resolution failure"

**Symptoms:**
- Download failures
- Connection timeouts
- DNS resolution errors
- Certificate validation errors

**Solutions:**

#### Test Basic Connectivity
```powershell
# Test GitHub connectivity
Test-NetConnection -ComputerName github.com -Port 443

# Test platform connectivity
Test-NetConnection -ComputerName api.repset.com -Port 443

# Test DNS resolution
Resolve-DnsName github.com
Resolve-DnsName api.repset.com
```

#### Configure Proxy Settings
```powershell
# Set proxy credentials
[System.Net.WebRequest]::DefaultWebProxy.Credentials = [System.Net.CredentialCache]::DefaultCredentials

# Or set specific proxy
$proxy = New-Object System.Net.WebProxy("http://proxy.company.com:8080")
$proxy.Credentials = [System.Net.CredentialCache]::DefaultCredentials
[System.Net.WebRequest]::DefaultWebProxy = $proxy
```

#### Bypass SSL Certificate Validation (Temporary)
```powershell
# WARNING: Only use for testing
[System.Net.ServicePointManager]::ServerCertificateValidationCallback = {$true}
```

#### Windows Firewall Configuration
```powershell
# Check firewall status
Get-NetFirewallProfile | Select-Object Name, Enabled

# Add firewall rule for bridge
New-NetFirewallRule -DisplayName "RepSet Bridge" -Direction Outbound -Protocol TCP -RemotePort 443 -Action Allow
```

### 4. .NET Framework Issues

**Error Messages:**
- "This application requires .NET Framework"
- "Could not load file or assembly"
- "The application failed to initialize properly"

**Symptoms:**
- Bridge executable won't start
- Runtime dependency errors
- Assembly loading failures

**Solutions:**

#### Check .NET Framework Version
```powershell
# Check installed .NET Framework versions
Get-ChildItem "HKLM:SOFTWARE\Microsoft\NET Framework Setup\NDP" -Recurse |
Get-ItemProperty -Name Version, Release -ErrorAction SilentlyContinue |
Where-Object { $_.PSChildName -match '^(?!S)\p{L}' } |
Select-Object PSChildName, Version, Release
```

#### Install Required .NET Framework
```powershell
# Download and install .NET Framework 4.7.2
$url = "https://download.microsoft.com/download/6/E/4/6E48E8AB-DC00-419E-9704-06DD46E5F81D/NDP472-KB4054530-x86-x64-AllOS-ENU.exe"
$output = "$env:TEMP\NDP472-KB4054530-x86-x64-AllOS-ENU.exe"
Invoke-WebRequest -Uri $url -OutFile $output
Start-Process -FilePath $output -ArgumentList "/quiet" -Wait
```

#### Verify Installation
```powershell
# Check if .NET 4.7.2 or later is installed
$release = Get-ItemProperty "HKLM:SOFTWARE\Microsoft\NET Framework Setup\NDP\v4\Full\" -Name Release -ErrorAction SilentlyContinue
if ($release.Release -ge 461808) {
    Write-Host ".NET Framework 4.7.2 or later is installed"
} else {
    Write-Host ".NET Framework 4.7.2 or later is NOT installed"
}
```

### 5. Windows Service Installation Issues

**Error Messages:**
- "Service installation failed"
- "The specified service already exists"
- "Service failed to start"
- "Service marked for deletion"

**Symptoms:**
- Service won't install
- Service installs but won't start
- Service crashes immediately
- Service shows as "marked for deletion"

**Solutions:**

#### Check Existing Service
```powershell
# Check if service exists
Get-Service -Name "RepSetBridge" -ErrorAction SilentlyContinue

# Check service configuration
Get-WmiObject -Class Win32_Service -Filter "Name='RepSetBridge'"
```

#### Remove Existing Service
```powershell
# Stop service if running
Stop-Service -Name "RepSetBridge" -Force -ErrorAction SilentlyContinue

# Remove service
sc.exe delete "RepSetBridge"

# Wait for deletion to complete
Start-Sleep -Seconds 5
```

#### Manual Service Installation
```powershell
# Install service manually
$servicePath = "C:\Program Files\RepSet\Bridge\gym-door-bridge.exe"
$serviceName = "RepSetBridge"
$displayName = "RepSet Bridge Service"
$description = "RepSet Bridge for biometric device integration"

sc.exe create $serviceName binPath= $servicePath start= auto DisplayName= $displayName
sc.exe description $serviceName $description
```

#### Service Startup Issues
```powershell
# Check service startup type
Get-Service -Name "RepSetBridge" | Select-Object Name, Status, StartType

# Set service to automatic startup
Set-Service -Name "RepSetBridge" -StartupType Automatic

# Start service
Start-Service -Name "RepSetBridge"

# Check service status
Get-Service -Name "RepSetBridge"
```

#### Check Event Logs for Service Errors
```powershell
# Check system event log for service errors
Get-EventLog -LogName System -Source "Service Control Manager" -Newest 10 | 
Where-Object { $_.Message -like "*RepSetBridge*" }

# Check application event log
Get-EventLog -LogName Application -Source "RepSetBridge" -Newest 10 -ErrorAction SilentlyContinue
```

### 6. Antivirus Software Interference

**Error Messages:**
- "File was deleted by antivirus"
- "Access denied" immediately after download
- "Executable blocked by security policy"

**Symptoms:**
- Downloaded files disappear
- Installation terminates unexpectedly
- Executable quarantined

**Solutions:**

#### Temporarily Disable Real-time Protection
```powershell
# Windows Defender - disable real-time protection temporarily
Set-MpPreference -DisableRealtimeMonitoring $true

# Re-enable after installation
Set-MpPreference -DisableRealtimeMonitoring $false
```

#### Add Exclusions to Windows Defender
```powershell
# Add installation directory to exclusions
Add-MpPreference -ExclusionPath "C:\Program Files\RepSet"

# Add executable to exclusions
Add-MpPreference -ExclusionProcess "gym-door-bridge.exe"

# Add temporary download location
Add-MpPreference -ExclusionPath "$env:TEMP"
```

#### Check Quarantine
```powershell
# Check Windows Defender quarantine
Get-MpThreatDetection | Where-Object { $_.Resources -like "*gym-door-bridge*" }

# Restore from quarantine if needed
# (Use Windows Security app for GUI restoration)
```

### 7. File Download and Integrity Issues

**Error Messages:**
- "Download failed"
- "File integrity check failed"
- "Corrupted download"
- "Hash mismatch"

**Symptoms:**
- Download interruptions
- Corrupted executable files
- Hash verification failures

**Solutions:**

#### Manual Download with Retry
```powershell
function Download-WithRetry {
    param(
        [string]$Url,
        [string]$OutputPath,
        [int]$MaxRetries = 3
    )
    
    for ($i = 1; $i -le $MaxRetries; $i++) {
        try {
            Write-Host "Download attempt $i of $MaxRetries"
            Invoke-WebRequest -Uri $Url -OutFile $OutputPath -UseBasicParsing
            Write-Host "Download successful"
            return $true
        }
        catch {
            Write-Host "Download attempt $i failed: $($_.Exception.Message)"
            if ($i -eq $MaxRetries) {
                throw
            }
            Start-Sleep -Seconds (5 * $i)
        }
    }
    return $false
}
```

#### Verify File Integrity
```powershell
# Calculate file hash
$filePath = "C:\path\to\gym-door-bridge.exe"
$hash = Get-FileHash -Path $filePath -Algorithm SHA256
Write-Host "File hash: $($hash.Hash)"

# Compare with expected hash (get from GitHub releases)
$expectedHash = "EXPECTED_SHA256_HASH_HERE"
if ($hash.Hash -eq $expectedHash) {
    Write-Host "File integrity verified"
} else {
    Write-Host "File integrity check failed - re-download required"
}
```

### 8. Configuration File Issues

**Error Messages:**
- "Configuration file not found"
- "Invalid configuration format"
- "Configuration validation failed"

**Symptoms:**
- Service starts but doesn't function
- Connection failures
- Invalid configuration errors

**Solutions:**

#### Validate Configuration File
```powershell
# Check if config file exists
$configPath = "C:\Program Files\RepSet\Bridge\config.yaml"
if (Test-Path $configPath) {
    Write-Host "Configuration file found"
    Get-Content $configPath
} else {
    Write-Host "Configuration file missing"
}
```

#### Create Default Configuration
```powershell
# Create default configuration
$defaultConfig = @"
device_id: "bridge-$(Get-Random)"
device_key: ""
server_url: "https://api.repset.com"
tier: "normal"

service:
  auto_start: true
  restart_on_failure: true
  failure_actions: ["restart", "restart", "none"]
  restart_delay: 60000

installation:
  version: "latest"
  installed_at: "$(Get-Date -Format 'yyyy-MM-ddTHH:mm:ssZ')"
  installed_by: "automated-installer"
"@

$configPath = "C:\Program Files\RepSet\Bridge\config.yaml"
$defaultConfig | Out-File -FilePath $configPath -Encoding UTF8
```

## Advanced Troubleshooting

### System Information Collection

```powershell
# Collect comprehensive system information
$systemInfo = @{
    OSVersion = (Get-WmiObject -Class Win32_OperatingSystem).Caption
    PowerShellVersion = $PSVersionTable.PSVersion.ToString()
    DotNetVersion = (Get-ItemProperty "HKLM:SOFTWARE\Microsoft\NET Framework Setup\NDP\v4\Full\" -Name Release -ErrorAction SilentlyContinue).Release
    IsAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")
    ExecutionPolicy = Get-ExecutionPolicy
    NetworkConnectivity = @{
        GitHub = (Test-NetConnection -ComputerName github.com -Port 443 -InformationLevel Quiet)
        Platform = (Test-NetConnection -ComputerName api.repset.com -Port 443 -InformationLevel Quiet)
    }
    ServiceStatus = (Get-Service -Name "RepSetBridge" -ErrorAction SilentlyContinue)
    FirewallProfiles = (Get-NetFirewallProfile | Select-Object Name, Enabled)
}

$systemInfo | ConvertTo-Json -Depth 3
```

### Log Analysis

```powershell
# Analyze installation logs
$logPath = "C:\Program Files\RepSet\Bridge\logs\installation.log"
if (Test-Path $logPath) {
    Write-Host "Recent installation log entries:"
    Get-Content $logPath | Select-Object -Last 50
}

# Check Windows Event Logs
Get-EventLog -LogName Application -Source "RepSetBridge" -Newest 10 -ErrorAction SilentlyContinue |
Select-Object TimeGenerated, EntryType, Message
```

### Network Diagnostics

```powershell
# Comprehensive network diagnostics
function Test-NetworkConnectivity {
    $results = @{}
    
    # Test DNS resolution
    try {
        $results.DNSResolution = @{
            GitHub = (Resolve-DnsName github.com -ErrorAction Stop).IPAddress
            Platform = (Resolve-DnsName api.repset.com -ErrorAction Stop).IPAddress
        }
    }
    catch {
        $results.DNSResolution = "Failed: $($_.Exception.Message)"
    }
    
    # Test HTTP connectivity
    try {
        $results.HTTPConnectivity = @{
            GitHub = (Invoke-WebRequest -Uri "https://github.com" -Method Head -TimeoutSec 10 -ErrorAction Stop).StatusCode
            Platform = (Invoke-WebRequest -Uri "https://api.repset.com/health" -Method Head -TimeoutSec 10 -ErrorAction Stop).StatusCode
        }
    }
    catch {
        $results.HTTPConnectivity = "Failed: $($_.Exception.Message)"
    }
    
    # Test proxy settings
    $proxy = [System.Net.WebRequest]::DefaultWebProxy
    $results.ProxySettings = @{
        ProxyEnabled = $proxy -ne $null
        ProxyAddress = if ($proxy) { $proxy.Address } else { "None" }
    }
    
    return $results
}

Test-NetworkConnectivity | ConvertTo-Json -Depth 3
```

## Recovery Procedures

### Complete Installation Reset

```powershell
# Stop and remove service
Stop-Service -Name "RepSetBridge" -Force -ErrorAction SilentlyContinue
sc.exe delete "RepSetBridge"

# Remove installation directory
Remove-Item -Path "C:\Program Files\RepSet" -Recurse -Force -ErrorAction SilentlyContinue

# Clean registry entries
Remove-Item -Path "HKLM:\SOFTWARE\RepSet" -Recurse -Force -ErrorAction SilentlyContinue

# Remove Windows Defender exclusions
Remove-MpPreference -ExclusionPath "C:\Program Files\RepSet" -ErrorAction SilentlyContinue
Remove-MpPreference -ExclusionProcess "gym-door-bridge.exe" -ErrorAction SilentlyContinue

Write-Host "Installation reset complete. You can now retry installation."
```

### Backup and Restore Configuration

```powershell
# Backup configuration
$configPath = "C:\Program Files\RepSet\Bridge\config.yaml"
$backupPath = "C:\Program Files\RepSet\Bridge\config.yaml.backup"
if (Test-Path $configPath) {
    Copy-Item $configPath $backupPath
    Write-Host "Configuration backed up to $backupPath"
}

# Restore configuration
if (Test-Path $backupPath) {
    Copy-Item $backupPath $configPath
    Write-Host "Configuration restored from backup"
}
```

## Getting Help

### Information to Collect Before Contacting Support

1. **System Information:**
   ```powershell
   Get-ComputerInfo | Select-Object WindowsProductName, WindowsVersion, TotalPhysicalMemory
   ```

2. **Error Messages:** Copy exact error messages and stack traces

3. **Log Files:** 
   - Installation logs: `C:\Program Files\RepSet\Bridge\logs\installation.log`
   - Service logs: `C:\Program Files\RepSet\Bridge\logs\bridge.log`
   - Windows Event Logs (Application and System)

4. **Network Configuration:** Results from network diagnostics

5. **Installation Command:** The exact command used for installation

### Support Channels

- **Documentation:** [Bridge Installation Guide](./README.md)
- **GitHub Issues:** [RepSet Bridge Repository](https://github.com/your-org/gym-door-bridge/issues)
- **Email Support:** bridge-support@repset.com
- **Emergency Support:** For critical production issues

### Self-Help Resources

- [System Requirements](./system-requirements.md)
- [FAQ](./faq.md)
- [Common Issues Database](./common-issues.md)
- [Video Tutorials](https://docs.repset.com/bridge/videos)

## Prevention Tips

1. **Always run PowerShell as Administrator**
2. **Temporarily disable antivirus during installation**
3. **Ensure stable internet connection**
4. **Keep Windows and .NET Framework updated**
5. **Test network connectivity before installation**
6. **Backup existing configurations before upgrades**
7. **Review system requirements before installation**

---

*Last updated: $(Get-Date -Format 'yyyy-MM-dd')*
*Version: 1.0*