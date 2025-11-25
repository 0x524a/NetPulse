# NetPulse

A resilient and efficient internet speed testing **library and CLI tool** built in Go with support for multiple speed test providers and automatic fallback.

> **Use NetPulse as a library in your Go projects or as a standalone CLI tool for network diagnostics and speed testing.**

## Features

### Core Capabilities
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
- **Provider benchmarking**: Compare performance across all providers

### As a Library
- Clean, minimal API with functional options pattern
- Import into any Go project: `github.com/0x524a/netpulse/pkg/speedtest`
- Customizable for your specific use case
- Full control over test parameters and provider selection

### As a CLI
- Standalone executable with intuitive command-line interface
- Provider selection or automatic fallback mode
- Verbose mode for real-time progress tracking
- Save results to file in structured format
- Benchmarking suite to compare all providers
- Clean, formatted output

### Additional Features
- **Custom implementations** - minimal external dependencies
- Provider agnostic - easily extensible
- Comprehensive test suite with high coverage
- CI/CD ready with GitHub Actions integration

## Project Structure

```
netpulse/
├── cmd/                      # CLI Application
│   └── main.go              # CLI entry point (executable only)
│
├── pkg/                      # Public Library API
│   ├── speedtest/           # Main speedtest library
│   │   └── client.go        # Public speedtest client with functional options
│   ├── reporter/            # Result formatting and reporting
│   │   └── reporter.go      # Console and file output
│   ├── benchmarks/          # Benchmarking suite
│   │   └── suite.go         # Provider comparison and scoring
│   ├── metrics/             # Metrics and classification
│   │   └── classification.go # Speed classification utilities
│   └── ookla/               # Ookla integration
│
├── internal/                 # Internal Implementation (not part of public API)
│   ├── speedtest/           # Speed test orchestrator
│   │   └── speedtest.go     # Test execution and provider fallback
│   ├── config/              # Configuration management
│   │   └── config.go        # Internal config structure
│   └── providers/           # Provider implementations
│       ├── provider/        # Provider interface
│       │   └── interface.go
│       ├── fastcom/         # Fast.com implementation
│       ├── cloudflare/      # Cloudflare implementation
│       ├── mlab/            # M-Lab implementation
│       ├── librespeed/      # LibreSpeed implementation
│       └── ookla/           # Ookla/Speedtest.net implementation
│
├── go.mod                    # Module definition
├── go.sum                    # Dependency checksums
└── README.md                 # Documentation
```

### Architecture Overview

NetPulse is designed with clear separation between the **public library API** and **internal implementations**:

- **`pkg/` (Public API)**: Stable, versioned library code intended for external use
  - Clean interfaces with functional options pattern
  - Minimal dependencies
  - Backward compatibility guarantees

- **`internal/` (Implementation)**: Provider-specific and orchestration logic
  - Network I/O and API integration details
  - May change between versions
  - Not intended for external use

- **`cmd/` (CLI)**: Command-line application
  - Uses the public library API
  - Demonstrates library usage
  - Independent of internal changes

## Installation

### As a Library

Add NetPulse to your Go project:

```bash
go get github.com/0x524a/netpulse@latest
```

Then import and use in your code:

```go
import "github.com/0x524a/netpulse/pkg/speedtest"

client := speedtest.New(
    speedtest.WithProvider("auto"),
    speedtest.WithVerbose(true),
)

result, err := client.Run()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Download: %.2f Mbps\n", result.DownloadMbps)
fmt.Printf("Upload: %.2f Mbps\n", result.UploadMbps)
```

### As a CLI Tool

Clone the repository and build the executable:

```bash
git clone https://github.com/0x524a/netpulse.git
cd netpulse
go mod tidy
go build -o netpulse ./cmd
```

Or use `go install` to install directly:

```bash
go install github.com/0x524a/netpulse/cmd@latest
```

## Usage

### CLI Usage

Run a simple speed test:

```bash
./netpulse
```

Or run directly with Go:

```bash
go run cmd/main.go
```

#### Command-Line Options

- `-provider` - Speed test provider: `auto`, `fastcom`, `cloudflare`, `mlab`, `librespeed`, or `ookla` (default: auto)
- `-server` - Server selection mode: `auto`, `random`, or `specific` (default: auto)
- `-duration` - Duration of the speed test (default: 15s)
- `-urls` - Number of URLs to test with (default: 5)
- `-verbose` - Enable verbose output (default: false)
- `-save` - Save results to file (default: false)
- `-output` - Output file name (default: speedtest_results.txt)

#### CLI Examples

Run a speed test with verbose output:
```bash
./netpulse -verbose
```

Use a specific provider:
```bash
./netpulse -provider fastcom -verbose
./netpulse -provider cloudflare
./netpulse -provider mlab
./netpulse -provider librespeed
./netpulse -provider ookla
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

Run a comprehensive test:
```bash
./netpulse -provider fastcom -urls 10 -duration 30s -verbose -save
```

Benchmark and compare all providers:
```bash
./netpulse benchmark -t 3 -d 15s --verbose
```

### Library Usage

#### Basic Example

```go
package main

