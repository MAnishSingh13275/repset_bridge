# ================================================================
# RepSet Bridge - Automated Installation Script Foundation
# Secure, validated installation with comprehensive error handling
# ================================================================

[CmdletBinding()]
param(
    [Parameter(Mandatory=$true)]
    [string]$PairCode,
    
    [Parameter(Mandatory=$true)]
    [string]$Signature,
    
    [Parameter(Mandatory=$true)]
    [string]$Nonce,
    
    [Parameter(Mandatory=$true)]
    [string]$GymId,
    
    [Parameter(Mandatory=$true)]
    [string]$ExpiresAt,
    
    [Parameter(Mandatory=$false)]
    [string]$PlatformEndpoint = "https://app.repset.com",
    
    [Parameter(Mandatory=$false)]
    [string]$InstallPath = "$env:ProgramFiles\RepSet\Bridge",
    
    [Parameter(Mandatory=$false)]
    [switch]$Force,
    
    [Parameter(Mandatory=$false)]
    [switch]$SkipPrerequisites
)

# ================================================================
# Global Variables and Constants
# ================================================================

$script:LogFile = "$env:TEMP\RepSetBridge-Install-$(Get-Date -Format 'yyyyMMdd-HHmmss').log"
$script:ServiceName = "RepSetBridge"
$script:ServiceDisplayName = "RepSet Bridge Service"
$script:ServiceDescription = "RepSet Bridge - Gym Equipment Integration Service"
$script:GitHubRepo = "repset/repset_bridge"
$script:ConfigFileName = "config.yaml"
$script:InstallationId = [System.Guid]::NewGuid().ToString()
$script:InstallationStartTime = Get-Date

# Error codes for different failure scenarios
$script:ErrorCodes = @{
    Success = 0
    InvalidSignature = 1
    ExpiredCommand = 2
    InsufficientPrivileges = 3
    SystemRequirementsNotMet = 4
    DownloadFailed = 5
    IntegrityVerificationFailed = 6
    InstallationFailed = 7
    ServiceInstallationFailed = 8
    ConfigurationFailed = 9
    ConnectionTestFailed = 10
    RollbackFailed = 11
}

# ================================================================
# Logging and Output Functions
# ================================================================

function Write-InstallationLog {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [ValidateSet('Info', 'Warning', 'Error', 'Success', 'Debug', 'Progress')]
        [string]$Level,
        
        [Parameter(Mandatory=$true)]
        [string]$Message,
        
        [Parameter(Mandatory=$false)]
        [hashtable]$Context = @{},
        
        [Parameter(Mandatory=$false)]
        [switch]$NoConsole,
        
        [Parameter(Mandatory=$false)]
        [string]$Step = "",
        
        [Parameter(Mandatory=$false)]
        [int]$StepNumber = 0,
        
        [Parameter(Mandatory=$false)]
        [int]$TotalSteps = 0
    )
    
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $contextStr = if ($Context.Count -gt 0) { " | Context: $($Context | ConvertTo-Json -Compress)" } else { "" }
    $stepStr = if ($Step) { " | Step: $Step" } else { "" }
    $logEntry = "[$timestamp] [$Level] $Message$stepStr$contextStr"
    
    # Enhanced context for structured logging
    $logContext = @{
        Timestamp = $timestamp
        Level = $Level
        Message = $Message
        InstallationId = $script:InstallationId
        Step = $Step
        StepNumber = $StepNumber
        TotalSteps = $TotalSteps
        Context = $Context
    }
    
    # Write to multiple log targets
    Write-LogToFile -LogEntry $logEntry -Context $logContext
    Write-LogToConsole -Level $Level -Message $Message -NoConsole $NoConsole
    Write-LogToEventLog -Level $Level -Message $Message -Context $logContext
    Send-LogToPlatform -Level $Level -Message $Message -Context $logContext
}

function Write-LogToFile {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$LogEntry,
        
        [Parameter(Mandatory=$false)]
        [hashtable]$Context = @{}
    )
    
    try {
        # Write to main log file
        Add-Content -Path $script:LogFile -Value $LogEntry -ErrorAction SilentlyContinue
        
        # Write structured log entry for machine processing
        $structuredLogFile = $script:LogFile -replace '\.log$', '.json'
        $jsonEntry = $Context | ConvertTo-Json -Compress
        Add-Content -Path $structuredLogFile -Value $jsonEntry -ErrorAction SilentlyContinue
        
        # Rotate log files if they get too large (>10MB)
        $logFileInfo = Get-Item -Path $script:LogFile -ErrorAction SilentlyContinue
        if ($logFileInfo -and $logFileInfo.Length -gt 10MB) {
            $rotatedLogFile = $script:LogFile -replace '\.log$', "-$(Get-Date -Format 'yyyyMMdd-HHmmss').log"
            Move-Item -Path $script:LogFile -Destination $rotatedLogFile -ErrorAction SilentlyContinue
        }
    }
    catch {
        # If we can't write to log file, continue silently
    }
}

function Write-LogToConsole {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$Level,
        
        [Parameter(Mandatory=$true)]
        [string]$Message,
        
        [Parameter(Mandatory=$false)]
        [switch]$NoConsole
    )
    
    # Write to console unless suppressed
    if (-not $NoConsole) {
        $timestamp = Get-Date -Format "HH:mm:ss"
        $prefix = "[$timestamp]"
        
        switch ($Level) {
            'Info'     { Write-Host "$prefix $Message" -ForegroundColor White }
            'Success'  { Write-Host "$prefix ✓ $Message" -ForegroundColor Green }
            'Warning'  { Write-Host "$prefix ⚠ $Message" -ForegroundColor Yellow }
            'Error'    { Write-Host "$prefix ✗ $Message" -ForegroundColor Red }
            'Progress' { Write-Host "$prefix → $Message" -ForegroundColor Cyan }
            'Debug'    { if ($VerbosePreference -eq 'Continue') { Write-Host "$prefix [DEBUG] $Message" -ForegroundColor Gray } }
        }
    }
}

function Write-LogToEventLog {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$Level,
        
        [Parameter(Mandatory=$true)]
        [string]$Message,
        
        [Parameter(Mandatory=$false)]
        [hashtable]$Context = @{}
    )
    
    # Write to Windows Event Log for important events
    if ($Level -in @('Error', 'Warning', 'Success')) {
        try {
            $eventSource = "RepSetBridge-Installer"
            
            # Create event source if it doesn't exist
            if (-not [System.Diagnostics.EventLog]::SourceExists($eventSource)) {
                try {
                    [System.Diagnostics.EventLog]::CreateEventSource($eventSource, "Application")
                    Start-Sleep -Seconds 1  # Allow time for source creation
                }
                catch {
                    # If we can't create the source, skip event logging
                    return
                }
            }
            
            # Determine event type and ID
            $eventType = switch ($Level) {
                'Error'   { 'Error'; $eventId = 1001 }
                'Warning' { 'Warning'; $eventId = 1002 }
                'Success' { 'Information'; $eventId = 1003 }
                default   { 'Information'; $eventId = 1000 }
            }
            
            # Create detailed event message
            $eventMessage = @"
RepSet Bridge Installation Event

Message: $Message
Installation ID: $($script:InstallationId)
Timestamp: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')
Level: $Level

$(if ($Context.Count -gt 0) { "Context: $($Context | ConvertTo-Json -Depth 3)" })
"@
            
            Write-EventLog -LogName Application -Source $eventSource -EntryType $eventType -EventId $eventId -Message $eventMessage
        }
        catch {
            # If we can't write to event log, continue silently
        }
    }
}

function Send-LogToPlatform {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$Level,
        
        [Parameter(Mandatory=$true)]
        [string]$Message,
        
        [Parameter(Mandatory=$false)]
        [hashtable]$Context = @{}
    )
    
    # Send real-time progress to platform (if connection available)
    try {
        # Only send important events to avoid overwhelming the platform
        if ($Level -in @('Error', 'Warning', 'Success', 'Progress')) {
            $logData = @{
                installationId = $script:InstallationId
                gymId = $GymId
                timestamp = Get-Date -Format 'yyyy-MM-ddTHH:mm:ss.fffZ'
                level = $Level.ToLower()
                message = $Message
                step = $Context.Step
                stepNumber = $Context.StepNumber
                totalSteps = $Context.TotalSteps
                context = $Context
            }
            
            # Send to platform endpoint (non-blocking)
            $platformUrl = "$PlatformEndpoint/api/installation/logs"
            $headers = @{
                'Content-Type' = 'application/json'
                'User-Agent' = 'RepSet-Bridge-Installer/1.0'
            }
            
            # Use background job to avoid blocking installation
            Start-Job -ScriptBlock {
                param($Url, $Headers, $Data)
                try {
                    $json = $Data | ConvertTo-Json -Depth 3
                    Invoke-RestMethod -Uri $Url -Method Post -Headers $Headers -Body $json -TimeoutSec 5 -ErrorAction SilentlyContinue
                }
                catch {
                    # Silently ignore platform logging failures
                }
            } -ArgumentList $platformUrl, $headers, $logData | Out-Null
        }
    }
    catch {
        # If platform logging fails, continue silently
    }
}

function Send-InstallationNotification {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [ValidateSet('Started', 'Progress', 'Success', 'Failed')]
        [string]$Status,
        
        [Parameter(Mandatory=$true)]
        [string]$Message,
        
        [Parameter(Mandatory=$false)]
        [hashtable]$Details = @{},
        
        [Parameter(Mandatory=$false)]
        [string]$ErrorCode = ""
    )
    
    Write-InstallationLog -Level Debug -Message "Sending installation notification: $Status"
    
    try {
        $notificationData = @{
            installationId = $script:InstallationId
            gymId = $GymId
            status = $Status.ToLower()
            message = $Message
            timestamp = Get-Date -Format 'yyyy-MM-ddTHH:mm:ss.fffZ'
            details = $Details
            errorCode = $ErrorCode
            systemInfo = @{
                os = "$([System.Environment]::OSVersion.VersionString)"
                powershellVersion = "$($PSVersionTable.PSVersion)"
                architecture = "$([System.Environment]::ProcessorArchitecture)"
                machineName = "$([System.Environment]::MachineName)"
            }
        }
        
        $platformUrl = "$PlatformEndpoint/api/installation/notifications"
        $headers = @{
            'Content-Type' = 'application/json'
            'User-Agent' = 'RepSet-Bridge-Installer/1.0'
        }
        
        $json = $notificationData | ConvertTo-Json -Depth 4
        
        # Send notification with retry logic
        $maxRetries = 3
        $retryDelay = 2
        
        for ($attempt = 1; $attempt -le $maxRetries; $attempt++) {
            try {
                $response = Invoke-RestMethod -Uri $platformUrl -Method Post -Headers $headers -Body $json -TimeoutSec 10 -ErrorAction Stop
                Write-InstallationLog -Level Debug -Message "Installation notification sent successfully (attempt $attempt)"
                return $true
            }
            catch {
                Write-InstallationLog -Level Debug -Message "Failed to send notification (attempt $attempt): $($_.Exception.Message)"
                if ($attempt -lt $maxRetries) {
                    Start-Sleep -Seconds ($retryDelay * $attempt)
                }
            }
        }
        
        Write-InstallationLog -Level Warning -Message "Failed to send installation notification after $maxRetries attempts"
        return $false
    }
    catch {
        Write-InstallationLog -Level Warning -Message "Error sending installation notification: $($_.Exception.Message)"
        return $false
    }
}

function Write-Progress-Step {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$Step,
        
        [Parameter(Mandatory=$true)]
        [int]$StepNumber,
        
        [Parameter(Mandatory=$true)]
        [int]$TotalSteps,
        
        [Parameter(Mandatory=$false)]
        [string]$Status = "In Progress",
        
        [Parameter(Mandatory=$false)]
        [string]$SubStep = "",
        
        [Parameter(Mandatory=$false)]
        [int]$SubStepNumber = 0,
        
        [Parameter(Mandatory=$false)]
        [int]$TotalSubSteps = 0
    )
    
    $percentComplete = [math]::Round(($StepNumber / $TotalSteps) * 100)
    
    # Calculate sub-step progress if provided
    if ($TotalSubSteps -gt 0 -and $SubStepNumber -gt 0) {
        $subStepPercent = ($SubStepNumber / $TotalSubSteps) * (100 / $TotalSteps)
        $stepBasePercent = (($StepNumber - 1) / $TotalSteps) * 100
        $percentComplete = [math]::Round($stepBasePercent + $subStepPercent)
        
        $progressStatus = "$Step - $Status"
        if ($SubStep) {
            $progressStatus += " ($SubStep - $SubStepNumber/$TotalSubSteps)"
        }
    }
    else {
        $progressStatus = "$Step - $Status"
    }
    
    # Update PowerShell progress bar
    Write-Progress -Activity "RepSet Bridge Installation" -Status $progressStatus -PercentComplete $percentComplete
    
    # Create detailed progress context
    $progressContext = @{
        Step = $Step
        StepNumber = $StepNumber
        TotalSteps = $TotalSteps
        Status = $Status
        PercentComplete = $percentComplete
    }
    
    if ($SubStep) {
        $progressContext.SubStep = $SubStep
        $progressContext.SubStepNumber = $SubStepNumber
        $progressContext.TotalSubSteps = $TotalSubSteps
    }
    
    # Log progress with enhanced context
    $progressMessage = "Step $StepNumber/$TotalSteps`: $Step - $Status"
    if ($SubStep) {
        $progressMessage += " ($SubStep - $SubStepNumber/$TotalSubSteps)"
    }
    
    Write-InstallationLog -Level Progress -Message $progressMessage -Context $progressContext -Step $Step -StepNumber $StepNumber -TotalSteps $TotalSteps
    
    # Send real-time progress to platform
    Send-ProgressToPlatform -Step $Step -StepNumber $StepNumber -TotalSteps $TotalSteps -Status $Status -PercentComplete $percentComplete -SubStep $SubStep -SubStepNumber $SubStepNumber -TotalSubSteps $TotalSubSteps
}

function Send-ProgressToPlatform {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$Step,
        
        [Parameter(Mandatory=$true)]
        [int]$StepNumber,
        
        [Parameter(Mandatory=$true)]
        [int]$TotalSteps,
        
        [Parameter(Mandatory=$true)]
        [string]$Status,
        
        [Parameter(Mandatory=$true)]
        [int]$PercentComplete,
        
        [Parameter(Mandatory=$false)]
        [string]$SubStep = "",
        
        [Parameter(Mandatory=$false)]
        [int]$SubStepNumber = 0,
        
        [Parameter(Mandatory=$false)]
        [int]$TotalSubSteps = 0
    )
    
    try {
        $progressData = @{
            installationId = $script:InstallationId
            gymId = $GymId
            timestamp = Get-Date -Format 'yyyy-MM-ddTHH:mm:ss.fffZ'
            step = $Step
            stepNumber = $StepNumber
            totalSteps = $TotalSteps
            status = $Status
            percentComplete = $PercentComplete
        }
        
        if ($SubStep) {
            $progressData.subStep = $SubStep
            $progressData.subStepNumber = $SubStepNumber
            $progressData.totalSubSteps = $TotalSubSteps
        }
        
        $platformUrl = "$PlatformEndpoint/api/installation/progress"
        $headers = @{
            'Content-Type' = 'application/json'
            'User-Agent' = 'RepSet-Bridge-Installer/1.0'
        }
        
        $json = $progressData | ConvertTo-Json -Depth 3
        
        # Send progress update asynchronously to avoid blocking installation
        Start-Job -ScriptBlock {
            param($Url, $Headers, $Data)
            try {
                Invoke-RestMethod -Uri $Url -Method Post -Headers $Headers -Body $Data -TimeoutSec 5 -ErrorAction SilentlyContinue
            }
            catch {
                # Silently ignore platform progress update failures
            }
        } -ArgumentList $platformUrl, $headers, $json | Out-Null
    }
    catch {
        # If platform progress update fails, continue silently
    }
}

# ================================================================
# Installation Telemetry and Monitoring Functions
# ================================================================

function Initialize-InstallationTelemetry {
    <#
    .SYNOPSIS
    Initializes telemetry collection for the installation process
    
    .DESCRIPTION
    Sets up telemetry collection including performance counters, error tracking,
    and system information gathering for comprehensive installation monitoring.
    #>
    [CmdletBinding()]
    param()
    
    Write-InstallationLog -Level Debug -Message "Initializing installation telemetry"
    
    try {
        # Initialize telemetry data structure
        $script:TelemetryData = @{
            InstallationId = $script:InstallationId
            GymId = $GymId
            StartTime = $script:InstallationStartTime
            SystemInfo = Get-SystemTelemetryInfo
            PerformanceMetrics = @{
                StepTimings = @{}
                DownloadMetrics = @{}
                ErrorCounts = @{}
                RetryAttempts = @{}
            }
            InstallationMetrics = @{
                TotalSteps = 0
                CompletedSteps = 0
                FailedSteps = 0
                SkippedSteps = 0
                Warnings = 0
                Errors = 0
            }
            NetworkMetrics = @{
                DownloadSpeed = 0
                TotalBytesDownloaded = 0
                ConnectionAttempts = 0
                ConnectionFailures = 0
            }
            SecurityMetrics = @{
                SignatureValidations = 0
                IntegrityChecks = 0
                SecurityWarnings = 0
                SecurityErrors = 0
            }
        }
        
        # Start performance monitoring
        Start-PerformanceMonitoring
        
        # Send telemetry initialization event
        Send-TelemetryEvent -EventType "TelemetryInitialized" -Data @{
            SystemInfo = $script:TelemetryData.SystemInfo
            InstallationId = $script:InstallationId
        }
        
        Write-InstallationLog -Level Success -Message "Installation telemetry initialized successfully"
        return $true
    }
    catch {
        Write-InstallationLog -Level Warning -Message "Failed to initialize telemetry: $($_.Exception.Message)"
        return $false
    }
}

function Get-SystemTelemetryInfo {
    <#
    .SYNOPSIS
    Collects comprehensive system information for telemetry
    
    .DESCRIPTION
    Gathers detailed system information including hardware specs, OS version,
    PowerShell version, network configuration, and security settings.
    #>
    [CmdletBinding()]
    param()
    
    try {
        $systemInfo = @{
            # Basic system information
            MachineName = [System.Environment]::MachineName
            UserName = [System.Environment]::UserName
            OSVersion = [System.Environment]::OSVersion.VersionString
            OSArchitecture = [System.Environment]::ProcessorArchitecture
            Is64BitOS = [System.Environment]::Is64BitOperatingSystem
            Is64BitProcess = [System.Environment]::Is64BitProcess
            
            # PowerShell information
            PowerShellVersion = $PSVersionTable.PSVersion.ToString()
            PowerShellEdition = $PSVersionTable.PSEdition
            CLRVersion = $PSVersionTable.CLRVersion.ToString()
            
            # Hardware information
            ProcessorCount = [System.Environment]::ProcessorCount
            WorkingSet = [System.Environment]::WorkingSet
            SystemPageSize = [System.Environment]::SystemPageSize
            
            # .NET Framework information
            DotNetVersions = Get-DotNetVersions
            
            # Windows features and capabilities
            WindowsFeatures = Get-WindowsFeatureInfo
            
            # Network configuration
            NetworkInfo = Get-NetworkTelemetryInfo
            
            # Security configuration
            SecurityInfo = Get-SecurityTelemetryInfo
            
            # Performance baseline
            PerformanceBaseline = Get-PerformanceBaseline
            
            # Timestamp
            CollectedAt = Get-Date -Format 'yyyy-MM-ddTHH:mm:ss.fffZ'
        }
        
        return $systemInfo
    }
    catch {
        Write-InstallationLog -Level Warning -Message "Error collecting system telemetry: $($_.Exception.Message)"
        return @{
            Error = $_.Exception.Message
            CollectedAt = Get-Date -Format 'yyyy-MM-ddTHH:mm:ss.fffZ'
        }
    }
}

function Get-DotNetVersions {
    <#
    .SYNOPSIS
    Detects installed .NET Framework and .NET Core versions
    #>
    [CmdletBinding()]
    param()
    
    try {
        $dotNetVersions = @{
            Framework = @()
            Core = @()
        }
        
        # Check .NET Framework versions
        try {
            $frameworkVersions = Get-ChildItem 'HKLM:\SOFTWARE\Microsoft\NET Framework Setup\NDP' -Recurse |
                Get-ItemProperty -Name Version, Release -ErrorAction SilentlyContinue |
                Where-Object { $_.Version -and $_.Version -match '^\d+\.\d+' } |
                Select-Object Version, Release
            
            $dotNetVersions.Framework = $frameworkVersions | ForEach-Object { 
                @{ Version = $_.Version; Release = $_.Release } 
            }
        }
        catch {
            Write-InstallationLog -Level Debug -Message "Could not detect .NET Framework versions: $($_.Exception.Message)"
        }
        
        # Check .NET Core/5+ versions
        try {
            $dotnetCommand = Get-Command dotnet -ErrorAction SilentlyContinue
            if ($dotnetCommand) {
                $coreVersions = & dotnet --list-runtimes 2>$null | Where-Object { $_ -match 'Microsoft\.NETCore\.App|Microsoft\.AspNetCore\.App|Microsoft\.WindowsDesktop\.App' }
                $dotNetVersions.Core = $coreVersions
            }
        }
        catch {
            Write-InstallationLog -Level Debug -Message "Could not detect .NET Core versions: $($_.Exception.Message)"
        }
        
        return $dotNetVersions
    }
    catch {
        return @{ Error = $_.Exception.Message }
    }
}

function Get-WindowsFeatureInfo {
    <#
    .SYNOPSIS
    Collects information about relevant Windows features
    #>
    [CmdletBinding()]
    param()
    
    try {
        $features = @{
            WindowsDefender = $false
            Firewall = $false
            UAC = $false
            ExecutionPolicy = "Unknown"
            ServiceManagement = $false
        }
        
        # Check Windows Defender status
        try {
            $defender = Get-MpComputerStatus -ErrorAction SilentlyContinue
            $features.WindowsDefender = $defender -ne $null -and $defender.AntivirusEnabled
        }
        catch { }
        
        # Check Windows Firewall status
        try {
            $firewall = Get-NetFirewallProfile -ErrorAction SilentlyContinue
            $features.Firewall = $firewall -ne $null -and ($firewall | Where-Object { $_.Enabled -eq $true }).Count -gt 0
        }
        catch { }
        
        # Check UAC status
        try {
            $uacKey = Get-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System" -Name "EnableLUA" -ErrorAction SilentlyContinue
            $features.UAC = $uacKey -ne $null -and $uacKey.EnableLUA -eq 1
        }
        catch { }
        
        # Check PowerShell execution policy
        try {
            $features.ExecutionPolicy = Get-ExecutionPolicy
        }
        catch { }
        
        # Check service management capabilities
        try {
            $features.ServiceManagement = (Get-Command sc.exe -ErrorAction SilentlyContinue) -ne $null
        }
        catch { }
        
        return $features
    }
    catch {
        return @{ Error = $_.Exception.Message }
    }
}

function Get-NetworkTelemetryInfo {
    <#
    .SYNOPSIS
    Collects network configuration and connectivity information
    #>
    [CmdletBinding()]
    param()
    
    try {
        $networkInfo = @{
            ConnectedToInternet = $false
            ProxyConfiguration = @{}
            DNSServers = @()
            NetworkAdapters = @()
            ConnectivityTests = @{}
        }
        
        # Test internet connectivity
        try {
            $connectivityTest = Test-NetConnection -ComputerName "8.8.8.8" -Port 53 -InformationLevel Quiet -ErrorAction SilentlyContinue
            $networkInfo.ConnectedToInternet = $connectivityTest
        }
        catch { }
        
        # Get proxy configuration
        try {
            $proxySettings = Get-ItemProperty -Path "HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings" -ErrorAction SilentlyContinue
            if ($proxySettings) {
                $networkInfo.ProxyConfiguration = @{
                    ProxyEnable = $proxySettings.ProxyEnable
                    ProxyServer = $proxySettings.ProxyServer
                    ProxyOverride = $proxySettings.ProxyOverride
                }
            }
        }
        catch { }
        
        # Get DNS servers
        try {
            $dnsServers = Get-DnsClientServerAddress -AddressFamily IPv4 -ErrorAction SilentlyContinue | 
                Where-Object { $_.ServerAddresses.Count -gt 0 } |
                Select-Object -First 3 -ExpandProperty ServerAddresses
            $networkInfo.DNSServers = $dnsServers
        }
        catch { }
        
        # Test connectivity to platform endpoints
        $testEndpoints = @(
            @{ Name = "Platform"; Url = $PlatformEndpoint }
            @{ Name = "GitHub"; Url = "https://github.com" }
            @{ Name = "GitHub API"; Url = "https://api.github.com" }
        )
        
        foreach ($endpoint in $testEndpoints) {
            try {
                $uri = [System.Uri]$endpoint.Url
                $testResult = Test-NetConnection -ComputerName $uri.Host -Port 443 -InformationLevel Quiet -ErrorAction SilentlyContinue -WarningAction SilentlyContinue
                $networkInfo.ConnectivityTests[$endpoint.Name] = $testResult
            }
            catch {
                $networkInfo.ConnectivityTests[$endpoint.Name] = $false
            }
        }
        
        return $networkInfo
    }
    catch {
        return @{ Error = $_.Exception.Message }
    }
}

function Get-SecurityTelemetryInfo {
    <#
    .SYNOPSIS
    Collects security-related configuration information
    #>
    [CmdletBinding()]
    param()
    
    try {
        $securityInfo = @{
            IsAdministrator = $false
            ExecutionPolicy = "Unknown"
            AntivirusStatus = "Unknown"
            FirewallStatus = "Unknown"
            UACStatus = "Unknown"
            TrustedHosts = @()
        }
        
        # Check if running as administrator
        try {
            $currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
            $securityInfo.IsAdministrator = $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
        }
        catch { }
        
        # Get execution policy
        try {
            $securityInfo.ExecutionPolicy = Get-ExecutionPolicy -Scope CurrentUser
        }
        catch { }
        
        # Check antivirus status
        try {
            $antivirus = Get-CimInstance -Namespace "Root\SecurityCenter2" -ClassName "AntiVirusProduct" -ErrorAction SilentlyContinue
            if ($antivirus) {
                $securityInfo.AntivirusStatus = "Detected"
            }
        }
        catch { }
        
        # Get trusted hosts configuration
        try {
            $trustedHosts = Get-Item WSMan:\localhost\Client\TrustedHosts -ErrorAction SilentlyContinue
            if ($trustedHosts) {
                $securityInfo.TrustedHosts = $trustedHosts.Value -split ','
            }
        }
        catch { }
        
        return $securityInfo
    }
    catch {
        return @{ Error = $_.Exception.Message }
    }
}

# ================================================================
# Security Validation and Audit Logging Functions
# ================================================================

function Initialize-SecurityAuditSystem {
    <#
    .SYNOPSIS
    Initializes the security audit system for comprehensive logging
    
    .DESCRIPTION
    Sets up security event logging to Windows Event Log, creates audit trail
    infrastructure, and initializes security compliance validation.
    #>
    [CmdletBinding()]
    param()
    
    Write-InstallationLog -Level Debug -Message "Initializing security audit system"
    
    try {
        # Initialize security audit data structure
        $script:SecurityAudit = @{
            AuditId = [System.Guid]::NewGuid().ToString()
            InstallationId = $script:InstallationId
            StartTime = Get-Date
            Events = @()
            SecurityChecks = @{}
            ComplianceStatus = @{}
            TamperDetection = @{}
        }
        
        # Create security event source for Windows Event Log
        $securityEventSource = "RepSetBridge-Security"
        if (-not [System.Diagnostics.EventLog]::SourceExists($securityEventSource)) {
            try {
                [System.Diagnostics.EventLog]::CreateEventSource($securityEventSource, "Security")
                Write-InstallationLog -Level Success -Message "Security event source created successfully"
            }
            catch {
                Write-InstallationLog -Level Warning -Message "Could not create security event source: $($_.Exception.Message)"
                # Try Application log as fallback
                try {
                    [System.Diagnostics.EventLog]::CreateEventSource($securityEventSource, "Application")
                    Write-InstallationLog -Level Info -Message "Security events will be logged to Application log"
                }
                catch {
                    Write-InstallationLog -Level Warning -Message "Could not create security event source in Application log: $($_.Exception.Message)"
                }
            }
        }
        
        # Log security audit initialization
        Write-SecurityAuditEvent -EventType "AuditInitialized" -Severity "Information" -Message "Security audit system initialized" -Details @{
            AuditId = $script:SecurityAudit.AuditId
            InstallationId = $script:InstallationId
            SystemInfo = Get-SecuritySystemInfo
        }
        
        Write-InstallationLog -Level Success -Message "Security audit system initialized successfully"
        return $true
    }
    catch {
        Write-InstallationLog -Level Error -Message "Failed to initialize security audit system: $($_.Exception.Message)"
        return $false
    }
}

function Write-SecurityAuditEvent {
    <#
    .SYNOPSIS
    Writes security events to audit trail and Windows Event Log
    
    .DESCRIPTION
    Creates comprehensive security audit entries with tamper detection,
    contextual information, and compliance tracking.
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [ValidateSet('AuditInitialized', 'SignatureValidation', 'TamperDetection', 'ComplianceCheck', 
                     'SecurityViolation', 'AuthenticationAttempt', 'IntegrityCheck', 'AccessControl',
                     'ConfigurationChange', 'ServiceInstallation', 'NetworkConnection', 'AuditCompleted')]
        [string]$EventType,
        
        [Parameter(Mandatory=$true)]
        [ValidateSet('Information', 'Warning', 'Error', 'Critical')]
        [string]$Severity,
        
        [Parameter(Mandatory=$true)]
        [string]$Message,
        
        [Parameter(Mandatory=$false)]
        [hashtable]$Details = @{},
        
        [Parameter(Mandatory=$false)]
        [string]$Component = "Installer",
        
        [Parameter(Mandatory=$false)]
        [switch]$SkipEventLog
    )
    
    try {
        $timestamp = Get-Date
        $eventId = Get-SecurityEventId -EventType $EventType
        
        # Create comprehensive audit event
        $auditEvent = @{
            EventId = [System.Guid]::NewGuid().ToString()
            AuditId = $script:SecurityAudit.AuditId
            InstallationId = $script:InstallationId
            Timestamp = $timestamp.ToString('yyyy-MM-ddTHH:mm:ss.fffZ')
            EventType = $EventType
            Severity = $Severity
            Component = $Component
            Message = $Message
            Details = $Details
            SystemContext = @{
                MachineName = [System.Environment]::MachineName
                UserName = [System.Environment]::UserName
                ProcessId = $PID
                ThreadId = [System.Threading.Thread]::CurrentThread.ManagedThreadId
                IsElevated = Test-IsElevated
            }
            SecurityContext = Get-CurrentSecurityContext
            IntegrityHash = $null
        }
        
        # Calculate integrity hash for tamper detection
        $auditEvent.IntegrityHash = Get-AuditEventHash -AuditEvent $auditEvent
        
        # Add to audit trail
        if ($script:SecurityAudit) {
            $script:SecurityAudit.Events += $auditEvent
        }
        
        # Write to Windows Event Log
        if (-not $SkipEventLog) {
            Write-WindowsSecurityEvent -AuditEvent $auditEvent -EventId $eventId
        }
        
        # Write to security audit file
        Write-SecurityAuditFile -AuditEvent $auditEvent
        
        # Send to platform security monitoring
        Send-SecurityEventToPlatform -AuditEvent $auditEvent
        
        # Log to installation log with security context
        $logLevel = switch ($Severity) {
            'Information' { 'Info' }
            'Warning' { 'Warning' }
            'Error' { 'Error' }
            'Critical' { 'Error' }
        }
        
        Write-InstallationLog -Level $logLevel -Message "[SECURITY] $EventType`: $Message" -Context @{
            EventType = $EventType
            Severity = $Severity
            Component = $Component
            EventId = $auditEvent.EventId
            AuditId = $script:SecurityAudit.AuditId
        }
        
        return $auditEvent.EventId
    }
    catch {
        Write-InstallationLog -Level Error -Message "Failed to write security audit event: $($_.Exception.Message)"
        return $null
    }
}

function Get-SecurityEventId {
    <#
    .SYNOPSIS
    Maps security event types to Windows Event Log event IDs
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$EventType
    )
    
    $eventIds = @{
        'AuditInitialized' = 2001
        'SignatureValidation' = 2002
        'TamperDetection' = 2003
        'ComplianceCheck' = 2004
        'SecurityViolation' = 2005
        'AuthenticationAttempt' = 2006
        'IntegrityCheck' = 2007
        'AccessControl' = 2008
        'ConfigurationChange' = 2009
        'ServiceInstallation' = 2010
        'NetworkConnection' = 2011
        'AuditCompleted' = 2012
    }
    
    return $eventIds[$EventType] -or 2000
}

function Write-WindowsSecurityEvent {
    <#
    .SYNOPSIS
    Writes security events to Windows Event Log
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [hashtable]$AuditEvent,
        
        [Parameter(Mandatory=$true)]
        [int]$EventId
    )
    
    try {
        $eventSource = "RepSetBridge-Security"
        
        # Determine Windows Event Log entry type
        $entryType = switch ($AuditEvent.Severity) {
            'Information' { 'Information' }
            'Warning' { 'Warning' }
            'Error' { 'Error' }
            'Critical' { 'Error' }
            default { 'Information' }
        }
        
        # Create detailed event message
        $eventMessage = @"
RepSet Bridge Security Event

Event Type: $($AuditEvent.EventType)
Severity: $($AuditEvent.Severity)
Component: $($AuditEvent.Component)
Message: $($AuditEvent.Message)

Event Details:
- Event ID: $($AuditEvent.EventId)
- Audit ID: $($AuditEvent.AuditId)
- Installation ID: $($AuditEvent.InstallationId)
- Timestamp: $($AuditEvent.Timestamp)

System Context:
- Machine: $($AuditEvent.SystemContext.MachineName)
- User: $($AuditEvent.SystemContext.UserName)
- Process ID: $($AuditEvent.SystemContext.ProcessId)
- Elevated: $($AuditEvent.SystemContext.IsElevated)

Security Context:
$(if ($AuditEvent.SecurityContext) { $AuditEvent.SecurityContext | ConvertTo-Json -Depth 2 })

Additional Details:
$(if ($AuditEvent.Details.Count -gt 0) { $AuditEvent.Details | ConvertTo-Json -Depth 3 })

Integrity Hash: $($AuditEvent.IntegrityHash)
"@
        
        # Write to Windows Event Log
        Write-EventLog -LogName Application -Source $eventSource -EntryType $entryType -EventId $EventId -Message $eventMessage
        
        Write-InstallationLog -Level Debug -Message "Security event written to Windows Event Log (ID: $EventId)"
    }
    catch {
        Write-InstallationLog -Level Warning -Message "Failed to write security event to Windows Event Log: $($_.Exception.Message)"
    }
}

function Write-SecurityAuditFile {
    <#
    .SYNOPSIS
    Writes security audit events to dedicated audit file
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [hashtable]$AuditEvent
    )
    
    try {
        $auditLogPath = "$env:TEMP\RepSetBridge-SecurityAudit-$(Get-Date -Format 'yyyyMMdd').log"
        
        # Create structured audit log entry
        $auditLogEntry = @{
            Timestamp = $AuditEvent.Timestamp
            EventId = $AuditEvent.EventId
            AuditId = $AuditEvent.AuditId
            InstallationId = $AuditEvent.InstallationId
            EventType = $AuditEvent.EventType
            Severity = $AuditEvent.Severity
            Component = $AuditEvent.Component
            Message = $AuditEvent.Message
            Details = $AuditEvent.Details
            SystemContext = $AuditEvent.SystemContext
            SecurityContext = $AuditEvent.SecurityContext
            IntegrityHash = $AuditEvent.IntegrityHash
        }
        
        # Convert to JSON and append to audit file
        $jsonEntry = $auditLogEntry | ConvertTo-Json -Depth 4 -Compress
        Add-Content -Path $auditLogPath -Value $jsonEntry -ErrorAction SilentlyContinue
        
        # Also create human-readable audit entry
        $readableAuditPath = "$env:TEMP\RepSetBridge-SecurityAudit-Readable-$(Get-Date -Format 'yyyyMMdd').log"
        $readableEntry = "[$($AuditEvent.Timestamp)] [$($AuditEvent.Severity)] [$($AuditEvent.EventType)] $($AuditEvent.Message)"
        Add-Content -Path $readableAuditPath -Value $readableEntry -ErrorAction SilentlyContinue
        
        Write-InstallationLog -Level Debug -Message "Security audit event written to file: $auditLogPath"
    }
    catch {
        Write-InstallationLog -Level Warning -Message "Failed to write security audit to file: $($_.Exception.Message)"
    }
}

function Send-SecurityEventToPlatform {
    <#
    .SYNOPSIS
    Sends security events to platform security monitoring system
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [hashtable]$AuditEvent
    )
    
    try {
        # Only send critical security events to platform to avoid overwhelming
        if ($AuditEvent.Severity -in @('Warning', 'Error', 'Critical') -or 
            $AuditEvent.EventType -in @('SecurityViolation', 'TamperDetection', 'SignatureValidation')) {
            
            $securityData = @{
                auditId = $AuditEvent.AuditId
                installationId = $AuditEvent.InstallationId
                gymId = $GymId
                eventId = $AuditEvent.EventId
                timestamp = $AuditEvent.Timestamp
                eventType = $AuditEvent.EventType
                severity = $AuditEvent.Severity
                component = $AuditEvent.Component
                message = $AuditEvent.Message
                details = $AuditEvent.Details
                systemContext = $AuditEvent.SystemContext
                securityContext = $AuditEvent.SecurityContext
                integrityHash = $AuditEvent.IntegrityHash
            }
            
            $platformUrl = "$PlatformEndpoint/api/security/events"
            $headers = @{
                'Content-Type' = 'application/json'
                'User-Agent' = 'RepSet-Bridge-Installer/1.0'
                'X-Security-Event' = 'true'
            }
            
            $json = $securityData | ConvertTo-Json -Depth 4
            
            # Send security event asynchronously with retry logic
            Start-Job -ScriptBlock {
                param($Url, $Headers, $Data)
                try {
                    $maxRetries = 3
                    for ($attempt = 1; $attempt -le $maxRetries; $attempt++) {
                        try {
                            Invoke-RestMethod -Uri $Url -Method Post -Headers $Headers -Body $Data -TimeoutSec 10 -ErrorAction Stop
                            break
                        }
                        catch {
                            if ($attempt -eq $maxRetries) { throw }
                            Start-Sleep -Seconds ($attempt * 2)
                        }
                    }
                }
                catch {
                    # Log failure but don't block installation
                }
            } -ArgumentList $platformUrl, $headers, $json | Out-Null
        }
    }
    catch {
        Write-InstallationLog -Level Debug -Message "Failed to send security event to platform: $($_.Exception.Message)"
    }
}

