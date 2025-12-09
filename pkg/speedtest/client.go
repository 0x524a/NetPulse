package speedtest

import (
	"fmt"
	"time"

	"github.com/0x524a/netpulse/internal/config"
	"github.com/0x524a/netpulse/internal/speedtest"
)

// Result represents the result of a speed test
type Result struct {
	Provider            string
	DownloadSpeed       float64       // Mbps
	UploadSpeed         float64       // Mbps
	Latency             time.Duration // RTT
	Jitter              time.Duration // Latency variation (ms)
	IdleLatency         time.Duration // Baseline latency without load
	PacketLoss          float64       // Percentage (0-100)
	DownloadTime        time.Duration
	UploadTime          time.Duration
	AvgDownload         float64 // Mbps
	MaxDownload         float64 // Mbps
	MinDownload         float64 // Mbps
	StdDevDownload      float64 // Standard deviation of download speeds (Mbps)
	AvgUpload           float64 // Mbps
	MaxUpload           float64 // Mbps
	MinUpload           float64 // Mbps
	StdDevUpload        float64 // Standard deviation of upload speeds (Mbps)
	Timestamp           time.Time
	LatencyMeasurements []time.Duration // For jitter calculation
}

// Client is the public interface for the speed test client
type Client struct {
	config  *config.Config
	tester  *speedtest.SpeedTest
	results []*Result // Store results when running multiple providers
}

// Option is a functional option for configuring the Client
type Option func(*Client)

// WithVerbose enables verbose output
func WithVerbose(verbose bool) Option {
	return func(c *Client) {
		c.config.Verbose = verbose
	}
}

// WithProvider sets a specific provider (fastcom, cloudflare, mlab, librespeed, ookla, or auto)
func WithProvider(provider string) Option {
	return func(c *Client) {
		c.config.Provider = provider
	}
}

// WithDuration sets the test duration
func WithDuration(duration time.Duration) Option {
	return func(c *Client) {
		c.config.Duration = duration
	}
}

// WithURLCount sets the number of URLs to test with
func WithURLCount(count int) Option {
	return func(c *Client) {
		c.config.URLCount = count
	}
}

// WithOutputFile sets the output file for saving results
func WithOutputFile(filename string) Option {
	return func(c *Client) {
		c.config.SaveToFile = true
		c.config.OutputFile = filename
	}
}

// WithServerMode sets the server selection mode (auto, random, or specific)
func WithServerMode(mode string) Option {
	return func(c *Client) {
		c.config.ServerMode = mode
	}
}

// New creates a new speed test client with the given options
func New(opts ...Option) *Client {
	cfg := config.NewConfig()

	client := &Client{
		config: cfg,
	}

	// Apply options
	for _, opt := range opts {
		opt(client)
	}

	client.tester = speedtest.New(cfg)

	return client
}

// Run executes the speed test and returns the results
func (c *Client) Run() (*Result, error) {
	// Handle "all" provider option to test with all providers
	if c.config.Provider == "all" {
		return c.runAllProviders()
	}

	// Handle "multiple" provider option for comma-separated providers
	if c.config.Provider == "multiple" {
		return c.runMultipleProviders(c.config.Providers)
	}

	if err := c.tester.Run(); err != nil {
		return nil, fmt.Errorf("speed test failed: %w", err)
	}

	// Get the result from internal speedtest
	internalResult := c.tester.GetResult()
	if internalResult == nil {
		return nil, fmt.Errorf("no result available")
	}

	// Convert to public Result type
	result := &Result{
		Provider:      internalResult.ProviderName,
		DownloadSpeed: internalResult.DownloadSpeed / 1000.0, // Convert Kbps to Mbps
		UploadSpeed:   internalResult.UploadSpeed / 1000.0,   // Convert Kbps to Mbps
		Latency:       internalResult.Latency,
		DownloadTime:  internalResult.DownloadTime,
		UploadTime:    internalResult.UploadTime,
		AvgDownload:   internalResult.AvgDownload / 1000.0,
		MaxDownload:   internalResult.MaxDownload / 1000.0,
		MinDownload:   internalResult.MinDownload / 1000.0,
		AvgUpload:     internalResult.AvgUpload / 1000.0,
		MaxUpload:     internalResult.MaxUpload / 1000.0,
		MinUpload:     internalResult.MinUpload / 1000.0,
		Timestamp:     internalResult.Timestamp,
	}

	c.results = []*Result{result}
	return result, nil
}

