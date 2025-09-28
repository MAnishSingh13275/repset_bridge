# ================================================================
# RepSet Bridge Installation - Mock System Configurations
# Provides mock configurations for testing various system scenarios
# ================================================================

# ================================================================
# Mock System Configuration Definitions
# ================================================================

$MockConfigurations = @{
    
    # Windows 10 Professional - Standard Configuration
    Windows10Pro = @{
        Name = "Windows 10 Professional"
        Description = "Standard Windows 10 Pro workstation"
        OSInfo = @{
            Caption = "Microsoft Windows 10 Pro"
            Version = "10.0.19041"
            OSArchitecture = "64-bit"
            BuildNumber = "19041"
            ServicePackMajorVersion = 0
        }
        PowerShellInfo = @{
            PSVersion = [Version]"5.1.19041.1682"
            PSEdition = "Desktop"
            CLRVersion = [Version]"4.0.30319.42000"
        }
        SecurityInfo = @{
            IsAdministrator = $true
            ExecutionPolicy = "RemoteSigned"
            UACEnabled = $true
            WindowsDefenderEnabled = $true
            FirewallEnabled = $true
        }
        NetworkInfo = @{
            InternetConnected = $true
            ProxyEnabled = $false
            DNSServers = @("8.8.8.8", "8.8.4.4")
        }
        DotNetVersions = @{
            Framework = @(
                @{ Version = "4.8.04084"; Release = 528040 }
            )
            Core = @()
        }
        ExpectedBehavior = @{
            ShouldInstallSuccessfully = $true
            RequiresPrerequisites = $false
            SecurityRisk = "Low"
        }
    }
    
    # Windows Server 2019 - Enterprise Configuration
    WindowsServer2019 = @{
        Name = "Windows Server 2019 Standard"
        Description = "Enterprise Windows Server 2019"
        OSInfo = @{
            Caption = "Microsoft Windows Server 2019 Standard"
            Version = "10.0.17763"
            OSArchitecture = "64-bit"
            BuildNumber = "17763"
            ServicePackMajorVersion = 0
        }
        PowerShellInfo = @{
            PSVersion = [Version]"5.1.17763.2268"
            PSEdition = "Desktop"
            CLRVersion = [Version]"4.0.30319.42000"
        }
        SecurityInfo = @{
            IsAdministrator = $true
            ExecutionPolicy = "RemoteSigned"
            UACEnabled = $true
            WindowsDefenderEnabled = $true
            FirewallEnabled = $true
        }
        NetworkInfo = @{
            InternetConnected = $true
            ProxyEnabled = $true
            ProxyServer = "proxy.company.com:8080"
            DNSServers = @("10.0.0.1", "10.0.0.2")
        }
        DotNetVersions = @{
            Framework = @(
                @{ Version = "4.7.2"; Release = 461808 },
                @{ Version = "4.8.04084"; Release = 528040 }
            )
            Core = @("Microsoft.NETCore.App 3.1.0", "Microsoft.AspNetCore.App 3.1.0")
        }
        ExpectedBehavior = @{
            ShouldInstallSuccessfully = $true
            RequiresPrerequisites = $false
            SecurityRisk = "Low"
        }
    }
    
    # Windows 11 - Modern Configuration
    Windows11 = @{
        Name = "Windows 11 Professional"
        Description = "Modern Windows 11 Pro workstation"
        OSInfo = @{
            Caption = "Microsoft Windows 11 Pro"
            Version = "10.0.22000"
            OSArchitecture = "64-bit"
            BuildNumber = "22000"
            ServicePackMajorVersion = 0
        }
        PowerShellInfo = @{
            PSVersion = [Version]"5.1.22000.282"
            PSEdition = "Desktop"
            CLRVersion = [Version]"4.0.30319.42000"
        }
        SecurityInfo = @{
            IsAdministrator = $true
            ExecutionPolicy = "RemoteSigned"
            UACEnabled = $true
            WindowsDefenderEnabled = $true
            FirewallEnabled = $true
        }
        NetworkInfo = @{
            InternetConnected = $true
            ProxyEnabled = $false
            DNSServers = @("1.1.1.1", "1.0.0.1")
        }
        DotNetVersions = @{
            Framework = @(
                @{ Version = "4.8.04084"; Release = 528040 }
            )
            Core = @("Microsoft.NETCore.App 6.0.0", "Microsoft.AspNetCore.App 6.0.0")
        }
        ExpectedBehavior = @{
            ShouldInstallSuccessfully = $true
            RequiresPrerequisites = $false
            SecurityRisk = "Low"
        }
    }
    
    # PowerShell Core 7.x Configuration
    PowerShellCore7 = @{
        Name = "PowerShell Core 7.2"
        Description = "System running PowerShell Core 7.x"
        OSInfo = @{
            Caption = "Microsoft Windows 10 Pro"
            Version = "10.0.19041"
            OSArchitecture = "64-bit"
            BuildNumber = "19041"
            ServicePackMajorVersion = 0
        }
        PowerShellInfo = @{
            PSVersion = [Version]"7.2.1"
            PSEdition = "Core"
            CLRVersion = [Version]"6.0.1"
        }
        SecurityInfo = @{
            IsAdministrator = $true
            ExecutionPolicy = "RemoteSigned"
            UACEnabled = $true
            WindowsDefenderEnabled = $true
            FirewallEnabled = $true
        }
        NetworkInfo = @{
            InternetConnected = $true
            ProxyEnabled = $false
            DNSServers = @("8.8.8.8", "8.8.4.4")
        }
        DotNetVersions = @{
            Framework = @(
                @{ Version = "4.8.04084"; Release = 528040 }
            )
            Core = @("Microsoft.NETCore.App 6.0.1", "Microsoft.AspNetCore.App 6.0.1")
        }
        ExpectedBehavior = @{
            ShouldInstallSuccessfully = $true
            RequiresPrerequisites = $false
            SecurityRisk = "Low"
        }
    }
    
    # Restricted Security Environment
    RestrictedSecurity = @{
        Name = "Restricted Security Environment"
        Description = "High-security environment with restrictions"
        OSInfo = @{
            Caption = "Microsoft Windows 10 Enterprise"
            Version = "10.0.19041"
            OSArchitecture = "64-bit"
            BuildNumber = "19041"
            ServicePackMajorVersion = 0
        }
        PowerShellInfo = @{
            PSVersion = [Version]"5.1.19041.1682"
            PSEdition = "Desktop"
            CLRVersion = [Version]"4.0.30319.42000"
        }
        SecurityInfo = @{
            IsAdministrator = $false  # Not running as admin
            ExecutionPolicy = "Restricted"  # Restricted execution policy
            UACEnabled = $true
            WindowsDefenderEnabled = $true
            FirewallEnabled = $true
        }
        NetworkInfo = @{
            InternetConnected = $true
            ProxyEnabled = $true
            ProxyServer = "secure-proxy.company.com:8080"
            DNSServers = @("10.0.0.1", "10.0.0.2")
        }
        DotNetVersions = @{
            Framework = @(
                @{ Version = "4.8.04084"; Release = 528040 }
            )
            Core = @()
        }
        ExpectedBehavior = @{
            ShouldInstallSuccessfully = $false
            RequiresPrerequisites = $true
            SecurityRisk = "High"
            ExpectedErrors = @(
                "Insufficient privileges",
                "Execution policy restriction",
                "Administrator rights required"
            )
        }
    }
    
    # Network Restricted Environment
    NetworkRestricted = @{
        Name = "Network Restricted Environment"
        Description = "Environment with limited network access"
        OSInfo = @{
            Caption = "Microsoft Windows 10 Pro"
            Version = "10.0.19041"
            OSArchitecture = "64-bit"
            BuildNumber = "19041"
            ServicePackMajorVersion = 0
        }
        PowerShellInfo = @{
            PSVersion = [Version]"5.1.19041.1682"
            PSEdition = "Desktop"
            CLRVersion = [Version]"4.0.30319.42000"
        }
        SecurityInfo = @{
            IsAdministrator = $true
            ExecutionPolicy = "RemoteSigned"
            UACEnabled = $true
            WindowsDefenderEnabled = $true
            FirewallEnabled = $true
        }
        NetworkInfo = @{
            InternetConnected = $false  # No internet access
            ProxyEnabled = $false
            DNSServers = @("192.168.1.1")
        }
        DotNetVersions = @{
            Framework = @(
                @{ Version = "4.8.04084"; Release = 528040 }
            )
            Core = @()
        }
        ExpectedBehavior = @{
            ShouldInstallSuccessfully = $false
            RequiresPrerequisites = $false
            SecurityRisk = "Medium"
            ExpectedErrors = @(
                "Unable to download bridge executable",
                "Network connectivity issues",
                "GitHub API access failed"
            )
        }
    }
    
    # Legacy System Configuration
    LegacySystem = @{
        Name = "Legacy Windows System"
        Description = "Older Windows system with legacy components"
        OSInfo = @{
            Caption = "Microsoft Windows 10 Home"
            Version = "10.0.17134"  # Older build
            OSArchitecture = "64-bit"
            BuildNumber = "17134"
            ServicePackMajorVersion = 0
        }
        PowerShellInfo = @{
            PSVersion = [Version]"5.1.17134.1"  # Older PowerShell
            PSEdition = "Desktop"
            CLRVersion = [Version]"4.0.30319.42000"
        }
        SecurityInfo = @{
            IsAdministrator = $true
            ExecutionPolicy = "Unrestricted"
            UACEnabled = $false  # UAC disabled
            WindowsDefenderEnabled = $false  # Defender disabled
            FirewallEnabled = $false  # Firewall disabled
        }
        NetworkInfo = @{
            InternetConnected = $true
            ProxyEnabled = $false
            DNSServers = @("8.8.8.8")
        }
        DotNetVersions = @{
            Framework = @(
                @{ Version = "4.7.1"; Release = 461308 }  # Older .NET Framework
            )
            Core = @()
        }
        ExpectedBehavior = @{
            ShouldInstallSuccessfully = $true
            RequiresPrerequisites = $true
            SecurityRisk = "High"
            ExpectedWarnings = @(
                "Outdated PowerShell version",
                "Security features disabled",
                "Older .NET Framework version"
            )
        }
    }
    
    # Corporate Proxy Environment
    CorporateProxy = @{
        Name = "Corporate Proxy Environment"
        Description = "Corporate environment with authenticated proxy"
        OSInfo = @{
            Caption = "Microsoft Windows 10 Enterprise"
            Version = "10.0.19041"
            OSArchitecture = "64-bit"
            BuildNumber = "19041"
            ServicePackMajorVersion = 0
        }
        PowerShellInfo = @{
            PSVersion = [Version]"5.1.19041.1682"
            PSEdition = "Desktop"
            CLRVersion = [Version]"4.0.30319.42000"
        }
        SecurityInfo = @{
            IsAdministrator = $true
            ExecutionPolicy = "RemoteSigned"
            UACEnabled = $true
            WindowsDefenderEnabled = $true
            FirewallEnabled = $true
        }
        NetworkInfo = @{
            InternetConnected = $true
            ProxyEnabled = $true
            ProxyServer = "proxy.corporate.com:8080"
            ProxyAuthRequired = $true
            DNSServers = @("10.1.1.1", "10.1.1.2")
        }
        DotNetVersions = @{
            Framework = @(
                @{ Version = "4.8.04084"; Release = 528040 }
            )
            Core = @()
        }
        ExpectedBehavior = @{
            ShouldInstallSuccessfully = $true
            RequiresPrerequisites = $false
            SecurityRisk = "Medium"
            ExpectedWarnings = @(
                "Proxy authentication may be required",
                "Corporate firewall may block downloads"
            )
        }
    }
    
    # Minimal System Configuration
    MinimalSystem = @{
        Name = "Minimal System Configuration"
        Description = "System with minimal components installed"
        OSInfo = @{
            Caption = "Microsoft Windows 10 Home"
            Version = "10.0.19041"
            OSArchitecture = "64-bit"
            BuildNumber = "19041"
            ServicePackMajorVersion = 0
        }
        PowerShellInfo = @{
            PSVersion = [Version]"5.1.19041.1682"
            PSEdition = "Desktop"
            CLRVersion = [Version]"4.0.30319.42000"
        }
        SecurityInfo = @{
            IsAdministrator = $true
            ExecutionPolicy = "RemoteSigned"
            UACEnabled = $true
            WindowsDefenderEnabled = $true
            FirewallEnabled = $true
        }
        NetworkInfo = @{
            InternetConnected = $true
            ProxyEnabled = $false
            DNSServers = @("8.8.8.8")
        }
        DotNetVersions = @{
            Framework = @(
                @{ Version = "4.6.1"; Release = 394254 }  # Minimal .NET Framework
            )
            Core = @()
        }
        ExpectedBehavior = @{
            ShouldInstallSuccessfully = $true
            RequiresPrerequisites = $true
            SecurityRisk = "Medium"
            ExpectedWarnings = @(
                "Older .NET Framework version",
                "May require .NET Framework update"
            )
        }
    }
}