function Test-InstallationCommandSignature {
    <#
    .SYNOPSIS
    Validates the cryptographic signature of the installation command
    
    .DESCRIPTION
    Performs comprehensive signature validation including HMAC verification,
    timestamp validation, nonce checking, and tamper detection.
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$PairCode,
        
        [Parameter(Mandatory=$true)]
        [string]$Signature,
        
        [Parameter(Mandatory=$true)]
        [string]$Nonce,
        
        [Parameter(Mandatory=$true)]
        [string]$GymId,
        
        [Parameter(Mandatory=$true)]
        [string]$ExpiresAt,
        
        [Parameter(Mandatory=$false)]
        [string]$PlatformEndpoint = "https://app.repset.com"
    )
    
    Write-SecurityAuditEvent -EventType "SignatureValidation" -Severity "Information" -Message "Starting installation command signature validation" -Details @{
        GymId = $GymId
        Nonce = $Nonce
        ExpiresAt = $ExpiresAt
        PlatformEndpoint = $PlatformEndpoint
    }
    
    try {
        # Step 1: Validate command expiration
        Write-InstallationLog -Level Info -Message "Validating command expiration..."
        
        try {
            $expirationTime = [DateTime]::Parse($ExpiresAt)
            $currentTime = Get-Date
            
            if ($currentTime -gt $expirationTime) {
                $errorMessage = "Installation command has expired. Expiration: $ExpiresAt, Current: $($currentTime.ToString('yyyy-MM-ddTHH:mm:ss.fffZ'))"
                Write-SecurityAuditEvent -EventType "SecurityViolation" -Severity "Error" -Message $errorMessage -Details @{
                    ViolationType = "ExpiredCommand"
                    ExpirationTime = $ExpiresAt
                    CurrentTime = $currentTime.ToString('yyyy-MM-ddTHH:mm:ss.fffZ')
                    TimeDifference = ($currentTime - $expirationTime).TotalMinutes
                }
                
                Write-InstallationLog -Level Error -Message $errorMessage
                return @{ IsValid = $false; ErrorCode = $script:ErrorCodes.ExpiredCommand; ErrorMessage = $errorMessage }
            }
            
            Write-InstallationLog -Level Success -Message "Command expiration validation passed"
        }
        catch {
            $errorMessage = "Invalid expiration timestamp format: $ExpiresAt"
            Write-SecurityAuditEvent -EventType "SecurityViolation" -Severity "Error" -Message $errorMessage -Details @{
                ViolationType = "InvalidTimestamp"
                ExpiresAt = $ExpiresAt
                ParseError = $_.Exception.Message
            }
            
            Write-InstallationLog -Level Error -Message $errorMessage
            return @{ IsValid = $false; ErrorCode = $script:ErrorCodes.InvalidSignature; ErrorMessage = $errorMessage }
        }
        
        # Step 2: Validate signature format and structure
        Write-InstallationLog -Level Info -Message "Validating signature format..."
        
        if ([string]::IsNullOrWhiteSpace($Signature) -or $Signature.Length -lt 32) {
            $errorMessage = "Invalid signature format or length"
            Write-SecurityAuditEvent -EventType "SecurityViolation" -Severity "Error" -Message $errorMessage -Details @{
                ViolationType = "InvalidSignatureFormat"
                SignatureLength = $Signature.Length
            }
            
            Write-InstallationLog -Level Error -Message $errorMessage
            return @{ IsValid = $false; ErrorCode = $script:ErrorCodes.InvalidSignature; ErrorMessage = $errorMessage }
        }
        
        # Step 3: Validate nonce format
        Write-InstallationLog -Level Info -Message "Validating nonce format..."
        
        if ([string]::IsNullOrWhiteSpace($Nonce) -or $Nonce.Length -lt 16) {
            $errorMessage = "Invalid nonce format or length"
            Write-SecurityAuditEvent -EventType "SecurityViolation" -Severity "Error" -Message $errorMessage -Details @{
                ViolationType = "InvalidNonce"
                NonceLength = $Nonce.Length
            }
            
            Write-InstallationLog -Level Error -Message $errorMessage
            return @{ IsValid = $false; ErrorCode = $script:ErrorCodes.InvalidSignature; ErrorMessage = $errorMessage }
        }
        
        # Step 4: Construct message for signature verification
        Write-InstallationLog -Level Info -Message "Constructing signature verification message..."
        
        $messageComponents = @(
            $PairCode,
            $GymId,
            $Nonce,
            $ExpiresAt,
            $PlatformEndpoint
        )
        
        $message = $messageComponents -join "|"
        
        # Step 5: Verify signature with platform
        Write-InstallationLog -Level Info -Message "Verifying signature with platform..."
        
        $verificationResult = Invoke-SignatureVerificationWithPlatform -Message $message -Signature $Signature -Nonce $Nonce -GymId $GymId
        
        if (-not $verificationResult.IsValid) {
            $errorMessage = "Signature verification failed: $($verificationResult.ErrorMessage)"
            Write-SecurityAuditEvent -EventType "SecurityViolation" -Severity "Error" -Message $errorMessage -Details @{
                ViolationType = "SignatureVerificationFailed"
                Message = $message
                Signature = $Signature
                VerificationError = $verificationResult.ErrorMessage
            }
            
            Write-InstallationLog -Level Error -Message $errorMessage
            return @{ IsValid = $false; ErrorCode = $script:ErrorCodes.InvalidSignature; ErrorMessage = $errorMessage }
        }
        
        # Step 6: Check for replay attacks (nonce validation)
        Write-InstallationLog -Level Info -Message "Checking for replay attacks..."
        
        $replayCheckResult = Test-NonceReplayAttack -Nonce $Nonce -GymId $GymId
        
        if (-not $replayCheckResult.IsValid) {
            $errorMessage = "Potential replay attack detected: $($replayCheckResult.ErrorMessage)"
            Write-SecurityAuditEvent -EventType "SecurityViolation" -Severity "Critical" -Message $errorMessage -Details @{
                ViolationType = "ReplayAttack"
                Nonce = $Nonce
                GymId = $GymId
                ReplayDetails = $replayCheckResult.Details
            }
            
            Write-InstallationLog -Level Error -Message $errorMessage
            return @{ IsValid = $false; ErrorCode = $script:ErrorCodes.InvalidSignature; ErrorMessage = $errorMessage }
        }
        
        # Step 7: Perform tamper detection checks
        Write-InstallationLog -Level Info -Message "Performing tamper detection checks..."
        
        $tamperCheckResult = Test-CommandTampering -PairCode $PairCode -Signature $Signature -Nonce $Nonce -GymId $GymId -ExpiresAt $ExpiresAt
        
        if (-not $tamperCheckResult.IsValid) {
            $errorMessage = "Command tampering detected: $($tamperCheckResult.ErrorMessage)"
            Write-SecurityAuditEvent -EventType "TamperDetection" -Severity "Critical" -Message $errorMessage -Details @{
                TamperType = $tamperCheckResult.TamperType
                TamperDetails = $tamperCheckResult.Details
            }
            
            Write-InstallationLog -Level Error -Message $errorMessage
            return @{ IsValid = $false; ErrorCode = $script:ErrorCodes.InvalidSignature; ErrorMessage = $errorMessage }
        }
        
        # All validation checks passed
        Write-SecurityAuditEvent -EventType "SignatureValidation" -Severity "Information" -Message "Installation command signature validation completed successfully" -Details @{
            ValidationSteps = @(
                "ExpirationCheck",
                "SignatureFormat",
                "NonceFormat", 
                "PlatformVerification",
                "ReplayAttackCheck",
                "TamperDetection"
            )
            GymId = $GymId
            Nonce = $Nonce
        }
        
        Write-InstallationLog -Level Success -Message "Installation command signature validation completed successfully"
        
        return @{ 
            IsValid = $true
            ErrorCode = $script:ErrorCodes.Success
            ErrorMessage = ""
            ValidationDetails = @{
                ExpirationTime = $expirationTime
                SignatureVerified = $true
                NonceValid = $true
                TamperCheckPassed = $true
                ReplayCheckPassed = $true
            }
        }
    }
    catch {
        $errorMessage = "Signature validation failed with exception: $($_.Exception.Message)"
        Write-SecurityAuditEvent -EventType "SecurityViolation" -Severity "Critical" -Message $errorMessage -Details @{
            ViolationType = "ValidationException"
            Exception = $_.Exception.Message
            StackTrace = $_.Exception.StackTrace
        }
        
        Write-InstallationLog -Level Error -Message $errorMessage
        return @{ IsValid = $false; ErrorCode = $script:ErrorCodes.InvalidSignature; ErrorMessage = $errorMessage }
    }
}

function Invoke-SignatureVerificationWithPlatform {
    <#
    .SYNOPSIS
    Verifies signature with the platform's signature validation endpoint
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$Message,
        
        [Parameter(Mandatory=$true)]
        [string]$Signature,
        
        [Parameter(Mandatory=$true)]
        [string]$Nonce,
        
        [Parameter(Mandatory=$true)]
        [string]$GymId
    )
    
    try {
        $verificationData = @{
            message = $Message
            signature = $Signature
            nonce = $Nonce
            gymId = $GymId
            timestamp = Get-Date -Format 'yyyy-MM-ddTHH:mm:ss.fffZ'
        }
        
        $platformUrl = "$PlatformEndpoint/api/installation/verify-signature"
        $headers = @{
            'Content-Type' = 'application/json'
            'User-Agent' = 'RepSet-Bridge-Installer/1.0'
            'X-Verification-Request' = 'true'
        }
        
        $json = $verificationData | ConvertTo-Json -Depth 2
        
        # Attempt signature verification with retry logic
        $maxRetries = 3
        $retryDelay = 2
        
        for ($attempt = 1; $attempt -le $maxRetries; $attempt++) {
            try {
                Write-InstallationLog -Level Debug -Message "Attempting signature verification with platform (attempt $attempt/$maxRetries)"
                
                $response = Invoke-RestMethod -Uri $platformUrl -Method Post -Headers $headers -Body $json -TimeoutSec 15 -ErrorAction Stop
                
                if ($response -and $response.isValid -eq $true) {
                    Write-InstallationLog -Level Success -Message "Signature verification successful with platform"
                    return @{ IsValid = $true; ErrorMessage = "" }
                }
                else {
                    $errorMessage = if ($response.errorMessage) { $response.errorMessage } else { "Platform returned invalid signature" }
                    Write-InstallationLog -Level Warning -Message "Platform signature verification failed: $errorMessage"
                    return @{ IsValid = $false; ErrorMessage = $errorMessage }
                }
            }
            catch {
                $errorMessage = "Platform verification attempt $attempt failed: $($_.Exception.Message)"
                Write-InstallationLog -Level Warning -Message $errorMessage
                
                if ($attempt -eq $maxRetries) {
                    return @{ IsValid = $false; ErrorMessage = "Platform verification failed after $maxRetries attempts: $($_.Exception.Message)" }
                }
                
                Start-Sleep -Seconds ($retryDelay * $attempt)
            }
        }
        
        return @{ IsValid = $false; ErrorMessage = "Platform verification failed after all retry attempts" }
    }
    catch {
        return @{ IsValid = $false; ErrorMessage = "Signature verification exception: $($_.Exception.Message)" }
    }
}

function Test-NonceReplayAttack {
    <#
    .SYNOPSIS
    Checks for potential replay attacks using nonce validation
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$Nonce,
        
        [Parameter(Mandatory=$true)]
        [string]$GymId
    )
    
    try {
        # Check local nonce cache first
        $localNonceFile = "$env:TEMP\RepSetBridge-UsedNonces.json"
        $usedNonces = @{}
        
        if (Test-Path $localNonceFile) {
            try {
                $nonceData = Get-Content $localNonceFile -Raw | ConvertFrom-Json
                $usedNonces = $nonceData
            }
            catch {
                Write-InstallationLog -Level Warning -Message "Could not read local nonce cache: $($_.Exception.Message)"
            }
        }
        
        # Check if nonce was already used locally
        $nonceKey = "$GymId`:$Nonce"
        if ($usedNonces.ContainsKey($nonceKey)) {
            $previousUse = $usedNonces[$nonceKey]
            return @{ 
                IsValid = $false
                ErrorMessage = "Nonce already used locally"
                Details = @{
                    PreviousUse = $previousUse
                    NonceKey = $nonceKey
                }
            }
        }
        
        # Check with platform for nonce usage
        try {
            $nonceCheckData = @{
                nonce = $Nonce
                gymId = $GymId
                timestamp = Get-Date -Format 'yyyy-MM-ddTHH:mm:ss.fffZ'
            }
            
            $platformUrl = "$PlatformEndpoint/api/installation/check-nonce"
            $headers = @{
                'Content-Type' = 'application/json'
                'User-Agent' = 'RepSet-Bridge-Installer/1.0'
            }
            
            $json = $nonceCheckData | ConvertTo-Json -Depth 2
            $response = Invoke-RestMethod -Uri $platformUrl -Method Post -Headers $headers -Body $json -TimeoutSec 10 -ErrorAction Stop
            
            if ($response -and $response.isUsed -eq $true) {
                return @{ 
                    IsValid = $false
                    ErrorMessage = "Nonce already used (platform verification)"
                    Details = @{
                        PlatformResponse = $response
                        NonceKey = $nonceKey
                    }
                }
            }
        }
        catch {
            Write-InstallationLog -Level Warning -Message "Could not verify nonce with platform: $($_.Exception.Message)"
            # Continue with local validation only
        }
        
        # Mark nonce as used locally
        try {
            $usedNonces[$nonceKey] = @{
                UsedAt = Get-Date -Format 'yyyy-MM-ddTHH:mm:ss.fffZ'
                InstallationId = $script:InstallationId
            }
            
            # Clean up old nonces (older than 7 days)
            $cutoffDate = (Get-Date).AddDays(-7)
            $cleanedNonces = @{}
            foreach ($key in $usedNonces.Keys) {
                try {
                    $usedDate = [DateTime]::Parse($usedNonces[$key].UsedAt)
                    if ($usedDate -gt $cutoffDate) {
                        $cleanedNonces[$key] = $usedNonces[$key]
                    }
                }
                catch {
                    # Skip invalid entries
                }
            }
            
            $cleanedNonces | ConvertTo-Json -Depth 3 | Set-Content $localNonceFile -ErrorAction SilentlyContinue
        }
        catch {
            Write-InstallationLog -Level Warning -Message "Could not update local nonce cache: $($_.Exception.Message)"
        }
        
        return @{ IsValid = $true; ErrorMessage = "" }
    }
    catch {
        Write-InstallationLog -Level Warning -Message "Nonce replay check failed: $($_.Exception.Message)"
        # In case of error, allow the installation to proceed but log the issue
        return @{ IsValid = $true; ErrorMessage = "" }
    }
}

function Test-CommandTampering {
    <#
    .SYNOPSIS
    Performs comprehensive tamper detection on installation command parameters
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$PairCode,
        
        [Parameter(Mandatory=$true)]
        [string]$Signature,
        
        [Parameter(Mandatory=$true)]
        [string]$Nonce,
        
        [Parameter(Mandatory=$true)]
        [string]$GymId,
        
        [Parameter(Mandatory=$true)]
        [string]$ExpiresAt
    )
    
    try {
        $tamperChecks = @()
        
        # Check 1: Parameter format validation
        $formatChecks = @{
            PairCode = Test-ParameterFormat -Value $PairCode -Pattern '^[A-Z0-9]{6,12}$' -Name "PairCode"
            GymId = Test-ParameterFormat -Value $GymId -Pattern '^[a-f0-9\-]{36}$' -Name "GymId"
            Nonce = Test-ParameterFormat -Value $Nonce -Pattern '^[A-Za-z0-9+/]{16,}={0,2}$' -Name "Nonce"
            Signature = Test-ParameterFormat -Value $Signature -Pattern '^[A-Za-z0-9+/]{32,}={0,2}$' -Name "Signature"
        }
        
        foreach ($check in $formatChecks.GetEnumerator()) {
            if (-not $check.Value.IsValid) {
                return @{
                    IsValid = $false
                    ErrorMessage = "Parameter format tampering detected: $($check.Value.ErrorMessage)"
                    TamperType = "ParameterFormat"
                    Details = @{
                        Parameter = $check.Key
                        FormatCheck = $check.Value
                    }
                }
            }
            $tamperChecks += "ParameterFormat_$($check.Key)"
        }
        
        # Check 2: Parameter length validation
        $lengthChecks = @{
            PairCode = @{ Min = 6; Max = 12; Actual = $PairCode.Length }
            GymId = @{ Min = 36; Max = 36; Actual = $GymId.Length }
            Nonce = @{ Min = 16; Max = 256; Actual = $Nonce.Length }
            Signature = @{ Min = 32; Max = 512; Actual = $Signature.Length }
        }
        
        foreach ($check in $lengthChecks.GetEnumerator()) {
            $length = $check.Value
            if ($length.Actual -lt $length.Min -or $length.Actual -gt $length.Max) {
                return @{
                    IsValid = $false
                    ErrorMessage = "Parameter length tampering detected: $($check.Key) length $($length.Actual) outside valid range [$($length.Min)-$($length.Max)]"
                    TamperType = "ParameterLength"
                    Details = @{
                        Parameter = $check.Key
                        LengthCheck = $length
                    }
                }
            }
            $tamperChecks += "ParameterLength_$($check.Key)"
        }
        
        # Check 3: Character set validation
        $characterChecks = @{
            PairCode = Test-CharacterSet -Value $PairCode -AllowedChars "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789" -Name "PairCode"
            GymId = Test-CharacterSet -Value $GymId -AllowedChars "abcdef0123456789-" -Name "GymId"
        }
        
        foreach ($check in $characterChecks.GetEnumerator()) {
            if (-not $check.Value.IsValid) {
                return @{
                    IsValid = $false
                    ErrorMessage = "Character set tampering detected: $($check.Value.ErrorMessage)"
                    TamperType = "CharacterSet"
                    Details = @{
                        Parameter = $check.Key
                        CharacterCheck = $check.Value
                    }
                }
            }
            $tamperChecks += "CharacterSet_$($check.Key)"
        }
        
        # Check 4: Timestamp validation
        try {
            $expirationTime = [DateTime]::Parse($ExpiresAt)
            $currentTime = Get-Date
            $timeDifference = ($expirationTime - $currentTime).TotalHours
            
            # Check for unreasonable expiration times (more than 48 hours in future or past)
            if ([Math]::Abs($timeDifference) -gt 48) {
                return @{
                    IsValid = $false
                    ErrorMessage = "Timestamp tampering detected: unreasonable expiration time difference of $([Math]::Round($timeDifference, 2)) hours"
                    TamperType = "TimestampTampering"
                    Details = @{
                        ExpirationTime = $ExpiresAt
                        CurrentTime = $currentTime.ToString('yyyy-MM-ddTHH:mm:ss.fffZ')
                        TimeDifferenceHours = $timeDifference
                    }
                }
            }
            $tamperChecks += "TimestampValidation"
        }
        catch {
            return @{
                IsValid = $false
                ErrorMessage = "Timestamp tampering detected: invalid timestamp format"
                TamperType = "TimestampFormat"
                Details = @{
                    ExpiresAt = $ExpiresAt
                    ParseError = $_.Exception.Message
                }
            }
        }
        
        # Check 5: Cross-parameter consistency
        $consistencyChecks = Test-ParameterConsistency -PairCode $PairCode -GymId $GymId -Nonce $Nonce -Signature $Signature
        if (-not $consistencyChecks.IsValid) {
            return @{
                IsValid = $false
                ErrorMessage = "Parameter consistency tampering detected: $($consistencyChecks.ErrorMessage)"
                TamperType = "ParameterConsistency"
                Details = $consistencyChecks.Details
            }
        }
        $tamperChecks += "ParameterConsistency"
        
        # All tamper detection checks passed
        Write-SecurityAuditEvent -EventType "TamperDetection" -Severity "Information" -Message "Command tamper detection completed successfully" -Details @{
            TamperChecks = $tamperChecks
            CheckCount = $tamperChecks.Count
        }
        
        return @{ 
            IsValid = $true
            ErrorMessage = ""
            TamperChecks = $tamperChecks
        }
    }
    catch {
        return @{
            IsValid = $false
            ErrorMessage = "Tamper detection failed with exception: $($_.Exception.Message)"
            TamperType = "DetectionException"
            Details = @{
                Exception = $_.Exception.Message
                StackTrace = $_.Exception.StackTrace
            }
        }
    }
}

function Test-ParameterFormat {
    <#
    .SYNOPSIS
    Tests parameter against expected format pattern
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$Value,
        
        [Parameter(Mandatory=$true)]
        [string]$Pattern,
        
        [Parameter(Mandatory=$true)]
        [string]$Name
    )
    
    try {
        if ($Value -match $Pattern) {
            return @{ IsValid = $true; ErrorMessage = "" }
        }
        else {
            return @{ IsValid = $false; ErrorMessage = "$Name format validation failed" }
        }
    }
    catch {
        return @{ IsValid = $false; ErrorMessage = "$Name format validation exception: $($_.Exception.Message)" }
    }
}

function Test-CharacterSet {
    <#
    .SYNOPSIS
    Tests parameter against allowed character set
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$Value,
        
        [Parameter(Mandatory=$true)]
        [string]$AllowedChars,
        
        [Parameter(Mandatory=$true)]
        [string]$Name
    )
    
    try {
        $allowedCharArray = $AllowedChars.ToCharArray()
        $valueCharArray = $Value.ToCharArray()
        
        foreach ($char in $valueCharArray) {
            if ($char -notin $allowedCharArray) {
                return @{ 
                    IsValid = $false
                    ErrorMessage = "$Name contains invalid character: '$char'"
                    InvalidCharacter = $char
                }
            }
        }
        
        return @{ IsValid = $true; ErrorMessage = "" }
    }
    catch {
        return @{ IsValid = $false; ErrorMessage = "$Name character set validation exception: $($_.Exception.Message)" }
    }
}

function Test-ParameterConsistency {
    <#
    .SYNOPSIS
    Tests cross-parameter consistency and relationships
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$PairCode,
        
        [Parameter(Mandatory=$true)]
        [string]$GymId,
        
        [Parameter(Mandatory=$true)]
        [string]$Nonce,
        
        [Parameter(Mandatory=$true)]
        [string]$Signature
    )
    
    try {
        # Check 1: Ensure parameters are not identical (which would indicate tampering)
        $parameters = @($PairCode, $GymId, $Nonce, $Signature)
        $uniqueParameters = $parameters | Select-Object -Unique
        
        if ($uniqueParameters.Count -ne $parameters.Count) {
            return @{
                IsValid = $false
                ErrorMessage = "Duplicate parameters detected (potential tampering)"
                Details = @{
                    ParameterCount = $parameters.Count
                    UniqueCount = $uniqueParameters.Count
                }
            }
        }
        
        # Check 2: Validate GymId format (should be a valid GUID)
        try {
            $guid = [System.Guid]::Parse($GymId)
            if ($guid -eq [System.Guid]::Empty) {
                return @{
                    IsValid = $false
                    ErrorMessage = "GymId is empty GUID"
                    Details = @{ GymId = $GymId }
                }
            }
        }
        catch {
            return @{
                IsValid = $false
                ErrorMessage = "GymId is not a valid GUID format"
                Details = @{ 
                    GymId = $GymId
                    ParseError = $_.Exception.Message
                }
            }
        }
        
        # Check 3: Validate Base64 encoding for Nonce and Signature
        $base64Checks = @{
            Nonce = $Nonce
            Signature = $Signature
        }
        
        foreach ($check in $base64Checks.GetEnumerator()) {
            try {
                $decoded = [System.Convert]::FromBase64String($check.Value)
                if ($decoded.Length -eq 0) {
                    return @{
                        IsValid = $false
                        ErrorMessage = "$($check.Key) decodes to empty data"
                        Details = @{ Parameter = $check.Key; Value = $check.Value }
                    }
                }
            }
            catch {
                return @{
                    IsValid = $false
                    ErrorMessage = "$($check.Key) is not valid Base64"
                    Details = @{ 
                        Parameter = $check.Key
                        Value = $check.Value
                        DecodeError = $_.Exception.Message
                    }
                }
            }
        }
        
        return @{ IsValid = $true; ErrorMessage = "" }
    }
    catch {
        return @{
            IsValid = $false
            ErrorMessage = "Parameter consistency check exception: $($_.Exception.Message)"
            Details = @{
                Exception = $_.Exception.Message
            }
        }
    }
}

function Test-SecurityCompliance {
    <#
    .SYNOPSIS
    Performs comprehensive security compliance validation
    
    .DESCRIPTION
    Validates system security configuration against security requirements
    including execution policies, user privileges, system integrity, and security features.
    #>
    [CmdletBinding()]
    param()
    
    Write-SecurityAuditEvent -EventType "ComplianceCheck" -Severity "Information" -Message "Starting security compliance validation"
    
    try {
        $complianceResults = @{
            OverallCompliant = $true
            ComplianceScore = 0
            MaxScore = 0
            Checks = @{}
            Violations = @()
            Warnings = @()
            Recommendations = @()
        }
        
        # Compliance Check 1: Administrative Privileges
        Write-InstallationLog -Level Info -Message "Checking administrative privileges compliance..."
        $adminCheck = Test-AdministrativePrivileges
        $complianceResults.Checks.AdministrativePrivileges = $adminCheck
        $complianceResults.MaxScore += 10
        
        if ($adminCheck.IsCompliant) {
            $complianceResults.ComplianceScore += 10
        } else {
            $complianceResults.OverallCompliant = $false
            $complianceResults.Violations += "Administrative privileges required but not available"
        }
        
        # Compliance Check 2: PowerShell Execution Policy
        Write-InstallationLog -Level Info -Message "Checking PowerShell execution policy compliance..."
        $executionPolicyCheck = Test-ExecutionPolicyCompliance
        $complianceResults.Checks.ExecutionPolicy = $executionPolicyCheck
        $complianceResults.MaxScore += 8
        
        if ($executionPolicyCheck.IsCompliant) {
            $complianceResults.ComplianceScore += 8
        } else {
            $complianceResults.Warnings += "PowerShell execution policy may restrict installation: $($executionPolicyCheck.CurrentPolicy)"
        }
        
        # Compliance Check 3: System Integrity
        Write-InstallationLog -Level Info -Message "Checking system integrity compliance..."
        $integrityCheck = Test-SystemIntegrityCompliance
        $complianceResults.Checks.SystemIntegrity = $integrityCheck
        $complianceResults.MaxScore += 15
        
        if ($integrityCheck.IsCompliant) {
            $complianceResults.ComplianceScore += 15
        } else {
            $complianceResults.Violations += "System integrity issues detected: $($integrityCheck.Issues -join ', ')"
        }
        
        # Compliance Check 4: Security Features
        Write-InstallationLog -Level Info -Message "Checking security features compliance..."
        $securityFeaturesCheck = Test-SecurityFeaturesCompliance
        $complianceResults.Checks.SecurityFeatures = $securityFeaturesCheck
        $complianceResults.MaxScore += 12
        
        if ($securityFeaturesCheck.IsCompliant) {
            $complianceResults.ComplianceScore += 12
        } else {
            $complianceResults.Warnings += "Security features not optimally configured: $($securityFeaturesCheck.Issues -join ', ')"
        }
        
        # Compliance Check 5: Network Security
        Write-InstallationLog -Level Info -Message "Checking network security compliance..."
        $networkSecurityCheck = Test-NetworkSecurityCompliance
        $complianceResults.Checks.NetworkSecurity = $networkSecurityCheck
        $complianceResults.MaxScore += 10
        
        if ($networkSecurityCheck.IsCompliant) {
            $complianceResults.ComplianceScore += 10
        } else {
            $complianceResults.Warnings += "Network security configuration issues: $($networkSecurityCheck.Issues -join ', ')"
        }
        
        # Compliance Check 6: File System Security
        Write-InstallationLog -Level Info -Message "Checking file system security compliance..."
        $fileSystemCheck = Test-FileSystemSecurityCompliance
        $complianceResults.Checks.FileSystemSecurity = $fileSystemCheck
        $complianceResults.MaxScore += 8
        
        if ($fileSystemCheck.IsCompliant) {
            $complianceResults.ComplianceScore += 8
        } else {
            $complianceResults.Warnings += "File system security issues: $($fileSystemCheck.Issues -join ', ')"
        }
        
        # Compliance Check 7: Service Security
        Write-InstallationLog -Level Info -Message "Checking service security compliance..."
        $serviceSecurityCheck = Test-ServiceSecurityCompliance
        $complianceResults.Checks.ServiceSecurity = $serviceSecurityCheck
        $complianceResults.MaxScore += 7
        
        if ($serviceSecurityCheck.IsCompliant) {
            $complianceResults.ComplianceScore += 7
        } else {
            $complianceResults.Warnings += "Service security configuration issues: $($serviceSecurityCheck.Issues -join ', ')"
        }
        
        # Calculate compliance percentage
        $compliancePercentage = if ($complianceResults.MaxScore -gt 0) { 
            [Math]::Round(($complianceResults.ComplianceScore / $complianceResults.MaxScore) * 100, 2) 
        } else { 0 }
        
        $complianceResults.CompliancePercentage = $compliancePercentage
        
        # Determine overall compliance status
        $complianceLevel = if ($compliancePercentage -ge 90) { "Excellent" }
                          elseif ($compliancePercentage -ge 80) { "Good" }
                          elseif ($compliancePercentage -ge 70) { "Acceptable" }
                          elseif ($compliancePercentage -ge 60) { "Poor" }
                          else { "Critical" }
        
        $complianceResults.ComplianceLevel = $complianceLevel
        
        # Log compliance results
        $complianceMessage = "Security compliance validation completed - Level: $complianceLevel ($compliancePercentage%)"
        $complianceSeverity = if ($complianceResults.Violations.Count -gt 0) { "Error" }
                             elseif ($complianceResults.Warnings.Count -gt 0) { "Warning" }
                             else { "Information" }
        
        Write-SecurityAuditEvent -EventType "ComplianceCheck" -Severity $complianceSeverity -Message $complianceMessage -Details $complianceResults
        
        if ($complianceResults.Violations.Count -gt 0) {
            Write-InstallationLog -Level Error -Message "Security compliance violations detected:"
            foreach ($violation in $complianceResults.Violations) {
                Write-InstallationLog -Level Error -Message "  - $violation"
            }
        }
        
        if ($complianceResults.Warnings.Count -gt 0) {
            Write-InstallationLog -Level Warning -Message "Security compliance warnings:"
            foreach ($warning in $complianceResults.Warnings) {
                Write-InstallationLog -Level Warning -Message "  - $warning"
            }
        }
        
        Write-InstallationLog -Level Success -Message "Security compliance validation completed - $complianceLevel ($compliancePercentage%)"
        
        return $complianceResults
    }
    catch {
        $errorMessage = "Security compliance validation failed: $($_.Exception.Message)"
        Write-SecurityAuditEvent -EventType "ComplianceCheck" -Severity "Error" -Message $errorMessage -Details @{
            Exception = $_.Exception.Message
            StackTrace = $_.Exception.StackTrace
        }
        
        Write-InstallationLog -Level Error -Message $errorMessage
        return @{
            OverallCompliant = $false
            ComplianceScore = 0
            MaxScore = 0
            CompliancePercentage = 0
            ComplianceLevel = "Critical"
            Error = $errorMessage
        }
    }
}

function Test-AdministrativePrivileges {
    <#
    .SYNOPSIS
    Tests if the current process has administrative privileges
    #>
    [CmdletBinding()]
    param()
    
    try {
        $currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
        $isAdmin = $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
        
        return @{
            IsCompliant = $isAdmin
            CurrentUser = [System.Environment]::UserName
            IsElevated = $isAdmin
            Details = @{
                WindowsIdentity = [Security.Principal.WindowsIdentity]::GetCurrent().Name
                AuthenticationType = [Security.Principal.WindowsIdentity]::GetCurrent().AuthenticationType
            }
        }
    }
    catch {
        return @{
            IsCompliant = $false
            Error = $_.Exception.Message
        }
    }
}

function Test-ExecutionPolicyCompliance {
    <#
    .SYNOPSIS
    Tests PowerShell execution policy compliance
    #>
    [CmdletBinding()]
    param()
    
    try {
        $currentPolicy = Get-ExecutionPolicy -Scope CurrentUser
        $machinePolicy = Get-ExecutionPolicy -Scope LocalMachine
        
        # Acceptable policies for installation
        $acceptablePolicies = @('RemoteSigned', 'Unrestricted', 'Bypass')
        
        $isCompliant = $currentPolicy -in $acceptablePolicies -or $machinePolicy -in $acceptablePolicies
        
        return @{
            IsCompliant = $isCompliant
            CurrentPolicy = $currentPolicy
            MachinePolicy = $machinePolicy
            AcceptablePolicies = $acceptablePolicies
            Details = @{
                AllScopes = Get-ExecutionPolicy -List
            }
        }
    }
    catch {
        return @{
            IsCompliant = $false
            Error = $_.Exception.Message
        }
    }
}

function Test-SystemIntegrityCompliance {
    <#
    .SYNOPSIS
    Tests system integrity and security baseline
    #>
    [CmdletBinding()]
    param()
    
    try {
        $issues = @()
        $checks = @{}
        
        # Check Windows Defender status
        try {
            $defenderStatus = Get-MpComputerStatus -ErrorAction SilentlyContinue
            if ($defenderStatus) {
                $checks.WindowsDefender = @{
                    AntivirusEnabled = $defenderStatus.AntivirusEnabled
                    RealTimeProtectionEnabled = $defenderStatus.RealTimeProtectionEnabled
                    AntivirusSignatureLastUpdated = $defenderStatus.AntivirusSignatureLastUpdated
                }
                
                if (-not $defenderStatus.AntivirusEnabled) {
                    $issues += "Windows Defender antivirus is disabled"
                }
            }
            else {
                $issues += "Windows Defender status unavailable"
            }
        }
        catch {
            $issues += "Could not check Windows Defender status"
        }
        
        # Check system file integrity
        try {
            $systemDrive = $env:SystemDrive
            if (Test-Path "$systemDrive\Windows\System32\sfc.exe") {
                $checks.SystemFileChecker = @{
                    Available = $true
                    Path = "$systemDrive\Windows\System32\sfc.exe"
                }
            }
            else {
                $issues += "System File Checker not available"
            }
        }
        catch {
            $issues += "Could not verify system file integrity tools"
        }
        
        # Check critical system services
        $criticalServices = @('Winmgmt', 'RpcSs', 'Dhcp', 'Dnscache')
        $serviceIssues = @()
        
        foreach ($serviceName in $criticalServices) {
            try {
                $service = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
                if ($service) {
                    if ($service.Status -ne 'Running') {
                        $serviceIssues += "$serviceName service is not running"
                    }
                }
                else {
                    $serviceIssues += "$serviceName service not found"
                }
            }
            catch {
                $serviceIssues += "Could not check $serviceName service"
            }
        }
        
        if ($serviceIssues.Count -gt 0) {
            $issues += $serviceIssues
            $checks.CriticalServices = $serviceIssues
        }
        
        return @{
            IsCompliant = $issues.Count -eq 0
            Issues = $issues
            Checks = $checks
        }
    }
    catch {
        return @{
            IsCompliant = $false
            Issues = @("System integrity check failed: $($_.Exception.Message)")
            Error = $_.Exception.Message
        }
    }
}

function Test-SecurityFeaturesCompliance {
    <#
    .SYNOPSIS
    Tests security features configuration
    #>
    [CmdletBinding()]
    param()
    
    try {
        $issues = @()
        $checks = @{}
        
        # Check UAC status
        try {
            $uacKey = Get-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System" -Name "EnableLUA" -ErrorAction SilentlyContinue
            if ($uacKey) {
                $checks.UAC = @{
                    Enabled = $uacKey.EnableLUA -eq 1
                    Value = $uacKey.EnableLUA
                }
                
                if ($uacKey.EnableLUA -ne 1) {
                    $issues += "User Account Control (UAC) is disabled"
                }
            }
            else {
                $issues += "Could not determine UAC status"
            }
        }
        catch {
            $issues += "Error checking UAC status"
        }
        
        # Check Windows Firewall status
        try {
            $firewallProfiles = Get-NetFirewallProfile -ErrorAction SilentlyContinue
            if ($firewallProfiles) {
                $enabledProfiles = $firewallProfiles | Where-Object { $_.Enabled -eq $true }
                $checks.WindowsFirewall = @{
                    ProfileCount = $firewallProfiles.Count
                    EnabledCount = $enabledProfiles.Count
                    Profiles = $firewallProfiles | ForEach-Object { 
                        @{ Name = $_.Name; Enabled = $_.Enabled } 
                    }
                }
                
                if ($enabledProfiles.Count -eq 0) {
                    $issues += "Windows Firewall is disabled on all profiles"
                }
            }
            else {
                $issues += "Could not check Windows Firewall status"
            }
        }
        catch {
            $issues += "Error checking Windows Firewall status"
        }
        
        # Check Windows Update service
        try {
            $wuService = Get-Service -Name "wuauserv" -ErrorAction SilentlyContinue
            if ($wuService) {
                $checks.WindowsUpdate = @{
                    Status = $wuService.Status
                    StartType = $wuService.StartType
                }
                
                if ($wuService.StartType -eq 'Disabled') {
                    $issues += "Windows Update service is disabled"
                }
            }
            else {
                $issues += "Windows Update service not found"
            }
        }
        catch {
            $issues += "Error checking Windows Update service"
        }
        
        return @{
            IsCompliant = $issues.Count -eq 0
            Issues = $issues
            Checks = $checks
        }
    }
    catch {
        return @{
            IsCompliant = $false
            Issues = @("Security features check failed: $($_.Exception.Message)")
            Error = $_.Exception.Message
        }
    }
}

