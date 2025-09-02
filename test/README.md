# Comprehensive Test Suite

This directory contains comprehensive tests for the Gym Door Bridge system, covering integration, load, security, and end-to-end testing scenarios.

## Test Categories

### Integration Tests (`integration/`)
- Complete hardware-to-cloud flow testing
- Multi-component interaction validation
- Real database and network simulation

### Load Tests (`load/`)
- Performance tier validation (Lite/Normal/Full)
- Queue capacity and throughput testing
- Resource constraint simulation

### Security Tests (`security/`)
- HMAC authentication flow validation
- Key rotation and device pairing security
- Attack scenario simulation

### End-to-End Tests (`e2e/`)
- Real deployment scenario simulation
- Installation and configuration testing
- Failure recovery and resilience testing

## Running Tests

```bash
# Run all comprehensive tests
go test ./test/... -v

# Run specific test categories
go test ./test/integration/... -v
go test ./test/load/... -v
go test ./test/security/... -v
go test ./test/e2e/... -v

# Run with race detection
go test ./test/... -race -v

# Run with coverage
go test ./test/... -cover -v
```

## Test Requirements Coverage

These tests validate all requirements from the specification:
- Requirement 1: Hardware adapter integration
- Requirement 2: Cross-platform service operation
- Requirement 3: Secure authentication flows
- Requirement 4: Offline queue functionality
- Requirement 5: Health monitoring and status
- Requirement 6: Installation and setup
- Requirement 7: User mapping functionality
- Requirement 8: Monitoring and alerting
- Requirement 9: Door unlock operations
- Requirement 10: Update and deployment