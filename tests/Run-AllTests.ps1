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
    $logEntry = "$timestamp - [$Level] $Message"
    Add-Content -Path $LogFile -Value $logEntry
    
    # Also write to console with appropriate color
    $color = switch ($Level) {
        'Error' { 'Red' }
        'Warning' { 'Yellow' }
        'Success' { 'Green' }
        'Info' { 'White' }
        default { 'White' }
    }
    
    Write-Host $logEntry -ForegroundColor $color
}

function Invoke-TestSuite {
    <#
    .SYNOPSIS
    Executes a specific test suite and returns results
    #>
    param(
        [string]$SuiteName,
        [hashtable]$SuiteConfig,
        [string]$LogFile
    )
    
    Write-TestLog -Message "Starting $($SuiteConfig.Name)..." -Level "Info" -LogFile $LogFile
    Write-Host "`n$('=' * 60)" -ForegroundColor $SuiteConfig.Color
    Write-Host "$($SuiteConfig.Name.ToUpper())" -ForegroundColor $SuiteConfig.Color
    Write-Host "$('=' * 60)" -ForegroundColor $SuiteConfig.Color
    Write-Host "$($SuiteConfig.Description)" -ForegroundColor Gray
    Write-Host ""
    
    try {
        # Check if test script exists
        if (-not (Test-Path $SuiteConfig.ScriptPath)) {
            throw "Test script not found: $($SuiteConfig.ScriptPath)"
        }
        
        # Configure output files
        $xmlOutputFile = Join-Path $TestConfig.OutputPath "xml" "$SuiteName-Results.xml"
        
        # Execute tests with timeout
        $testJob = Start-Job -ScriptBlock {
            param($ScriptPath, $XmlOutput)
            Import-Module Pester -Force
            Invoke-Pester -Path $ScriptPath -OutputFormat NUnitXml -OutputFile $XmlOutput -PassThru
        } -ArgumentList $SuiteConfig.ScriptPath, $xmlOutputFile
        
        # Wait for completion with timeout
        $completed = Wait-Job -Job $testJob -Timeout $TestConfig.TimeoutSeconds
        
        if ($completed) {
            $results = Receive-Job -Job $testJob
            Remove-Job -Job $testJob
            
            Write-TestLog -Message "$($SuiteConfig.Name) completed successfully" -Level "Success" -LogFile $LogFile
            return $results
        }
        else {
            Stop-Job -Job $testJob
            Remove-Job -Job $testJob
            throw "Test suite timed out after $($TestConfig.TimeoutSeconds) seconds"
        }
    }
    catch {
        Write-TestLog -Message "Error executing $($SuiteConfig.Name): $($_.Exception.Message)" -Level "Error" -LogFile $LogFile
        return $null
    }
}