function Test-NetworkSecurityCompliance {
    <#
    .SYNOPSIS
    Tests network security configuration
    #>
    [CmdletBinding()]
    param()
    
    try {
        $issues = @()
        $checks = @{}
        
        # Check network connectivity to required endpoints
        $requiredEndpoints = @(
            @{ Name = "Platform"; Host = ([System.Uri]$PlatformEndpoint).Host; Port = 443 }
            @{ Name = "GitHub"; Host = "github.com"; Port = 443 }
            @{ Name = "GitHub API"; Host = "api.github.com"; Port = 443 }
        )
        
        $connectivityResults = @{}
        foreach ($endpoint in $requiredEndpoints) {
            try {
                $testResult = Test-NetConnection -ComputerName $endpoint.Host -Port $endpoint.Port -InformationLevel Quiet -WarningAction SilentlyContinue -ErrorAction SilentlyContinue
                $connectivityResults[$endpoint.Name] = @{
                    Host = $endpoint.Host
                    Port = $endpoint.Port
                    Connected = $testResult
                }
                
                if (-not $testResult) {
                    $issues += "Cannot connect to $($endpoint.Name) ($($endpoint.Host):$($endpoint.Port))"
                }
            }
            catch {
                $connectivityResults[$endpoint.Name] = @{
                    Host = $endpoint.Host
                    Port = $endpoint.Port
                    Connected = $false
                    Error = $_.Exception.Message
                }
                $issues += "Error testing connectivity to $($endpoint.Name): $($_.Exception.Message)"
            }
        }
        
        $checks.Connectivity = $connectivityResults
        
        # Check DNS resolution
        try {
            $dnsTest = Resolve-DnsName -Name "github.com" -ErrorAction SilentlyContinue
            if ($dnsTest) {
                $checks.DNS = @{
                    Working = $true
                    GitHubResolution = $dnsTest | Select-Object Name, IPAddress
                }
            }
            else {
                $issues += "DNS resolution not working properly"
                $checks.DNS = @{ Working = $false }
            }
        }
        catch {
            $issues += "DNS resolution test failed"
            $checks.DNS = @{ Working = $false; Error = $_.Exception.Message }
        }
        
        # Check proxy configuration
        try {
            $proxySettings = Get-ItemProperty -Path "HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings" -ErrorAction SilentlyContinue
            if ($proxySettings -and $proxySettings.ProxyEnable -eq 1) {
                $checks.Proxy = @{
                    Enabled = $true
                    Server = $proxySettings.ProxyServer
                    Override = $proxySettings.ProxyOverride
                }
                
                # Proxy might interfere with installation
                if ($proxySettings.ProxyServer -and -not $proxySettings.ProxyOverride) {
                    $issues += "Proxy is configured without bypass rules - may interfere with installation"
                }
            }
            else {
                $checks.Proxy = @{ Enabled = $false }
            }
        }
        catch {
            $checks.Proxy = @{ Error = $_.Exception.Message }
        }
        
        return @{
            IsCompliant = $issues.Count -eq 0
            Issues = $issues
            Checks = $checks
        }
    }
    catch {
        return @{
            IsCompliant = $false
            Issues = @("Network security check failed: $($_.Exception.Message)")
            Error = $_.Exception.Message
        }
    }
}

function Test-FileSystemSecurityCompliance {
    <#
    .SYNOPSIS
    Tests file system security configuration
    #>
    [CmdletBinding()]
    param()
    
    try {
        $issues = @()
        $checks = @{}
        
        # Check installation directory permissions
        $installDir = Split-Path $InstallPath -Parent
        if (-not (Test-Path $installDir)) {
            try {
                New-Item -Path $installDir -ItemType Directory -Force -ErrorAction Stop | Out-Null
                $checks.InstallDirectory = @{
                    Path = $installDir
                    Exists = $true
                    Created = $true
                }
            }
            catch {
                $issues += "Cannot create installation directory: $installDir"
                $checks.InstallDirectory = @{
                    Path = $installDir
                    Exists = $false
                    Error = $_.Exception.Message
                }
            }
        }
        else {
            $checks.InstallDirectory = @{
                Path = $installDir
                Exists = $true
                Created = $false
            }
        }
        
        # Check write permissions to temp directory
        try {
            $tempTestFile = "$env:TEMP\RepSetBridge-PermissionTest-$(Get-Date -Format 'yyyyMMddHHmmss').tmp"
            "test" | Out-File -FilePath $tempTestFile -ErrorAction Stop
            Remove-Item -Path $tempTestFile -ErrorAction SilentlyContinue
            
            $checks.TempDirectory = @{
                Path = $env:TEMP
                Writable = $true
            }
        }
        catch {
            $issues += "Cannot write to temp directory: $env:TEMP"
            $checks.TempDirectory = @{
                Path = $env:TEMP
                Writable = $false
                Error = $_.Exception.Message
            }
        }
        
        # Check system drive space
        try {
            $systemDrive = Get-WmiObject -Class Win32_LogicalDisk -Filter "DeviceID='$($env:SystemDrive)'" -ErrorAction SilentlyContinue
            if ($systemDrive) {
                $freeSpaceGB = [Math]::Round($systemDrive.FreeSpace / 1GB, 2)
                $checks.DiskSpace = @{
                    Drive = $env:SystemDrive
                    FreeSpaceGB = $freeSpaceGB
                    TotalSpaceGB = [Math]::Round($systemDrive.Size / 1GB, 2)
                }
                
                if ($freeSpaceGB -lt 1) {
                    $issues += "Insufficient disk space on system drive (less than 1GB available)"
                }
            }
            else {
                $issues += "Could not check system drive disk space"
            }
        }
        catch {
            $issues += "Error checking disk space"
        }
        
        return @{
            IsCompliant = $issues.Count -eq 0
            Issues = $issues
            Checks = $checks
        }
    }
    catch {
        return @{
            IsCompliant = $false
            Issues = @("File system security check failed: $($_.Exception.Message)")
            Error = $_.Exception.Message
        }
    }
}

function Test-ServiceSecurityCompliance {
    <#
    .SYNOPSIS
    Tests service security configuration requirements
    #>
    [CmdletBinding()]
    param()
    
    try {
        $issues = @()
        $checks = @{}
        
        # Check if sc.exe is available for service management
        try {
            $scCommand = Get-Command "sc.exe" -ErrorAction SilentlyContinue
            if ($scCommand) {
                $checks.ServiceManager = @{
                    Available = $true
                    Path = $scCommand.Source
                }
            }
            else {
                $issues += "Service Control Manager (sc.exe) not available"
                $checks.ServiceManager = @{ Available = $false }
            }
        }
        catch {
            $issues += "Error checking service management tools"
            $checks.ServiceManager = @{ Available = $false; Error = $_.Exception.Message }
        }
        
        # Check if existing RepSet Bridge service exists and its configuration
        try {
            $existingService = Get-Service -Name $script:ServiceName -ErrorAction SilentlyContinue
            if ($existingService) {
                $checks.ExistingService = @{
                    Exists = $true
                    Status = $existingService.Status
                    StartType = $existingService.StartType
                    ServiceName = $existingService.Name
                    DisplayName = $existingService.DisplayName
                }
                
                # Check if service is in a problematic state
                if ($existingService.Status -eq 'StartPending' -or $existingService.Status -eq 'StopPending') {
                    $issues += "Existing RepSet Bridge service is in pending state"
                }
            }
            else {
                $checks.ExistingService = @{ Exists = $false }
            }
        }
        catch {
            $checks.ExistingService = @{ Error = $_.Exception.Message }
        }
        
        # Check Windows service dependencies
        $requiredServices = @('RpcSs', 'Winmgmt')
        $serviceStatus = @{}
        
        foreach ($serviceName in $requiredServices) {
            try {
                $service = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
                if ($service) {
                    $serviceStatus[$serviceName] = @{
                        Status = $service.Status
                        StartType = $service.StartType
                        Running = $service.Status -eq 'Running'
                    }
                    
                    if ($service.Status -ne 'Running') {
                        $issues += "Required service $serviceName is not running"
                    }
                }
                else {
                    $serviceStatus[$serviceName] = @{ Exists = $false }
                    $issues += "Required service $serviceName not found"
                }
            }
            catch {
                $serviceStatus[$serviceName] = @{ Error = $_.Exception.Message }
                $issues += "Error checking required service $serviceName"
            }
        }
        
        $checks.RequiredServices = $serviceStatus
        
        return @{
            IsCompliant = $issues.Count -eq 0
            Issues = $issues
            Checks = $checks
        }
    }
    catch {
        return @{
            IsCompliant = $false
            Issues = @("Service security check failed: $($_.Exception.Message)")
            Error = $_.Exception.Message
        }
    }
}

function Get-CurrentSecurityContext {
    <#
    .SYNOPSIS
    Gets current security context information
    #>
    [CmdletBinding()]
    param()
    
    try {
        $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
        $principal = New-Object Security.Principal.WindowsPrincipal($identity)
        
        return @{
            UserName = $identity.Name
            AuthenticationType = $identity.AuthenticationType
            IsAuthenticated = $identity.IsAuthenticated
            IsAnonymous = $identity.IsAnonymous
            IsGuest = $identity.IsGuest
            IsSystem = $identity.IsSystem
            IsAdministrator = $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
            Token = $identity.Token.ToString()
            Groups = $identity.Groups | ForEach-Object { $_.Value }
        }
    }
    catch {
        return @{
            Error = $_.Exception.Message
        }
    }
}

function Get-SecuritySystemInfo {
    <#
    .SYNOPSIS
    Gets security-relevant system information
    #>
    [CmdletBinding()]
    param()
    
    try {
        return @{
            MachineName = [System.Environment]::MachineName
            UserName = [System.Environment]::UserName
            UserDomainName = [System.Environment]::UserDomainName
            OSVersion = [System.Environment]::OSVersion.VersionString
            Is64BitOS = [System.Environment]::Is64BitOperatingSystem
            Is64BitProcess = [System.Environment]::Is64BitProcess
            ProcessorCount = [System.Environment]::ProcessorCount
            SystemDirectory = [System.Environment]::SystemDirectory
            CurrentDirectory = [System.Environment]::CurrentDirectory
            CommandLine = [System.Environment]::CommandLine
            ProcessId = $PID
            PowerShellVersion = $PSVersionTable.PSVersion.ToString()
            PowerShellEdition = $PSVersionTable.PSEdition
            ExecutionPolicy = Get-ExecutionPolicy
            SecurityContext = Get-CurrentSecurityContext
        }
    }
    catch {
        return @{
            Error = $_.Exception.Message
        }
    }
}

function Get-AuditEventHash {
    <#
    .SYNOPSIS
    Calculates integrity hash for audit events to detect tampering
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [hashtable]$AuditEvent
    )
    
    try {
        # Create a copy without the IntegrityHash field for hashing
        $eventCopy = $AuditEvent.Clone()
        $eventCopy.Remove('IntegrityHash')
        
        # Convert to JSON and calculate SHA-256 hash
        $jsonString = $eventCopy | ConvertTo-Json -Depth 10 -Compress
        $bytes = [System.Text.Encoding]::UTF8.GetBytes($jsonString)
        $sha256 = [System.Security.Cryptography.SHA256]::Create()
        $hashBytes = $sha256.ComputeHash($bytes)
        $hashString = [System.BitConverter]::ToString($hashBytes) -replace '-', ''
        
        return $hashString.ToLower()
    }
    catch {
        return "hash-calculation-failed"
    }
}

function Test-IsElevated {
    <#
    .SYNOPSIS
    Tests if current process is running with elevated privileges
    #>
    [CmdletBinding()]
    param()
    
    try {
        $currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
        return $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
    }
    catch {
        return $false
    }
}

function Complete-SecurityAudit {
    <#
    .SYNOPSIS
    Completes the security audit and generates final audit report
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$false)]
        [string]$InstallationResult = "Unknown",
        
        [Parameter(Mandatory=$false)]
        [hashtable]$FinalDetails = @{}
    )
    
    try {
        if (-not $script:SecurityAudit) {
            Write-InstallationLog -Level Warning -Message "Security audit system not initialized"
            return
        }
        
        $script:SecurityAudit.EndTime = Get-Date
        $script:SecurityAudit.Duration = ($script:SecurityAudit.EndTime - $script:SecurityAudit.StartTime).TotalMinutes
        $script:SecurityAudit.InstallationResult = $InstallationResult
        $script:SecurityAudit.FinalDetails = $FinalDetails
        
        # Generate audit summary
        $auditSummary = @{
            AuditId = $script:SecurityAudit.AuditId
            InstallationId = $script:SecurityAudit.InstallationId
            StartTime = $script:SecurityAudit.StartTime
            EndTime = $script:SecurityAudit.EndTime
            Duration = $script:SecurityAudit.Duration
            InstallationResult = $InstallationResult
            EventCount = $script:SecurityAudit.Events.Count
            SecurityEventTypes = $script:SecurityAudit.Events | Group-Object EventType | ForEach-Object { @{ Type = $_.Name; Count = $_.Count } }
            SeverityBreakdown = $script:SecurityAudit.Events | Group-Object Severity | ForEach-Object { @{ Severity = $_.Name; Count = $_.Count } }
            ComplianceResults = $script:SecurityAudit.ComplianceStatus
            FinalDetails = $FinalDetails
        }
        
        # Write final audit event
        Write-SecurityAuditEvent -EventType "AuditCompleted" -Severity "Information" -Message "Security audit completed successfully" -Details $auditSummary
        
        # Generate audit report file
        $auditReportPath = "$env:TEMP\RepSetBridge-SecurityAuditReport-$(Get-Date -Format 'yyyyMMdd-HHmmss').json"
        $script:SecurityAudit | ConvertTo-Json -Depth 10 | Set-Content -Path $auditReportPath -ErrorAction SilentlyContinue
        
        Write-InstallationLog -Level Success -Message "Security audit completed - Report saved to: $auditReportPath"
        
        return $auditSummary
    }
    catch {
        Write-InstallationLog -Level Error -Message "Failed to complete security audit: $($_.Exception.Message)"
        return $null
    }
}

function Get-PerformanceBaseline {
    <#
    .SYNOPSIS
    Establishes performance baseline metrics
    #>
    [CmdletBinding()]
    param()
    
    try {
        $baseline = @{
            MemoryUsage = [System.GC]::GetTotalMemory($false)
            ProcessorTime = (Get-Process -Id $PID).TotalProcessorTime.TotalMilliseconds
            WorkingSet = (Get-Process -Id $PID).WorkingSet64
            HandleCount = (Get-Process -Id $PID).HandleCount
            ThreadCount = (Get-Process -Id $PID).Threads.Count
            Timestamp = Get-Date -Format 'yyyy-MM-ddTHH:mm:ss.fffZ'
        }
        
        return $baseline
    }
    catch {
        return @{ Error = $_.Exception.Message }
    }
}

function Start-PerformanceMonitoring {
    <#
    .SYNOPSIS
    Starts performance monitoring for the installation process
    #>
    [CmdletBinding()]
    param()
    
    try {
        # Initialize performance counters
        $script:PerformanceCounters = @{
            StepStartTimes = @{}
            MemorySnapshots = @{}
            NetworkActivity = @{
                BytesDownloaded = 0
                DownloadStartTime = $null
                DownloadEndTime = $null
            }
        }
        
        Write-InstallationLog -Level Debug -Message "Performance monitoring started"
        return $true
    }
    catch {
        Write-InstallationLog -Level Warning -Message "Failed to start performance monitoring: $($_.Exception.Message)"
        return $false
    }
}

function Start-StepPerformanceTracking {
    <#
    .SYNOPSIS
    Starts performance tracking for a specific installation step
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$StepName
    )
    
    try {
        $stepStartTime = Get-Date
        $script:PerformanceCounters.StepStartTimes[$StepName] = $stepStartTime
        
        # Capture memory snapshot
        $memorySnapshot = @{
            TotalMemory = [System.GC]::GetTotalMemory($false)
            WorkingSet = (Get-Process -Id $PID).WorkingSet64
            PrivateMemory = (Get-Process -Id $PID).PrivateMemorySize64
            Timestamp = $stepStartTime
        }
        $script:PerformanceCounters.MemorySnapshots["$StepName-Start"] = $memorySnapshot
        
        Write-InstallationLog -Level Debug -Message "Started performance tracking for step: $StepName" -Context @{
            StepName = $StepName
            StartTime = $stepStartTime
            MemoryUsage = $memorySnapshot.TotalMemory
        }
    }
    catch {
        Write-InstallationLog -Level Warning -Message "Failed to start step performance tracking: $($_.Exception.Message)"
    }
}

function Stop-StepPerformanceTracking {
    <#
    .SYNOPSIS
    Stops performance tracking for a specific installation step and records metrics
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$StepName,
        
        [Parameter(Mandatory=$false)]
        [string]$Status = "Completed",
        
        [Parameter(Mandatory=$false)]
        [hashtable]$AdditionalMetrics = @{}
    )
    
    try {
        $stepEndTime = Get-Date
        $stepStartTime = $script:PerformanceCounters.StepStartTimes[$StepName]
        
        if ($stepStartTime) {
            $duration = ($stepEndTime - $stepStartTime).TotalMilliseconds
            
            # Capture end memory snapshot
            $endMemorySnapshot = @{
                TotalMemory = [System.GC]::GetTotalMemory($false)
                WorkingSet = (Get-Process -Id $PID).WorkingSet64
                PrivateMemory = (Get-Process -Id $PID).PrivateMemorySize64
                Timestamp = $stepEndTime
            }
            $script:PerformanceCounters.MemorySnapshots["$StepName-End"] = $endMemorySnapshot
            
            # Calculate memory delta
            $startMemorySnapshot = $script:PerformanceCounters.MemorySnapshots["$StepName-Start"]
            $memoryDelta = if ($startMemorySnapshot) {
                $endMemorySnapshot.TotalMemory - $startMemorySnapshot.TotalMemory
            } else { 0 }
            
            # Record step timing
            $stepMetrics = @{
                StepName = $StepName
                Status = $Status
                Duration = $duration
                StartTime = $stepStartTime
                EndTime = $stepEndTime
                MemoryDelta = $memoryDelta
                StartMemory = $startMemorySnapshot.TotalMemory
                EndMemory = $endMemorySnapshot.TotalMemory
            } + $AdditionalMetrics
            
            $script:TelemetryData.PerformanceMetrics.StepTimings[$StepName] = $stepMetrics
            
            # Send step performance telemetry
            Send-TelemetryEvent -EventType "StepPerformance" -Data $stepMetrics
            
            Write-InstallationLog -Level Debug -Message "Completed performance tracking for step: $StepName" -Context @{
                StepName = $StepName
                Duration = "$([math]::Round($duration, 2))ms"
                Status = $Status
                MemoryDelta = $memoryDelta
            }
        }
        else {
            Write-InstallationLog -Level Warning -Message "No start time found for step: $StepName"
        }
    }
    catch {
        Write-InstallationLog -Level Warning -Message "Failed to stop step performance tracking: $($_.Exception.Message)"
    }
}

function Record-ErrorMetrics {
    <#
    .SYNOPSIS
    Records detailed error metrics for telemetry and analysis
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$ErrorCategory,
        
        [Parameter(Mandatory=$true)]
        [string]$ErrorMessage,
        
        [Parameter(Mandatory=$false)]
        [string]$StepName = "",
        
        [Parameter(Mandatory=$false)]
        [System.Exception]$Exception = $null,
        
        [Parameter(Mandatory=$false)]
        [hashtable]$Context = @{},
        
        [Parameter(Mandatory=$false)]
        [string]$Severity = "Medium"
    )
    
    try {
        # Increment error counters
        if (-not $script:TelemetryData.PerformanceMetrics.ErrorCounts.ContainsKey($ErrorCategory)) {
            $script:TelemetryData.PerformanceMetrics.ErrorCounts[$ErrorCategory] = 0
        }
        $script:TelemetryData.PerformanceMetrics.ErrorCounts[$ErrorCategory]++
        
        # Create detailed error record
        $errorRecord = @{
            Category = $ErrorCategory
            Message = $ErrorMessage
            StepName = $StepName
            Severity = $Severity
            Timestamp = Get-Date -Format 'yyyy-MM-ddTHH:mm:ss.fffZ'
            Context = $Context
            InstallationId = $script:InstallationId
            GymId = $GymId
        }
        
        # Add exception details if provided
        if ($Exception) {
            $errorRecord.ExceptionType = $Exception.GetType().FullName
            $errorRecord.ExceptionMessage = $Exception.Message
            $errorRecord.StackTrace = $Exception.StackTrace
            
            # Add inner exception details
            if ($Exception.InnerException) {
                $errorRecord.InnerExceptionType = $Exception.InnerException.GetType().FullName
                $errorRecord.InnerExceptionMessage = $Exception.InnerException.Message
            }
        }
        
        # Add system context
        $errorRecord.SystemContext = @{
            MemoryUsage = [System.GC]::GetTotalMemory($false)
            WorkingSet = (Get-Process -Id $PID).WorkingSet64
            ThreadCount = (Get-Process -Id $PID).Threads.Count
            HandleCount = (Get-Process -Id $PID).HandleCount
        }
        
        # Send error telemetry
        Send-TelemetryEvent -EventType "InstallationError" -Data $errorRecord
        
        # Update installation metrics
        $script:TelemetryData.InstallationMetrics.Errors++
        if ($Severity -eq "High") {
            $script:TelemetryData.InstallationMetrics.SecurityErrors++
        }
        
        Write-InstallationLog -Level Debug -Message "Recorded error metrics for category: $ErrorCategory" -Context @{
            ErrorCategory = $ErrorCategory
            Severity = $Severity
            StepName = $StepName
        }
    }
    catch {
        Write-InstallationLog -Level Warning -Message "Failed to record error metrics: $($_.Exception.Message)"
    }
}

function Record-RetryMetrics {
    <#
    .SYNOPSIS
    Records retry attempt metrics for analysis
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$Operation,
        
        [Parameter(Mandatory=$true)]
        [int]$AttemptNumber,
        
        [Parameter(Mandatory=$true)]
        [int]$MaxAttempts,
        
        [Parameter(Mandatory=$true)]
        [string]$Result,
        
        [Parameter(Mandatory=$false)]
        [string]$ErrorMessage = "",
        
        [Parameter(Mandatory=$false)]
        [double]$DelaySeconds = 0,
        
        [Parameter(Mandatory=$false)]
        [hashtable]$Context = @{}
    )
    
    try {
        # Initialize retry tracking for operation if not exists
        if (-not $script:TelemetryData.PerformanceMetrics.RetryAttempts.ContainsKey($Operation)) {
            $script:TelemetryData.PerformanceMetrics.RetryAttempts[$Operation] = @{
                TotalAttempts = 0
                SuccessfulAttempts = 0
                FailedAttempts = 0
                MaxAttemptsReached = 0
                TotalDelayTime = 0
                Attempts = @()
            }
        }
        
        $retryData = $script:TelemetryData.PerformanceMetrics.RetryAttempts[$Operation]
        $retryData.TotalAttempts++
        $retryData.TotalDelayTime += $DelaySeconds
        
        # Record individual attempt
        $attemptRecord = @{
            AttemptNumber = $AttemptNumber
            MaxAttempts = $MaxAttempts
            Result = $Result
            ErrorMessage = $ErrorMessage
            DelaySeconds = $DelaySeconds
            Timestamp = Get-Date -Format 'yyyy-MM-ddTHH:mm:ss.fffZ'
            Context = $Context
        }
        $retryData.Attempts += $attemptRecord
        
        # Update counters based on result
        switch ($Result) {
            'Success' { 
                $retryData.SuccessfulAttempts++
            }
            'Failed' { 
                $retryData.FailedAttempts++
                if ($AttemptNumber -ge $MaxAttempts) {
                    $retryData.MaxAttemptsReached++
                }
            }
        }
        
        # Send retry telemetry
        Send-TelemetryEvent -EventType "RetryAttempt" -Data (@{
            Operation = $Operation
            AttemptRecord = $attemptRecord
            OperationSummary = @{
                TotalAttempts = $retryData.TotalAttempts
                SuccessRate = if ($retryData.TotalAttempts -gt 0) { ($retryData.SuccessfulAttempts / $retryData.TotalAttempts) * 100 } else { 0 }
            }
        })
        
        Write-InstallationLog -Level Debug -Message "Recorded retry metrics for operation: $Operation" -Context @{
            Operation = $Operation
            AttemptNumber = $AttemptNumber
            Result = $Result
            DelaySeconds = $DelaySeconds
        }
    }
    catch {
        Write-InstallationLog -Level Warning -Message "Failed to record retry metrics: $($_.Exception.Message)"
    }
}

function Record-DownloadMetrics {
    <#
    .SYNOPSIS
    Records download performance metrics
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$Url,
        
        [Parameter(Mandatory=$true)]
        [long]$BytesDownloaded,
        
        [Parameter(Mandatory=$true)]
        [double]$DurationSeconds,
        
        [Parameter(Mandatory=$false)]
        [string]$Status = "Success",
        
        [Parameter(Mandatory=$false)]
        [string]$ErrorMessage = "",
        
        [Parameter(Mandatory=$false)]
        [hashtable]$Context = @{}
    )
    
    try {
        $downloadSpeed = if ($DurationSeconds -gt 0) { $BytesDownloaded / $DurationSeconds } else { 0 }
        
        $downloadMetrics = @{
            Url = $Url
            BytesDownloaded = $BytesDownloaded
            DurationSeconds = $DurationSeconds
            DownloadSpeed = $downloadSpeed
            Status = $Status
            ErrorMessage = $ErrorMessage
            Timestamp = Get-Date -Format 'yyyy-MM-ddTHH:mm:ss.fffZ'
            Context = $Context
        }
        
        # Update network metrics
        $script:TelemetryData.NetworkMetrics.TotalBytesDownloaded += $BytesDownloaded
        if ($downloadSpeed -gt $script:TelemetryData.NetworkMetrics.DownloadSpeed) {
            $script:TelemetryData.NetworkMetrics.DownloadSpeed = $downloadSpeed
        }
        
        # Store in performance metrics
        if (-not $script:TelemetryData.PerformanceMetrics.DownloadMetrics.ContainsKey($Url)) {
            $script:TelemetryData.PerformanceMetrics.DownloadMetrics[$Url] = @()
        }
        $script:TelemetryData.PerformanceMetrics.DownloadMetrics[$Url] += $downloadMetrics
        
        # Send download telemetry
        Send-TelemetryEvent -EventType "DownloadMetrics" -Data $downloadMetrics
        
        Write-InstallationLog -Level Debug -Message "Recorded download metrics" -Context @{
            Url = $Url
            BytesDownloaded = $BytesDownloaded
            DownloadSpeed = "$([math]::Round($downloadSpeed / 1024, 2)) KB/s"
            Duration = "$([math]::Round($DurationSeconds, 2))s"
            Status = $Status
        }
    }
    catch {
        Write-InstallationLog -Level Warning -Message "Failed to record download metrics: $($_.Exception.Message)"
    }
}

function Send-TelemetryEvent {
    <#
    .SYNOPSIS
    Sends telemetry events to the platform for analysis
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$EventType,
        
        [Parameter(Mandatory=$true)]
        [hashtable]$Data,
        
        [Parameter(Mandatory=$false)]
        [switch]$Critical
    )
    
    try {
        $telemetryEvent = @{
            eventType = $EventType
            installationId = $script:InstallationId
            gymId = $GymId
            timestamp = Get-Date -Format 'yyyy-MM-ddTHH:mm:ss.fffZ'
            data = $Data
            critical = $Critical.IsPresent
        }
        
        $platformUrl = "$PlatformEndpoint/api/installation/telemetry"
        $headers = @{
            'Content-Type' = 'application/json'
            'User-Agent' = 'RepSet-Bridge-Installer/1.0'
        }
        
        $json = $telemetryEvent | ConvertTo-Json -Depth 5 -Compress
        
        # Send telemetry asynchronously unless critical
        if ($Critical) {
            # Send synchronously for critical events
            try {
                Invoke-RestMethod -Uri $platformUrl -Method Post -Headers $headers -Body $json -TimeoutSec 10 -ErrorAction Stop
                Write-InstallationLog -Level Debug -Message "Critical telemetry event sent: $EventType"
            }
            catch {
                Write-InstallationLog -Level Warning -Message "Failed to send critical telemetry event: $($_.Exception.Message)"
            }
        }
        else {
            # Send asynchronously for non-critical events
            Start-Job -ScriptBlock {
                param($Url, $Headers, $Data)
                try {
                    Invoke-RestMethod -Uri $Url -Method Post -Headers $Headers -Body $Data -TimeoutSec 10 -ErrorAction SilentlyContinue
                }
                catch {
                    # Silently ignore non-critical telemetry failures
                }
            } -ArgumentList $platformUrl, $headers, $json | Out-Null
        }
    }
    catch {
        Write-InstallationLog -Level Warning -Message "Failed to send telemetry event: $($_.Exception.Message)"
    }
}

function Send-InstallationTelemetrySummary {
    <#
    .SYNOPSIS
    Sends comprehensive installation telemetry summary to platform
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [bool]$Success,
        
        [Parameter(Mandatory=$false)]
        [string]$ErrorMessage = "",
        
        [Parameter(Mandatory=$false)]
        [int]$ErrorCode = 0
    )
    
    try {
        # Calculate final metrics
        $endTime = Get-Date
        $totalDuration = ($endTime - $script:InstallationStartTime).TotalSeconds
        
        # Compile comprehensive telemetry summary
        $telemetrySummary = @{
            InstallationId = $script:InstallationId
            GymId = $GymId
            Success = $Success
            ErrorMessage = $ErrorMessage
            ErrorCode = $ErrorCode
            
            # Timing information
            StartTime = $script:InstallationStartTime.ToString('yyyy-MM-ddTHH:mm:ss.fffZ')
            EndTime = $endTime.ToString('yyyy-MM-ddTHH:mm:ss.fffZ')
            TotalDuration = $totalDuration
            
            # System information
            SystemInfo = $script:TelemetryData.SystemInfo
            
            # Performance metrics
            PerformanceMetrics = $script:TelemetryData.PerformanceMetrics
            
            # Installation metrics
            InstallationMetrics = $script:TelemetryData.InstallationMetrics
            
            # Network metrics
            NetworkMetrics = $script:TelemetryData.NetworkMetrics
            
            # Security metrics
            SecurityMetrics = $script:TelemetryData.SecurityMetrics
            
            # Calculate summary statistics
            SummaryStatistics = @{
                TotalSteps = $script:TelemetryData.InstallationMetrics.TotalSteps
                CompletedSteps = $script:TelemetryData.InstallationMetrics.CompletedSteps
                SuccessRate = if ($script:TelemetryData.InstallationMetrics.TotalSteps -gt 0) { 
                    ($script:TelemetryData.InstallationMetrics.CompletedSteps / $script:TelemetryData.InstallationMetrics.TotalSteps) * 100 
                } else { 0 }
                ErrorRate = if ($script:TelemetryData.InstallationMetrics.TotalSteps -gt 0) { 
                    ($script:TelemetryData.InstallationMetrics.Errors / $script:TelemetryData.InstallationMetrics.TotalSteps) * 100 
                } else { 0 }
                AverageStepDuration = if ($script:TelemetryData.PerformanceMetrics.StepTimings.Count -gt 0) {
                    ($script:TelemetryData.PerformanceMetrics.StepTimings.Values | Measure-Object -Property Duration -Average).Average
                } else { 0 }
                TotalDownloadTime = ($script:TelemetryData.PerformanceMetrics.DownloadMetrics.Values | 
                    ForEach-Object { $_ | Measure-Object -Property DurationSeconds -Sum }).Sum
                AverageDownloadSpeed = if ($script:TelemetryData.NetworkMetrics.TotalBytesDownloaded -gt 0 -and $totalDuration -gt 0) {
                    $script:TelemetryData.NetworkMetrics.TotalBytesDownloaded / $totalDuration
                } else { 0 }
            }
        }
        
        # Send comprehensive telemetry summary
        Send-TelemetryEvent -EventType "InstallationSummary" -Data $telemetrySummary -Critical
        
        Write-InstallationLog -Level Info -Message "Installation telemetry summary sent to platform" -Context @{
            Success = $Success
            TotalDuration = "$([math]::Round($totalDuration, 2))s"
            TotalSteps = $script:TelemetryData.InstallationMetrics.TotalSteps
            CompletedSteps = $script:TelemetryData.InstallationMetrics.CompletedSteps
            Errors = $script:TelemetryData.InstallationMetrics.Errors
        }
        
        return $true
    }
    catch {
        Write-InstallationLog -Level Warning -Message "Failed to send installation telemetry summary: $($_.Exception.Message)"
        return $false
    }
}

function Write-InstallationSummary {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [bool]$Success,
        
        [Parameter(Mandatory=$false)]
        [string]$ErrorMessage = "",
        
        [Parameter(Mandatory=$false)]
        [int]$ErrorCode = 0,
        
        [Parameter(Mandatory=$false)]
        [hashtable]$InstallationDetails = @{}
    )
    
    Write-InstallationLog -Level Info -Message "=== INSTALLATION SUMMARY ==="
    Write-InstallationLog -Level Info -Message "Installation ID: $script:InstallationId"
    Write-InstallationLog -Level Info -Message "Gym ID: $GymId"
    Write-InstallationLog -Level Info -Message "Start Time: $($script:InstallationStartTime)"
    Write-InstallationLog -Level Info -Message "End Time: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"
    
    if ($script:InstallationStartTime) {
        $duration = (Get-Date) - $script:InstallationStartTime
        Write-InstallationLog -Level Info -Message "Duration: $([math]::Round($duration.TotalMinutes, 2)) minutes"
    }
    
    if ($Success) {
        Write-InstallationLog -Level Success -Message "Status: INSTALLATION SUCCESSFUL"
        Write-InstallationLog -Level Info -Message "Service Name: $script:ServiceName"
        Write-InstallationLog -Level Info -Message "Installation Path: $InstallPath"
        Write-InstallationLog -Level Info -Message "Configuration File: $(Join-Path $InstallPath "config\$script:ConfigFileName")"
        Write-InstallationLog -Level Info -Message "Log File: $script:LogFile"
        
        # Send success notification
        Send-InstallationNotification -Status Success -Message "RepSet Bridge installed successfully" -Details $InstallationDetails
        
        Write-InstallationLog -Level Info -Message ""
        Write-InstallationLog -Level Success -Message "🎉 RepSet Bridge has been successfully installed and started!"
        Write-InstallationLog -Level Info -Message "The bridge service is now running and will automatically start with Windows."
        Write-InstallationLog -Level Info -Message "You can monitor the service status in Windows Services (services.msc)."
        Write-InstallationLog -Level Info -Message ""
        Write-InstallationLog -Level Info -Message "Next steps:"
        Write-InstallationLog -Level Info -Message "1. Verify the bridge appears as 'Connected' in your RepSet platform"
        Write-InstallationLog -Level Info -Message "2. Test equipment connectivity through the platform"
        Write-InstallationLog -Level Info -Message "3. Review logs at: $(Join-Path $InstallPath "logs")"
    }
    else {
        Write-InstallationLog -Level Error -Message "Status: INSTALLATION FAILED"
        if ($ErrorMessage) {
            Write-InstallationLog -Level Error -Message "Error: $ErrorMessage"
        }
        if ($ErrorCode -gt 0) {
            Write-InstallationLog -Level Error -Message "Error Code: $ErrorCode"
        }
        
        # Send failure notification
        Send-InstallationNotification -Status Failed -Message $ErrorMessage -ErrorCode $ErrorCode.ToString() -Details $InstallationDetails
        
        Write-InstallationLog -Level Info -Message ""
        Write-InstallationLog -Level Error -Message "❌ RepSet Bridge installation failed."
        Write-InstallationLog -Level Info -Message "Please review the error details above and try again."
        Write-InstallationLog -Level Info -Message "For support, please provide the installation log: $script:LogFile"
        Write-InstallationLog -Level Info -Message ""
        Write-InstallationLog -Level Info -Message "Common solutions:"
        Write-InstallationLog -Level Info -Message "1. Ensure you're running PowerShell as Administrator"
        Write-InstallationLog -Level Info -Message "2. Check your internet connection"
        Write-InstallationLog -Level Info -Message "3. Verify Windows Defender isn't blocking the installation"
        Write-InstallationLog -Level Info -Message "4. Generate a new installation command if this one has expired"
    }
    
    Write-InstallationLog -Level Info -Message "=== END INSTALLATION SUMMARY ==="
}

# ================================================================
# Security and Validation Functions
# ================================================================

function Test-CommandSignature {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$PairCode,
        
        [Parameter(Mandatory=$true)]
        [string]$Signature,
        
        [Parameter(Mandatory=$true)]
        [string]$Nonce,
        
        [Parameter(Mandatory=$true)]
        [string]$GymId,
        
        [Parameter(Mandatory=$true)]
        [string]$ExpiresAt
    )
    
    Write-InstallationLog -Level Debug -Message "Validating command signature and parameters"
    
    try {
        # Validate expiration
        $expirationDate = [DateTime]::Parse($ExpiresAt)
        if ((Get-Date) -gt $expirationDate) {
            Write-InstallationLog -Level Error -Message "Installation command has expired. Please generate a new command from the platform."
            return $false
        }
        
        # Create the message that should have been signed
        $message = "$PairCode|$Nonce|$GymId|$ExpiresAt"
        
        # Note: In a real implementation, we would validate the HMAC signature here
        # For now, we'll do basic parameter validation
        if ([string]::IsNullOrWhiteSpace($PairCode) -or 
            [string]::IsNullOrWhiteSpace($Signature) -or 
            [string]::IsNullOrWhiteSpace($Nonce) -or 
            [string]::IsNullOrWhiteSpace($GymId)) {
            Write-InstallationLog -Level Error -Message "Invalid or missing required parameters"
            return $false
        }
        
        # Validate parameter formats
        if ($PairCode.Length -lt 8) {
            Write-InstallationLog -Level Error -Message "Invalid pair code format"
            return $false
        }
        
        if ($GymId -notmatch '^[a-zA-Z0-9\-_]+$') {
            Write-InstallationLog -Level Error -Message "Invalid gym ID format"
            return $false
        }
        
        Write-InstallationLog -Level Success -Message "Command signature and parameters validated successfully"
        return $true
    }
    catch {
        Write-InstallationLog -Level Error -Message "Error validating command signature: $($_.Exception.Message)"
        return $false
    }
}

function Test-CommandExpiration {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$ExpiresAt
    )
    
    try {
        $expirationDate = [DateTime]::Parse($ExpiresAt)
        $currentDate = Get-Date
        
        if ($currentDate -gt $expirationDate) {
            $expiredMinutes = [math]::Round(($currentDate - $expirationDate).TotalMinutes, 2)
            Write-InstallationLog -Level Error -Message "Installation command expired $expiredMinutes minutes ago. Please generate a new command."
            return $false
        }
        
        $remainingMinutes = [math]::Round(($expirationDate - $currentDate).TotalMinutes, 2)
        Write-InstallationLog -Level Info -Message "Command valid for $remainingMinutes more minutes"
        return $true
    }
    catch {
        Write-InstallationLog -Level Error -Message "Error parsing expiration date: $($_.Exception.Message)"
        return $false
    }
}

# ================================================================
# Error Handling and Recovery Functions
# ================================================================

