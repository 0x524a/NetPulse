# NetPulse

A resilient and efficient internet speed testing client built in Go with support for multiple speed test providers and automatic fallback.

## Features

- **Multi-provider support** with automatic fallback for resilience:
  - Fast.com (Netflix CDN)
  - Cloudflare (speed.cloudflare.com)
  - M-Lab (Measurement Lab)
  - LibreSpeed (open-source speed test)
  - Ookla/Speedtest.net
- Fast and accurate internet **download speed** testing
- **Upload speed** measurement
- **Latency measurement** to test servers
- **Detailed statistics**: min/max/average speeds during the test
- Configurable test parameters via command-line flags
- Provider selection or automatic fallback mode
- Verbose mode for real-time progress tracking
- Save results to file
- Clean, formatted output
- **Custom implementations** - minimal external dependencies

## Project Structure

```
netpulse
├── cmd
│   └── main.go               # Entry point of the application
├── internal
│   ├── speedtest
│   │   └── speedtest.go      # Speed test orchestrator with provider fallback
│   ├── config
│   │   └── config.go         # Configuration settings for the application
│   └── providers             # Internal provider implementations
│       ├── provider
│       │   └── interface.go  # Common interface for all speed test providers
│       ├── fastcom
│       │   └── client.go     # Fast.com (Netflix CDN) provider
│       ├── cloudflare
│       │   └── client.go     # Cloudflare provider
│       ├── mlab
│       │   └── client.go     # M-Lab provider
│       ├── librespeed
│       │   └── client.go     # LibreSpeed provider
│       └── ookla
│           └── client.go     # Ookla/Speedtest.net provider
├── pkg
│   ├── speedtest
│   │   └── client.go         # Public API for the speed test library
│   └── reporter
│       └── reporter.go       # Reporting results of the speed test
├── go.mod                     # Module dependencies
├── go.sum                     # Module dependency checksums
└── README.md                  # Project documentation
```

## Installation

To install the project, clone the repository and navigate to the project directory:

```bash
git clone https://github.com/0x524a/netpulse.git
cd netpulse
```

Then, run the following command to download the necessary dependencies:

```bash
go mod tidy
```

Build the executable:

```bash
go build -o netpulse ./cmd
```

## Usage

### Basic Usage

Run a simple speed test:

```bash
./netpulse
```

Or run directly with Go:

```bash
go run cmd/main.go
```

### Command-Line Options

- `-urls` - Number of URLs to test with (default: 5)
- `-duration` - Duration of the speed test (default: 15s)
- `-verbose` - Enable verbose output (default: false)
- `-save` - Save results to file (default: false)
- `-output` - Output file name (default: speedtest_results.txt)
- `-provider` - Speed test provider: `auto`, `fastcom`, `cloudflare`, `mlab`, `librespeed`, or `ookla` (default: auto)
- `-server` - Server selection mode: `auto` (best), `random`, or `specific` (default: auto)

### Provider Selection

The client supports five reliable speed test providers:

1. **Fast.com** (Netflix CDN) - Default first choice
   - Uses Netflix's global CDN infrastructure
   - Custom implementation with token-based authentication
   
2. **Cloudflare** (speed.cloudflare.com)
   - Cloudflare's global edge network
   - Fast and reliable worldwide coverage
   
3. **M-Lab** (Measurement Lab)
   - Open-source internet measurement platform
   - Operated by research institutions
   
4. **LibreSpeed** - Open-source speed test
   - Community-driven infrastructure
   - Privacy-focused with no tracking
   
5. **Ookla/Speedtest.net** - Industry standard
   - Widely used commercial speed test
   - Extensive server network

**Auto mode** (default) tries each provider in order until one succeeds, providing resilience against service outages.

**Specific provider mode** uses only the selected provider for consistent results.

### Examples

Run a speed test with verbose output (auto fallback mode):
```bash
./netpulse -verbose
```

Use a specific provider:
```bash
./netpulse -provider fastcom -verbose
./netpulse -provider cloudflare -verbose
./netpulse -provider mlab -verbose
./netpulse -provider librespeed -verbose
./netpulse -provider ookla -verbose
```

Use random server selection:
```bash
./netpulse -server random -verbose
```

Run a longer test with more URLs:
```bash
./netpulse -urls 10 -duration 30s
```

Save results to a file:
```bash
./netpulse -save -output results.txt
```

Run a comprehensive test with specific provider:
```bash
./netpulse -provider fastcom -urls 10 -duration 30s -verbose -save
```