import (
	"fmt"
	"log"
	"github.com/0x524a/netpulse/pkg/speedtest"
)

func main() {
	// Create a client with auto provider selection
	client := speedtest.New(
		speedtest.WithProvider("auto"),
		speedtest.WithVerbose(true),
		speedtest.WithDuration(15*time.Second),
	)

	// Run the speed test
	result, err := client.Run()
	if err != nil {
		log.Fatal(err)
	}

	// Use the results
	fmt.Printf("Provider: %s\n", result.ProviderName)
	fmt.Printf("Download Speed: %.2f Mbps\n", result.DownloadMbps)
	fmt.Printf("Upload Speed: %.2f Mbps\n", result.UploadMbps)
	fmt.Printf("Latency: %v\n", result.Latency)
}
```

#### Using Functional Options

```go
import "github.com/0x524a/netpulse/pkg/speedtest"

// Configure with functional options
client := speedtest.New(
	speedtest.WithProvider("fastcom"),        // Use Fast.com
	speedtest.WithDuration(20*time.Second),   // 20s test duration
	speedtest.WithURLCount(10),               // Use 10 URLs
	speedtest.WithServerMode("random"),       // Random server selection
	speedtest.WithVerbose(true),              // Enable progress output
	speedtest.WithOutputFile("results.txt"),  // Save results
)

result, err := client.Run()
```

#### Accessing Detailed Results

```go
// All results are available on the Result struct
fmt.Printf("Average Download: %.2f Mbps\n", result.AvgDownloadMbps)
fmt.Printf("Max Download: %.2f Mbps\n", result.MaxDownloadMbps)
fmt.Printf("Min Download: %.2f Mbps\n", result.MinDownloadMbps)
fmt.Printf("Std Dev Download: %.2f Mbps\n", result.StdDevDownloadMbps)

fmt.Printf("Average Upload: %.2f Mbps\n", result.AvgUploadMbps)
fmt.Printf("Jitter: %v\n", result.Jitter)
fmt.Printf("Timestamp: %v\n", result.Timestamp)
```

#### Running Benchmarks Programmatically

```go
// Benchmark all providers
client := speedtest.New(
	speedtest.WithProvider("all"),
	speedtest.WithDuration(10*time.Second),
)

results, err := client.RunBenchmark(3)  // Run 3 tests per provider
if err != nil {
	log.Fatal(err)
}

// Access benchmark results
for _, result := range results {
	fmt.Printf("%s - Score: %.2f\n", result.ProviderName, result.Score)
}
```

## Design Principles

NetPulse follows Go best practices and these core design principles:

### Public Library API (`pkg/`)
- **Functional Options Pattern**: Clean, extensible configuration API
- **Minimal Dependencies**: Only necessary external imports
- **Stable Interface**: Backward compatible across versions
- **Well-Documented**: Exported functions have clear documentation
- **Importable**: Easy to use as a library: `go get github.com/0x524a/netpulse@latest`

### Internal Implementation (`internal/`)
- **Provider Abstraction**: Common interface for all speed test providers
- **Extensible**: Add new providers by implementing the interface
- **Resilient**: Automatic fallback between providers
- **Testable**: Unit tested with mocked providers and integration tests

### CLI Application (`cmd/`)
- **Demonstrates Library Usage**: Shows how to use NetPulse as a library
- **Production-Ready**: Full feature set with benchmarking and result saving
- **User-Friendly**: Intuitive command-line interface with helpful options
- **Flexible**: Support for multiple providers, custom test parameters

### Key Architectural Features
- **Resilience**: If one provider is down, automatically falls back to alternatives
- **Flexibility**: Choose specific providers, use automatic selection, or run benchmarks
- **Consistency**: All providers return standardized results with identical metrics
- **Extensibility**: New providers can be added without modifying existing code
- **Clean Separation**: Clear boundary between public API and implementation details

## Supported Speed Test Providers

1. **Fast.com** (Netflix CDN)
   - Uses Netflix's global CDN infrastructure
   - Custom implementation with token-based authentication
   
2. **Cloudflare** (speed.cloudflare.com)
   - Cloudflare's global edge network
   - Fast and reliable worldwide coverage
   
3. **M-Lab** (Measurement Lab)
   - Open-source internet measurement platform
   - Operated by research institutions
   
4. **LibreSpeed**
   - Community-driven open-source speed test
   - Privacy-focused with no tracking
   
5. **Ookla/Speedtest.net**
   - Industry standard speed test
   - Extensive global server network

**Auto mode** (default) tries each provider in order until one succeeds, providing resilience against service outages.

## Result Output

Results include detailed metrics:
- **Download Speed**: Mbps with min/max/average
- **Upload Speed**: Mbps with min/max/average
- **Latency**: Round-trip time to test server
- **Jitter**: Latency variation (stability metric)
- **Standard Deviation**: Consistency of speeds
- **Timestamp**: When the test was run
- **Test Duration**: How long download/upload took

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