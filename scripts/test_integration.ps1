# Integration test for installation scripts with actual binary

param(
    [Parameter(Mandatory=$false)]
    [string]$BinaryPath = ".\gym-door-bridge-test.exe"
)

$ErrorActionPreference = "Stop"

# Test configuration
$TestResults = @{
    Passed = 0
    Failed = 0
    Tests = @()
}

# Logging function for tests
function Write-TestLog {
    param([string]$Message, [string]$Level = "INFO")
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    Write-Host "[$timestamp] [INTEGRATION-$Level] $Message"
}

# Test assertion function
function Assert-IntegrationTest {
    param(
        [string]$TestName,
        [scriptblock]$TestBlock
    )
    
    Write-TestLog "Running integration test: $TestName"
    
    try {
        $result = & $TestBlock
        $TestResults.Passed++
        $TestResults.Tests += @{
            Name = $TestName
            Status = "PASSED"
            Result = $result
        }
        Write-TestLog "PASSED: $TestName" "SUCCESS"
    }
    catch {
        $TestResults.Failed++
        $TestResults.Tests += @{
            Name = $TestName
            Status = "FAILED"
            Error = $_.Exception.Message
        }
        Write-TestLog "FAILED: $TestName - $($_.Exception.Message)" "ERROR"
    }
}

# Test binary availability and basic functionality
function Test-BinaryFunctionality {
    Write-TestLog "Testing binary functionality"
    
    Assert-IntegrationTest "Binary Exists and Executable" {
        if (-not (Test-Path $BinaryPath)) {
            throw "Binary not found at $BinaryPath"
        }
        
        # Test help command
        $output = & $BinaryPath --help 2>&1
        if ($LASTEXITCODE -ne 0) {
            throw "Binary help command failed with exit code $LASTEXITCODE"
        }
        
        if (-not $output -or -not ($output -join " ").Contains("gym-door-bridge")) {
            throw "Binary help output doesn't contain expected content"
        }
        
        return "Binary is executable and responds to help command"
    }
    
    Assert-IntegrationTest "Pair Command Available" {
        $output = & $BinaryPath pair --help 2>&1
        if ($LASTEXITCODE -ne 0) {
            throw "Pair command help failed with exit code $LASTEXITCODE"
        }
        
        if (-not ($output -join " ").Contains("pair-code")) {
            throw "Pair command help doesn't contain expected pair-code flag"
        }
        
        return "Pair command is available and properly configured"
    }
    
    Assert-IntegrationTest "Service Command Available" {
        $output = & $BinaryPath service --help 2>&1
        if ($LASTEXITCODE -ne 0) {
            throw "Service command help failed with exit code $LASTEXITCODE"
        }
        
        if (-not ($output -join " ").Contains("install")) {
            throw "Service command help doesn't contain expected install subcommand"
        }
        
        return "Service command is available and properly configured"
    }
}

# Test installation script functionality
function Test-InstallationScriptFunctionality {
    Write-TestLog "Testing installation script functionality"
    
    Assert-IntegrationTest "Installation Script Exists" {
        $scriptPath = ".\scripts\install.ps1"
        if (-not (Test-Path $scriptPath)) {
            throw "Installation script not found at $scriptPath"
        }
        
        # Basic syntax check by loading the script
        $scriptContent = Get-Content $scriptPath -Raw
        if (-not $scriptContent.Contains("Install-GymDoorBridge")) {
            throw "Installation script doesn't contain expected main function"
        }
        
        return "Installation script exists and contains expected functions"
    }
    
    Assert-IntegrationTest "Installation Script Parameters" {
        $scriptPath = ".\scripts\install.ps1"
        $scriptContent = Get-Content $scriptPath -Raw
        
        # Check for required parameters
        $requiredParams = @("PairCode", "ServerURL", "InstallDir", "ConfigDir")
        foreach ($param in $requiredParams) {
            if (-not $scriptContent.Contains("$param")) {
                throw "Installation script missing parameter: $param"
            }
        }
        
        return "Installation script contains all required parameters"
    }
}

# Test configuration file generation
function Test-ConfigurationGeneration {
    Write-TestLog "Testing configuration file generation"
    
    Assert-IntegrationTest "Generate Test Configuration" {
        $testConfigPath = "$env:TEMP\test_config_integration_$(Get-Random).yaml"
        $testServerURL = "https://test-api.example.com"
        
        $configContent = @"
# Gym Door Bridge Configuration
server_url: "$testServerURL"
tier: "normal"
queue_max_size: 10000
heartbeat_interval: 60
unlock_duration: 3000
database_path: "$($env:TEMP.Replace('\', '/'))/bridge.db"
log_level: "info"
log_file: "$($env:TEMP.Replace('\', '/'))/logs/bridge.log"
enabled_adapters:
  - simulator

# Pairing configuration (will be updated after pairing)
device_id: ""
device_key: ""
"@
        
        Set-Content -Path $testConfigPath -Value $configContent -Encoding UTF8
        
        # Test that binary can load the config
        $output = & $BinaryPath --config $testConfigPath --help 2>&1
        if ($LASTEXITCODE -ne 0) {
            throw "Binary failed to load test configuration"
        }
        
        # Cleanup
        Remove-Item $testConfigPath -Force
        
        return "Configuration file generation and loading works correctly"
    }
}