# ================================================================
# Mock Configuration Helper Functions
# ================================================================

function Get-MockConfiguration {
    <#
    .SYNOPSIS
    Retrieves a mock configuration by name
    #>
    param(
        [Parameter(Mandatory=$true)]
        [ValidateSet('Windows10Pro', 'WindowsServer2019', 'Windows11', 'PowerShellCore7', 
                     'RestrictedSecurity', 'NetworkRestricted', 'LegacySystem', 
                     'CorporateProxy', 'MinimalSystem')]
        [string]$ConfigurationName
    )
    
    return $MockConfigurations[$ConfigurationName]
}

function Get-AllMockConfigurations {
    <#
    .SYNOPSIS
    Returns all available mock configurations
    #>
    return $MockConfigurations
}

function Set-MockSystemEnvironment {
    <#
    .SYNOPSIS
    Applies a mock configuration to the current test environment
    #>
    param(
        [Parameter(Mandatory=$true)]
        [hashtable]$Configuration
    )
    
    Write-Host "Applying mock configuration: $($Configuration.Name)" -ForegroundColor Cyan
    Write-Host "Description: $($Configuration.Description)" -ForegroundColor Gray
    
    # Mock OS Information
    Mock -CommandName "Get-CimInstance" -ParameterFilter { $ClassName -eq "Win32_OperatingSystem" } -MockWith {
        return New-Object PSObject -Property $Configuration.OSInfo
    }
    
    # Mock PowerShell Version
    $global:PSVersionTable = @{
        PSVersion = $Configuration.PowerShellInfo.PSVersion
        PSEdition = $Configuration.PowerShellInfo.PSEdition
        CLRVersion = $Configuration.PowerShellInfo.CLRVersion
    }
    
    # Mock Security Information
    Mock -CommandName "New-Object" -ParameterFilter { $TypeName -eq "Security.Principal.WindowsPrincipal" } -MockWith {
        $principal = New-Object PSObject
        $principal | Add-Member -MemberType ScriptMethod -Name "IsInRole" -Value { 
            return $Configuration.SecurityInfo.IsAdministrator 
        }
        return $principal
    }
    
    Mock -CommandName "Get-ExecutionPolicy" -MockWith {
        return $Configuration.SecurityInfo.ExecutionPolicy
    }
    
    # Mock Network Information
    Mock -CommandName "Test-NetConnection" -MockWith {
        return @{ TcpTestSucceeded = $Configuration.NetworkInfo.InternetConnected }
    }
    
    if ($Configuration.NetworkInfo.ProxyEnabled) {
        Mock -CommandName "Get-ItemProperty" -ParameterFilter { $Path -like "*Internet Settings*" } -MockWith {
            return @{
                ProxyEnable = 1
                ProxyServer = $Configuration.NetworkInfo.ProxyServer
            }
        }
    }
    
    # Mock .NET Framework Detection
    Mock -CommandName "Get-ChildItem" -ParameterFilter { $Path -like "*NET Framework Setup*" } -MockWith {
        $frameworkVersions = @()
        foreach ($version in $Configuration.DotNetVersions.Framework) {
            $versionObj = New-Object PSObject
            $versionObj | Add-Member -MemberType NoteProperty -Name Version -Value $version.Version
            $versionObj | Add-Member -MemberType NoteProperty -Name Release -Value $version.Release
            $frameworkVersions += $versionObj
        }
        return $frameworkVersions
    }
    
    # Mock .NET Core Detection
    if ($Configuration.DotNetVersions.Core.Count -gt 0) {
        Mock -CommandName "Get-Command" -ParameterFilter { $Name -eq "dotnet" } -MockWith {
            return @{ Name = "dotnet"; Source = "C:\Program Files\dotnet\dotnet.exe" }
        }
        
        Mock -CommandName "Invoke-Expression" -ParameterFilter { $Command -like "*dotnet --list-runtimes*" } -MockWith {
            return $Configuration.DotNetVersions.Core
        }
    }
    
    Write-Host "✓ Mock configuration applied successfully" -ForegroundColor Green
}