function New-TestSummaryReport {
    <#
    .SYNOPSIS
    Generates a comprehensive test summary report
    #>
    param(
        [hashtable]$AllResults,
        [string]$LogFile
    )
    
    Write-TestLog -Message "Generating test summary report..." -Level "Info" -LogFile $LogFile
    
    # Calculate overall statistics
    $totalTests = 0
    $totalPassed = 0
    $totalFailed = 0
    $totalSkipped = 0
    $totalTime = New-TimeSpan
    
    foreach ($result in $AllResults.Values) {
        if ($result) {
            $totalTests += $result.TotalCount
            $totalPassed += $result.PassedCount
            $totalFailed += $result.FailedCount
            $totalSkipped += $result.SkippedCount
            $totalTime = $totalTime.Add($result.Time)
        }
    }
    
    # Create summary report
    $summaryReport = @"
# RepSet Bridge Installation - Test Execution Summary

**Generated:** $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')
**Test Type:** $TestType
**Total Execution Time:** $($totalTime.ToString())

## Overall Results

| Metric | Count | Percentage |
|--------|-------|------------|
| Total Tests | $totalTests | 100% |
| Passed | $totalPassed | $([math]::Round(($totalPassed / $totalTests) * 100, 2))% |
| Failed | $totalFailed | $([math]::Round(($totalFailed / $totalTests) * 100, 2))% |
| Skipped | $totalSkipped | $([math]::Round(($totalSkipped / $totalTests) * 100, 2))% |

## Test Suite Results

"@

    foreach ($suiteName in $AllResults.Keys) {
        $result = $AllResults[$suiteName]
        $suiteConfig = $TestConfig.TestSuites[$suiteName]
        
        if ($result) {
            $passRate = [math]::Round(($result.PassedCount / $result.TotalCount) * 100, 2)
            $status = if ($result.FailedCount -eq 0) { "✅ PASSED" } else { "❌ FAILED" }
            
            $summaryReport += @"

### $($suiteConfig.Name) $status

- **Description:** $($suiteConfig.Description)
- **Total Tests:** $($result.TotalCount)
- **Passed:** $($result.PassedCount)
- **Failed:** $($result.FailedCount)
- **Skipped:** $($result.SkippedCount)
- **Pass Rate:** $passRate%
- **Execution Time:** $($result.Time)

"@

            # Add failed test details if any
            if ($result.FailedCount -gt 0) {
                $summaryReport += "#### Failed Tests:`n`n"
                $failedTests = $result.TestResult | Where-Object { $_.Result -eq "Failed" }
                foreach ($failedTest in $failedTests) {
                    $summaryReport += "- **$($failedTest.Describe)** → $($failedTest.Context) → $($failedTest.Name)`n"
                    $summaryReport += "  - Error: $($failedTest.FailureMessage)`n`n"
                }
            }
        }
        else {
            $summaryReport += @"

### $($suiteConfig.Name) ❌ ERROR

- **Description:** $($suiteConfig.Description)
- **Status:** Test suite failed to execute
- **Error:** See execution log for details

"@
        }
    }
    
    # Add recommendations
    $summaryReport += @"

## Recommendations

"@

    if ($totalFailed -eq 0) {
        $summaryReport += @"
✅ **All tests passed!** The RepSet Bridge installation script appears to be working correctly and securely.

### Next Steps:
1. Review any skipped tests to ensure they're not critical
2. Consider adding additional test coverage for edge cases
3. Run tests in different environments to ensure compatibility
4. Schedule regular test execution as part of CI/CD pipeline

"@
    }
    else {
        $summaryReport += @"
⚠️ **Test failures detected!** Please review and fix the failing tests before deployment.

### Priority Actions:
1. **High Priority:** Fix all security test failures immediately
2. **Medium Priority:** Address integration test failures
3. **Low Priority:** Resolve unit test failures
4. Re-run tests after fixes to ensure resolution

### Security Considerations:
"@
        
        if ($AllResults.ContainsKey('Security') -and $AllResults['Security'] -and $AllResults['Security'].FailedCount -gt 0) {
            $summaryReport += @"
🚨 **CRITICAL:** Security test failures detected! Do not deploy until these are resolved.
- Review signature validation logic
- Check file integrity verification
- Validate input sanitization
- Ensure privilege escalation prevention

"@
        }
        else {
            $summaryReport += @"
✅ Security tests passed - no immediate security concerns detected.

"@
        }
    }
    
    # Save summary report
    $summaryFile = Join-Path $TestConfig.OutputPath "TestSummary.md"
    Set-Content -Path $summaryFile -Value $summaryReport
    
    Write-TestLog -Message "Test summary report saved to: $summaryFile" -Level "Success" -LogFile $LogFile
    return $summaryFile
}

