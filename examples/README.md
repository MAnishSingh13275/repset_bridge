# Examples

This directory contains example configurations, scripts, and usage patterns for the Gym Door Access Bridge.

## Configuration Examples

### Basic Configuration
- `config-template.yaml` - Complete configuration template with all options
- `config-basic.yaml` - Minimal configuration for single device
- `config-multi-device.yaml` - Configuration for multiple devices
- `config-production.yaml` - Production-ready configuration

### Device-Specific Examples
- `config-zkteco.yaml` - ZKTeco device configuration
- `config-essl.yaml` - ESSL device configuration
- `config-realtime.yaml` - Realtime device configuration

### Deployment Examples
- `docker-compose.example.yml` - Docker deployment
- `systemd.service` - Linux systemd service file
- `launchd.plist` - macOS launchd configuration

## Usage Examples

### API Integration
- `api-client.go` - Example API client implementation
- `webhook-handler.go` - Example webhook handler
- `event-processor.go` - Example event processing

### Testing Scripts
- `test-connection.ps1` - Test device connectivity
- `test-api.sh` - Test API endpoints
- `simulate-events.py` - Generate test events

## Development Examples

### Custom Adapters
- `custom-adapter/` - Example custom hardware adapter
- `protocol-example/` - Example device protocol implementation

### Integration Examples
- `gym-software-integration/` - Example gym software integration
- `monitoring-setup/` - Example monitoring configuration

## Getting Started

1. Copy the appropriate example configuration
2. Modify for your specific setup
3. Test the configuration
4. Deploy to production

For detailed documentation, see [docs/](../docs/)