// runAllProviders runs the speed test with all available providers
func (c *Client) runAllProviders() (*Result, error) {
	providers := ListProviders()
	c.results = make([]*Result, 0)
	totalProviders := len(providers)

	for i, provider := range providers {
		if c.config.Verbose {
			fmt.Printf("\n[%d/%d] Testing with provider: %s\n", i+1, totalProviders, provider)
			fmt.Println("─────────────────────────────────────────────────")
		}

		// Create a new tester for each provider
		tempConfig := config.NewConfig()
		tempConfig.Verbose = c.config.Verbose
		tempConfig.Provider = provider
		tempConfig.Duration = c.config.Duration
		tempConfig.URLCount = c.config.URLCount
		tempConfig.ServerMode = c.config.ServerMode

		tester := speedtest.New(tempConfig)

		if err := tester.Run(); err != nil {
			if c.config.Verbose {
				fmt.Printf("❌ Error testing with %s: %v\n", provider, err)
			}
			continue
		}

		internalResult := tester.GetResult()
		if internalResult == nil {
			if c.config.Verbose {
				fmt.Printf("❌ No result from provider %s\n", provider)
			}
			continue
		}

		result := &Result{
			Provider:      internalResult.ProviderName,
			DownloadSpeed: internalResult.DownloadSpeed / 1000.0,
			UploadSpeed:   internalResult.UploadSpeed / 1000.0,
			Latency:       internalResult.Latency,
			DownloadTime:  internalResult.DownloadTime,
			UploadTime:    internalResult.UploadTime,
			AvgDownload:   internalResult.AvgDownload / 1000.0,
			MaxDownload:   internalResult.MaxDownload / 1000.0,
			MinDownload:   internalResult.MinDownload / 1000.0,
			AvgUpload:     internalResult.AvgUpload / 1000.0,
			MaxUpload:     internalResult.MaxUpload / 1000.0,
			MinUpload:     internalResult.MinUpload / 1000.0,
			Timestamp:     internalResult.Timestamp,
		}

		c.results = append(c.results, result)
		
		if c.config.Verbose {
			fmt.Printf("✅ Completed %s: ↓ %.2f Mbps  ↑ %.2f Mbps  Latency: %.2f ms\n",
				provider, result.DownloadSpeed, result.UploadSpeed, 
				float64(result.Latency.Microseconds())/1000.0)
		}
	}

	if len(c.results) == 0 {
		return nil, fmt.Errorf("no providers successfully completed the test")
	}

	// Return the first result
	return c.results[0], nil
}

// runMultipleProviders runs the speed test with specific providers
func (c *Client) runMultipleProviders(providers []string) (*Result, error) {
	c.results = make([]*Result, 0)

	for _, provider := range providers {
		if c.config.Verbose {
			fmt.Printf("\n--- Testing with provider: %s ---\n", provider)
		}

		// Create a new tester for each provider
		tempConfig := config.NewConfig()
		tempConfig.Verbose = c.config.Verbose
		tempConfig.Provider = provider
		tempConfig.Duration = c.config.Duration
		tempConfig.URLCount = c.config.URLCount
		tempConfig.ServerMode = c.config.ServerMode

		tester := speedtest.New(tempConfig)

		if err := tester.Run(); err != nil {
			if c.config.Verbose {
				fmt.Printf("Error testing with %s: %v\n", provider, err)
			}
			continue
		}

		internalResult := tester.GetResult()
		if internalResult == nil {
			if c.config.Verbose {
				fmt.Printf("No result from provider %s\n", provider)
			}
			continue
		}

		result := &Result{
			Provider:      internalResult.ProviderName,
			DownloadSpeed: internalResult.DownloadSpeed / 1000.0,
			UploadSpeed:   internalResult.UploadSpeed / 1000.0,
			Latency:       internalResult.Latency,
			DownloadTime:  internalResult.DownloadTime,
			UploadTime:    internalResult.UploadTime,
			AvgDownload:   internalResult.AvgDownload / 1000.0,
			MaxDownload:   internalResult.MaxDownload / 1000.0,
			MinDownload:   internalResult.MinDownload / 1000.0,
			AvgUpload:     internalResult.AvgUpload / 1000.0,
			MaxUpload:     internalResult.MaxUpload / 1000.0,
			MinUpload:     internalResult.MinUpload / 1000.0,
			Timestamp:     internalResult.Timestamp,
		}

		c.results = append(c.results, result)
	}

	if len(c.results) == 0 {
		return nil, fmt.Errorf("no providers successfully completed the test")
	}

	// Return the first result
	return c.results[0], nil
}

// GetFormattedResult returns a formatted string representation of the result
func (c *Client) GetFormattedResult() string {
	// If we have multiple results from "all" providers, format them all
	if len(c.results) > 1 {
		output := "Internet Speed Test Results - All Providers\n"
		output += "==========================================\n\n"

		for _, result := range c.results {
			output += fmt.Sprintf("Provider: %s\n", result.Provider)
			output += fmt.Sprintf("Download: %.2f Mbps (Avg: %.2f, Max: %.2f, Min: %.2f)\n",
				result.DownloadSpeed, result.AvgDownload, result.MaxDownload, result.MinDownload)
			output += fmt.Sprintf("Upload: %.2f Mbps (Avg: %.2f, Max: %.2f, Min: %.2f)\n",
				result.UploadSpeed, result.AvgUpload, result.MaxUpload, result.MinUpload)
			output += fmt.Sprintf("Latency: %.2f ms\n", float64(result.Latency.Microseconds())/1000.0)
			output += fmt.Sprintf("Timestamp: %s\n", result.Timestamp.Format(time.RFC3339))
			output += "---\n\n"
		}

		return output
	}

	return c.tester.FormatResult()
}

// GetAllResults returns all results from running multiple providers
func (c *Client) GetAllResults() []*Result {
	return c.results
}

// ListProviders returns a list of available providers
func ListProviders() []string {
	return []string{
		"fastcom",
		"cloudflare",
		"mlab",
		"librespeed",
		"ookla",
	}
}
