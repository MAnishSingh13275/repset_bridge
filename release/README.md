# Gym Door Access Bridge

A lightweight local agent that connects gym door access hardware (fingerprint, RFID, or other devices) with our SaaS platform.

## 📚 Documentation

Complete documentation is available in the [`docs/`](docs/) directory:

- **[Installation Guide](docs/installation/README.md)** - Complete installation instructions
- **[Gym Owner Guide](docs/installation/gym-owner-guide.md)** - Simple guide for non-technical users
- **[Download Instructions](docs/installation/download.md)** - Download and install options
- **[Troubleshooting](docs/operations/troubleshooting.md)** - Common issues and solutions
- **[Deployment Guide](docs/operations/deployment.md)** - Production deployment guide

## Quick Installation (Windows)

### For Gym Owners (Non-Technical)

**🚀 One-Click Web Install:**
```powershell
# Run PowerShell as Administrator, then:
iex (iwr -useb https://raw.githubusercontent.com/your-org/gym-door-bridge/main/web-install.ps1).Content
```

**📦 Download & Install:**
1. Download: [gym-door-bridge-windows.zip](https://github.com/your-org/gym-door-bridge/releases/latest)
2. Extract and run `install.bat` as Administrator

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
├── internal/              # Internal packages (not importable)
├── pkg/                   # Public packages (importable by external projects)
├── docs/                  # Complete documentation
│   ├── installation/     # Installation guides
│   ├── development/      # Development documentation
│   ├── operations/       # Deployment and troubleshooting
│   └── testing/          # Testing documentation
├── examples/              # Configuration and usage examples
├── scripts/               # Build and deployment scripts
├── test/                  # Comprehensive test suite
├── build/                 # Build artifacts (generated)
├── data/                  # Runtime data (generated)
├── logs/                  # Log files (generated)
├── config.yaml.example   # Example configuration file
├── CONTRIBUTING.md        # Development guidelines
├── CHANGELOG.md           # Version history
├── LICENSE                # MIT License
├── go.mod                 # Go module definition
└── README.md              # This file
```

## Development

This project follows Go best practices:

- `cmd/` contains application entry points
- `internal/` contains private packages
- `pkg/` contains public packages
- `docs/` contains all documentation
- Structured logging with JSON output
- Configuration via files, environment variables, and CLI flags

## Support

For help and support:
- Check the [Troubleshooting Guide](docs/operations/troubleshooting.md)
- Review the [Installation Guide](docs/installation/README.md)
- Contact support with log files and error messages
