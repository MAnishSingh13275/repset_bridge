# WARP.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Project Overview

This is a **Gym Door Access Bridge** - a lightweight Go service that connects gym door hardware (fingerprint, RFID, and other biometric devices) to a cloud SaaS platform. The bridge acts as a local agent that discovers hardware devices, processes check-in events, and securely forwards them to the cloud platform with offline queuing capabilities.

## Architecture

### High-Level Design
The bridge follows a modular architecture with these key components:

- **Bridge Manager** (`internal/bridge/manager.go`) - Central coordinator that orchestrates all components
- **Adapter System** (`internal/adapters/`) - Hardware abstraction layer supporting multiple device types
- **Event Processing Pipeline** - Raw events → deduplication → standardization → queuing → cloud submission  
- **Queue Manager** - SQLite-based offline event storage with automatic submission
- **API Server** - REST API for local management and configuration
- **Authentication & Security** - Device pairing, JWT tokens, HMAC signatures

### Component Flow
```
Hardware Devices → Adapters → Event Processor → Queue → Submission Service → Cloud Platform
                                    ↓
                              Bridge Manager (orchestrates all)
                                    ↓
                              API Server (local management)
```

### Key Architectural Patterns
- **Adapter Pattern**: Hardware devices are abstracted through a common `HardwareAdapter` interface
- **Event-Driven**: Components communicate via typed events with callbacks
- **Tiered Performance**: Automatically detects system resources and adjusts performance (lite/normal/full)
- **Graceful Degradation**: Offline queue ensures no event loss during network issues
- **Service-Oriented**: Runs as Windows service or cross-platform daemon

## Common Development Commands

### Building
```powershell
# Build for current platform
make build
go build -o gym-door-bridge.exe ./cmd

# Build for all platforms
make build-all

# Build for specific platform
make build-windows  # or build-darwin, build-linux
```

### Testing
```powershell
# Run all tests
make test
go test ./...

# Run tests with coverage
make test-coverage

# Run integration tests only
make test-integration

# Run single test file/package
go test ./internal/adapters
go test ./internal/bridge -v
```

### Development Workflow
```powershell
# Run in development mode with file watching
make dev
make dev-watch  # requires entr tool

# Format and lint code
make fmt
make lint

# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Local Installation & Service Management
```powershell
# Install as Windows service
gym-door-bridge.exe install

# Service control
net start GymDoorBridge
net stop GymDoorBridge
sc query GymDoorBridge

# Uninstall service  
gym-door-bridge.exe uninstall
```

### Device Management
```powershell
# Pair with cloud platform
gym-door-bridge.exe pair YOUR_PAIR_CODE

# Check pairing status
gym-door-bridge.exe status

# Unpair device
gym-door-bridge.exe unpair

# Test connectivity
gym-door-bridge.exe trigger-heartbeat
gym-door-bridge.exe device-status
```

## Code Architecture Details

### Adapter System Architecture
Located in `internal/adapters/`, this implements a plugin-like architecture:

- **AdapterManager** - Lifecycle management, health monitoring, event routing
- **HardwareAdapter Interface** - Common contract for all device types
- **Adapter Types**:
  - `biometric/` - ZKTeco, ESSL, Realtime devices  
  - `fingerprint/` - Generic fingerprint readers
  - `rfid/` - RFID card readers
  - `simulator/` - Development/testing simulator
  - `webhook/` - HTTP webhook integration

New adapters register via `RegisterAdapter(name, factory)` and are loaded dynamically based on configuration.

### Event Processing Pipeline
Events flow through these stages:

1. **Raw Hardware Event** - Device-specific format from adapters
2. **Event Processor** (`internal/processor/`) - Deduplication, validation, normalization
3. **Standard Event** - Normalized format with device ID, timestamp, user ID, event type
4. **Queue Manager** (`internal/queue/`) - SQLite persistence with tier-based batching
5. **Submission Service** (`internal/client/`) - HTTP client with retry logic and authentication

### Configuration System
- **Main Config** (`internal/config/`) - YAML-based with environment variable overrides
- **Adapter Configs** - Per-device settings for network, authentication, device-specific options
- **Tier-based Settings** - Performance automatically adjusts based on system resources
- **Security Configuration** - Device keys stored in platform credential manager (Windows/macOS)

### Authentication Flow
1. **Device Pairing** - Exchange pair code for device ID + key from cloud platform
2. **Credential Storage** - Device key securely stored using OS credential manager
3. **Request Signing** - All API requests signed with HMAC-SHA256 using device key
4. **JWT Tokens** - Local API server uses JWT for session management

## Development Guidelines

### Adding New Hardware Adapters
1. Implement `HardwareAdapter` interface in `internal/adapters/[type]/`
2. Register adapter factory in `internal/adapters/manager.go`
3. Add configuration schema to `examples/config-template.yaml`
4. Include device auto-discovery logic if applicable
5. Add comprehensive tests including hardware simulation

### Working with Events
- Use `types.RawHardwareEvent` for device-specific events
- Process through `processor.EventProcessor` for standardization
- Handle event callbacks with proper error handling and logging
- Test event deduplication scenarios

### Database Operations
- Use `internal/database/` for SQLite operations with encryption
- Queue operations go through `queue.QueueManager` interface
- Include proper transaction handling and migrations
- Test with different tier configurations (lite/normal/full)

### API Development
- Add endpoints to `internal/api/handlers.go`
- Include authentication middleware for protected endpoints
- Use structured logging with request IDs
- Add comprehensive OpenAPI documentation
- Test with authentication and rate limiting

### Testing Strategy
- **Unit Tests**: Component-level testing with mocks
- **Integration Tests**: Database, queue, and adapter integration
- **End-to-End Tests**: Full service lifecycle with real/simulated hardware
- **Performance Tests**: Queue throughput, memory usage, tier detection

## Configuration Management

### Environment-Specific Configs
- `examples/config-template.yaml` - Full configuration template
- `examples/config-basic.yaml` - Minimal setup
- `examples/config-production.yaml` - Production-ready configuration
- `examples/config-multi-device.yaml` - Multiple hardware devices

### Key Configuration Areas
- **Device Pairing**: `device_id`, `device_key`, `server_url`
- **Performance Tuning**: `tier`, `queue_max_size`, `heartbeat_interval`
- **Adapter Management**: `enabled_adapters`, `adapter_configs`
- **API Server**: Authentication, rate limiting, CORS, security headers
- **Logging**: Structured JSON logging with configurable levels

Configuration is validated at startup with clear error messages for missing or invalid settings.

## Security Considerations

- Device credentials stored in OS secure storage (Windows Credential Manager, macOS Keychain)
- All cloud API requests signed with HMAC-SHA256
- Optional TLS for local API server
- Input validation and sanitization throughout
- Rate limiting and IP allowlisting for API endpoints
- Security headers (HSTS, CSP, XSS protection) configurable per deployment

## Cross-Platform Notes

This codebase is designed for cross-platform deployment with platform-specific optimizations:

- **Windows**: Service integration, credential manager, performance monitoring
- **macOS**: LaunchDaemon support, keychain integration
- **Linux**: SystemD service files, automatic hardware detection

Build system handles CGO dependencies for SQLite and platform-specific credential storage.