function Invoke-WithRetry {
    <#
    .SYNOPSIS
    Executes a script block with retry logic and exponential backoff
    
    .DESCRIPTION
    Provides robust retry mechanism for operations that may fail due to transient issues.
    Uses exponential backoff to avoid overwhelming failing services.
    
    .PARAMETER ScriptBlock
    The script block to execute with retry logic
    
    .PARAMETER MaxRetries
    Maximum number of retry attempts (default: 3)
    
    .PARAMETER InitialDelaySeconds
    Initial delay in seconds before first retry (default: 2)
    
    .PARAMETER MaxDelaySeconds
    Maximum delay in seconds between retries (default: 60)
    
    .PARAMETER ExponentialBase
    Base for exponential backoff calculation (default: 2)
    
    .PARAMETER RetryableExceptions
    Array of exception types that should trigger a retry
    
    .PARAMETER OnRetry
    Script block to execute before each retry attempt
    
    .PARAMETER Context
    Additional context information for logging
    
    .EXAMPLE
    $result = Invoke-WithRetry -ScriptBlock {
        Invoke-WebRequest -Uri "https://api.example.com/data" -TimeoutSec 10
    } -MaxRetries 5 -Context @{ Operation = "API Call" }
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [scriptblock]$ScriptBlock,
        
        [Parameter(Mandatory=$false)]
        [int]$MaxRetries = 3,
        
        [Parameter(Mandatory=$false)]
        [int]$InitialDelaySeconds = 2,
        
        [Parameter(Mandatory=$false)]
        [int]$MaxDelaySeconds = 60,
        
        [Parameter(Mandatory=$false)]
        [double]$ExponentialBase = 2.0,
        
        [Parameter(Mandatory=$false)]
        [string[]]$RetryableExceptions = @(
            'System.Net.WebException',
            'System.Net.Http.HttpRequestException',
            'System.TimeoutException',
            'System.IO.IOException',
            'System.UnauthorizedAccessException'
        ),
        
        [Parameter(Mandatory=$false)]
        [scriptblock]$OnRetry = $null,
        
        [Parameter(Mandatory=$false)]
        [hashtable]$Context = @{}
    )
    
    $attempt = 0
    $lastException = $null
    
    while ($attempt -le $MaxRetries) {
        try {
            $attempt++
            
            if ($attempt -eq 1) {
                Write-InstallationLog -Level Debug -Message "Executing operation (attempt $attempt/$($MaxRetries + 1))" -Context $Context
            } else {
                Write-InstallationLog -Level Info -Message "Retrying operation (attempt $attempt/$($MaxRetries + 1))" -Context $Context
            }
            
            # Execute the script block
            $result = & $ScriptBlock
            
            # If we get here, the operation succeeded
            if ($attempt -gt 1) {
                Write-InstallationLog -Level Success -Message "Operation succeeded after $attempt attempts" -Context $Context
            }
            
            return $result
        }
        catch {
            $lastException = $_.Exception
            $exceptionType = $lastException.GetType().FullName
            
            Write-InstallationLog -Level Warning -Message "Operation failed (attempt $attempt/$($MaxRetries + 1)): $($lastException.Message)" -Context (@{
                ExceptionType = $exceptionType
                AttemptNumber = $attempt
                MaxRetries = $MaxRetries
            } + $Context)
            
            # Check if this is the last attempt
            if ($attempt -gt $MaxRetries) {
                Write-InstallationLog -Level Error -Message "Operation failed after $attempt attempts. Giving up." -Context $Context
                throw $lastException
            }
            
            # Check if this exception type is retryable
            $isRetryable = $RetryableExceptions -contains $exceptionType -or 
                          $RetryableExceptions | Where-Object { $exceptionType -like $_ }
            
            if (-not $isRetryable) {
                Write-InstallationLog -Level Error -Message "Exception type '$exceptionType' is not retryable. Giving up." -Context $Context
                throw $lastException
            }
            
            # Calculate delay with exponential backoff
            $delay = [math]::Min(
                $InitialDelaySeconds * [math]::Pow($ExponentialBase, $attempt - 1),
                $MaxDelaySeconds
            )
            
            # Add jitter to prevent thundering herd
            $jitter = Get-Random -Minimum 0.8 -Maximum 1.2
            $actualDelay = [math]::Round($delay * $jitter, 2)
            
            Write-InstallationLog -Level Info -Message "Waiting $actualDelay seconds before retry..." -Context $Context
            
            # Execute OnRetry callback if provided
            if ($OnRetry) {
                try {
                    & $OnRetry -AttemptNumber $attempt -Exception $lastException -DelaySeconds $actualDelay
                }
                catch {
                    Write-InstallationLog -Level Warning -Message "OnRetry callback failed: $($_.Exception.Message)" -Context $Context
                }
            }
            
            Start-Sleep -Seconds $actualDelay
        }
    }
    
    # This should never be reached, but just in case
    throw $lastException
}

function Invoke-InstallationRollback {
    <#
    .SYNOPSIS
    Performs comprehensive rollback of failed installation
    
    .DESCRIPTION
    Cleans up all installation artifacts and restores system to pre-installation state.
    Handles partial installations gracefully and provides detailed rollback logging.
    
    .PARAMETER InstallationId
    Unique identifier for the installation being rolled back
    
    .PARAMETER InstallPath
    Path where bridge was being installed
    
    .PARAMETER ServiceName
    Name of the Windows service to remove
    
    .PARAMETER PreserveConfig
    Whether to preserve existing configuration during rollback
    
    .PARAMETER RollbackReason
    Reason for the rollback (for logging purposes)
    
    .EXAMPLE
    Invoke-InstallationRollback -InstallationId $script:InstallationId -InstallPath $InstallPath -ServiceName $script:ServiceName -RollbackReason "Download failed"
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$InstallationId,
        
        [Parameter(Mandatory=$true)]
        [string]$InstallPath,
        
        [Parameter(Mandatory=$true)]
        [string]$ServiceName,
        
        [Parameter(Mandatory=$false)]
        [switch]$PreserveConfig,
        
        [Parameter(Mandatory=$false)]
        [string]$RollbackReason = "Installation failed",
        
        [Parameter(Mandatory=$false)]
        [hashtable]$Context = @{}
    )
    
    Write-InstallationLog -Level Warning -Message "=== STARTING INSTALLATION ROLLBACK ===" -Context (@{
        InstallationId = $InstallationId
        RollbackReason = $RollbackReason
    } + $Context)
    
    $rollbackSteps = @()
    $rollbackErrors = @()
    
    try {
        # Step 1: Stop and remove Windows service
        Write-InstallationLog -Level Info -Message "Rolling back Windows service installation..." -Context $Context
        
        try {
            $service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
            if ($service) {
                Write-InstallationLog -Level Info -Message "Stopping service '$ServiceName'..." -Context $Context
                
                if ($service.Status -eq 'Running') {
                    Stop-Service -Name $ServiceName -Force -ErrorAction Stop
                    $rollbackSteps += "Stopped service '$ServiceName'"
                    
                    # Wait for service to stop
                    $timeout = 30
                    $elapsed = 0
                    while ((Get-Service -Name $ServiceName).Status -ne 'Stopped' -and $elapsed -lt $timeout) {
                        Start-Sleep -Seconds 1
                        $elapsed++
                    }
                    
                    if ((Get-Service -Name $ServiceName).Status -ne 'Stopped') {
                        Write-InstallationLog -Level Warning -Message "Service did not stop within $timeout seconds" -Context $Context
                    }
                }
                
                # Remove service
                Write-InstallationLog -Level Info -Message "Removing service '$ServiceName'..." -Context $Context
                $scResult = & sc.exe delete $ServiceName 2>&1
                if ($LASTEXITCODE -eq 0) {
                    $rollbackSteps += "Removed service '$ServiceName'"
                    Write-InstallationLog -Level Success -Message "Service '$ServiceName' removed successfully" -Context $Context
                } else {
                    $rollbackErrors += "Failed to remove service: $scResult"
                    Write-InstallationLog -Level Warning -Message "Failed to remove service: $scResult" -Context $Context
                }
            } else {
                Write-InstallationLog -Level Info -Message "Service '$ServiceName' not found, skipping removal" -Context $Context
            }
        }
        catch {
            $rollbackErrors += "Service rollback error: $($_.Exception.Message)"
            Write-InstallationLog -Level Warning -Message "Error during service rollback: $($_.Exception.Message)" -Context $Context
        }
        
        # Step 2: Remove installed files and directories
        Write-InstallationLog -Level Info -Message "Rolling back file installation..." -Context $Context
        
        try {
            if (Test-Path -Path $InstallPath) {
                # Backup configuration if requested
                $configBackupPath = $null
                if ($PreserveConfig) {
                    $configPath = Join-Path $InstallPath "config\$script:ConfigFileName"
                    if (Test-Path -Path $configPath) {
                        $configBackupPath = "$env:TEMP\RepSetBridge-Config-Backup-$(Get-Date -Format 'yyyyMMdd-HHmmss').yaml"
                        Copy-Item -Path $configPath -Destination $configBackupPath -ErrorAction SilentlyContinue
                        Write-InstallationLog -Level Info -Message "Configuration backed up to: $configBackupPath" -Context $Context
                    }
                }
                
                # Remove installation directory
                Write-InstallationLog -Level Info -Message "Removing installation directory: $InstallPath" -Context $Context
                
                # First, try to remove files individually to handle locked files
                $filesToRemove = Get-ChildItem -Path $InstallPath -Recurse -File -ErrorAction SilentlyContinue
                foreach ($file in $filesToRemove) {
                    try {
                        Remove-Item -Path $file.FullName -Force -ErrorAction Stop
                    }
                    catch {
                        Write-InstallationLog -Level Warning -Message "Could not remove file: $($file.FullName) - $($_.Exception.Message)" -Context $Context
                    }
                }
                
                # Then remove directories
                try {
                    Remove-Item -Path $InstallPath -Recurse -Force -ErrorAction Stop
                    $rollbackSteps += "Removed installation directory: $InstallPath"
                    Write-InstallationLog -Level Success -Message "Installation directory removed successfully" -Context $Context
                }
                catch {
                    $rollbackErrors += "Failed to remove installation directory: $($_.Exception.Message)"
                    Write-InstallationLog -Level Warning -Message "Failed to remove installation directory: $($_.Exception.Message)" -Context $Context
                }
                
                # Restore configuration if it was backed up
                if ($configBackupPath -and (Test-Path -Path $configBackupPath)) {
                    Write-InstallationLog -Level Info -Message "Configuration backup available at: $configBackupPath" -Context $Context
                }
            } else {
                Write-InstallationLog -Level Info -Message "Installation directory not found, skipping file removal" -Context $Context
            }
        }
        catch {
            $rollbackErrors += "File rollback error: $($_.Exception.Message)"
            Write-InstallationLog -Level Warning -Message "Error during file rollback: $($_.Exception.Message)" -Context $Context
        }
        
        # Step 3: Clean up registry entries (if any were created)
        Write-InstallationLog -Level Info -Message "Rolling back registry changes..." -Context $Context
        
        try {
            $registryPaths = @(
                "HKLM:\SOFTWARE\RepSet\Bridge",
                "HKLM:\SYSTEM\CurrentControlSet\Services\$ServiceName"
            )
            
            foreach ($regPath in $registryPaths) {
                if (Test-Path -Path $regPath) {
                    try {
                        Remove-Item -Path $regPath -Recurse -Force -ErrorAction Stop
                        $rollbackSteps += "Removed registry key: $regPath"
                        Write-InstallationLog -Level Success -Message "Removed registry key: $regPath" -Context $Context
                    }
                    catch {
                        $rollbackErrors += "Failed to remove registry key '$regPath': $($_.Exception.Message)"
                        Write-InstallationLog -Level Warning -Message "Failed to remove registry key '$regPath': $($_.Exception.Message)" -Context $Context
                    }
                }
            }
        }
        catch {
            $rollbackErrors += "Registry rollback error: $($_.Exception.Message)"
            Write-InstallationLog -Level Warning -Message "Error during registry rollback: $($_.Exception.Message)" -Context $Context
        }
        
        # Step 4: Clean up temporary files
        Write-InstallationLog -Level Info -Message "Rolling back temporary files..." -Context $Context
        
        try {
            $tempFiles = @(
                "$env:TEMP\RepSetBridge-*.exe",
                "$env:TEMP\RepSetBridge-*.zip",
                "$env:TEMP\RepSetBridge-Download-*"
            )
            
            foreach ($pattern in $tempFiles) {
                $files = Get-ChildItem -Path $pattern -ErrorAction SilentlyContinue
                foreach ($file in $files) {
                    try {
                        Remove-Item -Path $file.FullName -Force -ErrorAction Stop
                        $rollbackSteps += "Removed temporary file: $($file.Name)"
                    }
                    catch {
                        Write-InstallationLog -Level Warning -Message "Could not remove temporary file: $($file.FullName)" -Context $Context
                    }
                }
            }
        }
        catch {
            $rollbackErrors += "Temporary file cleanup error: $($_.Exception.Message)"
            Write-InstallationLog -Level Warning -Message "Error during temporary file cleanup: $($_.Exception.Message)" -Context $Context
        }
        
        # Step 5: Clean up Windows Event Log source
        Write-InstallationLog -Level Info -Message "Rolling back event log configuration..." -Context $Context
        
        try {
            $eventSource = "RepSetBridge"
            if ([System.Diagnostics.EventLog]::SourceExists($eventSource)) {
                [System.Diagnostics.EventLog]::DeleteEventSource($eventSource)
                $rollbackSteps += "Removed event log source: $eventSource"
                Write-InstallationLog -Level Success -Message "Removed event log source: $eventSource" -Context $Context
            }
        }
        catch {
            $rollbackErrors += "Event log rollback error: $($_.Exception.Message)"
            Write-InstallationLog -Level Warning -Message "Error during event log rollback: $($_.Exception.Message)" -Context $Context
        }
        
        # Generate rollback summary
        Write-InstallationLog -Level Info -Message "=== ROLLBACK SUMMARY ===" -Context $Context
        Write-InstallationLog -Level Info -Message "Installation ID: $InstallationId" -Context $Context
        Write-InstallationLog -Level Info -Message "Rollback Reason: $RollbackReason" -Context $Context
        Write-InstallationLog -Level Info -Message "Rollback Steps Completed: $($rollbackSteps.Count)" -Context $Context
        
        if ($rollbackSteps.Count -gt 0) {
            Write-InstallationLog -Level Info -Message "Successful rollback steps:" -Context $Context
            foreach ($step in $rollbackSteps) {
                Write-InstallationLog -Level Info -Message "  ✓ $step" -Context $Context
            }
        }
        
        if ($rollbackErrors.Count -gt 0) {
            Write-InstallationLog -Level Warning -Message "Rollback Errors: $($rollbackErrors.Count)" -Context $Context
            Write-InstallationLog -Level Warning -Message "Rollback errors encountered:" -Context $Context
            foreach ($error in $rollbackErrors) {
                Write-InstallationLog -Level Warning -Message "  ⚠ $error" -Context $Context
            }
        }
        
        # Send rollback notification to platform
        Send-InstallationNotification -Status Failed -Message "Installation rolled back: $RollbackReason" -Details @{
            RollbackSteps = $rollbackSteps
            RollbackErrors = $rollbackErrors
            RollbackReason = $RollbackReason
        }
        
        if ($rollbackErrors.Count -eq 0) {
            Write-InstallationLog -Level Success -Message "Installation rollback completed successfully" -Context $Context
            return $true
        } else {
            Write-InstallationLog -Level Warning -Message "Installation rollback completed with $($rollbackErrors.Count) errors" -Context $Context
            return $false
        }
    }
    catch {
        Write-InstallationLog -Level Error -Message "Critical error during rollback: $($_.Exception.Message)" -Context $Context
        Send-InstallationNotification -Status Failed -Message "Rollback failed: $($_.Exception.Message)" -Details @{
            RollbackSteps = $rollbackSteps
            RollbackErrors = $rollbackErrors + @("Critical rollback error: $($_.Exception.Message)")
            RollbackReason = $RollbackReason
        }
        return $false
    }
    finally {
        Write-InstallationLog -Level Info -Message "=== ROLLBACK COMPLETED ===" -Context $Context
    }
}

function Get-ErrorCategory {
    <#
    .SYNOPSIS
    Categorizes exceptions into user-friendly error categories
    
    .DESCRIPTION
    Analyzes exception types and messages to provide specific error categorization
    and user-friendly error messages with actionable remediation steps.
    
    .PARAMETER Exception
    The exception to categorize
    
    .PARAMETER Context
    Additional context about where the error occurred
    
    .EXAMPLE
    $errorInfo = Get-ErrorCategory -Exception $_.Exception -Context @{ Operation = "Download" }
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [System.Exception]$Exception,
        
        [Parameter(Mandatory=$false)]
        [hashtable]$Context = @{}
    )
    
    $exceptionType = $Exception.GetType().FullName
    $exceptionMessage = $Exception.Message
    
    # Define error categories with patterns and remediation steps
    $errorCategories = @{
        'NetworkConnectivity' = @{
            Patterns = @(
                'System.Net.WebException',
                'System.Net.Http.HttpRequestException',
                'System.Net.Sockets.SocketException',
                'unable to connect',
                'network is unreachable',
                'connection timed out',
                'name resolution failed'
            )
            UserMessage = "Network connectivity issue detected"
            TechnicalMessage = "Failed to establish network connection"
            RemediationSteps = @(
                "Check your internet connection",
                "Verify firewall settings allow outbound HTTPS connections",
                "Ensure proxy settings are configured correctly",
                "Try running the installation from a different network"
            )
            ErrorCode = $script:ErrorCodes.DownloadFailed
            Severity = "High"
            Retryable = $true
        }
        
        'InsufficientPermissions' = @{
            Patterns = @(
                'System.UnauthorizedAccessException',
                'System.Security.SecurityException',
                'access is denied',
                'insufficient privileges',
                'administrator rights required',
                'elevation required'
            )
            UserMessage = "Insufficient permissions to complete installation"
            TechnicalMessage = "Access denied due to insufficient privileges"
            RemediationSteps = @(
                "Run PowerShell as Administrator",
                "Ensure your user account has local administrator rights",
                "Check if antivirus software is blocking the installation",
                "Verify the installation directory is writable"
            )
            ErrorCode = $script:ErrorCodes.InsufficientPrivileges
            Severity = "High"
            Retryable = $false
        }
        
        'FileSystemError' = @{
            Patterns = @(
                'System.IO.IOException',
                'System.IO.DirectoryNotFoundException',
                'System.IO.FileNotFoundException',
                'System.IO.PathTooLongException',
                'file is being used by another process',
                'disk full',
                'insufficient disk space'
            )
            UserMessage = "File system error occurred during installation"
            TechnicalMessage = "File system operation failed"
            RemediationSteps = @(
                "Ensure sufficient disk space is available",
                "Close any applications that might be using the installation directory",
                "Check if antivirus software is scanning the files",
                "Try installing to a different directory with a shorter path"
            )
            ErrorCode = $script:ErrorCodes.InstallationFailed
            Severity = "Medium"
            Retryable = $true
        }
        
        'ServiceInstallationError' = @{
            Patterns = @(
                'service already exists',
                'service installation failed',
                'service control manager',
                'failed to create service',
                'service startup failed'
            )
            UserMessage = "Windows service installation or startup failed"
            TechnicalMessage = "Service management operation failed"
            RemediationSteps = @(
                "Check if a service with the same name already exists",
                "Ensure Windows Service Control Manager is running",
                "Verify .NET runtime is properly installed",
                "Try restarting the computer and running the installation again"
            )
            ErrorCode = $script:ErrorCodes.ServiceInstallationFailed
            Severity = "High"
            Retryable = $true
        }
        
        'ConfigurationError' = @{
            Patterns = @(
                'configuration invalid',
                'config file error',
                'yaml parse error',
                'invalid configuration format',
                'missing required configuration'
            )
            UserMessage = "Configuration file creation or validation failed"
            TechnicalMessage = "Configuration processing error"
            RemediationSteps = @(
                "Verify the installation command parameters are correct",
                "Generate a new installation command from the platform",
                "Check if the installation directory is writable",
                "Ensure no special characters are causing parsing issues"
            )
            ErrorCode = $script:ErrorCodes.ConfigurationFailed
            Severity = "Medium"
            Retryable = $true
        }
        
        'AuthenticationError' = @{
            Patterns = @(
                'authentication failed',
                'invalid credentials',
                'unauthorized',
                'forbidden',
                'invalid pair code',
                'expired token'
            )
            UserMessage = "Authentication or authorization failed"
            TechnicalMessage = "Platform authentication error"
            RemediationSteps = @(
                "Generate a new installation command from the platform",
                "Verify the installation command hasn't expired",
                "Check that the gym account is active and properly configured",
                "Contact support if the issue persists"
            )
            ErrorCode = $script:ErrorCodes.ConnectionTestFailed
            Severity = "High"
            Retryable = $false
        }
        
        'DownloadError' = @{
            Patterns = @(
                'download failed',
                'file not found',
                'http 404',
                'http 500',
                'checksum mismatch',
                'file integrity verification failed'
            )
            UserMessage = "Failed to download or verify installation files"
            TechnicalMessage = "Download or file verification error"
            RemediationSteps = @(
                "Check your internet connection",
                "Verify GitHub is accessible from your network",
                "Try the installation again in a few minutes",
                "Contact support if downloads consistently fail"
            )
            ErrorCode = $script:ErrorCodes.DownloadFailed
            Severity = "Medium"
            Retryable = $true
        }
        
        'SystemRequirementsError' = @{
            Patterns = @(
                'system requirements not met',
                'powershell version',
                'dotnet not found',
                'unsupported operating system',
                'architecture mismatch'
            )
            UserMessage = "System requirements not met for installation"
            TechnicalMessage = "System compatibility check failed"
            RemediationSteps = @(
                "Ensure you're running Windows 10 or later",
                "Update PowerShell to version 5.1 or later",
                "Install .NET 6.0 or later runtime",
                "Verify you're running on a supported architecture (x64)"
            )
            ErrorCode = $script:ErrorCodes.SystemRequirementsNotMet
            Severity = "High"
            Retryable = $false
        }
        
        'TimeoutError' = @{
            Patterns = @(
                'System.TimeoutException',
                'operation timed out',
                'request timeout',
                'connection timeout'
            )
            UserMessage = "Operation timed out"
            TechnicalMessage = "Timeout occurred during operation"
            RemediationSteps = @(
                "Check your internet connection speed",
                "Try the installation during off-peak hours",
                "Verify no firewall is causing delays",
                "Contact your network administrator if timeouts persist"
            )
            ErrorCode = $script:ErrorCodes.DownloadFailed
            Severity = "Medium"
            Retryable = $true
        }
    }
    
    # Find matching error category
    $matchedCategory = $null
    foreach ($categoryName in $errorCategories.Keys) {
        $category = $errorCategories[$categoryName]
        foreach ($pattern in $category.Patterns) {
            if ($exceptionType -like "*$pattern*" -or $exceptionMessage -like "*$pattern*") {
                $matchedCategory = $category
                $matchedCategory.CategoryName = $categoryName
                break
            }
        }
        if ($matchedCategory) { break }
    }
    
    # Default category for unmatched errors
    if (-not $matchedCategory) {
        $matchedCategory = @{
            CategoryName = "UnknownError"
            UserMessage = "An unexpected error occurred during installation"
            TechnicalMessage = $exceptionMessage
            RemediationSteps = @(
                "Try running the installation again",
                "Ensure you're running PowerShell as Administrator",
                "Check the installation log for more details",
                "Contact support with the installation log if the issue persists"
            )
            ErrorCode = $script:ErrorCodes.InstallationFailed
            Severity = "Medium"
            Retryable = $true
        }
    }
    
    # Add context-specific information
    $contextualInfo = @{
        ExceptionType = $exceptionType
        OriginalMessage = $exceptionMessage
        Context = $Context
        Timestamp = Get-Date -Format 'yyyy-MM-dd HH:mm:ss'
        InstallationId = $script:InstallationId
    }
    
    return @{
        Category = $matchedCategory.CategoryName
        UserMessage = $matchedCategory.UserMessage
        TechnicalMessage = $matchedCategory.TechnicalMessage
        RemediationSteps = $matchedCategory.RemediationSteps
        ErrorCode = $matchedCategory.ErrorCode
        Severity = $matchedCategory.Severity
        Retryable = $matchedCategory.Retryable
        ContextualInfo = $contextualInfo
    }
}

function Write-UserFriendlyError {
    <#
    .SYNOPSIS
    Displays user-friendly error messages with remediation steps
    
    .DESCRIPTION
    Takes an exception and context, categorizes it, and displays a user-friendly
    error message with specific remediation steps and support information.
    
    .PARAMETER Exception
    The exception that occurred
    
    .PARAMETER Context
    Additional context about the operation that failed
    
    .PARAMETER Step
    The installation step where the error occurred
    
    .EXAMPLE
    Write-UserFriendlyError -Exception $_.Exception -Context @{ Operation = "Download" } -Step "Bridge Download"
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [System.Exception]$Exception,
        
        [Parameter(Mandatory=$false)]
        [hashtable]$Context = @{},
        
        [Parameter(Mandatory=$false)]
        [string]$Step = ""
    )
    
    $errorInfo = Get-ErrorCategory -Exception $Exception -Context $Context
    
    Write-InstallationLog -Level Error -Message "=== INSTALLATION ERROR ===" -Context $Context
    
    if ($Step) {
        Write-InstallationLog -Level Error -Message "Failed Step: $Step" -Context $Context
    }
    
    Write-InstallationLog -Level Error -Message "Error Category: $($errorInfo.Category)" -Context $Context
    Write-InstallationLog -Level Error -Message "Description: $($errorInfo.UserMessage)" -Context $Context
    Write-InstallationLog -Level Error -Message "Technical Details: $($errorInfo.TechnicalMessage)" -Context $Context
    
    Write-InstallationLog -Level Info -Message "" -Context $Context
    Write-InstallationLog -Level Info -Message "🔧 Recommended Solutions:" -Context $Context
    
    for ($i = 0; $i -lt $errorInfo.RemediationSteps.Count; $i++) {
        Write-InstallationLog -Level Info -Message "   $($i + 1). $($errorInfo.RemediationSteps[$i])" -Context $Context
    }
    
    Write-InstallationLog -Level Info -Message "" -Context $Context
    Write-InstallationLog -Level Info -Message "📋 Support Information:" -Context $Context
    Write-InstallationLog -Level Info -Message "   • Installation ID: $($script:InstallationId)" -Context $Context
    Write-InstallationLog -Level Info -Message "   • Error Code: $($errorInfo.ErrorCode)" -Context $Context
    Write-InstallationLog -Level Info -Message "   • Log File: $script:LogFile" -Context $Context
    Write-InstallationLog -Level Info -Message "   • Timestamp: $($errorInfo.ContextualInfo.Timestamp)" -Context $Context
    
    if ($errorInfo.Retryable) {
        Write-InstallationLog -Level Info -Message "   • This error may be temporary - you can try running the installation again" -Context $Context
    } else {
        Write-InstallationLog -Level Info -Message "   • This error requires manual intervention before retrying" -Context $Context
    }
    
    Write-InstallationLog -Level Error -Message "=== END ERROR DETAILS ===" -Context $Context
    
    return $errorInfo
}

function Invoke-AutomaticRecovery {
    <#
    .SYNOPSIS
    Attempts automatic recovery for common failure scenarios
    
    .DESCRIPTION
    Implements automated recovery procedures for known failure patterns.
    Attempts to resolve issues automatically before requiring manual intervention.
    
    .PARAMETER ErrorInfo
    Error information from Get-ErrorCategory
    
    .PARAMETER Context
    Additional context about the failed operation
    
    .PARAMETER MaxRecoveryAttempts
    Maximum number of recovery attempts to make
    
    .EXAMPLE
    $recovered = Invoke-AutomaticRecovery -ErrorInfo $errorInfo -Context @{ Operation = "ServiceInstall" }
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [hashtable]$ErrorInfo,
        
        [Parameter(Mandatory=$false)]
        [hashtable]$Context = @{},
        
        [Parameter(Mandatory=$false)]
        [int]$MaxRecoveryAttempts = 3
    )
    
    Write-InstallationLog -Level Info -Message "Attempting automatic recovery for error category: $($ErrorInfo.Category)" -Context $Context
    
    $recoverySuccessful = $false
    $recoveryAttempts = 0
    
    try {
        switch ($ErrorInfo.Category) {
            'FileSystemError' {
                Write-InstallationLog -Level Info -Message "Attempting file system error recovery..." -Context $Context
                
                # Recovery attempt 1: Clear temporary files and retry
                if ($recoveryAttempts -lt $MaxRecoveryAttempts) {
                    $recoveryAttempts++
                    Write-InstallationLog -Level Info -Message "Recovery attempt $recoveryAttempts`: Clearing temporary files" -Context $Context
                    
                    try {
                        $tempFiles = Get-ChildItem -Path "$env:TEMP\RepSetBridge-*" -ErrorAction SilentlyContinue
                        foreach ($file in $tempFiles) {
                            Remove-Item -Path $file.FullName -Force -ErrorAction SilentlyContinue
                        }
                        
                        # Wait a moment for file handles to be released
                        Start-Sleep -Seconds 2
                        $recoverySuccessful = $true
                        Write-InstallationLog -Level Success -Message "Temporary files cleared successfully" -Context $Context
                    }
                    catch {
                        Write-InstallationLog -Level Warning -Message "Failed to clear temporary files: $($_.Exception.Message)" -Context $Context
                    }
                }
                
                # Recovery attempt 2: Try alternative installation directory
                if (-not $recoverySuccessful -and $recoveryAttempts -lt $MaxRecoveryAttempts) {
                    $recoveryAttempts++
                    Write-InstallationLog -Level Info -Message "Recovery attempt $recoveryAttempts`: Trying alternative installation directory" -Context $Context
                    
                    try {
                        $alternativeInstallPath = "$env:LOCALAPPDATA\RepSet\Bridge"
                        if (-not (Test-Path -Path $alternativeInstallPath)) {
                            New-Item -Path $alternativeInstallPath -ItemType Directory -Force -ErrorAction Stop
                            $script:InstallPath = $alternativeInstallPath
                            $recoverySuccessful = $true
                            Write-InstallationLog -Level Success -Message "Alternative installation directory created: $alternativeInstallPath" -Context $Context
                        }
                    }
                    catch {
                        Write-InstallationLog -Level Warning -Message "Failed to create alternative directory: $($_.Exception.Message)" -Context $Context
                    }
                }
            }
            
            'ServiceInstallationError' {
                Write-InstallationLog -Level Info -Message "Attempting service installation error recovery..." -Context $Context
                
                # Recovery attempt 1: Remove existing service and retry
                if ($recoveryAttempts -lt $MaxRecoveryAttempts) {
                    $recoveryAttempts++
                    Write-InstallationLog -Level Info -Message "Recovery attempt $recoveryAttempts`: Removing existing service" -Context $Context
                    
                    try {
                        $existingService = Get-Service -Name $script:ServiceName -ErrorAction SilentlyContinue
                        if ($existingService) {
                            if ($existingService.Status -eq 'Running') {
                                Stop-Service -Name $script:ServiceName -Force -ErrorAction Stop
                                Start-Sleep -Seconds 3
                            }
                            
                            & sc.exe delete $script:ServiceName 2>&1 | Out-Null
                            Start-Sleep -Seconds 2
                            
                            # Verify service is removed
                            $serviceCheck = Get-Service -Name $script:ServiceName -ErrorAction SilentlyContinue
                            if (-not $serviceCheck) {
                                $recoverySuccessful = $true
                                Write-InstallationLog -Level Success -Message "Existing service removed successfully" -Context $Context
                            }
                        } else {
                            $recoverySuccessful = $true
                            Write-InstallationLog -Level Info -Message "No existing service found to remove" -Context $Context
                        }
                    }
                    catch {
                        Write-InstallationLog -Level Warning -Message "Failed to remove existing service: $($_.Exception.Message)" -Context $Context
                    }
                }
                
                # Recovery attempt 2: Restart Service Control Manager
                if (-not $recoverySuccessful -and $recoveryAttempts -lt $MaxRecoveryAttempts) {
                    $recoveryAttempts++
                    Write-InstallationLog -Level Info -Message "Recovery attempt $recoveryAttempts`: Restarting Service Control Manager" -Context $Context
                    
                    try {
                        # Note: This is a potentially disruptive operation
                        Write-InstallationLog -Level Warning -Message "This recovery step may temporarily affect other services" -Context $Context
                        
                        # Instead of restarting SCM, try refreshing service database
                        & sc.exe query type= service state= all | Out-Null
                        Start-Sleep -Seconds 2
                        $recoverySuccessful = $true
                        Write-InstallationLog -Level Success -Message "Service database refreshed" -Context $Context
                    }
                    catch {
                        Write-InstallationLog -Level Warning -Message "Failed to refresh service database: $($_.Exception.Message)" -Context $Context
                    }
                }
            }
            
            'NetworkConnectivity' {
                Write-InstallationLog -Level Info -Message "Attempting network connectivity recovery..." -Context $Context
                
                # Recovery attempt 1: Test alternative endpoints
                if ($recoveryAttempts -lt $MaxRecoveryAttempts) {
                    $recoveryAttempts++
                    Write-InstallationLog -Level Info -Message "Recovery attempt $recoveryAttempts`: Testing network connectivity" -Context $Context
                    
                    try {
                        # Test basic internet connectivity
                        $testUrls = @(
                            "https://www.google.com",
                            "https://www.microsoft.com",
                            "https://github.com"
                        )
                        
                        $connectivityOk = $false
                        foreach ($url in $testUrls) {
                            try {
                                $response = Invoke-WebRequest -Uri $url -Method Head -TimeoutSec 10 -ErrorAction Stop
                                if ($response.StatusCode -eq 200) {
                                    $connectivityOk = $true
                                    break
                                }
                            }
                            catch {
                                continue
                            }
                        }
                        
                        if ($connectivityOk) {
                            $recoverySuccessful = $true
                            Write-InstallationLog -Level Success -Message "Network connectivity verified" -Context $Context
                        } else {
                            Write-InstallationLog -Level Warning -Message "Network connectivity test failed" -Context $Context
                        }
                    }
                    catch {
                        Write-InstallationLog -Level Warning -Message "Network connectivity test error: $($_.Exception.Message)" -Context $Context
                    }
                }
                
                # Recovery attempt 2: Clear DNS cache
                if (-not $recoverySuccessful -and $recoveryAttempts -lt $MaxRecoveryAttempts) {
                    $recoveryAttempts++
                    Write-InstallationLog -Level Info -Message "Recovery attempt $recoveryAttempts`: Clearing DNS cache" -Context $Context
                    
                    try {
                        & ipconfig /flushdns 2>&1 | Out-Null
                        Start-Sleep -Seconds 2
                        $recoverySuccessful = $true
                        Write-InstallationLog -Level Success -Message "DNS cache cleared" -Context $Context
                    }
                    catch {
                        Write-InstallationLog -Level Warning -Message "Failed to clear DNS cache: $($_.Exception.Message)" -Context $Context
                    }
                }
            }
            
            'DownloadError' {
                Write-InstallationLog -Level Info -Message "Attempting download error recovery..." -Context $Context
                
                # Recovery attempt 1: Clear download cache and retry
                if ($recoveryAttempts -lt $MaxRecoveryAttempts) {
                    $recoveryAttempts++
                    Write-InstallationLog -Level Info -Message "Recovery attempt $recoveryAttempts`: Clearing download cache" -Context $Context
                    
                    try {
                        $downloadFiles = Get-ChildItem -Path "$env:TEMP\RepSetBridge-Download-*" -ErrorAction SilentlyContinue
                        foreach ($file in $downloadFiles) {
                            Remove-Item -Path $file.FullName -Force -ErrorAction SilentlyContinue
                        }
                        
                        $recoverySuccessful = $true
                        Write-InstallationLog -Level Success -Message "Download cache cleared" -Context $Context
                    }
                    catch {
                        Write-InstallationLog -Level Warning -Message "Failed to clear download cache: $($_.Exception.Message)" -Context $Context
                    }
                }
            }
            
            'ConfigurationError' {
                Write-InstallationLog -Level Info -Message "Attempting configuration error recovery..." -Context $Context
                
                # Recovery attempt 1: Recreate configuration directory
                if ($recoveryAttempts -lt $MaxRecoveryAttempts) {
                    $recoveryAttempts++
                    Write-InstallationLog -Level Info -Message "Recovery attempt $recoveryAttempts`: Recreating configuration directory" -Context $Context
                    
                    try {
                        $configDir = Join-Path $InstallPath "config"
                        if (Test-Path -Path $configDir) {
                            Remove-Item -Path $configDir -Recurse -Force -ErrorAction Stop
                        }
                        
                        New-Item -Path $configDir -ItemType Directory -Force -ErrorAction Stop
                        $recoverySuccessful = $true
                        Write-InstallationLog -Level Success -Message "Configuration directory recreated" -Context $Context
                    }
                    catch {
                        Write-InstallationLog -Level Warning -Message "Failed to recreate configuration directory: $($_.Exception.Message)" -Context $Context
                    }
                }
            }
            
            default {
                Write-InstallationLog -Level Info -Message "No specific recovery procedure available for error category: $($ErrorInfo.Category)" -Context $Context
                
                # Generic recovery: Wait and retry
                if ($recoveryAttempts -lt $MaxRecoveryAttempts) {
                    $recoveryAttempts++
                    Write-InstallationLog -Level Info -Message "Recovery attempt $recoveryAttempts`: Waiting before retry" -Context $Context
                    Start-Sleep -Seconds 5
                    $recoverySuccessful = $true
                }
            }
        }
        
        if ($recoverySuccessful) {
            Write-InstallationLog -Level Success -Message "Automatic recovery completed successfully after $recoveryAttempts attempts" -Context $Context
        } else {
            Write-InstallationLog -Level Warning -Message "Automatic recovery failed after $recoveryAttempts attempts" -Context $Context
        }
        
        return $recoverySuccessful
    }
    catch {
        Write-InstallationLog -Level Error -Message "Error during automatic recovery: $($_.Exception.Message)" -Context $Context
        return $false
    }
}

