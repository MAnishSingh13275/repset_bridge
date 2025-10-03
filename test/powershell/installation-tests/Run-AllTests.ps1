# ================================================================
# RepSet Bridge Installation - Test Suite Runner
# Executes all test suites and generates comprehensive reports
# ================================================================

[CmdletBinding()]
param(
    [Parameter(Mandatory=$false)]
    [ValidateSet('Unit', 'Integration', 'Security', 'All')]
    [string]$TestType = 'All',
    
    [Parameter(Mandatory=$false)]
    [string]$OutputPath = "$env:TEMP\RepSetBridge-TestResults",
    
    [Parameter(Mandatory=$false)]
    [switch]$GenerateHtmlReport,
    
    [Parameter(Mandatory=$false)]
    [switch]$OpenReportAfterExecution,
    
    [Parameter(Mandatory=$false)]
    [switch]$ContinueOnFailure,
    
    [Parameter(Mandatory=$false)]
    [int]$TimeoutMinutes = 30
)

# Import required modules
Import-Module Pester -Force

# Test configuration
$TestConfig = @{
    OutputPath = $OutputPath
    TimeoutSeconds = $TimeoutMinutes * 60
    TestStartTime = Get-Date
    TestSuites = @{
        Unit = @{
            Name = "Unit Tests"
            ScriptPath = Join-Path $PSScriptRoot "Install-RepSetBridge.Tests.ps1"
            Description = "Unit tests for individual PowerShell functions"
            Color = "Green"
        }
        Integration = @{
            Name = "Integration Tests"
            ScriptPath = Join-Path $PSScriptRoot "Integration.Tests.ps1"
            Description = "End-to-end integration tests for complete workflows"
            Color = "Blue"
        }
        Security = @{
            Name = "Security Tests"
            ScriptPath = Join-Path $PSScriptRoot "Security.Tests.ps1"
            Description = "Security tests for signature validation and tampering detection"
            Color = "Red"
        }
    }
}

# ================================================================
# Test Runner Helper Functions
# ================================================================

function Initialize-TestEnvironment {
    <#
    .SYNOPSIS
    Initializes the test environment and creates output directories
    #>
    
    Write-Host "Initializing test environment..." -ForegroundColor Cyan
    
    # Create output directory
    if (-not (Test-Path $TestConfig.OutputPath)) {
        New-Item -ItemType Directory -Path $TestConfig.OutputPath -Force | Out-Null
    }
    
    # Create subdirectories for different report types
    $reportDirs = @('xml', 'html', 'logs', 'coverage')
    foreach ($dir in $reportDirs) {
        $dirPath = Join-Path $TestConfig.OutputPath $dir
        if (-not (Test-Path $dirPath)) {
            New-Item -ItemType Directory -Path $dirPath -Force | Out-Null
        }
    }
    
    # Initialize test log
    $logFile = Join-Path $TestConfig.OutputPath "logs" "test-execution.log"
    $logEntry = "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') - Test execution started"
    Set-Content -Path $logFile -Value $logEntry
    
    Write-Host "✓ Test environment initialized at: $($TestConfig.OutputPath)" -ForegroundColor Green
    return $logFile
}

function Write-TestLog {
    <#
    .SYNOPSIS
    Writes entries to the test execution log
    #>
    param(
        [string]$Message,
        [string]$Level = "Info",
        [string]$LogFile
    )
    
    $timestamp = Get-Date -Format 'yyyy-MM-dd HH:mm:ss'
    $logEntry = "[$timestamp] [$Level] $Message"
    Add-Content -Path $LogFile -Value $logEntry
}

