# Contributing to Gym Door Access Bridge

Thank you for your interest in contributing to the Gym Door Access Bridge project!

## Development Setup

### Prerequisites

- Go 1.21 or later
- Git
- SQLite3 development libraries
- Build tools (gcc/clang)

### Getting Started

1. **Clone the repository**
   ```bash
   git clone https://github.com/your-org/gym-door-bridge.git
   cd gym-door-bridge
   ```

2. **Install dependencies**
   ```bash
   go mod tidy
   ```

3. **Build the project**
   ```bash
   make build
   # or
   go build -o gym-door-bridge ./cmd
   ```

4. **Run tests**
   ```bash
   make test
   # or
   go test ./...
   ```

## Project Structure

```
gym-door-bridge/
├── cmd/                    # Application entry points
├── internal/              # Internal packages (not importable)
│   ├── adapters/          # Hardware adapter implementations
│   ├── api/               # API client and server
│   ├── auth/              # Authentication and security
│   ├── config/            # Configuration management
│   ├── database/          # Database operations
│   └── ...                # Other internal packages
├── pkg/                   # Public packages (importable)
├── docs/                  # Documentation
├── scripts/               # Build and deployment scripts
├── test/                  # Comprehensive test suite
├── build/                 # Build artifacts (generated)
├── data/                  # Runtime data (generated)
└── logs/                  # Log files (generated)
```

## Development Guidelines

### Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Run `golint` and `go vet`
- Write meaningful commit messages

### Testing

- Write tests for new functionality
- Maintain test coverage above 80%
- Run the full test suite before submitting PRs
- Include integration tests for new adapters

### Documentation

- Update documentation for new features
- Include code comments for complex logic
- Update README.md if needed
- Add examples for new APIs

## Submitting Changes

### Pull Request Process

1. **Fork the repository**
2. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make your changes**
   - Write code following the guidelines
   - Add tests for new functionality
   - Update documentation

4. **Test your changes**
   ```bash
   make test
   make lint
   ```

5. **Commit your changes**
   ```bash
   git commit -m "Add feature: description of changes"
   ```

6. **Push to your fork**
   ```bash
   git push origin feature/your-feature-name
   ```

7. **Create a Pull Request**

### Commit Message Format

```
type(scope): brief description

Longer description if needed

- List specific changes
- Reference issues: Fixes #123
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

## Adding Hardware Adapters

### Adapter Interface

All hardware adapters must implement the `HardwareAdapter` interface:

```go
type HardwareAdapter interface {
    Connect() error
    Disconnect() error
    GetEvents() ([]Event, error)
    GetStatus() AdapterStatus
    GetInfo() AdapterInfo
}
```

### Adapter Development

1. **Create adapter package**
   ```
   internal/adapters/yourdevice/
   ├── adapter.go
   ├── protocol.go
   ├── config.go
   └── adapter_test.go
   ```

2. **Implement the interface**
3. **Add configuration support**
4. **Write comprehensive tests**
5. **Add documentation**

### Testing Adapters

- Use the simulator for basic testing
- Test with real hardware when possible
- Include error handling tests
- Test connection recovery

## Release Process

### Version Numbering

We use semantic versioning (SemVer):
- `MAJOR.MINOR.PATCH`
- `MAJOR`: Breaking changes
- `MINOR`: New features (backward compatible)
- `PATCH`: Bug fixes

### Release Checklist

- [ ] Update version in code
- [ ] Update CHANGELOG.md
- [ ] Run full test suite
- [ ] Build for all platforms
- [ ] Test installation process
- [ ] Create release notes
- [ ] Tag release in Git

## Getting Help

### Resources

- [Documentation](docs/)
- [Troubleshooting Guide](docs/operations/troubleshooting.md)
- [API Documentation](docs/development/)

### Communication

- GitHub Issues for bugs and feature requests
- GitHub Discussions for questions
- Email: dev@repset.onezy.in

### Reporting Issues

When reporting issues, please include:

1. **Environment information**
   - OS and version
   - Go version
   - Hardware details

2. **Steps to reproduce**
   - Exact commands run
   - Configuration used
   - Expected vs actual behavior

3. **Logs and output**
   - Relevant log entries
   - Error messages
   - Debug output if available

4. **Additional context**
   - Screenshots if applicable
   - Related issues or PRs

## Code of Conduct

### Our Standards

- Be respectful and inclusive
- Focus on constructive feedback
- Help others learn and grow
- Maintain professionalism

### Enforcement

Violations of the code of conduct should be reported to dev@repset.onezy.in.

## License

By contributing to this project, you agree that your contributions will be licensed under the same license as the project.

Thank you for contributing to the Gym Door Access Bridge!