# macOS Service Management

This package provides macOS daemon management functionality for the Gym Door Bridge application. It handles installation, configuration, and lifecycle management of the bridge as a macOS daemon using launchd.

## Features

- **Daemon Management**: Install, start, stop, restart, and uninstall the bridge as a macOS daemon
- **Configuration Management**: Handle daemon configuration and directory setup
- **Code Signing & Notarization**: Support for macOS code signing and notarization workflows
- **Installation Scripts**: Generate installation and uninstallation scripts
- **Integration Tests**: Comprehensive testing for daemon operations

## Components

### Service (`service.go`)
- `Service`: Main service wrapper that handles daemon execution
- `RunService()`: Runs the application as a macOS daemon with proper signal handling
- `IsMacOSDaemon()`: Detects if running under launchd

### Manager (`manager.go`)
- `ServiceManager`: Handles daemon lifecycle operations using launchctl
- Install/uninstall daemon with plist generation
- Start/stop/restart daemon operations
- Status checking and service validation

### Configuration (`config.go`)
- `ServiceConfig`: Configuration structure for daemon settings
- Default configuration for macOS directory structure
- Directory creation and permission management
- Default configuration file generation

### Commands (`commands.go`)
- CLI commands for daemon management (`daemon install`, `daemon start`, etc.)
- Command-line argument parsing and validation
- Root privilege checking and error handling

### Notarization (`notarization.go`)
- `NotarizationManager`: Handles macOS code signing and notarization
- Binary signing with Developer ID certificates
- Notarization submission and status checking
- Keychain profile management for automation

### Scripts (`scripts.go`)
- `ScriptGenerator`: Generates installation and uninstallation scripts
- Customizable script templates with parameter substitution
- Bash scripts with comprehensive error handling and logging

## Usage

### Basic Daemon Management

```go
import "gym-door-bridge/internal/service/macos"

// Create service manager
sm, err := macos.NewServiceManager()
if err != nil {
    log.Fatal(err)
}

// Install daemon
err = sm.InstallService("/path/to/binary", "/path/to/config.yaml")
if err != nil {
    log.Fatal(err)
}

// Start daemon
err = sm.StartService()
if err != nil {
    log.Fatal(err)
}
```

### Running as Daemon

```go
import (
    "context"
    "gym-door-bridge/internal/config"
    "gym-door-bridge/internal/service/macos"
)

func main() {
    cfg, err := config.Load("config.yaml")
    if err != nil {
        log.Fatal(err)
    }
    
    // Define bridge main function
    bridgeFunc := func(ctx context.Context, cfg *config.Config) error {
        // Your bridge implementation here
        return nil
    }
    
    // Run as daemon
    err = macos.RunService(cfg, bridgeFunc)
    if err != nil {
        log.Fatal(err)
    }
}
```

### Code Signing and Notarization

```go
import "gym-door-bridge/internal/service/macos"

config := &macos.NotarizationConfig{
    DeveloperID:         "Developer ID Application: Your Name",
    TeamID:             "TEAM123456",
    BundleID:           "com.yourcompany.gymdoorbridge",
    AppleID:            "your-apple-id@example.com",
    AppSpecificPassword: "your-app-specific-password",
}

nm := macos.NewNotarizationManager(config)

// Sign and notarize binary
err := nm.NotarizeBinary("/path/to/binary")
if err != nil {
    log.Fatal(err)
}
```

### Generating Installation Scripts

```go
import "gym-door-bridge/internal/service/macos"

sg := macos.NewScriptGenerator()

// Generate both install and uninstall scripts
err := sg.GenerateScripts("/output/directory")
if err != nil {
    log.Fatal(err)
}
```

## CLI Commands

When integrated with the main application, the following commands are available:

```bash
# Install daemon
sudo gym-door-bridge daemon install --pair-code ABC123

# Start daemon
sudo gym-door-bridge daemon start

# Check status
gym-door-bridge daemon status

# Stop daemon
sudo gym-door-bridge daemon stop

# Restart daemon
sudo gym-door-bridge daemon restart

# Uninstall daemon
sudo gym-door-bridge daemon uninstall
```

## Directory Structure

The macOS daemon uses the following directory structure:

```
/usr/local/bin/gym-door-bridge              # Binary location
/usr/local/etc/gymdoorbridge/config.yaml    # Configuration file
/usr/local/var/lib/gymdoorbridge/           # Data directory
/usr/local/var/log/gymdoorbridge/           # Log directory
/Library/LaunchDaemons/com.gymdoorbridge.agent.plist  # Daemon plist
```

## Installation Scripts

The generated installation script supports the following options:

```bash
# Basic installation
sudo ./install.sh

# Installation with pair code
sudo ./install.sh --pair-code ABC123

# Installation with specific version
sudo ./install.sh --version 1.2.3 --pair-code ABC123

# Custom download URL
sudo ./install.sh --url https://custom.example.com/releases --pair-code ABC123
```

The uninstallation script supports:

```bash
# Basic uninstallation (preserves data)
sudo ./uninstall.sh

# Complete removal including data
sudo ./uninstall.sh --remove-data
```

## Requirements

- macOS 10.12 or later
- Root privileges for daemon installation and management
- Xcode Command Line Tools (for code signing and notarization)
- Valid Apple Developer ID (for code signing)
- Apple ID with app-specific password (for notarization)

## Testing

The package includes comprehensive unit and integration tests:

```bash
# Run unit tests
go test ./internal/service/macos

# Run integration tests (requires root)
sudo go test ./internal/service/macos -tags=integration

# Run with coverage
go test -cover ./internal/service/macos
```

## Security Considerations

- Daemon runs as root for system-level access
- Configuration files have appropriate permissions (644)
- Data directories have restricted permissions (755)
- Code signing ensures binary integrity
- Notarization provides additional security validation

## Troubleshooting

### Common Issues

1. **Permission Denied**: Ensure running with `sudo` for daemon operations
2. **Daemon Won't Start**: Check logs at `/usr/local/var/log/gymdoorbridge/bridge.log`
3. **Code Signing Fails**: Verify Developer ID certificate is installed
4. **Notarization Fails**: Check Apple ID credentials and app-specific password

### Log Locations

- Daemon logs: `/usr/local/var/log/gymdoorbridge/bridge.log`
- System logs: `sudo log show --predicate 'subsystem == "com.gymdoorbridge.agent"'`
- Installation logs: Console output during script execution

### Debugging

```bash
# Check daemon status
launchctl list com.gymdoorbridge.agent

# View daemon configuration
cat /Library/LaunchDaemons/com.gymdoorbridge.agent.plist

# Check file permissions
ls -la /usr/local/var/lib/gymdoorbridge/

# Test binary directly
/usr/local/bin/gym-door-bridge --help
```

## Integration with Main Application

This package integrates with the main application through:

1. **CLI Commands**: Added to root command in `cmd/main.go`
2. **Service Detection**: Used to determine if running as daemon
3. **Configuration**: Loads daemon-specific configuration
4. **Logging**: Integrates with application logging system

See the main application documentation for complete integration details.