function Test-MockConfiguration {
    <#
    .SYNOPSIS
    Tests a mock configuration to ensure it behaves as expected
    #>
    param(
        [Parameter(Mandatory=$true)]
        [hashtable]$Configuration,
        
        [Parameter(Mandatory=$false)]
        [switch]$Verbose
    )
    
    Write-Host "Testing mock configuration: $($Configuration.Name)" -ForegroundColor Yellow
    
    $testResults = @{
        ConfigurationName = $Configuration.Name
        TestsPassed = 0
        TestsFailed = 0
        Errors = @()
    }
    
    try {
        # Apply the mock configuration
        Set-MockSystemEnvironment -Configuration $Configuration
        
        # Test OS Information
        $osInfo = Get-CimInstance -ClassName Win32_OperatingSystem -ErrorAction SilentlyContinue
        if ($osInfo -and $osInfo.Caption -eq $Configuration.OSInfo.Caption) {
            $testResults.TestsPassed++
            if ($Verbose) { Write-Host "✓ OS Information mock working" -ForegroundColor Green }
        } else {
            $testResults.TestsFailed++
            $testResults.Errors += "OS Information mock failed"
        }
        
        # Test PowerShell Version
        if ($PSVersionTable.PSVersion -eq $Configuration.PowerShellInfo.PSVersion) {
            $testResults.TestsPassed++
            if ($Verbose) { Write-Host "✓ PowerShell Version mock working" -ForegroundColor Green }
        } else {
            $testResults.TestsFailed++
            $testResults.Errors += "PowerShell Version mock failed"
        }
        
        # Test Administrator Privileges
        $currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
        $isAdmin = $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
        if ($isAdmin -eq $Configuration.SecurityInfo.IsAdministrator) {
            $testResults.TestsPassed++
            if ($Verbose) { Write-Host "✓ Administrator Privileges mock working" -ForegroundColor Green }
        } else {
            $testResults.TestsFailed++
            $testResults.Errors += "Administrator Privileges mock failed"
        }
        
        # Test Network Connectivity
        $networkTest = Test-NetConnection -ComputerName "8.8.8.8" -Port 53 -InformationLevel Quiet -ErrorAction SilentlyContinue
        if ($networkTest -eq $Configuration.NetworkInfo.InternetConnected) {
            $testResults.TestsPassed++
            if ($Verbose) { Write-Host "✓ Network Connectivity mock working" -ForegroundColor Green }
        } else {
            $testResults.TestsFailed++
            $testResults.Errors += "Network Connectivity mock failed"
        }
        
    }
    catch {
        $testResults.TestsFailed++
        $testResults.Errors += "Exception during mock testing: $($_.Exception.Message)"
    }
    
    # Display results
    $totalTests = $testResults.TestsPassed + $testResults.TestsFailed
    $passRate = if ($totalTests -gt 0) { [math]::Round(($testResults.TestsPassed / $totalTests) * 100, 2) } else { 0 }
    
    Write-Host "Mock Configuration Test Results:" -ForegroundColor Yellow
    Write-Host "  Total Tests: $totalTests" -ForegroundColor White
    Write-Host "  Passed: $($testResults.TestsPassed)" -ForegroundColor Green
    Write-Host "  Failed: $($testResults.TestsFailed)" -ForegroundColor Red
    Write-Host "  Pass Rate: $passRate%" -ForegroundColor White
    
    if ($testResults.Errors.Count -gt 0) {
        Write-Host "  Errors:" -ForegroundColor Red
        foreach ($error in $testResults.Errors) {
            Write-Host "    - $error" -ForegroundColor DarkRed
        }
    }
    
    return $testResults
}

