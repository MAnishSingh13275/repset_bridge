# ================================================================
# RepSet Bridge - Complete Integration Test Executor
# Executes all integration tests and validates the complete workflow
# ================================================================

[CmdletBinding()]
param(
    [Parameter(Mandatory=$false)]
    [ValidateSet('Quick', 'Standard', 'Comprehensive')]
    [string]$TestLevel = 'Standard',
    
    [Parameter(Mandatory=$false)]
    [string]$OutputPath = "$env:TEMP\RepSetBridge-Complete-Integration-Results",
    
    [Parameter(Mandatory=$false)]
    [switch]$IncludeSecurityValidation,
    
    [Parameter(Mandatory=$false)]
    [switch]$IncludeUserAcceptanceTesting,
    
    [Parameter(Mandatory=$false)]
    [switch]$IncludeCrossPlatformTesting,
    
    [Parameter(Mandatory=$false)]
    [switch]$GenerateExecutiveSummary,
    
    [Parameter(Mandatory=$false)]
    [int]$TimeoutMinutes = 90
)

# ================================================================
# Complete Integration Test Configuration
# ================================================================

$IntegrationTestConfig = @{
    TestLevel = $TestLevel
    OutputPath = $OutputPath
    TimeoutSeconds = $TimeoutMinutes * 60
    ExecutionStartTime = Get-Date
    
    # Test suite configurations
    TestSuites = @{
        UnitTests = @{
            Name = "Unit Tests"
            ScriptPath = Join-Path $PSScriptRoot ".." "installation-tests" "Install-RepSetBridge.Tests.ps1"
            Description = "Unit tests for individual PowerShell functions"
            Priority = "High"
            EstimatedDuration = 300  # 5 minutes
            Required = $true
        }
        IntegrationTests = @{
            Name = "Integration Tests"
            ScriptPath = Join-Path $PSScriptRoot ".." "installation-tests" "Integration.Tests.ps1"
            Description = "End-to-end integration tests for complete workflows"
            Priority = "High"
            EstimatedDuration = 600  # 10 minutes
            Required = $true
        }
        SecurityTests = @{
            Name = "Security Tests"
            ScriptPath = Join-Path $PSScriptRoot ".." "installation-tests" "Security.Tests.ps1"
            Description = "Security tests for signature validation and tampering detection"
            Priority = "High"
            EstimatedDuration = 300  # 5 minutes
            Required = $true
        }
        CrossPlatformValidation = @{
            Name = "Cross-Platform Validation"
            ScriptPath = Join-Path $PSScriptRoot "Cross-Platform-Validator.ps1"
            Description = "Cross-platform compatibility validation"
            Priority = "Medium"
            EstimatedDuration = 900  # 15 minutes
            Required = $false
        }
        DeploymentReadiness = @{
            Name = "Deployment Readiness"
            ScriptPath = Join-Path $PSScriptRoot "Deployment-Readiness-Validator.ps1"
            Description = "Final deployment readiness validation"
            Priority = "High"
            EstimatedDuration = 1200  # 20 minutes
            Required = $true
        }
    }
}

# ================================================================
# Complete Integration Test Functions
# ================================================================

