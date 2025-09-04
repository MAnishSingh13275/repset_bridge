# Changelog

All notable changes to the Gym Door Access Bridge project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Comprehensive documentation structure in `docs/` directory
- Contributing guidelines and development setup
- Proper project structure with organized directories
- Build and deployment automation scripts

### Changed
- Reorganized all documentation into logical categories
- Updated project structure for better maintainability
- Improved .gitignore for better file organization

### Fixed
- Documentation consistency and navigation
- Project file organization and cleanup

## [1.0.0] - 2024-01-01

### Added
- Initial release of Gym Door Access Bridge
- Support for ZKTeco, ESSL, and Realtime biometric devices
- Automatic device discovery and configuration
- Windows service installation and management
- Offline queue functionality for network outages
- HMAC authentication with the SaaS platform
- SQLite database for local data storage
- Health monitoring and status reporting
- Automatic updates and deployment system
- Cross-platform build support (Windows, macOS, Linux)
- Docker containerization support
- Comprehensive test suite
- Installation scripts for easy deployment

### Hardware Support
- ZKTeco fingerprint devices (TCP/IP)
- ESSL biometric devices (TCP/IP, HTTP)
- Realtime access control devices (TCP/IP)
- Simulator for testing without hardware

### Features
- **Auto-Discovery**: Automatically finds and configures biometric devices
- **Offline Mode**: Queues events locally during network outages
- **Security**: HMAC authentication with key rotation
- **Performance Tiers**: Lite, Normal, and Full performance modes
- **Health Monitoring**: Real-time status and health checks
- **Easy Installation**: One-click installation for gym owners
- **Service Management**: Windows service with auto-start
- **Update System**: Automatic updates with staged rollout
- **Cross-Platform**: Support for Windows, macOS, and Linux

### Documentation
- Complete installation guides
- Troubleshooting documentation
- Development and integration guides
- Testing documentation
- Deployment guides

---

## Release Notes Format

### Types of Changes
- **Added** for new features
- **Changed** for changes in existing functionality
- **Deprecated** for soon-to-be removed features
- **Removed** for now removed features
- **Fixed** for any bug fixes
- **Security** for vulnerability fixes

### Version History
- **Major versions** (x.0.0) - Breaking changes
- **Minor versions** (x.y.0) - New features, backward compatible
- **Patch versions** (x.y.z) - Bug fixes, backward compatible