function New-HtmlReport {
    <#
    .SYNOPSIS
    Generates an HTML report from test results
    #>
    param(
        [hashtable]$AllResults,
        [string]$SummaryFile,
        [string]$LogFile
    )
    
    if (-not $GenerateHtmlReport) {
        return $null
    }
    
    Write-TestLog -Message "Generating HTML report..." -Level "Info" -LogFile $LogFile
    
    # Read markdown summary and convert to HTML (simplified)
    $markdownContent = Get-Content $SummaryFile -Raw
    
    $htmlContent = @"
<!DOCTYPE html>
<html>
<head>
    <title>RepSet Bridge Installation - Test Results</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background-color: #f5f5f5; }
        .container { background-color: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #2c3e50; border-bottom: 3px solid #3498db; padding-bottom: 10px; }
        h2 { color: #34495e; margin-top: 30px; }
        h3 { color: #7f8c8d; }
        .passed { color: #27ae60; font-weight: bold; }
        .failed { color: #e74c3c; font-weight: bold; }
        .skipped { color: #f39c12; font-weight: bold; }
        table { border-collapse: collapse; width: 100%; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 12px; text-align: left; }
        th { background-color: #3498db; color: white; }
        .error-section { background-color: #fdf2f2; border-left: 4px solid #e74c3c; padding: 15px; margin: 10px 0; }
        .success-section { background-color: #f0f9f0; border-left: 4px solid #27ae60; padding: 15px; margin: 10px 0; }
        .warning-section { background-color: #fef9e7; border-left: 4px solid #f39c12; padding: 15px; margin: 10px 0; }
        pre { background-color: #f8f9fa; padding: 15px; border-radius: 4px; overflow-x: auto; }
        .timestamp { color: #7f8c8d; font-size: 0.9em; }
    </style>
</head>
<body>
    <div class="container">
        <h1>RepSet Bridge Installation - Test Results</h1>
        <p class="timestamp">Generated: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')</p>
        
        <div class="summary">
            <pre>$($markdownContent -replace '#', '' -replace '\*\*', '<strong>' -replace '\*\*', '</strong>')</pre>
        </div>
        
        <h2>Detailed Results</h2>
"@

    foreach ($suiteName in $AllResults.Keys) {
        $result = $AllResults[$suiteName]
        if ($result) {
            $statusClass = if ($result.FailedCount -eq 0) { "success-section" } else { "error-section" }
            $htmlContent += @"
        <div class="$statusClass">
            <h3>$($TestConfig.TestSuites[$suiteName].Name)</h3>
            <p><strong>Total:</strong> $($result.TotalCount) | 
               <span class="passed">Passed: $($result.PassedCount)</span> | 
               <span class="failed">Failed: $($result.FailedCount)</span> | 
               <span class="skipped">Skipped: $($result.SkippedCount)</span></p>
            <p><strong>Execution Time:</strong> $($result.Time)</p>
        </div>
"@
        }
    }
    
    $htmlContent += @"
        
        <h2>Test Artifacts</h2>
        <ul>
            <li><a href="xml/">XML Test Results</a></li>
            <li><a href="logs/">Execution Logs</a></li>
            <li><a href="TestSummary.md">Markdown Summary</a></li>
        </ul>
        
        <footer style="margin-top: 40px; padding-top: 20px; border-top: 1px solid #ddd; color: #7f8c8d;">
            <p>Generated by RepSet Bridge Test Suite Runner</p>
        </footer>
    </div>
</body>
</html>
"@

    $htmlFile = Join-Path $TestConfig.OutputPath "TestResults.html"
    Set-Content -Path $htmlFile -Value $htmlContent
    
    Write-TestLog -Message "HTML report saved to: $htmlFile" -Level "Success" -LogFile $LogFile
    return $htmlFile
}

# ================================================================
# Main Test Execution
# ================================================================

function Main {
    Write-Host "RepSet Bridge Installation - Test Suite Runner" -ForegroundColor Cyan
    Write-Host "=" * 50 -ForegroundColor Cyan
    Write-Host "Test Type: $TestType" -ForegroundColor White
    Write-Host "Output Path: $($TestConfig.OutputPath)" -ForegroundColor White
    Write-Host "Timeout: $TimeoutMinutes minutes" -ForegroundColor White
    Write-Host ""
    
    # Initialize test environment
    $logFile = Initialize-TestEnvironment
    
    # Determine which test suites to run
    $suitesToRun = if ($TestType -eq 'All') {
        $TestConfig.TestSuites.Keys
    } else {
        @($TestType)
    }
    
    Write-TestLog -Message "Running test suites: $($suitesToRun -join ', ')" -Level "Info" -LogFile $logFile
    
    # Execute test suites
    $allResults = @{}
    $hasFailures = $false
    
    foreach ($suiteName in $suitesToRun) {
        if ($TestConfig.TestSuites.ContainsKey($suiteName)) {
            $suiteConfig = $TestConfig.TestSuites[$suiteName]
            $result = Invoke-TestSuite -SuiteName $suiteName -SuiteConfig $suiteConfig -LogFile $logFile
            $allResults[$suiteName] = $result
            
            if ($result -and $result.FailedCount -gt 0) {
                $hasFailures = $true
                if (-not $ContinueOnFailure) {
                    Write-TestLog -Message "Test failures detected and ContinueOnFailure is false. Stopping execution." -Level "Warning" -LogFile $logFile
                    break
                }
            }
        }
        else {
            Write-TestLog -Message "Unknown test suite: $suiteName" -Level "Error" -LogFile $logFile
        }
    }
    
    # Generate reports
    Write-Host "`n$('=' * 60)" -ForegroundColor Cyan
    Write-Host "GENERATING REPORTS" -ForegroundColor Cyan
    Write-Host "$('=' * 60)" -ForegroundColor Cyan
    
    $summaryFile = New-TestSummaryReport -AllResults $allResults -LogFile $logFile
    $htmlFile = New-HtmlReport -AllResults $allResults -SummaryFile $summaryFile -LogFile $logFile
    
    # Display final summary
    Write-Host "`n$('=' * 60)" -ForegroundColor Yellow
    Write-Host "TEST EXECUTION COMPLETE" -ForegroundColor Yellow
    Write-Host "$('=' * 60)" -ForegroundColor Yellow
    
    $totalExecutionTime = (Get-Date) - $TestConfig.TestStartTime
    Write-Host "Total Execution Time: $($totalExecutionTime.ToString())" -ForegroundColor White
    Write-Host "Results Location: $($TestConfig.OutputPath)" -ForegroundColor White
    
    if ($summaryFile) {
        Write-Host "Summary Report: $summaryFile" -ForegroundColor Cyan
    }
    
    if ($htmlFile) {
        Write-Host "HTML Report: $htmlFile" -ForegroundColor Cyan
        
        if ($OpenReportAfterExecution) {
            Write-Host "Opening HTML report..." -ForegroundColor Green
            Start-Process $htmlFile
        }
    }
    
    # Final status
    if ($hasFailures) {
        Write-Host "`n❌ TEST EXECUTION COMPLETED WITH FAILURES" -ForegroundColor Red -BackgroundColor Yellow
        Write-Host "Please review the failed tests and fix issues before deployment." -ForegroundColor Red
        Write-TestLog -Message "Test execution completed with failures" -Level "Error" -LogFile $logFile
        exit 1
    }
    else {
        Write-Host "`n✅ ALL TESTS PASSED SUCCESSFULLY" -ForegroundColor Green
        Write-Host "The RepSet Bridge installation script is ready for deployment." -ForegroundColor Green
        Write-TestLog -Message "All tests passed successfully" -Level "Success" -LogFile $logFile
        exit 0
    }
}

# Execute main function
try {
    Main
}
catch {
    Write-Host "`n💥 FATAL ERROR DURING TEST EXECUTION" -ForegroundColor Red -BackgroundColor Yellow
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "Stack Trace: $($_.ScriptStackTrace)" -ForegroundColor DarkRed
    exit 2
}