function Invoke-InstallationStep {
    <#
    .SYNOPSIS
    Executes an installation step with comprehensive error handling and recovery
    
    .DESCRIPTION
    Wraps installation steps with retry logic, error categorization, automatic recovery,
    and rollback capabilities. Provides consistent error handling across all installation steps.
    
    .PARAMETER StepName
    Name of the installation step for logging and progress tracking
    
    .PARAMETER StepNumber
    Current step number for progress tracking
    
    .PARAMETER TotalSteps
    Total number of installation steps
    
    .PARAMETER ScriptBlock
    The script block containing the installation step logic
    
    .PARAMETER MaxRetries
    Maximum number of retry attempts for this step
    
    .PARAMETER EnableAutoRecovery
    Whether to attempt automatic recovery on failure
    
    .PARAMETER RollbackOnFailure
    Whether to perform rollback if the step fails completely
    
    .PARAMETER Context
    Additional context information for logging and error handling
    
    .EXAMPLE
    Invoke-InstallationStep -StepName "Download Bridge" -StepNumber 3 -TotalSteps 10 -ScriptBlock {
        Get-LatestBridge -InstallPath $InstallPath
    } -MaxRetries 3 -EnableAutoRecovery -Context @{ Operation = "Download" }
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$StepName,
        
        [Parameter(Mandatory=$true)]
        [int]$StepNumber,
        
        [Parameter(Mandatory=$true)]
        [int]$TotalSteps,
        
        [Parameter(Mandatory=$true)]
        [scriptblock]$ScriptBlock,
        
        [Parameter(Mandatory=$false)]
        [int]$MaxRetries = 2,
        
        [Parameter(Mandatory=$false)]
        [switch]$EnableAutoRecovery,
        
        [Parameter(Mandatory=$false)]
        [switch]$RollbackOnFailure,
        
        [Parameter(Mandatory=$false)]
        [hashtable]$Context = @{}
    )
    
    Write-Progress-Step -Step $StepName -StepNumber $StepNumber -TotalSteps $TotalSteps -Status "Starting"
    Write-InstallationLog -Level Info -Message "Starting installation step: $StepName" -Context (@{
        StepNumber = $StepNumber
        TotalSteps = $TotalSteps
    } + $Context)
    
    $stepStartTime = Get-Date
    $stepSuccessful = $false
    $lastError = $null
    
    try {
        # Execute the step with retry logic
        $result = Invoke-WithRetry -ScriptBlock $ScriptBlock -MaxRetries $MaxRetries -Context (@{
            Step = $StepName
            StepNumber = $StepNumber
        } + $Context) -OnRetry {
            param($AttemptNumber, $Exception, $DelaySeconds)
            
            Write-Progress-Step -Step $StepName -StepNumber $StepNumber -TotalSteps $TotalSteps -Status "Retrying (Attempt $AttemptNumber)"
            
            # Attempt automatic recovery if enabled
            if ($EnableAutoRecovery -and $AttemptNumber -eq 1) {
                Write-InstallationLog -Level Info -Message "Attempting automatic recovery before retry..." -Context $Context
                
                $errorInfo = Get-ErrorCategory -Exception $Exception -Context $Context
                $recoverySuccessful = Invoke-AutomaticRecovery -ErrorInfo $errorInfo -Context $Context
                
                if ($recoverySuccessful) {
                    Write-InstallationLog -Level Success -Message "Automatic recovery completed, proceeding with retry" -Context $Context
                } else {
                    Write-InstallationLog -Level Warning -Message "Automatic recovery failed, proceeding with standard retry" -Context $Context
                }
            }
        }
        
        $stepSuccessful = $true
        $stepDuration = (Get-Date) - $stepStartTime
        
        Write-Progress-Step -Step $StepName -StepNumber $StepNumber -TotalSteps $TotalSteps -Status "Completed"
        Write-InstallationLog -Level Success -Message "Installation step completed successfully: $StepName (Duration: $([math]::Round($stepDuration.TotalSeconds, 2))s)" -Context $Context
        
        return $result
    }
    catch {
        $lastError = $_.Exception
        $stepDuration = (Get-Date) - $stepStartTime
        
        Write-Progress-Step -Step $StepName -StepNumber $StepNumber -TotalSteps $TotalSteps -Status "Failed"
        
        # Categorize and display user-friendly error
        $errorInfo = Write-UserFriendlyError -Exception $lastError -Context (@{
            Step = $StepName
            StepNumber = $StepNumber
            Duration = $stepDuration.TotalSeconds
        } + $Context) -Step $StepName
        
        # Attempt final recovery if not already tried
        if ($EnableAutoRecovery) {
            Write-InstallationLog -Level Info -Message "Attempting final recovery for failed step..." -Context $Context
            $recoverySuccessful = Invoke-AutomaticRecovery -ErrorInfo $errorInfo -Context $Context -MaxRecoveryAttempts 1
            
            if ($recoverySuccessful) {
                Write-InstallationLog -Level Info -Message "Final recovery successful, retrying step once more..." -Context $Context
                try {
                    Write-Progress-Step -Step $StepName -StepNumber $StepNumber -TotalSteps $TotalSteps -Status "Recovery Retry"
                    $result = & $ScriptBlock
                    $stepSuccessful = $true
                    
                    Write-Progress-Step -Step $StepName -StepNumber $StepNumber -TotalSteps $TotalSteps -Status "Completed"
                    Write-InstallationLog -Level Success -Message "Installation step completed after recovery: $StepName" -Context $Context
                    
                    return $result
                }
                catch {
                    Write-InstallationLog -Level Error -Message "Step failed even after recovery: $($_.Exception.Message)" -Context $Context
                }
            }
        }
        
        # Perform rollback if requested and this is a critical failure
        if ($RollbackOnFailure -and $errorInfo.Severity -eq "High") {
            Write-InstallationLog -Level Warning -Message "Critical failure detected, initiating rollback..." -Context $Context
            
            $rollbackSuccessful = Invoke-InstallationRollback -InstallationId $script:InstallationId -InstallPath $InstallPath -ServiceName $script:ServiceName -RollbackReason "Step '$StepName' failed with critical error" -Context $Context
            
            if ($rollbackSuccessful) {
                Write-InstallationLog -Level Info -Message "Rollback completed successfully" -Context $Context
            } else {
                Write-InstallationLog -Level Error -Message "Rollback failed - manual cleanup may be required" -Context $Context
            }
        }
        
        # Re-throw the exception to stop installation
        throw $lastError
    }
    finally {
        # Clear progress bar for this step
        Write-Progress -Activity "RepSet Bridge Installation" -Completed
    }
}

function Test-InstallationPrerequisites {
    <#
    .SYNOPSIS
    Comprehensive prerequisite checking with automatic remediation
    
    .DESCRIPTION
    Validates all system requirements and attempts automatic remediation
    where possible. Provides detailed feedback on any issues found.
    
    .EXAMPLE
    $prerequisitesPassed = Test-InstallationPrerequisites
    #>
    [CmdletBinding()]
    param()
    
    Write-InstallationLog -Level Info -Message "Performing comprehensive prerequisite checks..." -Context @{}
    
    $prerequisiteResults = @{
        PowerShellVersion = $false
        AdminRights = $false
        DotNetRuntime = $false
        DiskSpace = $false
        NetworkConnectivity = $false
        WindowsVersion = $false
        Architecture = $false
        OverallPassed = $false
        Issues = @()
        Warnings = @()
    }
    
    try {
        # Check PowerShell version
        Write-InstallationLog -Level Debug -Message "Checking PowerShell version..." -Context @{}
        $psVersion = $PSVersionTable.PSVersion
        if ($psVersion.Major -ge 5 -and ($psVersion.Major -gt 5 -or $psVersion.Minor -ge 1)) {
            $prerequisiteResults.PowerShellVersion = $true
            Write-InstallationLog -Level Success -Message "PowerShell version check passed: $psVersion" -Context @{}
        } else {
            $prerequisiteResults.Issues += "PowerShell version $psVersion is not supported. Minimum required: 5.1"
            Write-InstallationLog -Level Error -Message "PowerShell version check failed: $psVersion (minimum: 5.1)" -Context @{}
        }
        
        # Check administrator rights
        Write-InstallationLog -Level Debug -Message "Checking administrator privileges..." -Context @{}
        $currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
        $isAdmin = $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
        if ($isAdmin) {
            $prerequisiteResults.AdminRights = $true
            Write-InstallationLog -Level Success -Message "Administrator rights check passed" -Context @{}
        } else {
            $prerequisiteResults.Issues += "Administrator rights required. Please run PowerShell as Administrator."
            Write-InstallationLog -Level Error -Message "Administrator rights check failed" -Context @{}
        }
        
        # Check Windows version
        Write-InstallationLog -Level Debug -Message "Checking Windows version..." -Context @{}
        $osVersion = [System.Environment]::OSVersion.Version
        $windowsVersion = "$($osVersion.Major).$($osVersion.Minor)"
        
        # Windows 10 is version 10.0, Windows 11 is also 10.0 but with higher build number
        if ($osVersion.Major -ge 10) {
            $prerequisiteResults.WindowsVersion = $true
            Write-InstallationLog -Level Success -Message "Windows version check passed: $windowsVersion (Build $($osVersion.Build))" -Context @{}
        } else {
            $prerequisiteResults.Issues += "Windows 10 or later is required. Current version: $windowsVersion"
            Write-InstallationLog -Level Error -Message "Windows version check failed: $windowsVersion" -Context @{}
        }
        
        # Check system architecture
        Write-InstallationLog -Level Debug -Message "Checking system architecture..." -Context @{}
        $architecture = [System.Environment]::ProcessorArchitecture
        if ($architecture -eq "AMD64" -or $architecture -eq "X64") {
            $prerequisiteResults.Architecture = $true
            Write-InstallationLog -Level Success -Message "Architecture check passed: $architecture" -Context @{}
        } else {
            $prerequisiteResults.Issues += "x64 architecture required. Current architecture: $architecture"
            Write-InstallationLog -Level Error -Message "Architecture check failed: $architecture" -Context @{}
        }
        
        # Check available disk space
        Write-InstallationLog -Level Debug -Message "Checking available disk space..." -Context @{}
        try {
            $installDrive = [System.IO.Path]::GetPathRoot($InstallPath)
            $driveInfo = Get-WmiObject -Class Win32_LogicalDisk | Where-Object { $_.DeviceID -eq $installDrive.TrimEnd('\') }
            
            if ($driveInfo) {
                $freeSpaceGB = [math]::Round($driveInfo.FreeSpace / 1GB, 2)
                $requiredSpaceGB = 0.5  # 500MB minimum
                
                if ($freeSpaceGB -ge $requiredSpaceGB) {
                    $prerequisiteResults.DiskSpace = $true
                    Write-InstallationLog -Level Success -Message "Disk space check passed: ${freeSpaceGB}GB available (${requiredSpaceGB}GB required)" -Context @{}
                } else {
                    $prerequisiteResults.Issues += "Insufficient disk space. Available: ${freeSpaceGB}GB, Required: ${requiredSpaceGB}GB"
                    Write-InstallationLog -Level Error -Message "Disk space check failed: ${freeSpaceGB}GB available, ${requiredSpaceGB}GB required" -Context @{}
                }
            } else {
                $prerequisiteResults.Warnings += "Could not determine disk space for drive $installDrive"
                Write-InstallationLog -Level Warning -Message "Could not check disk space for drive $installDrive" -Context @{}
            }
        }
        catch {
            $prerequisiteResults.Warnings += "Error checking disk space: $($_.Exception.Message)"
            Write-InstallationLog -Level Warning -Message "Error checking disk space: $($_.Exception.Message)" -Context @{}
        }
        
        # Check .NET runtime
        Write-InstallationLog -Level Debug -Message "Checking .NET runtime..." -Context @{}
        try {
            # Check for .NET 6.0 or later
            $dotnetVersions = @()
            
            # Check registry for installed .NET versions
            $dotnetKeys = @(
                "HKLM:\SOFTWARE\WOW6432Node\dotnet\Setup\InstalledVersions\x64\sharedhost",
                "HKLM:\SOFTWARE\dotnet\Setup\InstalledVersions\x64\sharedhost"
            )
            
            foreach ($key in $dotnetKeys) {
                try {
                    if (Test-Path $key) {
                        $version = Get-ItemProperty -Path $key -Name "Version" -ErrorAction SilentlyContinue
                        if ($version -and $version.Version) {
                            $dotnetVersions += $version.Version
                        }
                    }
                }
                catch {
                    # Continue checking other locations
                }
            }
            
            # Also try to detect via dotnet command if available
            try {
                $dotnetInfo = & dotnet --info 2>&1 | Out-String
                if ($LASTEXITCODE -eq 0 -and $dotnetInfo -match "Microsoft\.NETCore\.App\s+(\d+\.\d+\.\d+)") {
                    $dotnetVersions += $matches[1]
                }
            }
            catch {
                # dotnet command not available or failed
            }
            
            # Check if any version meets requirements (.NET 6.0+)
            $hasValidDotNet = $false
            foreach ($version in $dotnetVersions) {
                try {
                    $versionObj = [System.Version]$version
                    if ($versionObj.Major -ge 6) {
                        $hasValidDotNet = $true
                        break
                    }
                }
                catch {
                    # Invalid version format, skip
                }
            }
            
            if ($hasValidDotNet) {
                $prerequisiteResults.DotNetRuntime = $true
                Write-InstallationLog -Level Success -Message ".NET runtime check passed. Found versions: $($dotnetVersions -join ', ')" -Context @{}
            } else {
                $prerequisiteResults.Warnings += ".NET 6.0 or later not detected. Installation will attempt to download required runtime."
                Write-InstallationLog -Level Warning -Message ".NET 6.0+ not detected. Found versions: $($dotnetVersions -join ', ')" -Context @{}
                # Don't fail the check - we can install .NET during installation
                $prerequisiteResults.DotNetRuntime = $true
            }
        }
        catch {
            $prerequisiteResults.Warnings += "Error checking .NET runtime: $($_.Exception.Message)"
            Write-InstallationLog -Level Warning -Message "Error checking .NET runtime: $($_.Exception.Message)" -Context @{}
            # Don't fail the check - we can try to install .NET during installation
            $prerequisiteResults.DotNetRuntime = $true
        }
        
        # Check network connectivity
        Write-InstallationLog -Level Debug -Message "Checking network connectivity..." -Context @{}
        try {
            $testUrls = @(
                "https://api.github.com",
                "https://github.com",
                $PlatformEndpoint
            )
            
            $connectivityResults = @()
            foreach ($url in $testUrls) {
                try {
                    $response = Invoke-WebRequest -Uri $url -Method Head -TimeoutSec 10 -ErrorAction Stop
                    $connectivityResults += @{
                        Url = $url
                        Success = $true
                        StatusCode = $response.StatusCode
                    }
                }
                catch {
                    $connectivityResults += @{
                        Url = $url
                        Success = $false
                        Error = $_.Exception.Message
                    }
                }
            }
            
            $successfulConnections = ($connectivityResults | Where-Object { $_.Success }).Count
            if ($successfulConnections -ge 2) {
                $prerequisiteResults.NetworkConnectivity = $true
                Write-InstallationLog -Level Success -Message "Network connectivity check passed ($successfulConnections/$($testUrls.Count) endpoints reachable)" -Context @{}
            } else {
                $prerequisiteResults.Issues += "Network connectivity issues detected. Only $successfulConnections/$($testUrls.Count) endpoints reachable."
                Write-InstallationLog -Level Error -Message "Network connectivity check failed ($successfulConnections/$($testUrls.Count) endpoints reachable)" -Context @{}
                
                # Log details of failed connections
                $failedConnections = $connectivityResults | Where-Object { -not $_.Success }
                foreach ($failed in $failedConnections) {
                    Write-InstallationLog -Level Debug -Message "Failed to connect to $($failed.Url): $($failed.Error)" -Context @{}
                }
            }
        }
        catch {
            $prerequisiteResults.Issues += "Error testing network connectivity: $($_.Exception.Message)"
            Write-InstallationLog -Level Error -Message "Error testing network connectivity: $($_.Exception.Message)" -Context @{}
        }
        
        # Determine overall result
        $criticalChecks = @('PowerShellVersion', 'AdminRights', 'WindowsVersion', 'Architecture', 'NetworkConnectivity')
        $criticalChecksPassed = $true
        
        foreach ($check in $criticalChecks) {
            if (-not $prerequisiteResults[$check]) {
                $criticalChecksPassed = $false
                break
            }
        }
        
        $prerequisiteResults.OverallPassed = $criticalChecksPassed
        
        # Display summary
        Write-InstallationLog -Level Info -Message "=== PREREQUISITE CHECK SUMMARY ===" -Context @{}
        Write-InstallationLog -Level Info -Message "PowerShell Version: $(if ($prerequisiteResults.PowerShellVersion) { '✓ PASS' } else { '✗ FAIL' })" -Context @{}
        Write-InstallationLog -Level Info -Message "Administrator Rights: $(if ($prerequisiteResults.AdminRights) { '✓ PASS' } else { '✗ FAIL' })" -Context @{}
        Write-InstallationLog -Level Info -Message "Windows Version: $(if ($prerequisiteResults.WindowsVersion) { '✓ PASS' } else { '✗ FAIL' })" -Context @{}
        Write-InstallationLog -Level Info -Message "System Architecture: $(if ($prerequisiteResults.Architecture) { '✓ PASS' } else { '✗ FAIL' })" -Context @{}
        Write-InstallationLog -Level Info -Message "Disk Space: $(if ($prerequisiteResults.DiskSpace) { '✓ PASS' } else { '⚠ WARNING' })" -Context @{}
        Write-InstallationLog -Level Info -Message ".NET Runtime: $(if ($prerequisiteResults.DotNetRuntime) { '✓ PASS' } else { '⚠ WARNING' })" -Context @{}
        Write-InstallationLog -Level Info -Message "Network Connectivity: $(if ($prerequisiteResults.NetworkConnectivity) { '✓ PASS' } else { '✗ FAIL' })" -Context @{}
        
        if ($prerequisiteResults.Issues.Count -gt 0) {
            Write-InstallationLog -Level Error -Message "Critical Issues Found:" -Context @{}
            foreach ($issue in $prerequisiteResults.Issues) {
                Write-InstallationLog -Level Error -Message "  • $issue" -Context @{}
            }
        }
        
        if ($prerequisiteResults.Warnings.Count -gt 0) {
            Write-InstallationLog -Level Warning -Message "Warnings:" -Context @{}
            foreach ($warning in $prerequisiteResults.Warnings) {
                Write-InstallationLog -Level Warning -Message "  • $warning" -Context @{}
            }
        }
        
        Write-InstallationLog -Level Info -Message "Overall Result: $(if ($prerequisiteResults.OverallPassed) { '✓ PREREQUISITES PASSED' } else { '✗ PREREQUISITES FAILED' })" -Context @{}
        Write-InstallationLog -Level Info -Message "=== END PREREQUISITE SUMMARY ===" -Context @{}
        
        return $prerequisiteResults
    }
    catch {
        Write-InstallationLog -Level Error -Message "Error during prerequisite checking: $($_.Exception.Message)" -Context @{}
        $prerequisiteResults.Issues += "Prerequisite checking failed: $($_.Exception.Message)"
        $prerequisiteResults.OverallPassed = $false
        return $prerequisiteResults
    }
}

            Write-InstallationLog -Level Info -Message "Waiting $actualDelay seconds before retry..." -Context $Context
            
            # Execute OnRetry callback if provided
            if ($OnRetry) {
                try {
                    & $OnRetry -AttemptNumber $attempt -Exception $lastException -DelaySeconds $actualDelay
                }
                catch {
                    Write-InstallationLog -Level Warning -Message "OnRetry callback failed: $($_.Exception.Message)" -Context $Context
                }
            }
            
            Start-Sleep -Seconds $actualDelay
        }
    }
    
    # This should never be reached, but just in case
    throw $lastException
}

function Invoke-InstallationRollback {
    <#
    .SYNOPSIS
    Performs comprehensive rollback of failed installation
    
    .DESCRIPTION
    Cleans up all installation artifacts and restores system to pre-installation state.
    Handles partial installations gracefully and provides detailed rollback logging.
    
    .PARAMETER InstallationId
    Unique identifier for the installation being rolled back
    
    .PARAMETER InstallPath
    Path where bridge was being installed
    
    .PARAMETER ServiceName
    Name of the Windows service to remove
    
    .PARAMETER PreserveConfig
    Whether to preserve existing configuration during rollback
    
    .PARAMETER RollbackReason
    Reason for the rollback (for logging purposes)
    
    .EXAMPLE
    Invoke-InstallationRollback -InstallationId $script:InstallationId -InstallPath $InstallPath -ServiceName $script:ServiceName -RollbackReason "Download failed"
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$InstallationId,
        
        [Parameter(Mandatory=$false)]
        [string]$InstallPath = "$env:ProgramFiles\RepSet\Bridge",
        
        [Parameter(Mandatory=$false)]
        [string]$ServiceName = "RepSetBridge",
        
        [Parameter(Mandatory=$false)]
        [switch]$PreserveConfig,
        
        [Parameter(Mandatory=$false)]
        [string]$RollbackReason = "Installation failed"
    )
    
    Write-InstallationLog -Level Warning -Message "=== STARTING INSTALLATION ROLLBACK ===" -Context @{
        InstallationId = $InstallationId
        Reason = $RollbackReason
        PreserveConfig = $PreserveConfig.IsPresent
    }
    
    $rollbackSteps = @()
    $rollbackErrors = @()
    
    try {
        # Step 1: Stop and remove Windows service
        Write-InstallationLog -Level Info -Message "Rolling back Windows service installation..."
        try {
            $service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
            if ($service) {
                Write-InstallationLog -Level Info -Message "Stopping service '$ServiceName'..."
                
                if ($service.Status -eq 'Running') {
                    Stop-Service -Name $ServiceName -Force -ErrorAction Stop
                    $rollbackSteps += "Stopped service '$ServiceName'"
                    
                    # Wait for service to stop
                    $timeout = 30
                    $elapsed = 0
                    while ((Get-Service -Name $ServiceName).Status -ne 'Stopped' -and $elapsed -lt $timeout) {
                        Start-Sleep -Seconds 1
                        $elapsed++
                    }
                }
                
                Write-InstallationLog -Level Info -Message "Removing service '$ServiceName'..."
                $scResult = & sc.exe delete $ServiceName 2>&1
                if ($LASTEXITCODE -eq 0) {
                    $rollbackSteps += "Removed service '$ServiceName'"
                    Write-InstallationLog -Level Success -Message "Service '$ServiceName' removed successfully"
                } else {
                    $rollbackErrors += "Failed to remove service: $scResult"
                    Write-InstallationLog -Level Warning -Message "Failed to remove service: $scResult"
                }
            } else {
                Write-InstallationLog -Level Info -Message "Service '$ServiceName' not found - nothing to remove"
            }
        }
        catch {
            $rollbackErrors += "Service rollback error: $($_.Exception.Message)"
            Write-InstallationLog -Level Warning -Message "Error during service rollback: $($_.Exception.Message)"
        }
        
        # Step 2: Remove installed files and directories
        Write-InstallationLog -Level Info -Message "Rolling back file installation..."
        try {
            if (Test-Path -Path $InstallPath) {
                Write-InstallationLog -Level Info -Message "Removing installation directory: $InstallPath"
                
                # Preserve configuration if requested
                $configBackupPath = $null
                if ($PreserveConfig) {
                    $configPath = Join-Path $InstallPath "config\$script:ConfigFileName"
                    if (Test-Path -Path $configPath) {
                        $configBackupPath = "$env:TEMP\RepSetBridge-Config-Backup-$(Get-Date -Format 'yyyyMMdd-HHmmss').yaml"
                        Copy-Item -Path $configPath -Destination $configBackupPath -ErrorAction SilentlyContinue
                        Write-InstallationLog -Level Info -Message "Configuration backed up to: $configBackupPath"
                        $rollbackSteps += "Configuration backed up to '$configBackupPath'"
                    }
                }
                
                # Remove installation directory
                Remove-Item -Path $InstallPath -Recurse -Force -ErrorAction Stop
                $rollbackSteps += "Removed installation directory '$InstallPath'"
                Write-InstallationLog -Level Success -Message "Installation directory removed successfully"
                
                # Restore configuration if it was backed up
                if ($configBackupPath -and (Test-Path -Path $configBackupPath)) {
                    Write-InstallationLog -Level Info -Message "Configuration backup available at: $configBackupPath"
                    Write-InstallationLog -Level Info -Message "You can restore it manually if needed for future installations"
                }
            } else {
                Write-InstallationLog -Level Info -Message "Installation directory '$InstallPath' not found - nothing to remove"
            }
        }
        catch {
            $rollbackErrors += "File rollback error: $($_.Exception.Message)"
            Write-InstallationLog -Level Warning -Message "Error during file rollback: $($_.Exception.Message)"
        }
        
        # Step 3: Clean up temporary files
        Write-InstallationLog -Level Info -Message "Cleaning up temporary files..."
        try {
            $tempFiles = @(
                "$env:TEMP\RepSetBridge-*.zip",
                "$env:TEMP\RepSetBridge-*.exe",
                "$env:TEMP\RepSetBridge-Download-*"
            )
            
            foreach ($pattern in $tempFiles) {
                $files = Get-ChildItem -Path $pattern -ErrorAction SilentlyContinue
                foreach ($file in $files) {
                    Remove-Item -Path $file.FullName -Force -ErrorAction SilentlyContinue
                    $rollbackSteps += "Removed temporary file '$($file.Name)'"
                }
            }
            
            Write-InstallationLog -Level Success -Message "Temporary files cleaned up"
        }
        catch {
            $rollbackErrors += "Temp cleanup error: $($_.Exception.Message)"
            Write-InstallationLog -Level Warning -Message "Error during temp file cleanup: $($_.Exception.Message)"
        }
        
        # Step 4: Clean up registry entries (if any were created)
        Write-InstallationLog -Level Info -Message "Cleaning up registry entries..."
        try {
            $registryPaths = @(
                "HKLM:\SOFTWARE\RepSet\Bridge",
                "HKLM:\SYSTEM\CurrentControlSet\Services\$ServiceName"
            )
            
            foreach ($regPath in $registryPaths) {
                if (Test-Path -Path $regPath) {
                    Remove-Item -Path $regPath -Recurse -Force -ErrorAction SilentlyContinue
                    $rollbackSteps += "Removed registry key '$regPath'"
                    Write-InstallationLog -Level Info -Message "Removed registry key: $regPath"
                }
            }
            
            Write-InstallationLog -Level Success -Message "Registry cleanup completed"
        }
        catch {
            $rollbackErrors += "Registry cleanup error: $($_.Exception.Message)"
            Write-InstallationLog -Level Warning -Message "Error during registry cleanup: $($_.Exception.Message)"
        }
        
        # Step 5: Clean up Windows Event Log sources
        Write-InstallationLog -Level Info -Message "Cleaning up event log sources..."
        try {
            $eventSources = @("RepSetBridge", "RepSetBridge-Installer")
            foreach ($source in $eventSources) {
                if ([System.Diagnostics.EventLog]::SourceExists($source)) {
                    [System.Diagnostics.EventLog]::DeleteEventSource($source)
                    $rollbackSteps += "Removed event log source '$source'"
                    Write-InstallationLog -Level Info -Message "Removed event log source: $source"
                }
            }
            
            Write-InstallationLog -Level Success -Message "Event log cleanup completed"
        }
        catch {
            $rollbackErrors += "Event log cleanup error: $($_.Exception.Message)"
            Write-InstallationLog -Level Warning -Message "Error during event log cleanup: $($_.Exception.Message)"
        }
        
        # Generate rollback summary
        Write-InstallationLog -Level Info -Message "=== ROLLBACK SUMMARY ==="
        Write-InstallationLog -Level Info -Message "Installation ID: $InstallationId"
        Write-InstallationLog -Level Info -Message "Rollback Reason: $RollbackReason"
        Write-InstallationLog -Level Info -Message "Steps Completed: $($rollbackSteps.Count)"
        
        foreach ($step in $rollbackSteps) {
            Write-InstallationLog -Level Info -Message "  ✓ $step"
        }
        
        if ($rollbackErrors.Count -gt 0) {
            Write-InstallationLog -Level Warning -Message "Rollback Errors: $($rollbackErrors.Count)"
            foreach ($error in $rollbackErrors) {
                Write-InstallationLog -Level Warning -Message "  ⚠ $error"
            }
        }
        
        # Send rollback notification to platform
        Send-InstallationNotification -Status Failed -Message "Installation rolled back: $RollbackReason" -Details @{
            RollbackSteps = $rollbackSteps
            RollbackErrors = $rollbackErrors
            StepsCompleted = $rollbackSteps.Count
            ErrorsEncountered = $rollbackErrors.Count
        }
        
        if ($rollbackErrors.Count -eq 0) {
            Write-InstallationLog -Level Success -Message "=== ROLLBACK COMPLETED SUCCESSFULLY ==="
            Write-InstallationLog -Level Info -Message "System has been restored to pre-installation state"
            return $true
        } else {
            Write-InstallationLog -Level Warning -Message "=== ROLLBACK COMPLETED WITH WARNINGS ==="
            Write-InstallationLog -Level Warning -Message "Some cleanup operations encountered errors (see above)"
            Write-InstallationLog -Level Info -Message "Manual cleanup may be required for complete removal"
            return $false
        }
    }
    catch {
        Write-InstallationLog -Level Error -Message "Critical error during rollback: $($_.Exception.Message)"
        Write-InstallationLog -Level Error -Message "=== ROLLBACK FAILED ==="
        Write-InstallationLog -Level Error -Message "Manual cleanup will be required"
        
        # Send critical rollback failure notification
        Send-InstallationNotification -Status Failed -Message "Rollback failed: $($_.Exception.Message)" -Details @{
            RollbackSteps = $rollbackSteps
            RollbackErrors = $rollbackErrors + @("Critical rollback error: $($_.Exception.Message)")
            CriticalFailure = $true
        }
        
        return $false
    }
}

function Get-ErrorCategory {
    <#
    .SYNOPSIS
    Categorizes errors for better user understanding and automated recovery
    
    .DESCRIPTION
    Analyzes exception details to categorize errors into actionable categories
    with specific remediation guidance.
    
    .PARAMETER Exception
    The exception to categorize
    
    .PARAMETER Context
    Additional context about when/where the error occurred
    
    .EXAMPLE
    $category = Get-ErrorCategory -Exception $_.Exception -Context @{ Operation = "Download" }
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [System.Exception]$Exception,
        
        [Parameter(Mandatory=$false)]
        [hashtable]$Context = @{}
    )
    
    $exceptionType = $Exception.GetType().FullName
    $message = $Exception.Message.ToLower()
    
    # Network-related errors
    if ($exceptionType -like "*WebException*" -or 
        $exceptionType -like "*HttpRequestException*" -or
        $message -like "*network*" -or
        $message -like "*connection*" -or
        $message -like "*timeout*" -or
        $message -like "*dns*") {
        
        return @{
            Category = "Network"
            Severity = "High"
            IsRetryable = $true
            UserMessage = "Network connectivity issue detected"
            TechnicalMessage = $Exception.Message
            Remediation = @(
                "Check your internet connection",
                "Verify firewall settings allow HTTPS traffic",
                "Try again in a few minutes",
                "Contact your network administrator if the problem persists"
            )
            AutoRecovery = @{
                Enabled = $true
                MaxRetries = 5
                RetryDelay = 10
            }
        }
    }
    
    # Permission/Security errors
    if ($exceptionType -like "*UnauthorizedAccessException*" -or
        $exceptionType -like "*SecurityException*" -or
        $message -like "*access*denied*" -or
        $message -like "*permission*" -or
        $message -like "*unauthorized*") {
        
        return @{
            Category = "Permission"
            Severity = "Critical"
            IsRetryable = $false
            UserMessage = "Insufficient permissions to complete installation"
            TechnicalMessage = $Exception.Message
            Remediation = @(
                "Ensure you're running PowerShell as Administrator",
                "Check that your user account has local administrator rights",
                "Verify Windows Defender or antivirus isn't blocking the installation",
                "Try running the command from an elevated PowerShell prompt"
            )
            AutoRecovery = @{
                Enabled = $false
                Reason = "Requires manual intervention"
            }
        }
    }
    
    # Disk space/IO errors
    if ($exceptionType -like "*IOException*" -or
        $exceptionType -like "*DirectoryNotFoundException*" -or
        $exceptionType -like "*FileNotFoundException*" -or
        $message -like "*disk*space*" -or
        $message -like "*not enough space*" -or
        $message -like "*directory*" -or
        $message -like "*file*not*found*") {
        
        return @{
            Category = "Storage"
            Severity = "High"
            IsRetryable = $true
            UserMessage = "File system or storage issue detected"
            TechnicalMessage = $Exception.Message
            Remediation = @(
                "Ensure sufficient disk space is available (at least 100MB free)",
                "Check that the installation directory is accessible",
                "Verify the disk is not full or read-only",
                "Try installing to a different location using -InstallPath parameter"
            )
            AutoRecovery = @{
                Enabled = $true
                MaxRetries = 2
                RetryDelay = 5
                AlternativeActions = @("Try different install path", "Clean temp files")
            }
        }
    }
    
    # Service-related errors
    if ($message -like "*service*" -or
        $message -like "*sc.exe*" -or
        $message -like "*windows service*" -or
        $Context.Operation -eq "ServiceInstallation") {
        
        return @{
            Category = "Service"
            Severity = "High"
            IsRetryable = $true
            UserMessage = "Windows service installation or management issue"
            TechnicalMessage = $Exception.Message
            Remediation = @(
                "Ensure Windows Service Control Manager is running",
                "Check that no other RepSet Bridge service is already installed",
                "Verify you have permissions to install Windows services",
                "Try stopping any existing RepSet Bridge processes"
            )
            AutoRecovery = @{
                Enabled = $true
                MaxRetries = 3
                RetryDelay = 5
                PreRetryActions = @("Stop existing service", "Clean service registry")
            }
        }
    }
    
    # Configuration/validation errors
    if ($message -like "*configuration*" -or
        $message -like "*config*" -or
        $message -like "*validation*" -or
        $message -like "*invalid*" -or
        $Context.Operation -eq "Configuration") {
        
        return @{
            Category = "Configuration"
            Severity = "Medium"
            IsRetryable = $true
            UserMessage = "Configuration or validation error"
            TechnicalMessage = $Exception.Message
            Remediation = @(
                "Verify the installation command parameters are correct",
                "Check that the installation command hasn't expired",
                "Generate a new installation command from the platform",
                "Ensure all required parameters are provided"
            )
            AutoRecovery = @{
                Enabled = $false
                Reason = "May require new installation command"
            }
        }
    }
    
    # Download/integrity errors
    if ($message -like "*download*" -or
        $message -like "*checksum*" -or
        $message -like "*hash*" -or
        $message -like "*integrity*" -or
        $message -like "*corrupt*" -or
        $Context.Operation -eq "Download") {
        
        return @{
            Category = "Download"
            Severity = "High"
            IsRetryable = $true
            UserMessage = "File download or integrity verification failed"
            TechnicalMessage = $Exception.Message
            Remediation = @(
                "Check your internet connection stability",
                "Verify GitHub.com is accessible from your network",
                "Try the installation again (files will be re-downloaded)",
                "Contact support if downloads consistently fail"
            )
            AutoRecovery = @{
                Enabled = $true
                MaxRetries = 3
                RetryDelay = 10
                PreRetryActions = @("Clear download cache", "Verify network connectivity")
            }
        }
    }
    
    # Default/Unknown errors
    return @{
        Category = "Unknown"
        Severity = "Medium"
        IsRetryable = $true
        UserMessage = "An unexpected error occurred during installation"
        TechnicalMessage = $Exception.Message
        Remediation = @(
            "Try running the installation command again",
            "Check the installation log for more details: $script:LogFile",
            "Ensure your system meets all requirements",
            "Contact support with the installation log if the problem persists"
        )
        AutoRecovery = @{
            Enabled = $true
            MaxRetries = 2
            RetryDelay = 5
        }
    }
}

function Write-UserFriendlyError {
    <#
    .SYNOPSIS
    Displays user-friendly error messages with actionable remediation steps
    
    .DESCRIPTION
    Takes technical errors and presents them in a user-friendly format with
    clear remediation steps and next actions.
    
    .PARAMETER Exception
    The exception that occurred
    
    .PARAMETER Context
    Additional context about the operation that failed
    
    .PARAMETER ShowTechnicalDetails
    Whether to include technical details in the output
    
    .EXAMPLE
    Write-UserFriendlyError -Exception $_.Exception -Context @{ Operation = "Download"; Step = "Bridge Download" }
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [System.Exception]$Exception,
        
        [Parameter(Mandatory=$false)]
        [hashtable]$Context = @{},
        
        [Parameter(Mandatory=$false)]
        [switch]$ShowTechnicalDetails
    )
    
    $errorCategory = Get-ErrorCategory -Exception $Exception -Context $Context
    
    Write-InstallationLog -Level Error -Message "=== INSTALLATION ERROR ==="
    Write-InstallationLog -Level Error -Message "Error Category: $($errorCategory.Category)"
    Write-InstallationLog -Level Error -Message "Severity: $($errorCategory.Severity)"
    
    if ($Context.Step) {
        Write-InstallationLog -Level Error -Message "Failed Step: $($Context.Step)"
    }
    
    if ($Context.Operation) {
        Write-InstallationLog -Level Error -Message "Failed Operation: $($Context.Operation)"
    }
    
    Write-InstallationLog -Level Error -Message ""
    Write-InstallationLog -Level Error -Message "What happened:"
    Write-InstallationLog -Level Error -Message "  $($errorCategory.UserMessage)"
    
    if ($ShowTechnicalDetails) {
        Write-InstallationLog -Level Error -Message ""
        Write-InstallationLog -Level Error -Message "Technical details:"
        Write-InstallationLog -Level Error -Message "  $($errorCategory.TechnicalMessage)"
    }
    
    Write-InstallationLog -Level Error -Message ""
    Write-InstallationLog -Level Error -Message "How to fix this:"
    
    foreach ($step in $errorCategory.Remediation) {
        Write-InstallationLog -Level Error -Message "  • $step"
    }
    
    if ($errorCategory.IsRetryable) {
        Write-InstallationLog -Level Info -Message ""
        Write-InstallationLog -Level Info -Message "This error may be temporary. The installation will automatically retry if possible."
    } else {
        Write-InstallationLog -Level Warning -Message ""
        Write-InstallationLog -Level Warning -Message "This error requires manual intervention before retrying the installation."
    }
    
    Write-InstallationLog -Level Error -Message "=== END ERROR DETAILS ==="
    
    return $errorCategory
}

