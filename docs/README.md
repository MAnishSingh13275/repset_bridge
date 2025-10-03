# Gym Door Access Bridge Documentation

A lightweight local agent that connects gym door access hardware (fingerprint, RFID, or other devices) with our SaaS platform.

## Quick Start

- **[Installation Guide](installation/README.md)** - Complete installation instructions
- **[Gym Owner Guide](installation/gym-owner-guide.md)** - Simple guide for non-technical users
- **[Download Instructions](installation/download.md)** - Download and install options

## Documentation Structure

### 📦 Installation & Setup
- [Installation Guide](installation/README.md) - Complete installation instructions
- [Gym Owner Guide](installation/gym-owner-guide.md) - Simple guide for non-technical users
- [Download Instructions](installation/download.md) - Download and install options

### 🔧 Development & Integration
- [Platform Integration](PLATFORM_INTEGRATION.md) - Repset SaaS platform integration
- [Fingerprint Integration](development/fingerprint-integration.md) - Biometric device integration
- [Testing Guide](development/testing.md) - Complete testing documentation
- [Build Scripts](development/build-scripts.md) - Build and deployment scripts

### 🚀 Operations & Deployment
- [Deployment Guide](operations/deployment.md) - Production deployment guide
- [Troubleshooting](operations/troubleshooting.md) - Common issues and solutions

### 🧪 Testing & Quality
- [Test Suite Overview](testing/README.md) - Test suite documentation
- [Complete Flow Testing](testing/complete-flow-testing.md) - End-to-end testing guide
- [Testing Documentation](testing/testing.md) - Comprehensive test documentation

## Quick Installation (Windows)

### For Gym Owners (Non-Technical)

**🚀 One-Click Web Install:**
```powershell
# Run PowerShell as Administrator, then:
iex (iwr -useb https://raw.githubusercontent.com/your-org/gym-door-bridge/main/public/install-bridge.ps1).Content
```

**📦 Download & Install:**
1. Download: [gym-door-bridge-windows.zip](https://github.com/your-org/gym-door-bridge/releases/latest)
2. Extract and run PowerShell as Administrator, then run `scripts\install.ps1`

### For Developers

**🔧 Build from Source:**
```bash
# Build executable
go build -o gym-door-bridge.exe ./cmd

# Install as service
gym-door-bridge.exe install
```

## Supported Devices (Auto-Discovered)

| Device Type | Ports | Auto-Config |
|-------------|-------|-------------|
| **ZKTeco** | 4370 | ✅ |
| **ESSL** | 80, 8080 | ✅ |
| **Realtime** | 5005, 9999 | ✅ |
| **Simulator** | - | ✅ (for testing) |

## Service Management

```cmd
# Check service status
sc query GymDoorBridge

# Start/Stop service
net start GymDoorBridge
net stop GymDoorBridge

# Uninstall service
gym-door-bridge.exe uninstall
```

## Project Structure

```
gym-door-bridge/
├── cmd/                    # Application entry points
│   └── main.go            # Main CLI application
├── internal/              # Internal packages (not importable)
│   ├── config/           # Configuration management
│   └── logging/          # Structured logging setup
├── pkg/                  # Public packages (importable by external projects)
├── docs/                 # Documentation (this directory)
├── examples/             # Configuration examples and templates
├── go.mod               # Go module definition
└── README.md           # This file
```

## Development

This project follows Go best practices:

- `cmd/` contains application entry points
- `internal/` contains private packages
- `pkg/` contains public packages
- Structured logging with JSON output
- Configuration via files, environment variables, and CLI flags

## Support

For help and support:
- Check the [Troubleshooting Guide](operations/troubleshooting.md)
- Review the [Installation Guide](installation/README.md)
- Contact support with log files and error messages