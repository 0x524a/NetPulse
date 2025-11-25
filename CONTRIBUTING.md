# Contributing to NetPulse

Thank you for your interest in contributing to NetPulse! We welcome contributions from the community.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/0x524a/netpulse.git`
3. Create a branch: `git checkout -b feature/your-feature-name`
4. Make your changes
5. Run tests: `go test ./...`
6. Commit with a clear message: `git commit -am 'Add your feature description'`
7. Push: `git push origin feature/your-feature-name`
8. Create a Pull Request

## Development Setup

```bash
# Clone the repository
git clone https://github.com/YOUR_USERNAME/netpulse.git
cd netpulse

# Download dependencies
go mod tidy

# Build
go build -o netpulse ./cmd/main.go

# Run tests
go test ./...
```

## Code Style

- Follow Go conventions (gofmt, go vet)
- Run `go fmt ./...` before committing
- Write tests for new features
- Update documentation as needed
- Keep commits small and focused

## Testing

Run tests before submitting a PR:
```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./pkg/speedtest
go test ./internal/speedtest
```

## Pull Request Process

1. Update the README.md or relevant documentation with details of changes
2. Ensure all tests pass: `go test ./...`
3. Run `go vet ./...` to check for issues
4. Add tests for new functionality
5. Use clear, descriptive commit messages
6. Reference any related issues in your PR description

## Reporting Issues

Please use GitHub Issues to report bugs or suggest features:

### For Bugs
- Clear title describing the issue
- Steps to reproduce
- Expected vs actual behavior
- Go version: `go version`
- Operating system
- NetPulse version

### For Features
- Clear description of the feature
- Use case and why it would be useful
- Examples if applicable

## Code Organization

```
netpulse/
├── cmd/              # Command-line application
├── internal/         # Internal packages
│   ├── speedtest/    # Speed test orchestrator
│   ├── config/       # Configuration management
│   └── providers/    # Provider implementations
├── pkg/              # Public API packages
│   ├── speedtest/    # Public speed test client
│   ├── benchmarks/   # Benchmarking suite
│   └── reporter/     # Result reporting
└── docs/             # Documentation
```

## Commit Message Guidelines

- Use the present tense: "Add feature" not "Added feature"
- Use the imperative mood: "Move cursor to..." not "Moves cursor to..."
- Limit the first line to 72 characters or less
- Reference issues and pull requests liberally after the first line
- Example: `Add support for custom test duration

Fixes #42. Allows users to specify custom duration via -duration flag.`

## Questions?

Feel free to:
- Open a GitHub Issue
- Start a GitHub Discussion
- Contact maintainers

Thank you for contributing! 🎉