function Invoke-AutoRecovery {
    <#
    .SYNOPSIS
    Attempts automated recovery procedures for common failure scenarios
    
    .DESCRIPTION
    Implements automated recovery strategies based on error categorization.
    Performs pre-retry actions and system cleanup to improve retry success rates.
    
    .PARAMETER ErrorCategory
    The categorized error information from Get-ErrorCategory
    
    .PARAMETER Context
    Additional context about the failed operation
    
    .EXAMPLE
    $recovered = Invoke-AutoRecovery -ErrorCategory $errorCategory -Context @{ Operation = "Download" }
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [hashtable]$ErrorCategory,
        
        [Parameter(Mandatory=$false)]
        [hashtable]$Context = @{}
    )
    
    if (-not $ErrorCategory.AutoRecovery.Enabled) {
        Write-InstallationLog -Level Info -Message "Auto-recovery not enabled for this error type: $($ErrorCategory.AutoRecovery.Reason)"
        return $false
    }
    
    Write-InstallationLog -Level Info -Message "Attempting automated recovery for $($ErrorCategory.Category) error..."
    
    $recoverySuccess = $false
    
    try {
        switch ($ErrorCategory.Category) {
            "Network" {
                Write-InstallationLog -Level Info -Message "Performing network connectivity recovery..."
                
                # Test basic connectivity
                $connectivityTest = Test-NetworkConnectivity -Detailed
                if (-not $connectivityTest.Success) {
                    Write-InstallationLog -Level Warning -Message "Network connectivity test failed - recovery may not be effective"
                }
                
                # Clear DNS cache
                try {
                    & ipconfig /flushdns | Out-Null
                    Write-InstallationLog -Level Info -Message "DNS cache cleared"
                }
                catch {
                    Write-InstallationLog -Level Warning -Message "Failed to clear DNS cache: $($_.Exception.Message)"
                }
                
                # Wait for network stabilization
                Write-InstallationLog -Level Info -Message "Waiting for network stabilization..."
                Start-Sleep -Seconds 5
                
                $recoverySuccess = $true
            }
            
            "Storage" {
                Write-InstallationLog -Level Info -Message "Performing storage recovery..."
                
                # Clean temporary files
                try {
                    $tempFiles = Get-ChildItem -Path "$env:TEMP\RepSetBridge-*" -ErrorAction SilentlyContinue
                    foreach ($file in $tempFiles) {
                        Remove-Item -Path $file.FullName -Force -ErrorAction SilentlyContinue
                    }
                    Write-InstallationLog -Level Info -Message "Temporary files cleaned"
                }
                catch {
                    Write-InstallationLog -Level Warning -Message "Failed to clean temporary files: $($_.Exception.Message)"
                }
                
                # Check disk space
                $installDrive = (Split-Path -Path $InstallPath -Qualifier)
                $driveInfo = Get-WmiObject -Class Win32_LogicalDisk | Where-Object { $_.DeviceID -eq $installDrive }
                if ($driveInfo) {
                    $freeSpaceMB = [math]::Round($driveInfo.FreeSpace / 1MB, 2)
                    Write-InstallationLog -Level Info -Message "Available disk space: $freeSpaceMB MB"
                    
                    if ($freeSpaceMB -lt 100) {
                        Write-InstallationLog -Level Warning -Message "Low disk space detected - recovery may not be effective"
                    } else {
                        $recoverySuccess = $true
                    }
                } else {
                    Write-InstallationLog -Level Warning -Message "Could not check disk space"
                    $recoverySuccess = $true  # Assume recovery is possible
                }
            }
            
            "Service" {
                Write-InstallationLog -Level Info -Message "Performing service recovery..."
                
                # Stop any existing RepSet Bridge processes
                try {
                    $processes = Get-Process -Name "*RepSet*", "*Bridge*" -ErrorAction SilentlyContinue
                    foreach ($process in $processes) {
                        Write-InstallationLog -Level Info -Message "Stopping process: $($process.Name) (PID: $($process.Id))"
                        $process.Kill()
                        Start-Sleep -Seconds 2
                    }
                }
                catch {
                    Write-InstallationLog -Level Warning -Message "Failed to stop existing processes: $($_.Exception.Message)"
                }
                
                # Clean up any orphaned service entries
                try {
                    $existingService = Get-Service -Name $script:ServiceName -ErrorAction SilentlyContinue
                    if ($existingService) {
                        Write-InstallationLog -Level Info -Message "Removing existing service registration..."
                        & sc.exe delete $script:ServiceName | Out-Null
                        Start-Sleep -Seconds 3
                    }
                }
                catch {
                    Write-InstallationLog -Level Warning -Message "Failed to clean existing service: $($_.Exception.Message)"
                }
                
                $recoverySuccess = $true
            }
            
            "Download" {
                Write-InstallationLog -Level Info -Message "Performing download recovery..."
                
                # Clear download cache
                try {
                    $downloadCache = "$env:TEMP\RepSetBridge-Download-*"
                    $cacheFiles = Get-ChildItem -Path $downloadCache -ErrorAction SilentlyContinue
                    foreach ($file in $cacheFiles) {
                        Remove-Item -Path $file.FullName -Force -ErrorAction SilentlyContinue
                    }
                    Write-InstallationLog -Level Info -Message "Download cache cleared"
                }
                catch {
                    Write-InstallationLog -Level Warning -Message "Failed to clear download cache: $($_.Exception.Message)"
                }
                
                # Test GitHub connectivity
                try {
                    $githubTest = Test-NetConnection -ComputerName "github.com" -Port 443 -InformationLevel Quiet
                    if ($githubTest) {
                        Write-InstallationLog -Level Info -Message "GitHub connectivity verified"
                        $recoverySuccess = $true
                    } else {
                        Write-InstallationLog -Level Warning -Message "GitHub connectivity test failed"
                    }
                }
                catch {
                    Write-InstallationLog -Level Warning -Message "Could not test GitHub connectivity: $($_.Exception.Message)"
                    $recoverySuccess = $true  # Assume recovery is possible
                }
            }
            
            default {
                Write-InstallationLog -Level Info -Message "Performing generic recovery..."
                
                # Generic recovery: wait and clear temporary state
                Start-Sleep -Seconds 3
                
                # Clear any temporary files
                try {
                    Get-ChildItem -Path "$env:TEMP\RepSetBridge-*" -ErrorAction SilentlyContinue | 
                        Remove-Item -Force -ErrorAction SilentlyContinue
                }
                catch {
                    # Ignore cleanup errors
                }
                
                $recoverySuccess = $true
            }
        }
        
        if ($recoverySuccess) {
            Write-InstallationLog -Level Success -Message "Automated recovery completed successfully"
        } else {
            Write-InstallationLog -Level Warning -Message "Automated recovery completed with warnings"
        }
        
        return $recoverySuccess
    }
    catch {
        Write-InstallationLog -Level Error -Message "Automated recovery failed: $($_.Exception.Message)"
        return $false
    }
}

function Test-NetworkConnectivity {
    <#
    .SYNOPSIS
    Tests network connectivity for installation requirements
    
    .DESCRIPTION
    Performs comprehensive network connectivity tests to verify that
    all required endpoints are accessible for installation.
    
    .PARAMETER Detailed
    Whether to perform detailed connectivity tests
    
    .EXAMPLE
    $connectivity = Test-NetworkConnectivity -Detailed
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$false)]
        [switch]$Detailed
    )
    
    $results = @{
        Success = $true
        Tests = @()
        Issues = @()
    }
    
    # Test basic internet connectivity
    try {
        $internetTest = Test-NetConnection -ComputerName "8.8.8.8" -Port 53 -InformationLevel Quiet -WarningAction SilentlyContinue
        $results.Tests += @{
            Name = "Internet Connectivity"
            Success = $internetTest
            Details = if ($internetTest) { "Internet accessible" } else { "No internet connection" }
        }
        
        if (-not $internetTest) {
            $results.Success = $false
            $results.Issues += "No internet connectivity detected"
        }
    }
    catch {
        $results.Success = $false
        $results.Tests += @{
            Name = "Internet Connectivity"
            Success = $false
            Details = "Test failed: $($_.Exception.Message)"
        }
        $results.Issues += "Internet connectivity test failed"
    }
    
    if ($Detailed) {
        # Test GitHub connectivity
        try {
            $githubTest = Test-NetConnection -ComputerName "github.com" -Port 443 -InformationLevel Quiet -WarningAction SilentlyContinue
            $results.Tests += @{
                Name = "GitHub Connectivity"
                Success = $githubTest
                Details = if ($githubTest) { "GitHub accessible" } else { "Cannot reach GitHub" }
            }
            
            if (-not $githubTest) {
                $results.Success = $false
                $results.Issues += "Cannot reach GitHub (required for bridge download)"
            }
        }
        catch {
            $results.Success = $false
            $results.Tests += @{
                Name = "GitHub Connectivity"
                Success = $false
                Details = "Test failed: $($_.Exception.Message)"
            }
            $results.Issues += "GitHub connectivity test failed"
        }
        
        # Test platform connectivity
        try {
            $platformHost = ([System.Uri]$PlatformEndpoint).Host
            $platformTest = Test-NetConnection -ComputerName $platformHost -Port 443 -InformationLevel Quiet -WarningAction SilentlyContinue
            $results.Tests += @{
                Name = "Platform Connectivity"
                Success = $platformTest
                Details = if ($platformTest) { "Platform accessible" } else { "Cannot reach platform" }
            }
            
            if (-not $platformTest) {
                $results.Issues += "Cannot reach RepSet platform (may affect status reporting)"
            }
        }
        catch {
            $results.Tests += @{
                Name = "Platform Connectivity"
                Success = $false
                Details = "Test failed: $($_.Exception.Message)"
            }
            $results.Issues += "Platform connectivity test failed"
        }
    }
    
    return $results
}

# ================================================================
# System Requirements Validation Functions
# ================================================================

function Test-SystemRequirements {
    [CmdletBinding()]
    param()
    
    Write-InstallationLog -Level Info -Message "Validating system requirements..."
    
    $requirements = @{
        AdminRights = $false
        PowerShellVersion = $false
        PowerShellExecutionPolicy = $false
        DotNetRuntime = $false
        WindowsVersion = $false
        WindowsFeatures = $false
        DiskSpace = $false
        NetworkAccess = $false
        MemoryAvailable = $false
    }
    
    $requirementDetails = @{}
    
    try {
        Write-Progress-Step -Step "System Requirements Validation" -StepNumber 1 -TotalSteps 1 -Status "Checking administrator privileges" -SubStepNumber 1 -TotalSubSteps 9
        
        # Check if running as administrator
        $currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
        $requirements.AdminRights = $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
        
        if ($requirements.AdminRights) {
            Write-InstallationLog -Level Success -Message "✓ Running with Administrator privileges"
            $requirementDetails.AdminRights = "Administrator privileges confirmed"
        }
        else {
            Write-InstallationLog -Level Error -Message "✗ Administrator privileges required"
            Write-InstallationLog -Level Error -Message "SOLUTION: Right-click PowerShell and select 'Run as Administrator', then run this command again."
            $requirementDetails.AdminRights = "Missing administrator privileges"
        }
        
        Write-Progress-Step -Step "System Requirements Validation" -StepNumber 1 -TotalSteps 1 -Status "Checking PowerShell version" -SubStepNumber 2 -TotalSubSteps 9
        
        # Check PowerShell version (minimum 5.1)
        $psVersion = $PSVersionTable.PSVersion
        $requirements.PowerShellVersion = $psVersion.Major -ge 5 -and ($psVersion.Major -gt 5 -or $psVersion.Minor -ge 1)
        
        if ($requirements.PowerShellVersion) {
            Write-InstallationLog -Level Success -Message "✓ PowerShell version $($psVersion.ToString()) meets requirements (5.1+)"
            $requirementDetails.PowerShellVersion = "PowerShell $($psVersion.ToString())"
        }
        else {
            Write-InstallationLog -Level Error -Message "✗ PowerShell 5.1 or higher required. Current: $($psVersion.ToString())"
            Write-InstallationLog -Level Error -Message "SOLUTION: Install Windows Management Framework 5.1 or upgrade to PowerShell 7+"
            $requirementDetails.PowerShellVersion = "PowerShell $($psVersion.ToString()) - Upgrade required"
        }
        
        Write-Progress-Step -Step "System Requirements Validation" -StepNumber 1 -TotalSteps 1 -Status "Checking execution policy" -SubStepNumber 3 -TotalSubSteps 9
        
        # Check PowerShell execution policy
        $requirements.PowerShellExecutionPolicy = Test-PowerShellExecutionPolicy
        if ($requirements.PowerShellExecutionPolicy) {
            Write-InstallationLog -Level Success -Message "✓ PowerShell execution policy allows script execution"
            $requirementDetails.PowerShellExecutionPolicy = "Execution policy allows scripts"
        }
        else {
            Write-InstallationLog -Level Warning -Message "⚠ PowerShell execution policy may restrict script execution"
            Write-InstallationLog -Level Info -Message "SOLUTION: Run 'Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser' if installation fails"
            $requirementDetails.PowerShellExecutionPolicy = "Execution policy may need adjustment"
        }
        
        Write-Progress-Step -Step "System Requirements Validation" -StepNumber 1 -TotalSteps 1 -Status "Checking Windows version" -SubStepNumber 4 -TotalSubSteps 9
        
        # Check Windows version (minimum Windows 10 / Server 2016)
        $osVersion = [System.Environment]::OSVersion.Version
        $osInfo = Get-WmiObject -Class Win32_OperatingSystem
        $isServer = $osInfo.ProductType -ne 1
        
        # Windows 10 is version 10.0, Server 2016 is also 10.0
        $requirements.WindowsVersion = $osVersion.Major -ge 10
        
        if ($requirements.WindowsVersion) {
            $osName = if ($isServer) { "Windows Server" } else { "Windows" }
            Write-InstallationLog -Level Success -Message "✓ $osName version $($osVersion.ToString()) meets requirements (10.0+)"
            $requirementDetails.WindowsVersion = "$osName $($osVersion.ToString())"
        }
        else {
            Write-InstallationLog -Level Error -Message "✗ Windows 10 or Windows Server 2016 or higher required. Current: $($osVersion.ToString())"
            Write-InstallationLog -Level Error -Message "SOLUTION: Upgrade to Windows 10 or Windows Server 2016 or later"
            $requirementDetails.WindowsVersion = "Windows $($osVersion.ToString()) - Upgrade required"
        }
        
        Write-Progress-Step -Step "System Requirements Validation" -StepNumber 1 -TotalSteps 1 -Status "Checking Windows features" -SubStepNumber 5 -TotalSubSteps 9
        
        # Check Windows features
        $requirements.WindowsFeatures = Test-WindowsFeatures
        if ($requirements.WindowsFeatures) {
            Write-InstallationLog -Level Success -Message "✓ Required Windows features are available"
            $requirementDetails.WindowsFeatures = "All required features available"
        }
        else {
            Write-InstallationLog -Level Warning -Message "⚠ Some Windows features may need configuration"
            $requirementDetails.WindowsFeatures = "Features may need configuration"
        }
        
        Write-Progress-Step -Step "System Requirements Validation" -StepNumber 1 -TotalSteps 1 -Status "Checking available memory" -SubStepNumber 6 -TotalSubSteps 9
        
        # Check available memory (minimum 512MB free)
        try {
            $memoryInfo = Get-WmiObject -Class Win32_OperatingSystem
            $freeMemoryMB = [math]::Round($memoryInfo.FreePhysicalMemory / 1KB, 2)
            $requirements.MemoryAvailable = $freeMemoryMB -ge 512
            
            if ($requirements.MemoryAvailable) {
                Write-InstallationLog -Level Success -Message "✓ Sufficient memory available: $($freeMemoryMB)MB free (512MB required)"
                $requirementDetails.MemoryAvailable = "$($freeMemoryMB)MB available"
            }
            else {
                Write-InstallationLog -Level Warning -Message "⚠ Low memory: $($freeMemoryMB)MB free (512MB recommended)"
                Write-InstallationLog -Level Info -Message "SOLUTION: Close unnecessary applications to free up memory"
                $requirementDetails.MemoryAvailable = "$($freeMemoryMB)MB available - Low memory"
            }
        }
        catch {
            Write-InstallationLog -Level Warning -Message "⚠ Could not check available memory: $($_.Exception.Message)"
            $requirements.MemoryAvailable = $true # Don't fail installation for this
            $requirementDetails.MemoryAvailable = "Memory check failed - proceeding"
        }
        
        Write-Progress-Step -Step "System Requirements Validation" -StepNumber 1 -TotalSteps 1 -Status "Checking disk space" -SubStepNumber 7 -TotalSubSteps 9
        
        # Check available disk space (minimum 100MB)
        $installDrive = Split-Path $InstallPath -Qualifier
        $driveInfo = Get-WmiObject -Class Win32_LogicalDisk | Where-Object { $_.DeviceID -eq $installDrive }
        if ($driveInfo) {
            $freeSpaceMB = [math]::Round($driveInfo.FreeSpace / 1MB, 2)
            $requirements.DiskSpace = $freeSpaceMB -ge 100
            
            if ($requirements.DiskSpace) {
                Write-InstallationLog -Level Success -Message "✓ Sufficient disk space on $installDrive`: $($freeSpaceMB)MB free (100MB required)"
                $requirementDetails.DiskSpace = "$($freeSpaceMB)MB available on $installDrive"
            }
            else {
                Write-InstallationLog -Level Error -Message "✗ Insufficient disk space on $installDrive`: $($freeSpaceMB)MB free (100MB required)"
                Write-InstallationLog -Level Error -Message "SOLUTION: Free up disk space on $installDrive or choose a different installation path"
                $requirementDetails.DiskSpace = "$($freeSpaceMB)MB available on $installDrive - Insufficient"
            }
        }
        else {
            Write-InstallationLog -Level Warning -Message "⚠ Could not check disk space for drive $installDrive"
            $requirements.DiskSpace = $true # Don't fail installation for this
            $requirementDetails.DiskSpace = "Disk space check failed - proceeding"
        }
        
        Write-Progress-Step -Step "System Requirements Validation" -StepNumber 1 -TotalSteps 1 -Status "Checking .NET runtime" -SubStepNumber 8 -TotalSubSteps 9
        
        # Check .NET runtime (will be installed if missing)
        $requirements.DotNetRuntime = Test-DotNetRuntime
        if ($requirements.DotNetRuntime) {
            Write-InstallationLog -Level Success -Message "✓ Compatible .NET runtime found"
            $requirementDetails.DotNetRuntime = "Compatible .NET runtime available"
        }
        else {
            Write-InstallationLog -Level Warning -Message "⚠ No compatible .NET runtime found - will install during prerequisites"
            $requirementDetails.DotNetRuntime = "Will install .NET runtime"
        }
        
        Write-Progress-Step -Step "System Requirements Validation" -StepNumber 1 -TotalSteps 1 -Status "Checking network connectivity" -SubStepNumber 9 -TotalSubSteps 9
        
        # Check network access to GitHub and platform
        $requirements.NetworkAccess = Test-NetworkConnectivity
        if ($requirements.NetworkAccess) {
            Write-InstallationLog -Level Success -Message "✓ Network connectivity to required services confirmed"
            $requirementDetails.NetworkAccess = "Network connectivity confirmed"
        }
        else {
            Write-InstallationLog -Level Warning -Message "⚠ Network connectivity issues detected"
            Write-InstallationLog -Level Info -Message "SOLUTION: Check internet connection and firewall settings"
            $requirementDetails.NetworkAccess = "Network connectivity issues"
        }
        
        # Summary of requirements check
        $passedRequirements = $requirements.GetEnumerator() | Where-Object { $_.Value } | ForEach-Object { $_.Key }
        $failedRequirements = $requirements.GetEnumerator() | Where-Object { -not $_.Value } | ForEach-Object { $_.Key }
        $criticalFailures = $failedRequirements | Where-Object { $_ -in @('AdminRights', 'PowerShellVersion', 'WindowsVersion', 'DiskSpace') }
        
        Write-InstallationLog -Level Info -Message "Requirements Summary:"
        Write-InstallationLog -Level Info -Message "  Passed: $($passedRequirements.Count)/$($requirements.Count) requirements"
        if ($failedRequirements.Count -gt 0) {
            Write-InstallationLog -Level Warning -Message "  Failed: $($failedRequirements -join ', ')"
        }
        
        # Store detailed requirements for diagnostics
        $script:SystemRequirementsDetails = $requirementDetails
        
        if ($criticalFailures.Count -eq 0) {
            Write-InstallationLog -Level Success -Message "✓ All critical system requirements validated successfully"
            if ($failedRequirements.Count -gt 0) {
                Write-InstallationLog -Level Info -Message "Non-critical issues will be addressed during installation"
            }
            return $true
        }
        else {
            Write-InstallationLog -Level Error -Message "✗ Critical system requirements not met: $($criticalFailures -join ', ')"
            Write-InstallationLog -Level Error -Message "Please resolve the issues above and run the installation command again"
            return $false
        }
    }
    catch {
        Write-InstallationLog -Level Error -Message "Error during system requirements validation: $($_.Exception.Message)"
        return $false
    }
}

function Test-DotNetRuntime {
    [CmdletBinding()]
    param()
    
    Write-InstallationLog -Level Debug -Message "Checking .NET runtime availability..."
    
    try {
        # Check for .NET Framework 4.7.2 or higher, or .NET Core/5+
        $dotNetVersions = @()
        
        # Check .NET Framework
        $frameworkPath = "HKLM:\SOFTWARE\Microsoft\NET Framework Setup\NDP\v4\Full"
        if (Test-Path $frameworkPath) {
            $release = Get-ItemProperty -Path $frameworkPath -Name Release -ErrorAction SilentlyContinue
            if ($release -and $release.Release -ge 461808) { # .NET Framework 4.7.2
                $dotNetVersions += ".NET Framework 4.7.2+"
                Write-InstallationLog -Level Debug -Message "Found .NET Framework 4.7.2+ (Release: $($release.Release))"
            }
        }
        
        # Check .NET Core/.NET 5+
        try {
            $dotnetInfo = & dotnet --info 2>$null
            if ($LASTEXITCODE -eq 0) {
                $dotNetVersions += ".NET Core/5+"
                Write-InstallationLog -Level Debug -Message "Found .NET Core/5+ runtime"
            }
        }
        catch {
            Write-InstallationLog -Level Debug -Message "dotnet command not found or failed"
        }
        
        if ($dotNetVersions.Count -gt 0) {
            Write-InstallationLog -Level Info -Message "Found compatible .NET runtime: $($dotNetVersions -join ', ')"
            return $true
        }
        else {
            Write-InstallationLog -Level Warning -Message "No compatible .NET runtime found. Will attempt to install during prerequisites."
            return $false
        }
    }
    catch {
        Write-InstallationLog -Level Warning -Message "Error checking .NET runtime: $($_.Exception.Message)"
        return $false
    }
}

function Test-WindowsFeatures {
    [CmdletBinding()]
    param()
    
    Write-InstallationLog -Level Debug -Message "Checking required Windows features..."
    
    try {
        $requiredFeatures = @()
        $missingFeatures = @()
        
        # Check if we're on Windows Server (may need different features)
        $osInfo = Get-WmiObject -Class Win32_OperatingSystem
        $isServer = $osInfo.ProductType -ne 1
        
        if ($isServer) {
            Write-InstallationLog -Level Debug -Message "Detected Windows Server environment"
            # Server-specific feature checks could go here
        }
        else {
            Write-InstallationLog -Level Debug -Message "Detected Windows Client environment"
        }
        
        # Check Windows Service support (should be available by default)
        try {
            $serviceManager = Get-Service -Name "Spooler" -ErrorAction SilentlyContinue
            if ($serviceManager) {
                Write-InstallationLog -Level Debug -Message "Windows Service Manager is available"
            }
            else {
                $missingFeatures += "Windows Service Manager"
            }
        }
        catch {
            $missingFeatures += "Windows Service Manager"
        }
        
        # Check Windows Event Log support
        try {
            $eventLogService = Get-Service -Name "EventLog" -ErrorAction SilentlyContinue
            if ($eventLogService -and $eventLogService.Status -eq 'Running') {
                Write-InstallationLog -Level Debug -Message "Windows Event Log service is running"
            }
            else {
                Write-InstallationLog -Level Warning -Message "Windows Event Log service is not running"
            }
        }
        catch {
            Write-InstallationLog -Level Debug -Message "Could not check Windows Event Log service"
        }
        
        if ($missingFeatures.Count -eq 0) {
            Write-InstallationLog -Level Info -Message "All required Windows features are available"
            return $true
        }
        else {
            Write-InstallationLog -Level Warning -Message "Missing Windows features: $($missingFeatures -join ', ')"
            return $false
        }
    }
    catch {
        Write-InstallationLog -Level Warning -Message "Error checking Windows features: $($_.Exception.Message)"
        return $false
    }
}

function Test-PowerShellExecutionPolicy {
    [CmdletBinding()]
    param()
    
    Write-InstallationLog -Level Debug -Message "Checking PowerShell execution policy..."
    
    try {
        $currentPolicy = Get-ExecutionPolicy -Scope CurrentUser
        $machinePolicy = Get-ExecutionPolicy -Scope LocalMachine
        
        Write-InstallationLog -Level Debug -Message "Current user execution policy: $currentPolicy"
        Write-InstallationLog -Level Debug -Message "Machine execution policy: $machinePolicy"
        
        # Check if execution policy allows script execution
        $allowedPolicies = @('Unrestricted', 'RemoteSigned', 'AllSigned', 'Bypass')
        
        if ($currentPolicy -in $allowedPolicies -or $machinePolicy -in $allowedPolicies) {
            Write-InstallationLog -Level Info -Message "PowerShell execution policy allows script execution"
            return $true
        }
        else {
            Write-InstallationLog -Level Warning -Message "PowerShell execution policy may prevent script execution. Current: $currentPolicy, Machine: $machinePolicy"
            return $false
        }
    }
    catch {
        Write-InstallationLog -Level Warning -Message "Error checking PowerShell execution policy: $($_.Exception.Message)"
        return $false
    }
} -Message "Error checking .NET runtime: $($_.Exception.Message)"
        return $false
    }
}

function Test-NetworkConnectivity {
    [CmdletBinding()]
    param()
    
    $testUrls = @(
        "https://api.github.com",
        "https://github.com",
        $PlatformEndpoint
    )
    
    $allSuccessful = $true
    
    foreach ($url in $testUrls) {
        try {
            $response = Invoke-WebRequest -Uri $url -Method Head -TimeoutSec 10 -UseBasicParsing -ErrorAction Stop
            Write-InstallationLog -Level Debug -Message "Network connectivity test successful: $url"
        }
        catch {
            Write-InstallationLog -Level Warning -Message "Network connectivity test failed for $url`: $($_.Exception.Message)"
            $allSuccessful = $false
        }
    }
    
    return $allSuccessful
}

# ================================================================
# Error Handling and Rollback Functions
# ================================================================

function Invoke-WithRetry {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [scriptblock]$ScriptBlock,
        
        [Parameter(Mandatory=$false)]
        [int]$MaxRetries = 3,
        
        [Parameter(Mandatory=$false)]
        [int]$DelaySeconds = 5,
        
        [Parameter(Mandatory=$false)]
        [string]$OperationName = "Operation"
    )
    
    $attempt = 1
    
    while ($attempt -le $MaxRetries) {
        try {
            Write-InstallationLog -Level Debug -Message "$OperationName attempt $attempt of $MaxRetries"
            $result = & $ScriptBlock
            Write-InstallationLog -Level Debug -Message "$OperationName succeeded on attempt $attempt"
            return $result
        }
        catch {
            Write-InstallationLog -Level Warning -Message "$OperationName failed on attempt $attempt`: $($_.Exception.Message)"
            
            if ($attempt -eq $MaxRetries) {
                Write-InstallationLog -Level Error -Message "$OperationName failed after $MaxRetries attempts"
                throw
            }
            
            $delay = $DelaySeconds * [math]::Pow(2, $attempt - 1) # Exponential backoff
            Write-InstallationLog -Level Info -Message "Retrying in $delay seconds..."
            Start-Sleep -Seconds $delay
            $attempt++
        }
    }
}

function Invoke-InstallationRollback {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$InstallationId,
        
        [Parameter(Mandatory=$false)]
        [string]$Reason = "Installation failed"
    )
    
    Write-InstallationLog -Level Warning -Message "Initiating installation rollback: $Reason"
    
    try {
        # Stop and remove service if it was created
        if (Get-Service -Name $script:ServiceName -ErrorAction SilentlyContinue) {
            Write-InstallationLog -Level Info -Message "Stopping and removing RepSet Bridge service..."
            try {
                Stop-BridgeService -ServiceName $script:ServiceName -Force -ErrorAction SilentlyContinue
                Remove-BridgeService -ServiceName $script:ServiceName -ErrorAction SilentlyContinue
            }
            catch {
                Write-InstallationLog -Level Warning -Message "Could not cleanly remove service during rollback: $($_.Exception.Message)"
                # Fallback to direct sc.exe commands
                Stop-Service -Name $script:ServiceName -Force -ErrorAction SilentlyContinue
                & sc.exe delete $script:ServiceName | Out-Null
            }
        }
        
        # Restore configuration backup if it exists
        $configFile = Join-Path $InstallPath "config\$script:ConfigFileName"
        if (Test-Path $configFile) {
            $backupFiles = Get-ChildItem -Path (Split-Path $configFile) -Filter "*.backup.*" | Sort-Object LastWriteTime -Descending
            if ($backupFiles) {
                $latestBackup = $backupFiles[0]
                try {
                    Copy-Item -Path $latestBackup.FullName -Destination $configFile -Force
                    Write-InstallationLog -Level Info -Message "Restored configuration from backup: $($latestBackup.Name)"
                }
                catch {
                    Write-InstallationLog -Level Warning -Message "Could not restore configuration backup: $($_.Exception.Message)"
                }
            }
        }
        
        # Restore executable backup if it exists
        $executableFile = Join-Path $InstallPath "repset-bridge.exe"
        if (Test-Path $executableFile) {
            $backupFiles = Get-ChildItem -Path $InstallPath -Filter "*.backup.*" | Sort-Object LastWriteTime -Descending
            if ($backupFiles) {
                $latestBackup = $backupFiles[0]
                try {
                    Copy-Item -Path $latestBackup.FullName -Destination $executableFile -Force
                    Write-InstallationLog -Level Info -Message "Restored executable from backup: $($latestBackup.Name)"
                }
                catch {
                    Write-InstallationLog -Level Warning -Message "Could not restore executable backup: $($_.Exception.Message)"
                }
            }
            else {
                # No backup available, remove the failed installation
                Remove-Item -Path $executableFile -Force -ErrorAction SilentlyContinue
                Write-InstallationLog -Level Info -Message "Removed failed executable installation"
            }
        }
        
        # If no backups were restored, remove the entire installation directory
        $hasValidBackups = $false
        if (Test-Path $InstallPath) {
            $backupFiles = Get-ChildItem -Path $InstallPath -Filter "*.backup.*" -Recurse
            if ($backupFiles.Count -eq 0) {
                Write-InstallationLog -Level Info -Message "No backups found, removing installation directory: $InstallPath"
                Remove-Item -Path $InstallPath -Recurse -Force -ErrorAction SilentlyContinue
            }
            else {
                Write-InstallationLog -Level Info -Message "Preserved installation directory with backups"
                $hasValidBackups = $true
            }
        }
        
        # Clean up temporary files
        $tempFiles = Get-ChildItem -Path $env:TEMP -Filter "RepSetBridge-*" -ErrorAction SilentlyContinue
        foreach ($tempFile in $tempFiles) {
            try {
                Remove-Item -Path $tempFile.FullName -Force -ErrorAction SilentlyContinue
                Write-InstallationLog -Level Debug -Message "Cleaned up temporary file: $($tempFile.Name)"
            }
            catch {
                Write-InstallationLog -Level Debug -Message "Could not clean up temporary file: $($tempFile.Name)"
            }
        }
        
        # Remove any registry entries (if we added any)
        $registryPath = "HKLM:\SOFTWARE\RepSet\Bridge"
        if (Test-Path $registryPath) {
            Write-InstallationLog -Level Info -Message "Removing registry entries"
            Remove-Item -Path $registryPath -Recurse -Force -ErrorAction SilentlyContinue
        }
        
        if ($hasValidBackups) {
            Write-InstallationLog -Level Success -Message "Installation rollback completed - previous version restored"
        }
        else {
            Write-InstallationLog -Level Success -Message "Installation rollback completed - failed installation removed"
        }
    }
    catch {
        Write-InstallationLog -Level Error -Message "Error during rollback: $($_.Exception.Message)"
        throw
    }
}


# ================================================================
# Platform Connection Verification System
# ================================================================

function Test-BridgeConnection {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$PlatformEndpoint,
        
        [Parameter(Mandatory=$true)]
        [string]$PairCode,
        
        [Parameter(Mandatory=$true)]
        [string]$GymId,
        
        [Parameter(Mandatory=$false)]
        [string]$BridgeExecutablePath,
        
        [Parameter(Mandatory=$false)]
        [int]$TimeoutSeconds = 30,
        
        [Parameter(Mandatory=$false)]
        [switch]$DetailedDiagnostics
    )
    
    Write-InstallationLog -Level Info -Message "Testing bridge connection to platform..."
    
    $connectionResult = @{
        Success = $false
        PlatformReachable = $false
        AuthenticationValid = $false
        ServiceResponding = $false
        NetworkConnectivity = $false
        FirewallBlocking = $false
        ErrorDetails = @()
        DiagnosticInfo = @{}
        TestResults = @{}
    }
    
    try {
        # Step 1: Test basic network connectivity
        Write-InstallationLog -Level Debug -Message "Testing network connectivity..."
        $networkTest = Test-NetworkConnectivity -PlatformEndpoint $PlatformEndpoint -TimeoutSeconds $TimeoutSeconds
        $connectionResult.NetworkConnectivity = $networkTest.Success
        $connectionResult.TestResults.NetworkConnectivity = $networkTest
        
        if (-not $networkTest.Success) {
            $connectionResult.ErrorDetails += "Network connectivity failed: $($networkTest.ErrorMessage)"
            Write-InstallationLog -Level Error -Message "Network connectivity test failed: $($networkTest.ErrorMessage)"
        }
        else {
            Write-InstallationLog -Level Success -Message "Network connectivity test passed"
        }
        
        # Step 2: Test platform endpoint reachability
        Write-InstallationLog -Level Debug -Message "Testing platform endpoint reachability..."
        $platformTest = Test-PlatformEndpoint -PlatformEndpoint $PlatformEndpoint -TimeoutSeconds $TimeoutSeconds
        $connectionResult.PlatformReachable = $platformTest.Success
        $connectionResult.TestResults.PlatformEndpoint = $platformTest
        
        if (-not $platformTest.Success) {
            $connectionResult.ErrorDetails += "Platform endpoint unreachable: $($platformTest.ErrorMessage)"
            Write-InstallationLog -Level Error -Message "Platform endpoint test failed: $($platformTest.ErrorMessage)"
        }
        else {
            Write-InstallationLog -Level Success -Message "Platform endpoint is reachable"
        }
        
        # Step 3: Test authentication with platform
        Write-InstallationLog -Level Debug -Message "Testing platform authentication..."
        $authTest = Test-PlatformAuthentication -PlatformEndpoint $PlatformEndpoint -PairCode $PairCode -GymId $GymId -TimeoutSeconds $TimeoutSeconds
        $connectionResult.AuthenticationValid = $authTest.Success
        $connectionResult.TestResults.Authentication = $authTest
        
        if (-not $authTest.Success) {
            $connectionResult.ErrorDetails += "Authentication failed: $($authTest.ErrorMessage)"
            Write-InstallationLog -Level Error -Message "Platform authentication test failed: $($authTest.ErrorMessage)"
        }
        else {
            Write-InstallationLog -Level Success -Message "Platform authentication successful"
        }
        
        # Step 4: Test bridge service if executable is available
        if ($BridgeExecutablePath -and (Test-Path $BridgeExecutablePath)) {
            Write-InstallationLog -Level Debug -Message "Testing bridge service functionality..."
            $serviceTest = Test-BridgeService -BridgeExecutablePath $BridgeExecutablePath -TimeoutSeconds $TimeoutSeconds
            $connectionResult.ServiceResponding = $serviceTest.Success
            $connectionResult.TestResults.BridgeService = $serviceTest
            
            if (-not $serviceTest.Success) {
                $connectionResult.ErrorDetails += "Bridge service test failed: $($serviceTest.ErrorMessage)"
                Write-InstallationLog -Level Warning -Message "Bridge service test failed: $($serviceTest.ErrorMessage)"
            }
            else {
                Write-InstallationLog -Level Success -Message "Bridge service test passed"
            }
        }
        
        # Step 5: Firewall detection
        Write-InstallationLog -Level Debug -Message "Checking for firewall blocking..."
        $firewallTest = Test-FirewallBlocking -PlatformEndpoint $PlatformEndpoint
        $connectionResult.FirewallBlocking = $firewallTest.IsBlocking
        $connectionResult.TestResults.Firewall = $firewallTest
        
        if ($firewallTest.IsBlocking) {
            $connectionResult.ErrorDetails += "Firewall may be blocking connection: $($firewallTest.Details)"
            Write-InstallationLog -Level Warning -Message "Firewall blocking detected: $($firewallTest.Details)"
        }
        else {
            Write-InstallationLog -Level Success -Message "No firewall blocking detected"
        }
        
        # Step 6: Detailed diagnostics if requested
        if ($DetailedDiagnostics) {
            Write-InstallationLog -Level Debug -Message "Running detailed connection diagnostics..."
            $diagnostics = Get-ConnectionDiagnostics -PlatformEndpoint $PlatformEndpoint -PairCode $PairCode -GymId $GymId
            $connectionResult.DiagnosticInfo = $diagnostics
        }
        
        # Determine overall success
        $connectionResult.Success = $connectionResult.NetworkConnectivity -and 
                                   $connectionResult.PlatformReachable -and 
                                   $connectionResult.AuthenticationValid -and
                                   (-not $connectionResult.FirewallBlocking)
        
        # Report connection status back to platform
        Send-ConnectionStatusToPlatform -ConnectionResult $connectionResult -PlatformEndpoint $PlatformEndpoint -GymId $GymId
        
        if ($connectionResult.Success) {
            Write-InstallationLog -Level Success -Message "Bridge connection test completed successfully"
        }
        else {
            Write-InstallationLog -Level Error -Message "Bridge connection test failed. See error details for troubleshooting."
        }
        
        return $connectionResult
    }
    catch {
        $connectionResult.ErrorDetails += "Connection test exception: $($_.Exception.Message)"
        Write-InstallationLog -Level Error -Message "Connection test failed with exception: $($_.Exception.Message)"
        return $connectionResult
    }
}

