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
}

# ================================================================
# Mock Configuration Functions
# ================================================================

function Get-MockConfiguration {
    <#
    .SYNOPSIS
    Gets a specific mock configuration by name
    #>
    param(
        [Parameter(Mandatory=$true)]
        [ValidateSet('Windows10Pro', 'WindowsServer2019')]
        [string]$ConfigurationName
    )
    
    return $MockConfigurations[$ConfigurationName]
}

function Set-MockSystemEnvironment {
    <#
    .SYNOPSIS
    Applies a mock system environment for testing
    #>
    param(
        [Parameter(Mandatory=$true)]
        [hashtable]$Configuration
    )
    
    # Mock system information cmdlets
    Mock -CommandName "Get-CimInstance" -MockWith {
        param($ClassName)
        
        switch ($ClassName) {
            "Win32_OperatingSystem" {
                return $Configuration.OSInfo
            }
            default {
                return $null
            }
        }
    }
    
    # Mock PowerShell version information
    $global:PSVersionTable = $Configuration.PowerShellInfo
    
    Write-Host "Mock environment applied: $($Configuration.Name)" -ForegroundColor Green
}

function Test-MockConfiguration {
    <#
    .SYNOPSIS
    Tests a mock configuration for completeness
    #>
    param(
        [Parameter(Mandatory=$true)]
        [hashtable]$Configuration
    )
    
    $requiredKeys = @('Name', 'Description', 'OSInfo', 'PowerShellInfo', 'SecurityInfo', 'NetworkInfo', 'ExpectedBehavior')
    
    foreach ($key in $requiredKeys) {
        if (-not $Configuration.ContainsKey($key)) {
            Write-Warning "Mock configuration missing required key: $key"
            return $false
        }
    }
    
    Write-Host "Mock configuration validation passed: $($Configuration.Name)" -ForegroundColor Green
    return $true
}

# Additional mock configuration functions...
# Note: This is a truncated version for the migration. The full file contains comprehensive mock configurations.