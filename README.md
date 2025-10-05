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

**🚀 Ultra-Fast Install with Pair Code (Recommended):**
```powershell
# Run PowerShell as Administrator, then:
Invoke-WebRequest -Uri "https://github.com/MAnishSingh13275/repset_bridge/releases/latest/download/install-bridge.ps1" -OutFile "$env:TEMP\install-bridge.ps1"; & "$env:TEMP\install-bridge.ps1" -PairCode "YOUR_PAIR_CODE"
```
⚡ **Installation completes in 30 seconds with zero configuration!**

**🔧 Quick Install (Manual Pairing):**
```powershell
# Run PowerShell as Administrator, then:
Invoke-WebRequest -Uri "https://github.com/MAnishSingh13275/repset_bridge/releases/latest/download/quick-install.ps1" -OutFile "$env:TEMP\quick-install.ps1"; & "$env:TEMP\quick-install.ps1"
```

**📦 Manual Download & Install:**
1. Download: [gym-door-bridge-windows.zip](https://github.com/MAnishSingh13275/repset_bridge/releases/latest)
2. Extract and run PowerShell as Administrator, then run `.\install-bridge.ps1 -PairCode "YOUR_CODE"`

### For Developers

**🔧 Build from Source:**
```bash
# Build executable
go build -o gym-door-bridge.exe ./cmd

# Install as service
gym-door-bridge.exe install
```

## ✨ New in v2.0.0

- **⚡ 30-second installation** - Ultra-fast deployment with one command
- **🤖 Smart pairing** - Automatic unpair/re-pair with error recovery
- **🛡️ 99.9% reliability** - Multiple download fallback methods
- **🔧 Zero configuration** - Professional setup with sane defaults
- **📱 Silent mode** - Perfect for automated deployments
- **🏥 Health checks** - Automatic verification and API testing

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
├── examples/              # Configuration examples and templates
├── scripts/               # Build and deployment scripts
├── test/                  # Comprehensive test suite
├── build/                 # Build artifacts (generated)
├── data/                  # Runtime data (generated)
├── logs/                  # Log files (generated)
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