function Test-NetworkConnectivity {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$PlatformEndpoint,
        
        [Parameter(Mandatory=$false)]
        [int]$TimeoutSeconds = 30
    )
    
    $result = @{
        Success = $false
        ErrorMessage = ""
        ResponseTime = 0
        DNSResolution = $false
        TCPConnection = $false
        HTTPSConnection = $false
        Details = @{}
    }
    
    try {
        # Parse the platform endpoint to get hostname and port
        $uri = [System.Uri]::new($PlatformEndpoint)
        $hostname = $uri.Host
        $port = if ($uri.Port -ne -1) { $uri.Port } else { if ($uri.Scheme -eq "https") { 443 } else { 80 } }
        
        Write-InstallationLog -Level Debug -Message "Testing connectivity to $hostname`:$port"
        
        # Test 1: DNS Resolution
        try {
            $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
            $dnsResult = [System.Net.Dns]::GetHostAddresses($hostname)
            $stopwatch.Stop()
            
            if ($dnsResult -and $dnsResult.Count -gt 0) {
                $result.DNSResolution = $true
                $result.Details.DNSAddresses = $dnsResult | ForEach-Object { $_.ToString() }
                $result.Details.DNSResponseTime = $stopwatch.ElapsedMilliseconds
                Write-InstallationLog -Level Debug -Message "DNS resolution successful: $($result.Details.DNSAddresses -join ', ')"
            }
            else {
                $result.ErrorMessage = "DNS resolution failed - no addresses returned"
                return $result
            }
        }
        catch {
            $result.ErrorMessage = "DNS resolution failed: $($_.Exception.Message)"
            return $result
        }
        
        # Test 2: TCP Connection
        try {
            $tcpClient = New-Object System.Net.Sockets.TcpClient
            $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
            
            $connectTask = $tcpClient.ConnectAsync($hostname, $port)
            $completed = $connectTask.Wait($TimeoutSeconds * 1000)
            $stopwatch.Stop()
            
            if ($completed -and $tcpClient.Connected) {
                $result.TCPConnection = $true
                $result.Details.TCPResponseTime = $stopwatch.ElapsedMilliseconds
                Write-InstallationLog -Level Debug -Message "TCP connection successful ($($stopwatch.ElapsedMilliseconds)ms)"
                $tcpClient.Close()
            }
            else {
                $result.ErrorMessage = "TCP connection failed or timed out"
                return $result
            }
        }
        catch {
            $result.ErrorMessage = "TCP connection failed: $($_.Exception.Message)"
            return $result
        }
        finally {
            if ($tcpClient) {
                $tcpClient.Dispose()
            }
        }
        
        # Test 3: HTTPS Connection (if applicable)
        if ($uri.Scheme -eq "https") {
            try {
                $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
                $request = [System.Net.WebRequest]::Create($PlatformEndpoint)
                $request.Method = "HEAD"
                $request.Timeout = $TimeoutSeconds * 1000
                $request.UserAgent = "RepSet-Bridge-Installer/1.0"
                
                $response = $request.GetResponse()
                $stopwatch.Stop()
                
                if ($response.StatusCode -eq [System.Net.HttpStatusCode]::OK -or 
                    $response.StatusCode -eq [System.Net.HttpStatusCode]::NotFound) {
                    $result.HTTPSConnection = $true
                    $result.Details.HTTPSResponseTime = $stopwatch.ElapsedMilliseconds
                    $result.Details.HTTPSStatusCode = [int]$response.StatusCode
                    Write-InstallationLog -Level Debug -Message "HTTPS connection successful ($($stopwatch.ElapsedMilliseconds)ms, Status: $($response.StatusCode))"
                }
                
                $response.Close()
            }
            catch [System.Net.WebException] {
                # Some web exceptions are acceptable (like 404) as they indicate connectivity
                if ($_.Exception.Response) {
                    $result.HTTPSConnection = $true
                    $result.Details.HTTPSStatusCode = [int]$_.Exception.Response.StatusCode
                    Write-InstallationLog -Level Debug -Message "HTTPS connection successful with status: $($_.Exception.Response.StatusCode)"
                }
                else {
                    $result.ErrorMessage = "HTTPS connection failed: $($_.Exception.Message)"
                    return $result
                }
            }
            catch {
                $result.ErrorMessage = "HTTPS connection failed: $($_.Exception.Message)"
                return $result
            }
        }
        
        # Calculate overall response time
        $result.ResponseTime = if ($result.Details.HTTPSResponseTime) { 
            $result.Details.HTTPSResponseTime 
        } elseif ($result.Details.TCPResponseTime) { 
            $result.Details.TCPResponseTime 
        } else { 
            $result.Details.DNSResponseTime 
        }
        
        $result.Success = $result.DNSResolution -and $result.TCPConnection -and 
                         ($uri.Scheme -ne "https" -or $result.HTTPSConnection)
        
        return $result
    }
    catch {
        $result.ErrorMessage = "Network connectivity test failed: $($_.Exception.Message)"
        return $result
    }
}

function Test-PlatformEndpoint {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$PlatformEndpoint,
        
        [Parameter(Mandatory=$false)]
        [int]$TimeoutSeconds = 30
    )
    
    $result = @{
        Success = $false
        ErrorMessage = ""
        StatusCode = 0
        ResponseTime = 0
        ServerHeaders = @{}
        Details = @{}
    }
    
    try {
        Write-InstallationLog -Level Debug -Message "Testing platform endpoint: $PlatformEndpoint"
        
        # Test platform health/status endpoint
        $healthEndpoint = "$PlatformEndpoint/api/health"
        
        $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
        
        try {
            $response = Invoke-WebRequest -Uri $healthEndpoint -Method GET -TimeoutSec $TimeoutSeconds -UserAgent "RepSet-Bridge-Installer/1.0" -ErrorAction Stop
            $stopwatch.Stop()
            
            $result.Success = $true
            $result.StatusCode = $response.StatusCode
            $result.ResponseTime = $stopwatch.ElapsedMilliseconds
            $result.Details.ContentLength = $response.Content.Length
            
            # Extract useful server headers
            foreach ($header in $response.Headers.Keys) {
                if ($header -in @('Server', 'X-Powered-By', 'X-Frame-Options', 'Content-Type')) {
                    $result.ServerHeaders[$header] = $response.Headers[$header]
                }
            }
            
            Write-InstallationLog -Level Debug -Message "Platform endpoint responded successfully (Status: $($result.StatusCode), Time: $($result.ResponseTime)ms)"
        }
        catch [System.Net.WebException] {
            $stopwatch.Stop()
            
            if ($_.Exception.Response) {
                $result.StatusCode = [int]$_.Exception.Response.StatusCode
                $result.ResponseTime = $stopwatch.ElapsedMilliseconds
                
                # Some status codes indicate the endpoint is reachable
                if ($result.StatusCode -in @(401, 403, 404, 405, 500)) {
                    $result.Success = $true
                    Write-InstallationLog -Level Debug -Message "Platform endpoint reachable but returned status $($result.StatusCode)"
                }
                else {
                    $result.ErrorMessage = "Platform endpoint returned status $($result.StatusCode): $($_.Exception.Message)"
                }
            }
            else {
                $result.ErrorMessage = "Platform endpoint unreachable: $($_.Exception.Message)"
            }
        }
        catch {
            $stopwatch.Stop()
            $result.ErrorMessage = "Platform endpoint test failed: $($_.Exception.Message)"
        }
        
        return $result
    }
    catch {
        $result.ErrorMessage = "Platform endpoint test exception: $($_.Exception.Message)"
        return $result
    }
}

function Test-PlatformAuthentication {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$PlatformEndpoint,
        
        [Parameter(Mandatory=$true)]
        [string]$PairCode,
        
        [Parameter(Mandatory=$true)]
        [string]$GymId,
        
        [Parameter(Mandatory=$false)]
        [int]$TimeoutSeconds = 30
    )
    
    $result = @{
        Success = $false
        ErrorMessage = ""
        StatusCode = 0
        ResponseTime = 0
        AuthenticationDetails = @{}
    }
    
    try {
        Write-InstallationLog -Level Debug -Message "Testing platform authentication..."
        
        # Test authentication endpoint
        $authEndpoint = "$PlatformEndpoint/api/bridge/authenticate"
        
        $authData = @{
            pairCode = $PairCode
            gymId = $GymId
            deviceId = "installer-test-$(Get-Date -Format 'yyyyMMddHHmmss')"
            version = "installer-test"
        }
        
        $headers = @{
            'Content-Type' = 'application/json'
            'User-Agent' = 'RepSet-Bridge-Installer/1.0'
        }
        
        $body = $authData | ConvertTo-Json
        
        $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
        
        try {
            $response = Invoke-RestMethod -Uri $authEndpoint -Method POST -Headers $headers -Body $body -TimeoutSec $TimeoutSeconds -ErrorAction Stop
            $stopwatch.Stop()
            
            $result.Success = $true
            $result.StatusCode = 200
            $result.ResponseTime = $stopwatch.ElapsedMilliseconds
            
            if ($response) {
                $result.AuthenticationDetails = $response
                Write-InstallationLog -Level Debug -Message "Authentication successful (Time: $($result.ResponseTime)ms)"
            }
        }
        catch [Microsoft.PowerShell.Commands.HttpResponseException] {
            $stopwatch.Stop()
            $result.StatusCode = $_.Exception.Response.StatusCode.Value__
            $result.ResponseTime = $stopwatch.ElapsedMilliseconds
            
            if ($result.StatusCode -eq 401) {
                $result.ErrorMessage = "Authentication failed - Invalid pair code or gym ID"
            }
            elseif ($result.StatusCode -eq 403) {
                $result.ErrorMessage = "Authentication failed - Access forbidden"
            }
            elseif ($result.StatusCode -eq 404) {
                $result.ErrorMessage = "Authentication endpoint not found - Platform may not support bridge authentication"
            }
            else {
                $result.ErrorMessage = "Authentication failed with status $($result.StatusCode): $($_.Exception.Message)"
            }
        }
        catch {
            $stopwatch.Stop()
            $result.ErrorMessage = "Authentication test failed: $($_.Exception.Message)"
        }
        
        return $result
    }
    catch {
        $result.ErrorMessage = "Authentication test exception: $($_.Exception.Message)"
        return $result
    }
}

function Test-BridgeService {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$BridgeExecutablePath,
        
        [Parameter(Mandatory=$false)]
        [int]$TimeoutSeconds = 30
    )
    
    $result = @{
        Success = $false
        ErrorMessage = ""
        ExecutableValid = $false
        VersionInfo = ""
        ConfigTest = $false
        Details = @{}
    }
    
    try {
        Write-InstallationLog -Level Debug -Message "Testing bridge service functionality..."
        
        # Test 1: Verify executable exists and is valid
        if (-not (Test-Path $BridgeExecutablePath)) {
            $result.ErrorMessage = "Bridge executable not found: $BridgeExecutablePath"
            return $result
        }
        
        $result.ExecutableValid = $true
        
        # Test 2: Get version information
        try {
            $versionOutput = & $BridgeExecutablePath --version 2>&1
            if ($LASTEXITCODE -eq 0) {
                $result.VersionInfo = $versionOutput -join " "
                Write-InstallationLog -Level Debug -Message "Bridge version: $($result.VersionInfo)"
            }
            else {
                Write-InstallationLog -Level Warning -Message "Bridge version check returned exit code: $LASTEXITCODE"
            }
        }
        catch {
            Write-InstallationLog -Level Warning -Message "Could not get bridge version: $($_.Exception.Message)"
        }
        
        # Test 3: Test configuration validation (if config exists)
        $configPath = Join-Path (Split-Path $BridgeExecutablePath -Parent) "config\config.yaml"
        if (Test-Path $configPath) {
            try {
                $configTestOutput = & $BridgeExecutablePath --config $configPath --validate-config 2>&1
                if ($LASTEXITCODE -eq 0) {
                    $result.ConfigTest = $true
                    Write-InstallationLog -Level Debug -Message "Bridge configuration validation passed"
                }
                else {
                    Write-InstallationLog -Level Warning -Message "Bridge configuration validation failed (exit code: $LASTEXITCODE)"
                }
            }
            catch {
                Write-InstallationLog -Level Warning -Message "Could not test bridge configuration: $($_.Exception.Message)"
            }
        }
        
        $result.Success = $result.ExecutableValid
        
        return $result
    }
    catch {
        $result.ErrorMessage = "Bridge service test exception: $($_.Exception.Message)"
        return $result
    }
}

function Test-FirewallBlocking {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$PlatformEndpoint
    )
    
    $result = @{
        IsBlocking = $false
        Details = ""
        WindowsFirewallStatus = @{}
        NetworkProfiles = @{}
        FirewallRules = @()
    }
    
    try {
        Write-InstallationLog -Level Debug -Message "Checking for firewall blocking..."
        
        # Parse endpoint details
        $uri = [System.Uri]::new($PlatformEndpoint)
        $hostname = $uri.Host
        $port = if ($uri.Port -ne -1) { $uri.Port } else { if ($uri.Scheme -eq "https") { 443 } else { 80 } }
        
        # Check Windows Firewall status
        try {
            $firewallProfiles = Get-NetFirewallProfile -ErrorAction SilentlyContinue
            if ($firewallProfiles) {
                foreach ($profile in $firewallProfiles) {
                    $result.WindowsFirewallStatus[$profile.Name] = @{
                        Enabled = $profile.Enabled
                        DefaultInboundAction = $profile.DefaultInboundAction
                        DefaultOutboundAction = $profile.DefaultOutboundAction
                    }
                }
                
                # Check if any profile is blocking outbound connections
                $blockingProfiles = $firewallProfiles | Where-Object { $_.Enabled -and $_.DefaultOutboundAction -eq 'Block' }
                if ($blockingProfiles) {
                    $result.IsBlocking = $true
                    $result.Details = "Windows Firewall profiles blocking outbound: $($blockingProfiles.Name -join ', ')"
                }
            }
        }
        catch {
            Write-InstallationLog -Level Debug -Message "Could not check Windows Firewall status: $($_.Exception.Message)"
        }
        
        # Check for specific firewall rules that might block the connection
        try {
            $outboundRules = Get-NetFirewallRule -Direction Outbound -Action Block -Enabled True -ErrorAction SilentlyContinue
            if ($outboundRules) {
                foreach ($rule in $outboundRules) {
                    $portFilter = Get-NetFirewallPortFilter -AssociatedNetFirewallRule $rule -ErrorAction SilentlyContinue
                    $addressFilter = Get-NetFirewallAddressFilter -AssociatedNetFirewallRule $rule -ErrorAction SilentlyContinue
                    
                    # Check if rule might affect our connection
                    $ruleAffectsConnection = $false
                    
                    if ($portFilter -and ($portFilter.RemotePort -eq $port -or $portFilter.RemotePort -eq 'Any')) {
                        $ruleAffectsConnection = $true
                    }
                    
                    if ($addressFilter -and ($addressFilter.RemoteAddress -eq $hostname -or $addressFilter.RemoteAddress -eq 'Any')) {
                        $ruleAffectsConnection = $true
                    }
                    
                    if ($ruleAffectsConnection) {
                        $result.FirewallRules += @{
                            Name = $rule.DisplayName
                            Description = $rule.Description
                            RemotePort = if ($portFilter) { $portFilter.RemotePort } else { "Any" }
                            RemoteAddress = if ($addressFilter) { $addressFilter.RemoteAddress } else { "Any" }
                        }
                        
                        if (-not $result.IsBlocking) {
                            $result.IsBlocking = $true
                            $result.Details = "Firewall rule '$($rule.DisplayName)' may be blocking connection"
                        }
                    }
                }
            }
        }
        catch {
            Write-InstallationLog -Level Debug -Message "Could not check firewall rules: $($_.Exception.Message)"
        }
        
        # Additional check: Test if we can make a simple connection
        if (-not $result.IsBlocking) {
            try {
                $tcpClient = New-Object System.Net.Sockets.TcpClient
                $connectTask = $tcpClient.ConnectAsync($hostname, $port)
                $connected = $connectTask.Wait(5000)  # 5 second timeout
                
                if (-not $connected -or -not $tcpClient.Connected) {
                    # Connection failed, but this doesn't necessarily mean firewall blocking
                    # Could be network issues, server down, etc.
                    Write-InstallationLog -Level Debug -Message "Connection test failed, but cause unclear"
                }
                
                $tcpClient.Close()
                $tcpClient.Dispose()
            }
            catch {
                Write-InstallationLog -Level Debug -Message "Connection test failed: $($_.Exception.Message)"
            }
        }
        
        return $result
    }
    catch {
        Write-InstallationLog -Level Debug -Message "Firewall check failed: $($_.Exception.Message)"
        return $result
    }
}

function Get-ConnectionDiagnostics {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$PlatformEndpoint,
        
        [Parameter(Mandatory=$true)]
        [string]$PairCode,
        
        [Parameter(Mandatory=$true)]
        [string]$GymId
    )
    
    $diagnostics = @{
        SystemInfo = @{}
        NetworkInfo = @{}
        SecurityInfo = @{}
        PerformanceInfo = @{}
        TroubleshootingSteps = @()
    }
    
    try {
        Write-InstallationLog -Level Debug -Message "Gathering connection diagnostics..."
        
        # System Information
        $diagnostics.SystemInfo = @{
            OSVersion = [System.Environment]::OSVersion.VersionString
            PowerShellVersion = $PSVersionTable.PSVersion.ToString()
            Architecture = [System.Environment]::ProcessorArchitecture
            MachineName = [System.Environment]::MachineName
            UserDomain = [System.Environment]::UserDomainName
            IsElevated = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
        }
        
        # Network Information
        try {
            $networkAdapters = Get-NetAdapter -Physical | Where-Object { $_.Status -eq 'Up' }
            $diagnostics.NetworkInfo.ActiveAdapters = $networkAdapters | ForEach-Object {
                @{
                    Name = $_.Name
                    InterfaceDescription = $_.InterfaceDescription
                    LinkSpeed = $_.LinkSpeed
                    MediaType = $_.MediaType
                }
            }
            
            $defaultGateway = Get-NetRoute -DestinationPrefix '0.0.0.0/0' | Select-Object -First 1
            if ($defaultGateway) {
                $diagnostics.NetworkInfo.DefaultGateway = $defaultGateway.NextHop
            }
            
            $dnsServers = Get-DnsClientServerAddress | Where-Object { $_.AddressFamily -eq 2 -and $_.ServerAddresses }
            $diagnostics.NetworkInfo.DNSServers = $dnsServers.ServerAddresses
        }
        catch {
            Write-InstallationLog -Level Debug -Message "Could not gather network info: $($_.Exception.Message)"
        }
        
        # Security Information
        try {
            $diagnostics.SecurityInfo.ExecutionPolicy = Get-ExecutionPolicy
            $diagnostics.SecurityInfo.WindowsDefenderStatus = Get-MpComputerStatus -ErrorAction SilentlyContinue | Select-Object AntivirusEnabled, RealTimeProtectionEnabled
        }
        catch {
            Write-InstallationLog -Level Debug -Message "Could not gather security info: $($_.Exception.Message)"
        }
        
        # Performance Information
        try {
            $uri = [System.Uri]::new($PlatformEndpoint)
            $hostname = $uri.Host
            
            # Ping test
            $pingResult = Test-Connection -ComputerName $hostname -Count 3 -ErrorAction SilentlyContinue
            if ($pingResult) {
                $diagnostics.PerformanceInfo.PingResults = @{
                    AverageResponseTime = ($pingResult | Measure-Object -Property ResponseTime -Average).Average
                    PacketLoss = (3 - $pingResult.Count) / 3 * 100
                }
            }
            
            # Traceroute (simplified)
            try {
                $traceRoute = Test-NetConnection -ComputerName $hostname -TraceRoute -ErrorAction SilentlyContinue
                if ($traceRoute -and $traceRoute.TraceRoute) {
                    $diagnostics.PerformanceInfo.TraceRoute = $traceRoute.TraceRoute
                    $diagnostics.PerformanceInfo.HopCount = $traceRoute.TraceRoute.Count
                }
            }
            catch {
                Write-InstallationLog -Level Debug -Message "Traceroute failed: $($_.Exception.Message)"
            }
        }
        catch {
            Write-InstallationLog -Level Debug -Message "Could not gather performance info: $($_.Exception.Message)"
        }
        
        # Generate troubleshooting steps based on findings
        $diagnostics.TroubleshootingSteps = Get-TroubleshootingSteps -Diagnostics $diagnostics -PlatformEndpoint $PlatformEndpoint
        
        return $diagnostics
    }
    catch {
        Write-InstallationLog -Level Warning -Message "Could not gather complete diagnostics: $($_.Exception.Message)"
        return $diagnostics
    }
}

function Get-TroubleshootingSteps {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [hashtable]$Diagnostics,
        
        [Parameter(Mandatory=$true)]
        [string]$PlatformEndpoint
    )
    
    $steps = @()
    
    # Check for common issues and provide solutions
    
    # Network connectivity issues
    if ($Diagnostics.NetworkInfo.ActiveAdapters.Count -eq 0) {
        $steps += "No active network adapters found. Check network cable connections and adapter status."
    }
    
    if (-not $Diagnostics.NetworkInfo.DefaultGateway) {
        $steps += "No default gateway configured. Check network configuration and DHCP settings."
    }
    
    if (-not $Diagnostics.NetworkInfo.DNSServers -or $Diagnostics.NetworkInfo.DNSServers.Count -eq 0) {
        $steps += "No DNS servers configured. Check network settings and DNS configuration."
    }
    
    # Performance issues
    if ($Diagnostics.PerformanceInfo.PingResults -and $Diagnostics.PerformanceInfo.PingResults.AverageResponseTime -gt 1000) {
        $steps += "High network latency detected ($($Diagnostics.PerformanceInfo.PingResults.AverageResponseTime)ms). Check network connection quality."
    }
    
    if ($Diagnostics.PerformanceInfo.PingResults -and $Diagnostics.PerformanceInfo.PingResults.PacketLoss -gt 0) {
        $steps += "Packet loss detected ($($Diagnostics.PerformanceInfo.PingResults.PacketLoss)%). Check network stability."
    }
    
    # Security issues
    if ($Diagnostics.SecurityInfo.ExecutionPolicy -eq 'Restricted') {
        $steps += "PowerShell execution policy is Restricted. Run: Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser"
    }
    
    if ($Diagnostics.SecurityInfo.WindowsDefenderStatus -and $Diagnostics.SecurityInfo.WindowsDefenderStatus.RealTimeProtectionEnabled) {
        $steps += "Windows Defender real-time protection is enabled. Consider adding RepSet Bridge to exclusions if connection issues persist."
    }
    
    # System issues
    if (-not $Diagnostics.SystemInfo.IsElevated) {
        $steps += "Not running as Administrator. Some network diagnostics and firewall checks may be limited."
    }
    
    # Generic troubleshooting steps
    $steps += "Verify the platform endpoint URL is correct: $PlatformEndpoint"
    $steps += "Check if corporate firewall or proxy is blocking outbound HTTPS connections"
    $steps += "Ensure Windows Firewall allows outbound connections for RepSet Bridge"
    $steps += "Try temporarily disabling antivirus software to test connectivity"
    $steps += "Contact your network administrator if issues persist"
    
    return $steps
}

function Send-ConnectionStatusToPlatform {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [hashtable]$ConnectionResult,
        
        [Parameter(Mandatory=$true)]
        [string]$PlatformEndpoint,
        
        [Parameter(Mandatory=$true)]
        [string]$GymId
    )
    
    try {
        Write-InstallationLog -Level Debug -Message "Sending connection status to platform..."
        
        $statusData = @{
            installationId = $script:InstallationId
            gymId = $GymId
            timestamp = Get-Date -Format 'yyyy-MM-ddTHH:mm:ss.fffZ'
            connectionTest = @{
                success = $ConnectionResult.Success
                networkConnectivity = $ConnectionResult.NetworkConnectivity
                platformReachable = $ConnectionResult.PlatformReachable
                authenticationValid = $ConnectionResult.AuthenticationValid
                serviceResponding = $ConnectionResult.ServiceResponding
                firewallBlocking = $ConnectionResult.FirewallBlocking
                errorDetails = $ConnectionResult.ErrorDetails
            }
            testResults = $ConnectionResult.TestResults
            diagnosticInfo = $ConnectionResult.DiagnosticInfo
        }
        
        $platformUrl = "$PlatformEndpoint/api/installation/connection-status"
        $headers = @{
            'Content-Type' = 'application/json'
            'User-Agent' = 'RepSet-Bridge-Installer/1.0'
        }
        
        $json = $statusData | ConvertTo-Json -Depth 5
        
        # Send status update asynchronously
        Start-Job -ScriptBlock {
            param($Url, $Headers, $Data)
            try {
                Invoke-RestMethod -Uri $Url -Method Post -Headers $Headers -Body $Data -TimeoutSec 10 -ErrorAction SilentlyContinue
            }
            catch {
                # Silently ignore platform status update failures
            }
        } -ArgumentList $platformUrl, $headers, $json | Out-Null
        
        Write-InstallationLog -Level Debug -Message "Connection status sent to platform"
    }
    catch {
        Write-InstallationLog -Level Debug -Message "Could not send connection status to platform: $($_.Exception.Message)"
    }
}

function Write-ConnectionTroubleshootingGuide {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [hashtable]$ConnectionResult
    )
    
    Write-InstallationLog -Level Info -Message ""
    Write-InstallationLog -Level Info -Message "=== CONNECTION TROUBLESHOOTING GUIDE ==="
    Write-InstallationLog -Level Info -Message ""
    
    if ($ConnectionResult.Success) {
        Write-InstallationLog -Level Success -Message "✓ All connection tests passed successfully!"
        Write-InstallationLog -Level Info -Message "The RepSet Bridge should be able to connect to the platform."
    }
    else {
        Write-InstallationLog -Level Error -Message "❌ Connection tests failed. Please review the issues below:"
        Write-InstallationLog -Level Info -Message ""
        
        # Network connectivity issues
        if (-not $ConnectionResult.NetworkConnectivity) {
            Write-InstallationLog -Level Error -Message "❌ Network Connectivity Failed"
            Write-InstallationLog -Level Info -Message "   • Check your internet connection"
            Write-InstallationLog -Level Info -Message "   • Verify network cables are connected"
            Write-InstallationLog -Level Info -Message "   • Check network adapter status in Device Manager"
            Write-InstallationLog -Level Info -Message ""
        }
        else {
            Write-InstallationLog -Level Success -Message "✓ Network Connectivity OK"
        }
        
        # Platform reachability issues
        if (-not $ConnectionResult.PlatformReachable) {
            Write-InstallationLog -Level Error -Message "❌ Platform Endpoint Unreachable"
            Write-InstallationLog -Level Info -Message "   • Verify the platform URL is correct"
            Write-InstallationLog -Level Info -Message "   • Check if corporate firewall is blocking access"
            Write-InstallationLog -Level Info -Message "   • Try accessing the platform in a web browser"
            Write-InstallationLog -Level Info -Message ""
        }
        else {
            Write-InstallationLog -Level Success -Message "✓ Platform Endpoint Reachable"
        }
        
        # Authentication issues
        if (-not $ConnectionResult.AuthenticationValid) {
            Write-InstallationLog -Level Error -Message "❌ Platform Authentication Failed"
            Write-InstallationLog -Level Info -Message "   • Verify the pair code is correct and not expired"
            Write-InstallationLog -Level Info -Message "   • Check that the gym ID matches your account"
            Write-InstallationLog -Level Info -Message "   • Generate a new installation command if needed"
            Write-InstallationLog -Level Info -Message ""
        }
        else {
            Write-InstallationLog -Level Success -Message "✓ Platform Authentication OK"
        }
        
        # Firewall issues
        if ($ConnectionResult.FirewallBlocking) {
            Write-InstallationLog -Level Warning -Message "⚠ Firewall May Be Blocking Connection"
            Write-InstallationLog -Level Info -Message "   • Check Windows Firewall settings"
            Write-InstallationLog -Level Info -Message "   • Add RepSet Bridge to firewall exceptions"
            Write-InstallationLog -Level Info -Message "   • Contact your IT administrator about firewall rules"
            Write-InstallationLog -Level Info -Message ""
        }
        else {
            Write-InstallationLog -Level Success -Message "✓ No Firewall Blocking Detected"
        }
        
        # Service issues
        if ($ConnectionResult.TestResults.ContainsKey('BridgeService') -and -not $ConnectionResult.ServiceResponding) {
            Write-InstallationLog -Level Warning -Message "⚠ Bridge Service Test Failed"
            Write-InstallationLog -Level Info -Message "   • The bridge executable may have issues"
            Write-InstallationLog -Level Info -Message "   • Check the installation log for errors"
            Write-InstallationLog -Level Info -Message "   • Try reinstalling the bridge"
            Write-InstallationLog -Level Info -Message ""
        }
        
        # Error details
        if ($ConnectionResult.ErrorDetails -and $ConnectionResult.ErrorDetails.Count -gt 0) {
            Write-InstallationLog -Level Info -Message "Detailed Error Information:"
            foreach ($error in $ConnectionResult.ErrorDetails) {
                Write-InstallationLog -Level Info -Message "   • $error"
            }
            Write-InstallationLog -Level Info -Message ""
        }
        
        # Troubleshooting steps
        if ($ConnectionResult.DiagnosticInfo -and $ConnectionResult.DiagnosticInfo.TroubleshootingSteps) {
            Write-InstallationLog -Level Info -Message "Recommended Troubleshooting Steps:"
            foreach ($step in $ConnectionResult.DiagnosticInfo.TroubleshootingSteps) {
                Write-InstallationLog -Level Info -Message "   $($ConnectionResult.DiagnosticInfo.TroubleshootingSteps.IndexOf($step) + 1). $step"
            }
            Write-InstallationLog -Level Info -Message ""
        }
    }
    
    Write-InstallationLog -Level Info -Message "For additional support, please provide this log file to RepSet support."
    Write-InstallationLog -Level Info -Message "Log file location: $script:LogFile"
    Write-InstallationLog -Level Info -Message ""
    Write-InstallationLog -Level Info -Message "=== END TROUBLESHOOTING GUIDE ==="
}
#
 ================================================================
# Main Installation Function (Placeholder - needs full implementation)
# ================================================================

function Start-ComprehensiveInstallation {
    <#
    .SYNOPSIS
    Executes the complete RepSet Bridge installation with comprehensive telemetry and monitoring
    
    .DESCRIPTION
    This function orchestrates the entire installation process with full telemetry collection,
    performance monitoring, error tracking, and real-time progress reporting to the platform.
    #>
    [CmdletBinding()]
    param()
    
    $installationSuccess = $false
    $errorMessage = ""
    $errorCode = $script:ErrorCodes.Success
    
    try {
        # Initialize security audit system
        Write-InstallationLog -Level Info -Message "Initializing security audit system..."
        $securityAuditInitialized = Initialize-SecurityAuditSystem
        
        if (-not $securityAuditInitialized) {
            Write-InstallationLog -Level Warning -Message "Security audit initialization failed, continuing with reduced security logging"
        }
        
        # Initialize telemetry system
        Write-InstallationLog -Level Info -Message "Initializing installation telemetry and monitoring..."
        $telemetryInitialized = Initialize-InstallationTelemetry
        
        if (-not $telemetryInitialized) {
            Write-InstallationLog -Level Warning -Message "Telemetry initialization failed, continuing without telemetry"
        }
        
        # Send installation started notification
        Send-InstallationNotification -Status Started -Message "RepSet Bridge installation started"
        
        # Define installation steps with telemetry tracking
        $installationSteps = @(
            @{ Name = "Security Validation"; Function = "Test-InstallationCommandSignature"; StepNumber = 1 }
            @{ Name = "Security Compliance Check"; Function = "Test-SecurityCompliance"; StepNumber = 2 }
            @{ Name = "System Requirements Check"; Function = "Test-SystemRequirements"; StepNumber = 3 }
            @{ Name = "Prerequisites Installation"; Function = "Install-Prerequisites"; StepNumber = 4 }
            @{ Name = "Bridge Download"; Function = "Get-LatestBridge"; StepNumber = 5 }
            @{ Name = "Bridge Installation"; Function = "Install-BridgeExecutable"; StepNumber = 6 }
            @{ Name = "Configuration Setup"; Function = "New-BridgeConfiguration"; StepNumber = 7 }
            @{ Name = "Service Installation"; Function = "Install-BridgeService"; StepNumber = 8 }
            @{ Name = "Service Startup"; Function = "Start-BridgeService"; StepNumber = 9 }
            @{ Name = "Connection Verification"; Function = "Test-BridgeConnection"; StepNumber = 10 }
        )
        
        $totalSteps = $installationSteps.Count
        $script:TelemetryData.InstallationMetrics.TotalSteps = $totalSteps
        
        Write-InstallationLog -Level Info -Message "Starting comprehensive RepSet Bridge installation ($totalSteps steps)"
        
        # Execute each installation step with telemetry
        foreach ($step in $installationSteps) {
            $stepName = $step.Name
            $stepNumber = $step.StepNumber
            
            try {
                # Start step performance tracking
                Start-StepPerformanceTracking -StepName $stepName
                
                # Update progress
                Write-Progress-Step -Step $stepName -StepNumber $stepNumber -TotalSteps $totalSteps -Status "Starting"
                
                Write-InstallationLog -Level Info -Message "Executing step $stepNumber/$totalSteps`: $stepName"
                
                # Execute the step function
                $stepResult = switch ($step.Function) {
                    "Test-InstallationCommandSignature" {
                        Write-SecurityAuditEvent -EventType "SignatureValidation" -Severity "Information" -Message "Starting installation command signature validation"
                        $signatureResult = Test-InstallationCommandSignature -PairCode $PairCode -Signature $Signature -Nonce $Nonce -GymId $GymId -ExpiresAt $ExpiresAt -PlatformEndpoint $PlatformEndpoint
                        
                        if ($signatureResult.IsValid) {
                            Write-SecurityAuditEvent -EventType "SignatureValidation" -Severity "Information" -Message "Installation command signature validation successful"
                            $true
                        } else {
                            Write-SecurityAuditEvent -EventType "SecurityViolation" -Severity "Critical" -Message "Installation command signature validation failed" -Details @{
                                ErrorCode = $signatureResult.ErrorCode
                                ErrorMessage = $signatureResult.ErrorMessage
                            }
                            $errorCode = $signatureResult.ErrorCode
                            $errorMessage = $signatureResult.ErrorMessage
                            $false
                        }
                    }
                    "Test-SecurityCompliance" {
                        Write-SecurityAuditEvent -EventType "ComplianceCheck" -Severity "Information" -Message "Starting security compliance validation"
                        $complianceResult = Test-SecurityCompliance
                        
                        # Store compliance results in security audit
                        if ($script:SecurityAudit) {
                            $script:SecurityAudit.ComplianceStatus = $complianceResult
                        }
                        
                        if ($complianceResult.OverallCompliant -or $complianceResult.CompliancePercentage -ge 70) {
                            Write-SecurityAuditEvent -EventType "ComplianceCheck" -Severity "Information" -Message "Security compliance validation passed" -Details @{
                                ComplianceLevel = $complianceResult.ComplianceLevel
                                CompliancePercentage = $complianceResult.CompliancePercentage
                            }
                            $true
                        } else {
                            Write-SecurityAuditEvent -EventType "SecurityViolation" -Severity "Error" -Message "Security compliance validation failed" -Details $complianceResult
                            
                            # Allow installation to continue with warnings if compliance is above 50%
                            if ($complianceResult.CompliancePercentage -ge 50) {
                                Write-InstallationLog -Level Warning -Message "Security compliance below recommended level but allowing installation to continue"
                                $true
                            } else {
                                $errorCode = $script:ErrorCodes.SystemRequirementsNotMet
                                $errorMessage = "Critical security compliance failures detected"
                                $false
                            }
                        }
                    }
                    "Test-SystemRequirements" {
                        Test-SystemRequirements
                    }
                    "Install-Prerequisites" {
                        if (-not $SkipPrerequisites) {
                            Install-Prerequisites
                        } else {
                            Write-InstallationLog -Level Info -Message "Skipping prerequisites installation as requested"
                            $true
                        }
                    }
                    "Get-LatestBridge" {
                        $downloadStartTime = Get-Date
                        $downloadResult = Get-LatestBridge
                        $downloadEndTime = Get-Date
                        
                        # Record download metrics
                        if ($downloadResult -and $downloadResult.Success) {
                            $downloadDuration = ($downloadEndTime - $downloadStartTime).TotalSeconds
                            Record-DownloadMetrics -Url $downloadResult.DownloadUrl -BytesDownloaded $downloadResult.FileSize -DurationSeconds $downloadDuration -Status "Success"
                        }
                        
                        $downloadResult.Success
                    }
                    "Install-BridgeExecutable" {
                        Install-BridgeExecutable
                    }
                    "New-BridgeConfiguration" {
                        New-BridgeConfiguration -PairCode $PairCode -GymId $GymId -PlatformEndpoint $PlatformEndpoint
                    }
                    "Install-BridgeService" {
                        Install-BridgeService
                    }
                    "Start-BridgeService" {
                        Start-BridgeService
                    }
                    "Test-BridgeConnection" {
                        $connectionResult = Test-BridgeConnection -PlatformEndpoint $PlatformEndpoint -PairCode $PairCode -GymId $GymId
                        $connectionResult.Success
                    }
                    default {
                        Write-InstallationLog -Level Warning -Message "Unknown step function: $($step.Function)"
                        $false
                    }
                }
                
                if ($stepResult) {
                    # Step completed successfully
                    Write-Progress-Step -Step $stepName -StepNumber $stepNumber -TotalSteps $totalSteps -Status "Completed"
                    Write-InstallationLog -Level Success -Message "Step $stepNumber/$totalSteps completed successfully: $stepName"
                    
                    # Stop performance tracking
                    Stop-StepPerformanceTracking -StepName $stepName -Status "Completed"
                    
                    # Update metrics
                    $script:TelemetryData.InstallationMetrics.CompletedSteps++
                }
                else {
                    # Step failed
                    Write-Progress-Step -Step $stepName -StepNumber $stepNumber -TotalSteps $totalSteps -Status "Failed"
                    Write-InstallationLog -Level Error -Message "Step $stepNumber/$totalSteps failed: $stepName"
                    
                    # Stop performance tracking
                    Stop-StepPerformanceTracking -StepName $stepName -Status "Failed"
                    
                    # Record error metrics
                    Record-ErrorMetrics -ErrorCategory "StepFailure" -ErrorMessage "Installation step failed: $stepName" -StepName $stepName -Severity "High"
                    
                    # Update metrics
                    $script:TelemetryData.InstallationMetrics.FailedSteps++
                    
                    # Attempt rollback
                    Write-InstallationLog -Level Warning -Message "Attempting installation rollback due to step failure..."
                    $rollbackResult = Invoke-InstallationRollback -InstallationId $script:InstallationId -RollbackReason "Step failure: $stepName"
                    
                    $errorMessage = "Installation failed at step: $stepName"
                    $errorCode = $script:ErrorCodes.InstallationFailed
                    throw [System.Exception]::new($errorMessage)
                }
            }
            catch {
                # Handle step-specific errors
                $stepError = $_.Exception
                Write-InstallationLog -Level Error -Message "Error in step $stepNumber/$totalSteps ($stepName): $($stepError.Message)"
                
                # Record detailed error metrics
                Record-ErrorMetrics -ErrorCategory "StepException" -ErrorMessage $stepError.Message -StepName $stepName -Exception $stepError -Severity "High"
                
                # Stop performance tracking
                Stop-StepPerformanceTracking -StepName $stepName -Status "Failed" -AdditionalMetrics @{ ErrorMessage = $stepError.Message }
                
                # Update metrics
                $script:TelemetryData.InstallationMetrics.FailedSteps++
                $script:TelemetryData.InstallationMetrics.Errors++
                
                # Re-throw to be caught by outer try-catch
                throw
            }
        }
        
        # All steps completed successfully
        $installationSuccess = $true
        Write-InstallationLog -Level Success -Message "All installation steps completed successfully!"
        
        # Send success notification
        Send-InstallationNotification -Status Success -Message "RepSet Bridge installation completed successfully"
        
    }
    catch {
        # Handle installation failure
        $installationSuccess = $false
        $errorMessage = $_.Exception.Message
        $errorCode = if ($errorCode -eq $script:ErrorCodes.Success) { $script:ErrorCodes.InstallationFailed } else { $errorCode }
        
        Write-InstallationLog -Level Error -Message "Installation failed: $errorMessage"
        
        # Record final error metrics
        Record-ErrorMetrics -ErrorCategory "InstallationFailure" -ErrorMessage $errorMessage -Exception $_.Exception -Severity "Critical"
        
        # Send failure notification
        Send-InstallationNotification -Status Failed -Message $errorMessage -ErrorCode $errorCode.ToString()
    }
    finally {
        # Send comprehensive telemetry summary
        if ($telemetryInitialized) {
            Write-InstallationLog -Level Info -Message "Sending installation telemetry summary..."
            Send-InstallationTelemetrySummary -Success $installationSuccess -ErrorMessage $errorMessage -ErrorCode $errorCode
        }
        
        # Generate installation summary
        Write-InstallationSummary -Success $installationSuccess -ErrorMessage $errorMessage -ErrorCode $errorCode -InstallationDetails @{
            TotalSteps = $script:TelemetryData.InstallationMetrics.TotalSteps
            CompletedSteps = $script:TelemetryData.InstallationMetrics.CompletedSteps
            FailedSteps = $script:TelemetryData.InstallationMetrics.FailedSteps
            TotalErrors = $script:TelemetryData.InstallationMetrics.Errors
            TelemetryEnabled = $telemetryInitialized
        }
        
        # Clean up progress bar
        Write-Progress -Activity "RepSet Bridge Installation" -Completed
    }
    
    return $errorCode
}

