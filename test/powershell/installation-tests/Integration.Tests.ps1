# ================================================================
# RepSet Bridge Installation - Integration Tests
# End-to-end testing of complete installation workflows
# ================================================================

Import-Module Pester -Force

# Test configuration
$IntegrationTestConfig = @{
    TestEnvironmentPath = "$env:TEMP\RepSetBridge-Integration-Tests"
    MockPlatformEndpoint = "https://test-platform.repset.com"
    MockGitHubEndpoint = "https://api.github.com"
    TestTimeout = 300  # 5 minutes
}

# ================================================================
# Integration Test Helper Functions
# ================================================================

function New-IntegrationTestEnvironment {
    <#
    .SYNOPSIS
    Creates a complete test environment for integration testing
    #>
    param(
        [string]$TestName
    )
    
    $testDir = Join-Path $IntegrationTestConfig.TestEnvironmentPath $TestName
    
    # Create directory structure
    New-Item -ItemType Directory -Path $testDir -Force | Out-Null
    New-Item -ItemType Directory -Path "$testDir\logs" -Force | Out-Null
    New-Item -ItemType Directory -Path "$testDir\config" -Force | Out-Null
    New-Item -ItemType Directory -Path "$testDir\downloads" -Force | Out-Null
    
    # Create mock executable
    $mockExecutable = Join-Path $testDir "repset-bridge.exe"
    Set-Content -Path $mockExecutable -Value "Mock RepSet Bridge Executable"
    
    # Create mock configuration
    $mockConfig = @"
device_id: "integration-test-device"
device_key: "integration-test-key"
server_url: "$($IntegrationTestConfig.MockPlatformEndpoint)"
tier: "normal"
service:
  auto_start: true
  restart_on_failure: true
"@
    Set-Content -Path "$testDir\config\config.yaml" -Value $mockConfig
    
    return $testDir
}

function Remove-IntegrationTestEnvironment {
    <#
    .SYNOPSIS
    Cleans up integration test environment
    #>
    param(
        [string]$TestDirectory
    )
    
    if (Test-Path $TestDirectory) {
        try {
            # Stop any test services
            $testServices = Get-Service -Name "*RepSetBridge*Test*" -ErrorAction SilentlyContinue
            foreach ($service in $testServices) {
                Stop-Service -Name $service.Name -Force -ErrorAction SilentlyContinue
                & sc.exe delete $service.Name 2>$null
            }
            
            # Remove test directory
            Remove-Item -Path $TestDirectory -Recurse -Force -ErrorAction SilentlyContinue
        }
        catch {
            Write-Warning "Could not fully clean up integration test environment: $($_.Exception.Message)"
        }
    }
}

# Additional integration test functions and test cases...
# Note: This is a truncated version for the migration. The full file contains comprehensive integration testing.