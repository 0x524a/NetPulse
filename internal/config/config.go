package config

import (
	"flag"
	"strings"
	"time"
)

// Config holds the configuration for the speed test
type Config struct {
	URLCount   int
	Duration   time.Duration
	Verbose    bool
	SaveToFile bool
	OutputFile string
	Provider   string
	Providers  []string // List of providers to test with
	ServerMode string   // "specific", "random", or "auto" (best available)
}

// NewConfig creates a new Config with default values
func NewConfig() *Config {
	return &Config{
		URLCount:   5,
		Duration:   15 * time.Second,
		Verbose:    false,
		SaveToFile: false,
		OutputFile: "speedtest_results.txt",
		Provider:   "auto",
		ServerMode: "auto",
	}
}

// LoadFromFlags loads configuration from command line flags
func (c *Config) LoadFromFlags() {
	flag.IntVar(&c.URLCount, "urls", c.URLCount, "Number of URLs to test with")
	flag.DurationVar(&c.Duration, "duration", c.Duration, "Duration of the speed test")
	flag.BoolVar(&c.Verbose, "verbose", c.Verbose, "Enable verbose output")
	flag.BoolVar(&c.SaveToFile, "save", c.SaveToFile, "Save results to file")
	flag.StringVar(&c.OutputFile, "output", c.OutputFile, "Output file name")
	flag.StringVar(&c.Provider, "provider", c.Provider, "Speed test provider: auto, fastcom, cloudflare, mlab, librespeed, ookla, all, or comma-separated list")
	flag.StringVar(&c.ServerMode, "server", c.ServerMode, "Server selection mode: auto (best), random, or specific")
	flag.Parse()

	// Parse comma-separated providers if provided
	if c.Provider != "auto" && c.Provider != "all" && strings.Contains(c.Provider, ",") {
		c.Providers = strings.Split(strings.TrimSpace(c.Provider), ",")
		// Clean up each provider name
		for i, p := range c.Providers {
			c.Providers[i] = strings.TrimSpace(p)
		}
		// Set Provider to a special value to indicate multiple providers
		c.Provider = "multiple"
	}
}