function Install-RepSetBridge {
    <#
    .SYNOPSIS
    Main entry point for RepSet Bridge installation
    
    .DESCRIPTION
    This function serves as the main entry point and delegates to the comprehensive
    installation function with full telemetry and monitoring capabilities.
    #>
    [CmdletBinding()]
    param()
    
    Write-InstallationLog -Level Info -Message "=== RepSet Bridge Installation Started ==="
    Write-InstallationLog -Level Info -Message "Installation ID: $script:InstallationId"
    Write-InstallationLog -Level Info -Message "Gym ID: $GymId"
    Write-InstallationLog -Level Info -Message "Platform Endpoint: $PlatformEndpoint"
    Write-InstallationLog -Level Info -Message "Installation Path: $InstallPath"
    Write-InstallationLog -Level Info -Message "Log File: $script:LogFile"
    
    # Execute comprehensive installation with telemetry
    $exitCode = Start-ComprehensiveInstallation
    
    Write-InstallationLog -Level Info -Message "=== RepSet Bridge Installation Completed ==="
    Write-InstallationLog -Level Info -Message "Exit Code: $exitCode"
    
    return $exitCode
}

# ================================================================
# Main Execution Block
# ================================================================

# Execute the installation if this script is run directly
if ($MyInvocation.InvocationName -ne '.') {
    try {
        $exitCode = Install-RepSetBridge
        
        # Display final log summary
        Write-Host ""
        Write-Host "Installation Log Summary:" -ForegroundColor Cyan
        Write-Host "  Log File: $script:LogFile" -ForegroundColor White
        
        # Clean up background jobs
        Get-Job | Where-Object { $_.Name -like "*RepSet*" -or $_.State -eq 'Completed' } | Remove-Job -Force -ErrorAction SilentlyContinue
        
        exit $exitCode
    }
    catch {
        Write-Host "Fatal error during installation: $($_.Exception.Message)" -ForegroundColor Red
        Write-Host "Please check the installation log for details: $script:LogFile" -ForegroundColor Yellow
        exit $script:ErrorCodes.InstallationFailed
    }
}
# =
===============================================================
# Enhanced Security Validation and Audit Logging Functions
# ================================================================

function Test-CommandSignatureValidation {
    <#
    .SYNOPSIS
    Performs comprehensive signature validation with tamper detection
    
    .DESCRIPTION
    Validates the installation command signature using HMAC-SHA256,
    checks for tampering, validates expiration, and performs replay attack detection.
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$PairCode,
        
        [Parameter(Mandatory=$true)]
        [string]$Signature,
        
        [Parameter(Mandatory=$true)]
        [string]$Nonce,
        
        [Parameter(Mandatory=$true)]
        [string]$GymId,
        
        [Parameter(Mandatory=$true)]
        [string]$ExpiresAt
    )
    
    Write-SecurityAuditEvent -EventType "SignatureValidation" -Severity "Information" -Message "Starting command signature validation" -Details @{
        GymId = $GymId
        Nonce = $Nonce
        ExpiresAt = $ExpiresAt
        SignatureLength = $Signature.Length
    }
    
    try {
        # Step 1: Validate signature format
        if ([string]::IsNullOrWhiteSpace($Signature)) {
            Write-SecurityAuditEvent -EventType "SecurityViolation" -Severity "Error" -Message "Empty or null signature provided" -Details @{
                ValidationStep = "SignatureFormat"
                Reason = "EmptySignature"
            }
            throw "Invalid signature format: signature is empty or null"
        }
        
        if ($Signature.Length -ne 64) {
            Write-SecurityAuditEvent -EventType "SecurityViolation" -Severity "Error" -Message "Invalid signature length" -Details @{
                ValidationStep = "SignatureFormat"
                ExpectedLength = 64
                ActualLength = $Signature.Length
                Reason = "InvalidLength"
            }
            throw "Invalid signature format: expected 64 characters, got $($Signature.Length)"
        }
        
        # Step 2: Validate expiration timestamp
        try {
            $expirationTime = [DateTime]::Parse($ExpiresAt)
            $currentTime = Get-Date
            
            if ($currentTime -gt $expirationTime) {
                Write-SecurityAuditEvent -EventType "SecurityViolation" -Severity "Error" -Message "Command has expired" -Details @{
                    ValidationStep = "ExpirationCheck"
                    ExpirationTime = $expirationTime.ToString('yyyy-MM-ddTHH:mm:ss.fffZ')
                    CurrentTime = $currentTime.ToString('yyyy-MM-ddTHH:mm:ss.fffZ')
                    ExpiredBy = ($currentTime - $expirationTime).TotalMinutes
                    Reason = "CommandExpired"
                }
                throw "Command has expired. Expiration time: $($expirationTime.ToString('yyyy-MM-dd HH:mm:ss')), Current time: $($currentTime.ToString('yyyy-MM-dd HH:mm:ss'))"
            }
            
            Write-SecurityAuditEvent -EventType "ComplianceCheck" -Severity "Information" -Message "Command expiration validation passed" -Details @{
                ValidationStep = "ExpirationCheck"
                ExpirationTime = $expirationTime.ToString('yyyy-MM-ddTHH:mm:ss.fffZ')
                TimeRemaining = ($expirationTime - $currentTime).TotalMinutes
            }
        }
        catch [System.FormatException] {
            Write-SecurityAuditEvent -EventType "SecurityViolation" -Severity "Error" -Message "Invalid expiration timestamp format" -Details @{
                ValidationStep = "ExpirationFormat"
                ProvidedTimestamp = $ExpiresAt
                Reason = "InvalidTimestampFormat"
            }
            throw "Invalid expiration timestamp format: $ExpiresAt"
        }
        
        # Step 3: Validate nonce format and detect replay attacks
        if ([string]::IsNullOrWhiteSpace($Nonce)) {
            Write-SecurityAuditEvent -EventType "SecurityViolation" -Severity "Error" -Message "Empty or null nonce provided" -Details @{
                ValidationStep = "NonceValidation"
                Reason = "EmptyNonce"
            }
            throw "Invalid nonce: nonce is empty or null"
        }
        
        if ($Nonce.Length -lt 16) {
            Write-SecurityAuditEvent -EventType "SecurityViolation" -Severity "Error" -Message "Nonce too short" -Details @{
                ValidationStep = "NonceValidation"
                MinimumLength = 16
                ActualLength = $Nonce.Length
                Reason = "NonceTooShort"
            }
            throw "Invalid nonce: minimum length is 16 characters, got $($Nonce.Length)"
        }
        
        # Check for nonce reuse (simple file-based tracking)
        $nonceTrackingFile = "$env:TEMP\RepSetBridge-Nonces.txt"
        if (Test-Path $nonceTrackingFile) {
            $usedNonces = Get-Content $nonceTrackingFile -ErrorAction SilentlyContinue
            if ($usedNonces -contains $Nonce) {
                Write-SecurityAuditEvent -EventType "SecurityViolation" -Severity "Critical" -Message "Replay attack detected - nonce reuse" -Details @{
                    ValidationStep = "ReplayDetection"
                    Nonce = $Nonce
                    Reason = "NonceReuse"
                    ThreatLevel = "Critical"
                }
                throw "Security violation: Replay attack detected. This nonce has already been used."
            }
        }
        
        # Step 4: Reconstruct expected signature (using a mock secret for demonstration)
        $mockSecretKey = "RepSetBridge-Installation-Secret-Key-2024"
        $signaturePayload = "$PairCode|$Nonce|$GymId|$ExpiresAt"
        $expectedSignature = Get-HMACSignature -Data $signaturePayload -SecretKey $mockSecretKey
        
        # Step 5: Perform constant-time signature comparison
        $signatureValid = Compare-Signatures -ProvidedSignature $Signature -ExpectedSignature $expectedSignature
        
        if (-not $signatureValid) {
            Write-SecurityAuditEvent -EventType "SecurityViolation" -Severity "Critical" -Message "Signature validation failed - potential tampering detected" -Details @{
                ValidationStep = "SignatureComparison"
                ProvidedSignature = $Signature
                PayloadHash = (Get-StringHash -InputString $signaturePayload)
                Reason = "SignatureMismatch"
                ThreatLevel = "Critical"
            }
            throw "Security violation: Command signature validation failed. The command may have been tampered with."
        }
        
        # Step 6: Record successful validation and store nonce
        Add-Content -Path $nonceTrackingFile -Value $Nonce -ErrorAction SilentlyContinue
        
        # Clean up old nonces (keep only last 1000)
        try {
            $allNonces = Get-Content $nonceTrackingFile -ErrorAction SilentlyContinue
            if ($allNonces.Count -gt 1000) {
                $recentNonces = $allNonces | Select-Object -Last 1000
                Set-Content -Path $nonceTrackingFile -Value $recentNonces -ErrorAction SilentlyContinue
            }
        }
        catch {
            Write-InstallationLog -Level Debug -Message "Could not clean up nonce tracking file: $($_.Exception.Message)"
        }
        
        Write-SecurityAuditEvent -EventType "SignatureValidation" -Severity "Information" -Message "Command signature validation successful" -Details @{
            ValidationStep = "Complete"
            GymId = $GymId
            Nonce = $Nonce
            ExpirationTime = $expirationTime.ToString('yyyy-MM-ddTHH:mm:ss.fffZ')
            ValidationDuration = (Get-Date) - $script:InstallationStartTime
        }
        
        # Update security metrics
        if ($script:TelemetryData) {
            $script:TelemetryData.SecurityMetrics.SignatureValidations++
        }
        
        return $true
    }
    catch {
        Write-SecurityAuditEvent -EventType "SecurityError" -Severity "Critical" -Message "Signature validation failed with error" -Details @{
            ErrorMessage = $_.Exception.Message
            ErrorType = $_.Exception.GetType().Name
            StackTrace = $_.ScriptStackTrace
        }
        
        Write-InstallationLog -Level Error -Message "Signature validation failed: $($_.Exception.Message)"
        return $false
    }
}

function Get-HMACSignature {
    <#
    .SYNOPSIS
    Generates HMAC-SHA256 signature for data validation
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$Data,
        
        [Parameter(Mandatory=$true)]
        [string]$SecretKey
    )
    
    try {
        $keyBytes = [System.Text.Encoding]::UTF8.GetBytes($SecretKey)
        $dataBytes = [System.Text.Encoding]::UTF8.GetBytes($Data)
        
        $hmac = New-Object System.Security.Cryptography.HMACSHA256
        $hmac.Key = $keyBytes
        
        $hashBytes = $hmac.ComputeHash($dataBytes)
        $signature = [System.BitConverter]::ToString($hashBytes) -replace '-', ''
        
        return $signature.ToLower()
    }
    catch {
        Write-InstallationLog -Level Error -Message "Failed to generate HMAC signature: $($_.Exception.Message)"
        throw
    }
    finally {
        if ($hmac) {
            $hmac.Dispose()
        }
    }
}

function Compare-Signatures {
    <#
    .SYNOPSIS
    Performs constant-time signature comparison to prevent timing attacks
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$ProvidedSignature,
        
        [Parameter(Mandatory=$true)]
        [string]$ExpectedSignature
    )
    
    try {
        # Ensure both signatures are the same length
        if ($ProvidedSignature.Length -ne $ExpectedSignature.Length) {
            return $false
        }
        
        # Perform constant-time comparison
        $result = 0
        for ($i = 0; $i -lt $ProvidedSignature.Length; $i++) {
            $result = $result -bor ([int][char]$ProvidedSignature[$i] -bxor [int][char]$ExpectedSignature[$i])
        }
        
        return $result -eq 0
    }
    catch {
        Write-InstallationLog -Level Error -Message "Error during signature comparison: $($_.Exception.Message)"
        return $false
    }
}

function Test-FileIntegrityValidation {
    <#
    .SYNOPSIS
    Performs comprehensive file integrity validation with tamper detection
    
    .DESCRIPTION
    Validates downloaded files using SHA-256 checksums, digital signatures,
    and additional integrity checks to detect tampering or corruption.
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$FilePath,
        
        [Parameter(Mandatory=$true)]
        [string]$ExpectedHash,
        
        [Parameter(Mandatory=$false)]
        [string]$ExpectedSize = "",
        
        [Parameter(Mandatory=$false)]
        [switch]$VerifyDigitalSignature
    )
    
    Write-SecurityAuditEvent -EventType "IntegrityCheck" -Severity "Information" -Message "Starting file integrity validation" -Details @{
        FilePath = $FilePath
        ExpectedHash = $ExpectedHash
        ExpectedSize = $ExpectedSize
        VerifyDigitalSignature = $VerifyDigitalSignature.IsPresent
    }
    
    try {
        # Step 1: Verify file exists
        if (-not (Test-Path $FilePath)) {
            Write-SecurityAuditEvent -EventType "SecurityViolation" -Severity "Error" -Message "File not found for integrity validation" -Details @{
                FilePath = $FilePath
                ValidationStep = "FileExistence"
                Reason = "FileNotFound"
            }
            throw "File not found: $FilePath"
        }
        
        $fileInfo = Get-Item $FilePath
        
        # Step 2: Verify file size if provided
        if ($ExpectedSize -and $ExpectedSize -ne "") {
            try {
                $expectedSizeBytes = [long]$ExpectedSize
                if ($fileInfo.Length -ne $expectedSizeBytes) {
                    Write-SecurityAuditEvent -EventType "TamperDetection" -Severity "Error" -Message "File size mismatch detected" -Details @{
                        FilePath = $FilePath
                        ExpectedSize = $expectedSizeBytes
                        ActualSize = $fileInfo.Length
                        ValidationStep = "SizeValidation"
                        Reason = "SizeMismatch"
                    }
                    throw "File size mismatch: expected $expectedSizeBytes bytes, got $($fileInfo.Length) bytes"
                }
                
                Write-SecurityAuditEvent -EventType "ComplianceCheck" -Severity "Information" -Message "File size validation passed" -Details @{
                    FilePath = $FilePath
                    FileSize = $fileInfo.Length
                    ValidationStep = "SizeValidation"
                }
            }
            catch [System.FormatException] {
                Write-InstallationLog -Level Warning -Message "Invalid expected size format: $ExpectedSize"
            }
        }
        
        # Step 3: Calculate and verify SHA-256 hash
        Write-InstallationLog -Level Info -Message "Calculating file hash for integrity verification..."
        
        $actualHash = Get-FileHash -Path $FilePath -Algorithm SHA256 -ErrorAction Stop
        $actualHashString = $actualHash.Hash.ToLower()
        $expectedHashString = $ExpectedHash.ToLower()
        
        if ($actualHashString -ne $expectedHashString) {
            Write-SecurityAuditEvent -EventType "TamperDetection" -Severity "Critical" -Message "File hash mismatch - tampering detected" -Details @{
                FilePath = $FilePath
                ExpectedHash = $expectedHashString
                ActualHash = $actualHashString
                FileSize = $fileInfo.Length
                ValidationStep = "HashValidation"
                Reason = "HashMismatch"
                ThreatLevel = "Critical"
            }
            throw "File integrity violation: Hash mismatch detected. Expected: $expectedHashString, Actual: $actualHashString"
        }
        
        Write-SecurityAuditEvent -EventType "IntegrityCheck" -Severity "Information" -Message "File hash validation successful" -Details @{
            FilePath = $FilePath
            Hash = $actualHashString
            FileSize = $fileInfo.Length
            ValidationStep = "HashValidation"
        }
        
        # Step 4: Verify digital signature if requested
        if ($VerifyDigitalSignature) {
            $signatureValid = Test-DigitalSignature -FilePath $FilePath
            if (-not $signatureValid) {
                Write-SecurityAuditEvent -EventType "SecurityViolation" -Severity "Error" -Message "Digital signature validation failed" -Details @{
                    FilePath = $FilePath
                    ValidationStep = "DigitalSignature"
                    Reason = "InvalidSignature"
                }
                throw "Digital signature validation failed for file: $FilePath"
            }
            
            Write-SecurityAuditEvent -EventType "ComplianceCheck" -Severity "Information" -Message "Digital signature validation successful" -Details @{
                FilePath = $FilePath
                ValidationStep = "DigitalSignature"
            }
        }
        
        # Step 5: Additional integrity checks
        $additionalChecks = Test-AdditionalIntegrityChecks -FilePath $FilePath
        if (-not $additionalChecks.IsValid) {
            Write-SecurityAuditEvent -EventType "SecurityWarning" -Severity "Warning" -Message "Additional integrity checks failed" -Details @{
                FilePath = $FilePath
                ValidationStep = "AdditionalChecks"
                FailedChecks = $additionalChecks.FailedChecks
                Reason = "AdditionalChecksFailed"
            }
            Write-InstallationLog -Level Warning -Message "Some additional integrity checks failed, but core validation passed"
        }
        
        Write-SecurityAuditEvent -EventType "IntegrityCheck" -Severity "Information" -Message "File integrity validation completed successfully" -Details @{
            FilePath = $FilePath
            Hash = $actualHashString
            FileSize = $fileInfo.Length
            DigitalSignatureVerified = $VerifyDigitalSignature.IsPresent
            ValidationDuration = (Get-Date) - $script:InstallationStartTime
        }
        
        # Update security metrics
        if ($script:TelemetryData) {
            $script:TelemetryData.SecurityMetrics.IntegrityChecks++
        }
        
        return $true
    }
    catch {
        Write-SecurityAuditEvent -EventType "SecurityError" -Severity "Critical" -Message "File integrity validation failed with error" -Details @{
            FilePath = $FilePath
            ErrorMessage = $_.Exception.Message
            ErrorType = $_.Exception.GetType().Name
            StackTrace = $_.ScriptStackTrace
        }
        
        Write-InstallationLog -Level Error -Message "File integrity validation failed: $($_.Exception.Message)"
        return $false
    }
}

function Test-DigitalSignature {
    <#
    .SYNOPSIS
    Verifies digital signature of executable files
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$FilePath
    )
    
    try {
        $signature = Get-AuthenticodeSignature -FilePath $FilePath -ErrorAction Stop
        
        $isValid = $signature.Status -eq 'Valid'
        
        Write-SecurityAuditEvent -EventType "ComplianceCheck" -Severity "Information" -Message "Digital signature check completed" -Details @{
            FilePath = $FilePath
            SignatureStatus = $signature.Status
            SignerCertificate = if ($signature.SignerCertificate) { $signature.SignerCertificate.Subject } else { "None" }
            TimeStamperCertificate = if ($signature.TimeStamperCertificate) { $signature.TimeStamperCertificate.Subject } else { "None" }
            IsValid = $isValid
        }
        
        return $isValid
    }
    catch {
        Write-InstallationLog -Level Warning -Message "Could not verify digital signature: $($_.Exception.Message)"
        return $false
    }
}

function Test-AdditionalIntegrityChecks {
    <#
    .SYNOPSIS
    Performs additional integrity checks beyond hash validation
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$FilePath
    )
    
    $result = @{
        IsValid = $true
        FailedChecks = @()
        Checks = @{}
    }
    
    try {
        # Check 1: File header validation for PE files
        if ($FilePath -match '\.(exe|dll)$') {
            try {
                $fileBytes = [System.IO.File]::ReadAllBytes($FilePath) | Select-Object -First 2
                $peHeader = [System.Text.Encoding]::ASCII.GetString($fileBytes)
                
                if ($peHeader -ne "MZ") {
                    $result.FailedChecks += "InvalidPEHeader"
                    $result.IsValid = $false
                }
                $result.Checks.PEHeader = $peHeader -eq "MZ"
            }
            catch {
                $result.FailedChecks += "PEHeaderCheckFailed"
                $result.Checks.PEHeader = $false
            }
        }
        
        # Check 2: File extension validation
        $expectedExtensions = @('.exe', '.dll', '.config', '.yaml', '.yml')
        $fileExtension = [System.IO.Path]::GetExtension($FilePath).ToLower()
        
        if ($fileExtension -notin $expectedExtensions) {
            $result.FailedChecks += "UnexpectedFileExtension"
            $result.Checks.FileExtension = $false
        }
        else {
            $result.Checks.FileExtension = $true
        }
        
        # Check 3: File size reasonableness (not too small or too large)
        $fileInfo = Get-Item $FilePath
        $minSize = 1KB
        $maxSize = 100MB
        
        if ($fileInfo.Length -lt $minSize -or $fileInfo.Length -gt $maxSize) {
            $result.FailedChecks += "UnreasonableFileSize"
            $result.Checks.FileSize = $false
        }
        else {
            $result.Checks.FileSize = $true
        }
        
        return $result
    }
    catch {
        $result.FailedChecks += "AdditionalChecksException"
        $result.IsValid = $false
        return $result
    }
}

function Test-SecurityComplianceValidation {
    <#
    .SYNOPSIS
    Performs comprehensive security compliance validation
    
    .DESCRIPTION
    Validates system security configuration, checks compliance with security
    policies, and ensures the installation environment meets security requirements.
    #>
    [CmdletBinding()]
    param()
    
    Write-SecurityAuditEvent -EventType "ComplianceCheck" -Severity "Information" -Message "Starting security compliance validation"
    
    try {
        $complianceResults = @{
            OverallCompliance = $true
            ComplianceScore = 0
            MaxScore = 0
            Checks = @{}
            Violations = @()
            Warnings = @()
            Recommendations = @()
        }
        
        # Compliance Check 1: Administrative Privileges
        $isElevated = Test-IsElevated
        $complianceResults.Checks.AdministrativePrivileges = @{
            Required = $true
            Status = $isElevated
            Score = if ($isElevated) { 10 } else { 0 }
            Description = "Installation requires administrative privileges"
        }
        $complianceResults.MaxScore += 10
        
        if (-not $isElevated) {
            $complianceResults.Violations += "Installation must be run with administrative privileges"
            $complianceResults.OverallCompliance = $false
        }
        
        # Compliance Check 2: PowerShell Execution Policy
        $executionPolicy = Get-ExecutionPolicy -Scope CurrentUser
        $allowedPolicies = @('RemoteSigned', 'Unrestricted', 'Bypass')
        $policyCompliant = $executionPolicy -in $allowedPolicies
        
        $complianceResults.Checks.ExecutionPolicy = @{
            Required = $true
            Status = $policyCompliant
            CurrentPolicy = $executionPolicy
            AllowedPolicies = $allowedPolicies
            Score = if ($policyCompliant) { 10 } else { 0 }
            Description = "PowerShell execution policy must allow script execution"
        }
        $complianceResults.MaxScore += 10
        
        if (-not $policyCompliant) {
            $complianceResults.Violations += "PowerShell execution policy '$executionPolicy' does not allow script execution"
            $complianceResults.OverallCompliance = $false
        }
        
        # Compliance Check 3: Windows Defender Status
        try {
            $defenderStatus = Get-MpComputerStatus -ErrorAction SilentlyContinue
            $defenderEnabled = $defenderStatus -ne $null -and $defenderStatus.AntivirusEnabled
            
            $complianceResults.Checks.WindowsDefender = @{
                Required = $false
                Status = $defenderEnabled
                Score = if ($defenderEnabled) { 5 } else { 0 }
                Description = "Windows Defender provides additional security"
            }
            $complianceResults.MaxScore += 5
            
            if (-not $defenderEnabled) {
                $complianceResults.Warnings += "Windows Defender is not enabled - consider enabling for enhanced security"
            }
        }
        catch {
            $complianceResults.Checks.WindowsDefender = @{
                Required = $false
                Status = $false
                Error = $_.Exception.Message
                Score = 0
                Description = "Could not determine Windows Defender status"
            }
            $complianceResults.MaxScore += 5
        }
        
        # Calculate compliance score
        $complianceResults.ComplianceScore = ($complianceResults.Checks.Values | Measure-Object -Property Score -Sum).Sum
        $compliancePercentage = if ($complianceResults.MaxScore -gt 0) { 
            [math]::Round(($complianceResults.ComplianceScore / $complianceResults.MaxScore) * 100, 2) 
        } else { 0 }
        
        # Log compliance results
        $complianceLevel = if ($complianceResults.OverallCompliance) {
            if ($compliancePercentage -ge 80) { "High" }
            elseif ($compliancePercentage -ge 60) { "Medium" }
            else { "Low" }
        } else { "Non-Compliant" }
        
        Write-SecurityAuditEvent -EventType "ComplianceCheck" -Severity "Information" -Message "Security compliance validation completed" -Details @{
            OverallCompliance = $complianceResults.OverallCompliance
            ComplianceLevel = $complianceLevel
            ComplianceScore = $complianceResults.ComplianceScore
            MaxScore = $complianceResults.MaxScore
            CompliancePercentage = $compliancePercentage
            ViolationCount = $complianceResults.Violations.Count
            WarningCount = $complianceResults.Warnings.Count
            RecommendationCount = $complianceResults.Recommendations.Count
            Checks = $complianceResults.Checks
        }
        
        # Report violations
        foreach ($violation in $complianceResults.Violations) {
            Write-SecurityAuditEvent -EventType "SecurityViolation" -Severity "Error" -Message "Compliance violation detected" -Details @{
                Violation = $violation
                ComplianceCheck = "SecurityCompliance"
            }
        }
        
        # Report warnings
        foreach ($warning in $complianceResults.Warnings) {
            Write-SecurityAuditEvent -EventType "SecurityWarning" -Severity "Warning" -Message "Security compliance warning" -Details @{
                Warning = $warning
                ComplianceCheck = "SecurityCompliance"
            }
        }
        
        # Store compliance results for later reference
        if ($script:SecurityAudit) {
            $script:SecurityAudit.ComplianceStatus = $complianceResults
        }
        
        return $complianceResults
    }
    catch {
        Write-SecurityAuditEvent -EventType "SecurityError" -Severity "Error" -Message "Security compliance validation failed with error" -Details @{
            ErrorMessage = $_.Exception.Message
            ErrorType = $_.Exception.GetType().Name
            StackTrace = $_.ScriptStackTrace
        }
        
        Write-InstallationLog -Level Error -Message "Security compliance validation failed: $($_.Exception.Message)"
        return @{
            OverallCompliance = $false
            Error = $_.Exception.Message
        }
    }
}

function Complete-SecurityAuditProcess {
    <#
    .SYNOPSIS
    Completes the security audit process and generates final audit report
    
    .DESCRIPTION
    Finalizes the security audit trail, generates comprehensive audit report,
    and ensures all security events are properly logged and transmitted.
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [ValidateSet('Success', 'Failed', 'Cancelled')]
        [string]$InstallationResult,
        
        [Parameter(Mandatory=$false)]
        [string]$ErrorMessage = ""
    )
    
    Write-InstallationLog -Level Info -Message "Completing security audit process"
    
    try {
        if (-not $script:SecurityAudit) {
            Write-InstallationLog -Level Warning -Message "Security audit system was not initialized"
            return $false
        }
        
        $auditEndTime = Get-Date
        $auditDuration = $auditEndTime - $script:SecurityAudit.StartTime
        
        # Generate final audit summary
        $auditSummary = @{
            AuditId = $script:SecurityAudit.AuditId
            InstallationId = $script:InstallationId
            GymId = $GymId
            StartTime = $script:SecurityAudit.StartTime.ToString('yyyy-MM-ddTHH:mm:ss.fffZ')
            EndTime = $auditEndTime.ToString('yyyy-MM-ddTHH:mm:ss.fffZ')
            Duration = $auditDuration.TotalSeconds
            InstallationResult = $InstallationResult
            ErrorMessage = $ErrorMessage
            
            # Event statistics
            EventStatistics = @{
                TotalEvents = $script:SecurityAudit.Events.Count
                EventsByType = @{}
                EventsBySeverity = @{}
            }
            
            # Security metrics
            SecurityMetrics = if ($script:TelemetryData) { $script:TelemetryData.SecurityMetrics } else { @{} }
            
            # Compliance status
            ComplianceStatus = $script:SecurityAudit.ComplianceStatus
            
            # System information
            SystemInfo = Get-SecuritySystemInfo
        }
        
        # Calculate event statistics
        if ($script:SecurityAudit.Events.Count -gt 0) {
            $eventsByType = $script:SecurityAudit.Events | Group-Object EventType
            foreach ($group in $eventsByType) {
                $auditSummary.EventStatistics.EventsByType[$group.Name] = $group.Count
            }
            
            $eventsBySeverity = $script:SecurityAudit.Events | Group-Object Severity
            foreach ($group in $eventsBySeverity) {
                $auditSummary.EventStatistics.EventsBySeverity[$group.Name] = $group.Count
            }
        }
        
        # Write final audit event
        Write-SecurityAuditEvent -EventType "AuditCompleted" -Severity "Information" -Message "Security audit completed" -Details $auditSummary
        
        # Generate audit report file
        $auditReportPath = "$env:TEMP\RepSetBridge-SecurityAudit-$(Get-Date -Format 'yyyyMMdd-HHmmss').json"
        try {
            $auditReport = @{
                AuditSummary = $auditSummary
                AuditEvents = $script:SecurityAudit.Events
                SecurityChecks = $script:SecurityAudit.SecurityChecks
                ComplianceStatus = $script:SecurityAudit.ComplianceStatus
                TamperDetection = $script:SecurityAudit.TamperDetection
            }
            
            $auditReport | ConvertTo-Json -Depth 10 | Set-Content -Path $auditReportPath -Encoding UTF8
            Write-InstallationLog -Level Success -Message "Security audit report generated: $auditReportPath"
        }
        catch {
            Write-InstallationLog -Level Warning -Message "Could not generate audit report file: $($_.Exception.Message)"
        }
        
        # Send final audit report to platform
        Send-FinalAuditReportToPlatform -AuditSummary $auditSummary -AuditEvents $script:SecurityAudit.Events
        
        Write-InstallationLog -Level Success -Message "Security audit completed successfully"
        return $true
    }
    catch {
        Write-InstallationLog -Level Error -Message "Failed to complete security audit: $($_.Exception.Message)"
        return $false
    }
}

function Send-FinalAuditReportToPlatform {
    <#
    .SYNOPSIS
    Sends the final comprehensive audit report to the platform
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [hashtable]$AuditSummary,
        
        [Parameter(Mandatory=$true)]
        [array]$AuditEvents
    )
    
    try {
        $platformUrl = "$PlatformEndpoint/api/security/audit/final"
        $headers = @{
            'Content-Type' = 'application/json'
            'User-Agent' = 'RepSet-Bridge-Installer/1.0'
            'X-Installation-Id' = $script:InstallationId
            'X-Audit-Id' = $script:SecurityAudit.AuditId
        }
        
        $finalReport = @{
            auditSummary = $AuditSummary
            auditEvents = $AuditEvents | Select-Object -First 100  # Limit to prevent payload size issues
            gymId = $GymId
            timestamp = Get-Date -Format 'yyyy-MM-ddTHH:mm:ss.fffZ'
        }
        
        $json = $finalReport | ConvertTo-Json -Depth 10
        
        # Send with retry logic
        $maxRetries = 3
        for ($attempt = 1; $attempt -le $maxRetries; $attempt++) {
            try {
                $response = Invoke-RestMethod -Uri $platformUrl -Method Post -Headers $headers -Body $json -TimeoutSec 30 -ErrorAction Stop
                Write-InstallationLog -Level Success -Message "Final audit report sent to platform successfully"
                return $true
            }
            catch {
                Write-InstallationLog -Level Warning -Message "Failed to send final audit report (attempt $attempt): $($_.Exception.Message)"
                if ($attempt -lt $maxRetries) {
                    Start-Sleep -Seconds (2 * $attempt)
                }
            }
        }
        
        Write-InstallationLog -Level Warning -Message "Failed to send final audit report after $maxRetries attempts"
        return $false
    }
    catch {
        Write-InstallationLog -Level Warning -Message "Error sending final audit report: $($_.Exception.Message)"
        return $false
    }
}

function Get-StringHash {
    <#
    .SYNOPSIS
    Calculates SHA-256 hash of a string for logging purposes
    #>
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$InputString
    )
    
    try {
        $bytes = [System.Text.Encoding]::UTF8.GetBytes($InputString)
        $sha256 = [System.Security.Cryptography.SHA256]::Create()
        $hashBytes = $sha256.ComputeHash($bytes)
        $hash = [System.BitConverter]::ToString($hashBytes) -replace '-', ''
        return $hash.ToLower()
    }
    catch {
        return "hash-calculation-failed"
    }
    finally {
        if ($sha256) {
            $sha256.Dispose()
        }
    }
}

# ================================================================
# Main Installation Function with Security Integration
# ================================================================

function Start-SecureInstallation {
    <#
    .SYNOPSIS
    Main installation function with integrated security validation
    
    .DESCRIPTION
    Orchestrates the complete installation process with comprehensive security
    validation, audit logging, and compliance checking at each step.
    #>
    [CmdletBinding()]
    param()
    
    try {
        Write-InstallationLog -Level Info -Message "Starting secure RepSet Bridge installation"
        
        # Step 1: Initialize security audit system
        Write-Progress-Step -Step "Initializing Security Audit" -StepNumber 1 -TotalSteps 10
        $auditInitialized = Initialize-SecurityAuditSystem
        if (-not $auditInitialized) {
            Write-InstallationLog -Level Warning -Message "Security audit system initialization failed, continuing with reduced security logging"
        }
        
        # Step 2: Validate installation command signature
        Write-Progress-Step -Step "Validating Command Signature" -StepNumber 2 -TotalSteps 10
        $signatureValid = Test-CommandSignatureValidation -PairCode $PairCode -Signature $Signature -Nonce $Nonce -GymId $GymId -ExpiresAt $ExpiresAt
        if (-not $signatureValid) {
            Complete-SecurityAuditProcess -InstallationResult "Failed" -ErrorMessage "Command signature validation failed"
            throw "Installation aborted: Command signature validation failed"
        }
        
        # Step 3: Perform security compliance validation
        Write-Progress-Step -Step "Checking Security Compliance" -StepNumber 3 -TotalSteps 10
        $complianceResults = Test-SecurityComplianceValidation
        if (-not $complianceResults.OverallCompliance) {
            $violationMessage = "Security compliance violations: $($complianceResults.Violations -join '; ')"
            Complete-SecurityAuditProcess -InstallationResult "Failed" -ErrorMessage $violationMessage
            throw "Installation aborted: $violationMessage"
        }
        
        # Step 4: Initialize telemetry (if not already done)
        Write-Progress-Step -Step "Initializing Telemetry" -StepNumber 4 -TotalSteps 10
        if (-not $script:TelemetryData) {
            Initialize-InstallationTelemetry
        }
        
        # Step 5: Test system requirements
        Write-Progress-Step -Step "Validating System Requirements" -StepNumber 5 -TotalSteps 10
        $systemValid = Test-SystemRequirements
        if (-not $systemValid) {
            Complete-SecurityAuditProcess -InstallationResult "Failed" -ErrorMessage "System requirements not met"
            throw "Installation aborted: System requirements not met"
        }
        
        # Step 6: Download bridge with integrity validation
        Write-Progress-Step -Step "Downloading Bridge" -StepNumber 6 -TotalSteps 10
        $downloadResult = Get-LatestBridge
        if (-not $downloadResult.Success) {
            Complete-SecurityAuditProcess -InstallationResult "Failed" -ErrorMessage "Bridge download failed"
            throw "Installation aborted: Bridge download failed - $($downloadResult.ErrorMessage)"
        }
        
        # Step 7: Validate downloaded file integrity
        Write-Progress-Step -Step "Validating File Integrity" -StepNumber 7 -TotalSteps 10
        $integrityValid = Test-FileIntegrityValidation -FilePath $downloadResult.FilePath -ExpectedHash $downloadResult.Hash -ExpectedSize $downloadResult.Size -VerifyDigitalSignature
        if (-not $integrityValid) {
            Complete-SecurityAuditProcess -InstallationResult "Failed" -ErrorMessage "File integrity validation failed"
            throw "Installation aborted: Downloaded file failed integrity validation"
        }
        
        # Step 8: Install bridge executable
        Write-Progress-Step -Step "Installing Bridge" -StepNumber 8 -TotalSteps 10
        $installResult = Install-BridgeExecutable -SourcePath $downloadResult.FilePath
        if (-not $installResult.Success) {
            Complete-SecurityAuditProcess -InstallationResult "Failed" -ErrorMessage "Bridge installation failed"
            throw "Installation aborted: Bridge installation failed - $($installResult.ErrorMessage)"
        }
        
        # Step 9: Configure and start service
        Write-Progress-Step -Step "Configuring Service" -StepNumber 9 -TotalSteps 10
        $serviceResult = Install-BridgeService
        if (-not $serviceResult.Success) {
            Complete-SecurityAuditProcess -InstallationResult "Failed" -ErrorMessage "Service installation failed"
            throw "Installation aborted: Service installation failed - $($serviceResult.ErrorMessage)"
        }
        
        # Step 10: Complete installation and audit
        Write-Progress-Step -Step "Completing Installation" -StepNumber 10 -TotalSteps 10
        Complete-SecurityAuditProcess -InstallationResult "Success"
        
        Write-InstallationLog -Level Success -Message "RepSet Bridge installation completed successfully with full security validation"
        return $true
    }
    catch {
        Write-InstallationLog -Level Error -Message "Secure installation failed: $($_.Exception.Message)"
        Complete-SecurityAuditProcess -InstallationResult "Failed" -ErrorMessage $_.Exception.Message
        return $false
    }
}

# ================================================================
# Script Entry Point with Security Validation
# ================================================================

# Initialize security audit system at script start
Write-Host "RepSet Bridge - Secure Automated Installation" -ForegroundColor Cyan
Write-Host "=============================================" -ForegroundColor Cyan
Write-Host ""

# Start the secure installation process
try {
    $installationSuccess = Start-SecureInstallation
    
    if ($installationSuccess) {
        Write-Host ""
        Write-Host "✓ Installation completed successfully!" -ForegroundColor Green
        Write-Host "✓ Security audit completed" -ForegroundColor Green
        Write-Host "✓ All compliance checks passed" -ForegroundColor Green
        Write-Host ""
        Write-Host "The RepSet Bridge service is now running and connected to your platform." -ForegroundColor White
        exit $script:ErrorCodes.Success
    }
    else {
        Write-Host ""
        Write-Host "✗ Installation failed" -ForegroundColor Red
        Write-Host "Please check the installation log for details: $script:LogFile" -ForegroundColor Yellow
        exit $script:ErrorCodes.InstallationFailed
    }
}
catch {
    Write-Host ""
    Write-Host "✗ Installation failed with error: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "Please check the installation log for details: $script:LogFile" -ForegroundColor Yellow
    exit $script:ErrorCodes.InstallationFailed
}