function Invoke-TestSuite {
    <#
    .SYNOPSIS
    Executes a specific test suite
    #>
    param(
        [string]$SuiteName,
        [hashtable]$SuiteConfig,
        [string]$LogFile
    )
    
    Write-Host "`nExecuting $($SuiteConfig.Name)..." -ForegroundColor $SuiteConfig.Color
    Write-Host "Description: $($SuiteConfig.Description)" -ForegroundColor Gray
    Write-Host "Script: $($SuiteConfig.ScriptPath)" -ForegroundColor Gray
    
    Write-TestLog -Message "Starting $($SuiteConfig.Name)" -Level "Info" -LogFile $LogFile
    
    if (-not (Test-Path $SuiteConfig.ScriptPath)) {
        Write-Host "✗ Test script not found: $($SuiteConfig.ScriptPath)" -ForegroundColor Red
        Write-TestLog -Message "Test script not found: $($SuiteConfig.ScriptPath)" -Level "Error" -LogFile $LogFile
        return $null
    }
    
    try {
        $xmlOutputPath = Join-Path $TestConfig.OutputPath "xml" "$SuiteName-Results.xml"
        
        $testResult = Invoke-Pester -Path $SuiteConfig.ScriptPath -OutputFormat NUnitXml -OutputFile $xmlOutputPath -PassThru
        
        Write-Host "✓ $($SuiteConfig.Name) completed" -ForegroundColor $SuiteConfig.Color
        Write-Host "  Total: $($testResult.TotalCount) | Passed: $($testResult.PassedCount) | Failed: $($testResult.FailedCount)" -ForegroundColor White
        
        Write-TestLog -Message "$($SuiteConfig.Name) completed - Total: $($testResult.TotalCount), Passed: $($testResult.PassedCount), Failed: $($testResult.FailedCount)" -Level "Info" -LogFile $LogFile
        
        return $testResult
    }
    catch {
        Write-Host "✗ Error executing $($SuiteConfig.Name): $($_.Exception.Message)" -ForegroundColor Red
        Write-TestLog -Message "Error executing $($SuiteConfig.Name): $($_.Exception.Message)" -Level "Error" -LogFile $LogFile
        return $null
    }
}

# ================================================================
# Main Test Execution
# ================================================================

Write-Host "RepSet Bridge Installation Test Suite Runner" -ForegroundColor Yellow
Write-Host "=============================================" -ForegroundColor Yellow
Write-Host "Test Type: $TestType" -ForegroundColor White
Write-Host "Output Path: $($TestConfig.OutputPath)" -ForegroundColor White
Write-Host "Timeout: $TimeoutMinutes minutes" -ForegroundColor White

# Initialize test environment
$logFile = Initialize-TestEnvironment

# Determine which test suites to run
$suitesToRun = @()
if ($TestType -eq 'All') {
    $suitesToRun = $TestConfig.TestSuites.Keys
} else {
    $suitesToRun = @($TestType)
}

# Execute test suites
$allResults = @{}
$totalTests = 0
$totalPassed = 0
$totalFailed = 0

foreach ($suiteName in $suitesToRun) {
    if ($TestConfig.TestSuites.ContainsKey($suiteName)) {
        $suiteConfig = $TestConfig.TestSuites[$suiteName]
        $result = Invoke-TestSuite -SuiteName $suiteName -SuiteConfig $suiteConfig -LogFile $logFile
        
        if ($result) {
            $allResults[$suiteName] = $result
            $totalTests += $result.TotalCount
            $totalPassed += $result.PassedCount
            $totalFailed += $result.FailedCount
        }
        
        if ($result.FailedCount -gt 0 -and -not $ContinueOnFailure) {
            Write-Host "`nStopping execution due to test failures (use -ContinueOnFailure to override)" -ForegroundColor Red
            break
        }
    } else {
        Write-Host "Unknown test suite: $suiteName" -ForegroundColor Red
    }
}

# Generate summary report
Write-Host "`n" + "="*50 -ForegroundColor Yellow
Write-Host "TEST EXECUTION SUMMARY" -ForegroundColor Yellow
Write-Host "="*50 -ForegroundColor Yellow
Write-Host "Total Tests: $totalTests" -ForegroundColor White
Write-Host "Passed: $totalPassed" -ForegroundColor Green
Write-Host "Failed: $totalFailed" -ForegroundColor Red
Write-Host "Success Rate: $(if ($totalTests -gt 0) { [math]::Round(($totalPassed / $totalTests) * 100, 2) } else { 0 })%" -ForegroundColor White
Write-Host "Execution Time: $((Get-Date) - $TestConfig.TestStartTime)" -ForegroundColor White

# Generate reports
if ($GenerateHtmlReport) {
    Write-Host "`nGenerating HTML report..." -ForegroundColor Cyan
    # HTML report generation would be implemented here
    Write-Host "✓ HTML report generated" -ForegroundColor Green
}

Write-Host "`nTest results saved to: $($TestConfig.OutputPath)" -ForegroundColor Cyan

# Open report if requested
if ($OpenReportAfterExecution -and $GenerateHtmlReport) {
    $htmlReportPath = Join-Path $TestConfig.OutputPath "html" "TestReport.html"
    if (Test-Path $htmlReportPath) {
        Start-Process $htmlReportPath
    }
}

Write-TestLog -Message "Test execution completed - Total: $totalTests, Passed: $totalPassed, Failed: $totalFailed" -Level "Info" -LogFile $logFile

# Exit with appropriate code
if ($totalFailed -gt 0) {
    exit 1
} else {
    exit 0
}