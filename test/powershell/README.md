# PowerShell Test Suite

This directory contains PowerShell-based tests for the RepSet Bridge installation and validation processes.

## Test Categories

### Installation Tests
- Unit tests for individual PowerShell functions
- Integration tests for complete installation workflows
- Security tests for signature validation and tampering detection

### Validation Tests
- Cross-platform compatibility validation
- Deployment readiness validation
- Complete workflow integration testing

### Mock Configurations
- Mock system configurations for testing various scenarios
- Test environment setup and teardown utilities

## Running Tests

```powershell
# Navigate to the PowerShell test directory
cd test\powershell

# Run all test suites
.\Run-AllTests.ps1

# Run specific test categories
.\Run-AllTests.ps1 -TestType Unit
.\Run-AllTests.ps1 -TestType Integration
.\Run-AllTests.ps1 -TestType Security
```

For detailed documentation on individual test suites, see the original README content in the installation-tests subdirectory.