package speedtest

import (
	"fmt"
	"time"

	"github.com/0x524a/netpulse/internal/config"
	"github.com/0x524a/netpulse/internal/speedtest"
	"github.com/0x524a/netpulse/pkg/benchmarks"
)

// BenchmarkOptions holds options for benchmarking
type BenchmarkOptions struct {
	TestsPerProvider int
	Duration         time.Duration
	Verbose          bool
	Providers        []string
}

// RunBenchmark runs a comprehensive benchmark across all providers
func (c *Client) RunBenchmark(opts BenchmarkOptions) (*benchmarks.BenchmarkSuite, error) {
	suite := benchmarks.NewBenchmarkSuite()

	// Determine which providers to test
	providers := opts.Providers
	if len(providers) == 0 {
		providers = ListProviders()
	}

	totalTests := len(providers) * opts.TestsPerProvider
	currentTest := 0

	// Run tests for each provider
	for _, providerName := range providers {
		for test := 0; test < opts.TestsPerProvider; test++ {
			currentTest++
			if opts.Verbose {
				fmt.Printf("[%d/%d] Testing with %s (%d/%d)...\n",
					currentTest, totalTests, providerName, test+1, opts.TestsPerProvider)
			}

			// Create a new tester for this test
			tempConfig := config.NewConfig()
			tempConfig.Verbose = opts.Verbose
			tempConfig.Provider = providerName
			tempConfig.Duration = opts.Duration
			tempConfig.URLCount = c.config.URLCount
			tempConfig.ServerMode = c.config.ServerMode

			tester := speedtest.New(tempConfig)

			// Run the test
			err := tester.Run()
			if err != nil {
				if opts.Verbose {
					fmt.Printf("Error testing with %s: %v\n", providerName, err)
				}
				// Add failed result
				suite.AddResult(&benchmarks.ProviderResult{
					ProviderName: providerName,
					Success:      false,
					Error:        err.Error(),
				})
				continue
			}

			// Get result
			internalResult := tester.GetResult()
			if internalResult == nil {
				if opts.Verbose {
					fmt.Printf("No result from provider %s\n", providerName)
				}
				suite.AddResult(&benchmarks.ProviderResult{
					ProviderName: providerName,
					Success:      false,
					Error:        "no result returned",
				})
				continue
			}

			// Convert to benchmark result
			benchmarkResult := &benchmarks.ProviderResult{
				ProviderName:        internalResult.ProviderName,
				DownloadMbps:        internalResult.DownloadSpeed / 1000.0,
				UploadMbps:          internalResult.UploadSpeed / 1000.0,
				Latency:             internalResult.Latency,
				Jitter:              internalResult.Jitter,
				MinDownload:         internalResult.MinDownload / 1000.0,
				MaxDownload:         internalResult.MaxDownload / 1000.0,
				AvgDownload:         internalResult.AvgDownload / 1000.0,
				MinUpload:           internalResult.MinUpload / 1000.0,
				MaxUpload:           internalResult.MaxUpload / 1000.0,
				AvgUpload:           internalResult.AvgUpload / 1000.0,
				Success:             true,
				DownloadTime:        internalResult.DownloadTime,
				UploadTime:          internalResult.UploadTime,
				Timestamp:           internalResult.Timestamp,
				LatencyMeasurements: internalResult.LatencyMeasurements,
			}

			suite.AddResult(benchmarkResult)
		}
	}

	// Finalize the benchmark suite
	err := suite.Finalize()
	if err != nil {
		return nil, fmt.Errorf("error finalizing benchmark: %w", err)
	}

	return suite, nil
}
