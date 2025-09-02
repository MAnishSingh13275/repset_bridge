# Gym Door Access Bridge

A lightweight local agent that connects gym door access hardware (fingerprint, RFID, or other devices) with our SaaS platform.

## Prerequisites

- Go 1.21 or later
- Windows or macOS operating system

## Setup

1. **Install Go dependencies:**

   ```bash
   go mod tidy
   ```

2. **Create configuration file:**

   ```bash
   cp config.yaml.example config.yaml
   ```

3. **Build the application:**

   ```bash
   go build -o gym-door-bridge ./cmd
   ```

4. **Run the application:**
   ```bash
   ./gym-door-bridge --help
   ```

## Configuration

The bridge can be configured through:

- Configuration file (`config.yaml`)
- Environment variables (prefixed with `BRIDGE_`)
- Command line flags

See `config.yaml.example` for all available configuration options.

## Usage

### Basic usage:

```bash
./gym-door-bridge
```

### With custom config file:

```bash
./gym-door-bridge --config /path/to/config.yaml
```

### With debug logging:

```bash
./gym-door-bridge --log-level debug
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
├── config.yaml.example  # Example configuration file
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