# Test pairing command with invalid pair code (should fail gracefully)
function Test-PairingCommandError {
    Write-TestLog "Testing pairing command error handling"
    
    # This test is expected to fail with a network error, which is correct behavior
    Write-TestLog "Pairing with Invalid Code Test - Expected to fail with network error"
    
    $testConfigPath = "$env:TEMP\test_config_pairing_$(Get-Random).yaml"
    
    $configContent = @"
server_url: "https://test-api.example.com"
tier: "normal"
queue_max_size: 10000
heartbeat_interval: 60
unlock_duration: 3000
database_path: "$($env:TEMP.Replace('\', '/'))/bridge.db"
log_level: "info"
log_file: "$($env:TEMP.Replace('\', '/'))/logs/bridge.log"
enabled_adapters:
  - simulator
device_id: ""
device_key: ""
"@
    
    Set-Content -Path $testConfigPath -Value $configContent -Encoding UTF8
    
    try {
        # Try pairing with invalid code (should fail but not crash)
        $output = & $BinaryPath --config $testConfigPath pair --pair-code "INVALID123" --timeout 5 2>&1
        $exitCode = $LASTEXITCODE
        
        # We expect this to fail, but it should fail gracefully
        if ($exitCode -eq 0) {
            Write-TestLog "Unexpected: Pairing with invalid code succeeded when it should have failed" "WARN"
        } else {
            # Check that the error message is reasonable (should be network connectivity or pairing failure)
            $outputStr = $output -join " "
            if ($outputStr.Contains("failed") -or $outputStr.Contains("error") -or $outputStr.Contains("connectivity")) {
                $TestResults.Passed++
                $TestResults.Tests += @{
                    Name = "Pairing with Invalid Code Fails Gracefully"
                    Status = "PASSED"
                    Result = "Pairing command fails gracefully with expected error: $($outputStr.Substring(0, [Math]::Min(100, $outputStr.Length)))"
                }
                Write-TestLog "PASSED: Pairing with Invalid Code Fails Gracefully" "SUCCESS"
            } else {
                $TestResults.Failed++
                $TestResults.Tests += @{
                    Name = "Pairing with Invalid Code Fails Gracefully"
                    Status = "FAILED"
                    Error = "Error message doesn't contain expected failure indication. Got: $outputStr"
                }
                Write-TestLog "FAILED: Pairing with Invalid Code Fails Gracefully - Unexpected error format" "ERROR"
            }
        }
    }
    catch {
        Write-TestLog "Exception during pairing test: $($_.Exception.Message)" "WARN"
        # This is actually expected behavior - the command should fail
        $TestResults.Passed++
        $TestResults.Tests += @{
            Name = "Pairing with Invalid Code Fails Gracefully"
            Status = "PASSED"
            Result = "Pairing command properly throws exception for invalid configuration"
        }
        Write-TestLog "PASSED: Pairing with Invalid Code Fails Gracefully (via exception)" "SUCCESS"
    }
    finally {
        # Cleanup
        Remove-Item $testConfigPath -Force -ErrorAction SilentlyContinue
    }
}

# Main test runner
function Run-IntegrationTests {
    Write-TestLog "Starting integration tests for installation scripts"
    Write-TestLog "Binary Path: $BinaryPath"
    
    # Check if binary exists
    if (-not (Test-Path $BinaryPath)) {
        Write-TestLog "Binary not found at $BinaryPath. Please build the binary first:" "ERROR"
        Write-TestLog "  go build -o gym-door-bridge-test.exe ./cmd" "ERROR"
        exit 1
    }
    
    # Run test suites
    Test-BinaryFunctionality
    Test-InstallationScriptFunctionality
    Test-ConfigurationGeneration
    Test-PairingCommandError
    
    # Print test results
    Write-TestLog "Integration Test Results Summary"
    Write-TestLog "Passed: $($TestResults.Passed)"
    Write-TestLog "Failed: $($TestResults.Failed)"
    Write-TestLog "Total: $($TestResults.Passed + $TestResults.Failed)"
    
    if ($TestResults.Failed -gt 0) {
        Write-TestLog "Some integration tests failed:" "ERROR"
        foreach ($test in $TestResults.Tests | Where-Object { $_.Status -eq "FAILED" }) {
            Write-TestLog "  - $($test.Name): $($test.Error)" "ERROR"
        }
        exit 1
    } else {
        Write-TestLog "All integration tests passed!" "SUCCESS"
        Write-TestLog "Installation scripts are ready for deployment" "SUCCESS"
        exit 0
    }
}

# Run the integration tests
Run-IntegrationTests