function New-MockConfigurationReport {
    <#
    .SYNOPSIS
    Generates a comprehensive report of all mock configurations
    #>
    param(
        [Parameter(Mandatory=$false)]
        [string]$OutputPath = "$env:TEMP\MockConfigurationReport.html"
    )
    
    Write-Host "Generating mock configuration report..." -ForegroundColor Cyan
    
    $htmlContent = @"
<!DOCTYPE html>
<html>
<head>
    <title>RepSet Bridge - Mock System Configurations</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background-color: #f5f5f5; }
        .container { background-color: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #2c3e50; border-bottom: 3px solid #3498db; padding-bottom: 10px; }
        h2 { color: #34495e; margin-top: 30px; }
        .config-card { border: 1px solid #ddd; border-radius: 8px; padding: 15px; margin: 15px 0; background-color: #f9f9f9; }
        .config-name { font-size: 1.2em; font-weight: bold; color: #2c3e50; }
        .config-description { color: #7f8c8d; margin: 5px 0; }
        .config-details { margin-top: 10px; }
        .detail-section { margin: 10px 0; }
        .detail-title { font-weight: bold; color: #34495e; }
        .risk-low { color: #27ae60; font-weight: bold; }
        .risk-medium { color: #f39c12; font-weight: bold; }
        .risk-high { color: #e74c3c; font-weight: bold; }
        .expected-success { color: #27ae60; }
        .expected-failure { color: #e74c3c; }
        table { border-collapse: collapse; width: 100%; margin: 10px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #3498db; color: white; }
        .code { background-color: #f8f9fa; padding: 2px 4px; border-radius: 3px; font-family: monospace; }
    </style>
</head>
<body>
    <div class="container">
        <h1>RepSet Bridge - Mock System Configurations</h1>
        <p>This document describes the various mock system configurations available for testing the RepSet Bridge installation script.</p>
        <p><strong>Generated:</strong> $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')</p>
        
        <h2>Configuration Overview</h2>
        <table>
            <tr>
                <th>Configuration</th>
                <th>OS</th>
                <th>PowerShell</th>
                <th>Security Risk</th>
                <th>Expected Result</th>
            </tr>
"@

    foreach ($configName in $MockConfigurations.Keys) {
        $config = $MockConfigurations[$configName]
        $riskClass = "risk-$($config.ExpectedBehavior.SecurityRisk.ToLower())"
        $resultClass = if ($config.ExpectedBehavior.ShouldInstallSuccessfully) { "expected-success" } else { "expected-failure" }
        $resultText = if ($config.ExpectedBehavior.ShouldInstallSuccessfully) { "Success" } else { "Failure" }
        
        $htmlContent += @"
            <tr>
                <td>$($config.Name)</td>
                <td>$($config.OSInfo.Caption)</td>
                <td>$($config.PowerShellInfo.PSVersion) ($($config.PowerShellInfo.PSEdition))</td>
                <td class="$riskClass">$($config.ExpectedBehavior.SecurityRisk)</td>
                <td class="$resultClass">$resultText</td>
            </tr>
"@
    }
    
    $htmlContent += @"
        </table>
        
        <h2>Detailed Configurations</h2>
"@

    foreach ($configName in $MockConfigurations.Keys) {
        $config = $MockConfigurations[$configName]
        $riskClass = "risk-$($config.ExpectedBehavior.SecurityRisk.ToLower())"
        
        $htmlContent += @"
        <div class="config-card">
            <div class="config-name">$($config.Name)</div>
            <div class="config-description">$($config.Description)</div>
            
            <div class="config-details">
                <div class="detail-section">
                    <div class="detail-title">Operating System</div>
                    <ul>
                        <li><strong>Caption:</strong> $($config.OSInfo.Caption)</li>
                        <li><strong>Version:</strong> $($config.OSInfo.Version)</li>
                        <li><strong>Architecture:</strong> $($config.OSInfo.OSArchitecture)</li>
                        <li><strong>Build:</strong> $($config.OSInfo.BuildNumber)</li>
                    </ul>
                </div>
                
                <div class="detail-section">
                    <div class="detail-title">PowerShell Environment</div>
                    <ul>
                        <li><strong>Version:</strong> $($config.PowerShellInfo.PSVersion)</li>
                        <li><strong>Edition:</strong> $($config.PowerShellInfo.PSEdition)</li>
                        <li><strong>CLR Version:</strong> $($config.PowerShellInfo.CLRVersion)</li>
                    </ul>
                </div>
                
                <div class="detail-section">
                    <div class="detail-title">Security Configuration</div>
                    <ul>
                        <li><strong>Administrator:</strong> $($config.SecurityInfo.IsAdministrator)</li>
                        <li><strong>Execution Policy:</strong> $($config.SecurityInfo.ExecutionPolicy)</li>
                        <li><strong>UAC Enabled:</strong> $($config.SecurityInfo.UACEnabled)</li>
                        <li><strong>Windows Defender:</strong> $($config.SecurityInfo.WindowsDefenderEnabled)</li>
                        <li><strong>Firewall:</strong> $($config.SecurityInfo.FirewallEnabled)</li>
                    </ul>
                </div>
                
                <div class="detail-section">
                    <div class="detail-title">Network Configuration</div>
                    <ul>
                        <li><strong>Internet Connected:</strong> $($config.NetworkInfo.InternetConnected)</li>
                        <li><strong>Proxy Enabled:</strong> $($config.NetworkInfo.ProxyEnabled)</li>
"@
        
        if ($config.NetworkInfo.ProxyEnabled -and $config.NetworkInfo.ProxyServer) {
            $htmlContent += "<li><strong>Proxy Server:</strong> $($config.NetworkInfo.ProxyServer)</li>"
        }
        
        $htmlContent += @"
                        <li><strong>DNS Servers:</strong> $($config.NetworkInfo.DNSServers -join ', ')</li>
                    </ul>
                </div>
                
                <div class="detail-section">
                    <div class="detail-title">.NET Framework Versions</div>
                    <ul>
"@
        
        foreach ($framework in $config.DotNetVersions.Framework) {
            $htmlContent += "<li><strong>Framework:</strong> $($framework.Version) (Release: $($framework.Release))</li>"
        }
        
        foreach ($core in $config.DotNetVersions.Core) {
            $htmlContent += "<li><strong>Core:</strong> $core</li>"
        }
        
        $htmlContent += @"
                    </ul>
                </div>
                
                <div class="detail-section">
                    <div class="detail-title">Expected Behavior</div>
                    <ul>
                        <li><strong>Should Install Successfully:</strong> $($config.ExpectedBehavior.ShouldInstallSuccessfully)</li>
                        <li><strong>Requires Prerequisites:</strong> $($config.ExpectedBehavior.RequiresPrerequisites)</li>
                        <li><strong>Security Risk:</strong> <span class="$riskClass">$($config.ExpectedBehavior.SecurityRisk)</span></li>
"@
        
        if ($config.ExpectedBehavior.ExpectedErrors) {
            $htmlContent += "<li><strong>Expected Errors:</strong><ul>"
            foreach ($error in $config.ExpectedBehavior.ExpectedErrors) {
                $htmlContent += "<li>$error</li>"
            }
            $htmlContent += "</ul></li>"
        }
        
        if ($config.ExpectedBehavior.ExpectedWarnings) {
            $htmlContent += "<li><strong>Expected Warnings:</strong><ul>"
            foreach ($warning in $config.ExpectedBehavior.ExpectedWarnings) {
                $htmlContent += "<li>$warning</li>"
            }
            $htmlContent += "</ul></li>"
        }
        
        $htmlContent += @"
                    </ul>
                </div>
            </div>
        </div>
"@
    }
    
    $htmlContent += @"
        
        <h2>Usage Instructions</h2>
        <p>To use these mock configurations in your tests:</p>
        <ol>
            <li>Import the MockConfigurations.ps1 file: <span class="code">. .\MockConfigurations.ps1</span></li>
            <li>Get a configuration: <span class="code">`$config = Get-MockConfiguration -ConfigurationName 'Windows10Pro'</span></li>
            <li>Apply the configuration: <span class="code">Set-MockSystemEnvironment -Configuration `$config</span></li>
            <li>Run your tests with the mocked environment</li>
        </ol>
        
        <h2>Testing Mock Configurations</h2>
        <p>To test that a mock configuration is working correctly:</p>
        <p><span class="code">Test-MockConfiguration -Configuration `$config -Verbose</span></p>
        
        <footer style="margin-top: 40px; padding-top: 20px; border-top: 1px solid #ddd; color: #7f8c8d;">
            <p>Generated by RepSet Bridge Mock Configuration System</p>
        </footer>
    </div>
</body>
</html>
"@

    Set-Content -Path $OutputPath -Value $htmlContent
    Write-Host "✓ Mock configuration report saved to: $OutputPath" -ForegroundColor Green
    
    return $OutputPath
}

# ================================================================
# Export Functions
# ================================================================

Export-ModuleMember -Function @(
    'Get-MockConfiguration',
    'Get-AllMockConfigurations', 
    'Set-MockSystemEnvironment',
    'Test-MockConfiguration',
    'New-MockConfigurationReport'
)