## Output Example

```
Internet Speed Test Client
===========================

Starting speed test...
Using provider: auto
---
Trying provider: Fast.com...
Using provider: Fast.com
---
Latency: 9.82 ms
---
Download speed: 682.65 Mbps
---
Testing upload speed...
Upload speed: 67.11 Mbps

Speed Test Results
==================
Provider:       Fast.com

Download Speed: 682.65 Mbps (682652.47 Kbps)
  - Average:    651.65 Mbps
  - Maximum:    682.65 Mbps
  - Minimum:    538.67 Mbps

Upload Speed:   67.11 Mbps (67106.57 Kbps)
  - Average:    60.67 Mbps
  - Maximum:    67.11 Mbps
  - Minimum:    41.93 Mbps

Latency:        9.82 ms
Download Time:  11.08s
Upload Time:    5.392s
Timestamp:      2025-11-24T23:13:36-05:00
```

## Configuration

The application uses command-line flags for configuration. You can customize:
- Number of test URLs
- Test duration
- Output verbosity
- Provider selection (auto fallback or specific provider)
- File saving options

## Architecture

The client uses a clean layered architecture:

### Public API Layer (`pkg/speedtest`)
- **Clean Interface**: Exposes simple functional options pattern for configuration
- **Abstraction**: Hides provider implementation details from users
- **Easy Integration**: Can be imported as a library in other Go projects

### Internal Layer (`internal/`)
- **Provider Interface**: Common contract for all speed test providers
- **Provider Implementations**: Each provider (Fast.com, Cloudflare, M-Lab, LibreSpeed, Ookla)
- **Orchestrator**: Manages provider fallback and test execution
- **Configuration**: Command-line flag management

### Key Features
- **Resilience**: If one provider is down, the client automatically falls back to alternatives
- **Flexibility**: Users can choose specific providers, use automatic selection, or random selection
- **Consistency**: All providers return standardized results
- **Extensibility**: New providers can be easily added by implementing the interface
- **Clean Separation**: Provider implementations are internal, public API is stable

## Reporting

The results of the speed test can be reported to the console or saved to a file using the methods provided in the `reporter.go` file located in the `pkg/reporter` directory.

## Testing

The project includes comprehensive unit tests and benchmarks to ensure reliability and performance.

### Running Tests

Run all tests:
```bash
go test ./...
```

Run tests with verbose output:
```bash
go test -v ./...
```

Run tests for a specific package:
```bash
go test ./pkg/speedtest
go test ./internal/speedtest
```

### Running Benchmarks

Run all benchmarks:
```bash
go test -bench=. ./...
```

Run benchmarks with memory allocation stats:
```bash
go test -bench=. -benchmem ./...
```

### Test Coverage

View test coverage:
```bash
go test -cover ./...
```

Generate detailed coverage report:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Coverage by Package Type

**Public API** (90-93% coverage):
- `pkg/speedtest`: 92.9% - Public client API with functional options
- `pkg/reporter`: 90.9% - Result formatting and file operations
- `internal/config`: 100% - Configuration management

**Provider Implementations** (3-28% coverage):
- `internal/providers/*`: Network-dependent integration code
- These require live network connections and are tested via integration tests
- Unit tests cover initialization and basic functionality

**Overall**: ~14% total coverage (weighted by provider implementation size)

The lower overall coverage reflects that provider implementations (70% of codebase) are primarily network I/O code that requires integration testing with live services. The core business logic and public API have excellent test coverage (90%+).

### Test Summary

The test suite includes:
- **Unit Tests**: For configuration, provider interface, reporter, and public API
- **Benchmarks**: For performance-critical operations like initialization and result formatting
- **Integration Tests**: Testing provider fallback mechanism and result formatting

Current test coverage focuses on:
- ✅ Configuration defaults and initialization
- ✅ Provider interface implementation
- ✅ Mock provider for testing
- ✅ File I/O operations
- ✅ Public API functional options pattern
- ✅ Provider listing and selection
- ✅ Result formatting and display
- ✅ Provider name mapping

All tests pass successfully with comprehensive coverage of the public API surface.

## Dependencies

- [golang.org/x/net](https://pkg.go.dev/golang.org/x/net) - HTML parsing for API interaction

The client includes custom implementations of multiple speed test providers, minimizing external dependencies while maintaining reliability through provider diversity.

## License

This project is licensed under the MIT License. See the LICENSE file for more details.