function Initialize-IntegrationTestEnvironment {
    <#
    .SYNOPSIS
    Initializes the complete integration test environment
    #>
    
    Write-Host "Initializing Complete Integration Test Environment..." -ForegroundColor Cyan
    
    # Create output directory structure
    if (-not (Test-Path $IntegrationTestConfig.OutputPath)) {
        New-Item -ItemType Directory -Path $IntegrationTestConfig.OutputPath -Force | Out-Null
    }
    
    $subDirectories = @('results', 'logs', 'reports', 'artifacts')
    foreach ($dir in $subDirectories) {
        $dirPath = Join-Path $IntegrationTestConfig.OutputPath $dir
        if (-not (Test-Path $dirPath)) {
            New-Item -ItemType Directory -Path $dirPath -Force | Out-Null
        }
    }
    
    # Initialize execution log
    $logFile = Join-Path $IntegrationTestConfig.OutputPath "logs" "complete-integration-execution.log"
    $logEntry = "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') - Complete Integration Test Execution Started"
    Set-Content -Path $logFile -Value $logEntry
    
    Write-Host "✓ Integration test environment initialized" -ForegroundColor Green
    return $logFile
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
    
    Write-Host "`nExecuting $($SuiteConfig.Name)..." -ForegroundColor Yellow
    Write-Host "Description: $($SuiteConfig.Description)" -ForegroundColor Gray
    Write-Host "Priority: $($SuiteConfig.Priority)" -ForegroundColor Gray
    Write-Host "Estimated Duration: $([math]::Round($SuiteConfig.EstimatedDuration / 60, 1)) minutes" -ForegroundColor Gray
    
    # Log test suite start
    Add-Content -Path $LogFile -Value "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') - Starting $($SuiteConfig.Name)"
    
    if (-not (Test-Path $SuiteConfig.ScriptPath)) {
        Write-Host "✗ Test script not found: $($SuiteConfig.ScriptPath)" -ForegroundColor Red
        Add-Content -Path $LogFile -Value "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') - ERROR: Test script not found: $($SuiteConfig.ScriptPath)"
        return @{
            SuiteName = $SuiteName
            Success = $false
            Error = "Test script not found"
            Duration = 0
        }
    }
    
    try {
        $startTime = Get-Date
        
        # Execute test suite based on type
        $result = switch ($SuiteName) {
            "UnitTests" { 
                Import-Module Pester -Force
                Invoke-Pester -Path $SuiteConfig.ScriptPath -PassThru
            }
            "IntegrationTests" { 
                Import-Module Pester -Force
                Invoke-Pester -Path $SuiteConfig.ScriptPath -PassThru
            }
            "SecurityTests" { 
                Import-Module Pester -Force
                Invoke-Pester -Path $SuiteConfig.ScriptPath -PassThru
            }
            "CrossPlatformValidation" { 
                & $SuiteConfig.ScriptPath
                @{ Success = $LASTEXITCODE -eq 0; TotalCount = 1; PassedCount = if ($LASTEXITCODE -eq 0) { 1 } else { 0 }; FailedCount = if ($LASTEXITCODE -eq 0) { 0 } else { 1 } }
            }
            "DeploymentReadiness" { 
                & $SuiteConfig.ScriptPath
                @{ Success = $LASTEXITCODE -eq 0; TotalCount = 1; PassedCount = if ($LASTEXITCODE -eq 0) { 1 } else { 0 }; FailedCount = if ($LASTEXITCODE -eq 0) { 0 } else { 1 } }
            }
            default { 
                throw "Unknown test suite: $SuiteName"
            }
        }
        
        $endTime = Get-Date
        $duration = ($endTime - $startTime).TotalSeconds
        
        $success = if ($result.GetType().Name -eq "Hashtable") { 
            $result.Success 
        } else { 
            $result.FailedCount -eq 0 
        }
        
        # Display results
        if ($success) {
            Write-Host "✓ $($SuiteConfig.Name) completed successfully" -ForegroundColor Green
            if ($result.TotalCount) {
                Write-Host "  Tests: $($result.TotalCount) | Passed: $($result.PassedCount) | Failed: $($result.FailedCount)" -ForegroundColor White
            }
        } else {
            Write-Host "✗ $($SuiteConfig.Name) failed" -ForegroundColor Red
            if ($result.TotalCount) {
                Write-Host "  Tests: $($result.TotalCount) | Passed: $($result.PassedCount) | Failed: $($result.FailedCount)" -ForegroundColor White
            }
        }
        
        Write-Host "  Duration: $([math]::Round($duration, 1)) seconds" -ForegroundColor White
        
        # Log results
        Add-Content -Path $LogFile -Value "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') - $($SuiteConfig.Name) completed - Success: $success, Duration: $([math]::Round($duration, 1))s"
        
        return @{
            SuiteName = $SuiteName
            Success = $success
            Duration = $duration
            TestResult = $result
        }
    }
    catch {
        $endTime = Get-Date
        $duration = ($endTime - $startTime).TotalSeconds
        
        Write-Host "✗ Error executing $($SuiteConfig.Name): $($_.Exception.Message)" -ForegroundColor Red
        Add-Content -Path $LogFile -Value "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') - ERROR in $($SuiteConfig.Name): $($_.Exception.Message)"
        
        return @{
            SuiteName = $SuiteName
            Success = $false
            Error = $_.Exception.Message
            Duration = $duration
        }
    }
}

# ================================================================
# Main Complete Integration Test Execution
# ================================================================

Write-Host "RepSet Bridge Complete Integration Test Executor" -ForegroundColor Yellow
Write-Host "===============================================" -ForegroundColor Yellow
Write-Host "Test Level: $TestLevel" -ForegroundColor White
Write-Host "Output Path: $($IntegrationTestConfig.OutputPath)" -ForegroundColor White
Write-Host "Timeout: $TimeoutMinutes minutes" -ForegroundColor White

# Initialize test environment
$logFile = Initialize-IntegrationTestEnvironment

# Determine which test suites to run based on test level and parameters
$suitesToRun = @()

switch ($TestLevel) {
    "Quick" {
        $suitesToRun = @("UnitTests", "SecurityTests")
    }
    "Standard" {
        $suitesToRun = @("UnitTests", "IntegrationTests", "SecurityTests", "DeploymentReadiness")
    }
    "Comprehensive" {
        $suitesToRun = $IntegrationTestConfig.TestSuites.Keys
    }
}

# Add optional test suites based on parameters
if ($IncludeCrossPlatformTesting -and $suitesToRun -notcontains "CrossPlatformValidation") {
    $suitesToRun += "CrossPlatformValidation"
}

# Execute test suites
$executionResults = @{}
$totalDuration = 0
$successfulSuites = 0
$totalSuites = $suitesToRun.Count

foreach ($suiteName in $suitesToRun) {
    if ($IntegrationTestConfig.TestSuites.ContainsKey($suiteName)) {
        $suiteConfig = $IntegrationTestConfig.TestSuites[$suiteName]
        $result = Invoke-TestSuite -SuiteName $suiteName -SuiteConfig $suiteConfig -LogFile $logFile
        
        $executionResults[$suiteName] = $result
        $totalDuration += $result.Duration
        
        if ($result.Success) {
            $successfulSuites++
        }
        
        # Stop on critical failures for required test suites
        if (-not $result.Success -and $suiteConfig.Required) {
            Write-Host "`nCritical test suite failed. Stopping execution." -ForegroundColor Red
            break
        }
    } else {
        Write-Host "Unknown test suite: $suiteName" -ForegroundColor Red
    }
}

# Generate execution summary
Write-Host "`n" + "="*60 -ForegroundColor Yellow
Write-Host "COMPLETE INTEGRATION TEST EXECUTION SUMMARY" -ForegroundColor Yellow
Write-Host "="*60 -ForegroundColor Yellow
Write-Host "Test Level: $TestLevel" -ForegroundColor White
Write-Host "Successful Suites: $successfulSuites / $totalSuites" -ForegroundColor White
Write-Host "Success Rate: $(if ($totalSuites -gt 0) { [math]::Round(($successfulSuites / $totalSuites) * 100, 2) } else { 0 })%" -ForegroundColor White
Write-Host "Total Duration: $([math]::Round($totalDuration / 60, 1)) minutes" -ForegroundColor White
Write-Host "Execution Time: $((Get-Date) - $IntegrationTestConfig.ExecutionStartTime)" -ForegroundColor White

# Display individual suite results
Write-Host "`nIndividual Suite Results:" -ForegroundColor Yellow
foreach ($result in $executionResults.Values) {
    $status = if ($result.Success) { "✓ PASSED" } else { "✗ FAILED" }
    $color = if ($result.Success) { "Green" } else { "Red" }
    Write-Host "  $status - $($result.SuiteName) ($([math]::Round($result.Duration, 1))s)" -ForegroundColor $color
}

# Save execution results
$resultsPath = Join-Path $IntegrationTestConfig.OutputPath "results" "complete-integration-results.json"
$executionResults | ConvertTo-Json -Depth 4 | Set-Content -Path $resultsPath

Write-Host "`nComplete integration test results saved to: $resultsPath" -ForegroundColor Cyan

# Final status
$overallSuccess = $successfulSuites -eq $totalSuites
$finalStatus = if ($overallSuccess) { "✓ ALL TESTS PASSED" } else { "✗ SOME TESTS FAILED" }
$finalColor = if ($overallSuccess) { "Green" } else { "Red" }

Write-Host "`n$finalStatus" -ForegroundColor $finalColor

# Exit with appropriate code
if ($overallSuccess) {
    exit 0
} else {
    exit 1
}