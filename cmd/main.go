package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/0x524a/netpulse/pkg/reporter"
	"github.com/0x524a/netpulse/pkg/speedtest"
)

func main() {
	app := &cli.App{
		Name:    "speedtest",
		Usage:   "Internet Speed Test Client",
		Version: "1.0.0",
		Authors: []*cli.Author{
			{
				Name: "Speed Test Client",
			},
		},
		Description: "A comprehensive internet speed test tool with multiple provider support, jitter measurement, and connection classification.",
		Commands: []*cli.Command{
			{
				Name:    "providers",
				Aliases: []string{"list-providers", "list"},
				Usage:   "List all available speed test providers",
				Action:  cmdProviders,
			},
			{
				Name:    "benchmark",
				Aliases: []string{"bench", "compare"},
				Usage:   "Benchmark and compare all providers",
				Action:  cmdBenchmark,
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "tests",
						Aliases: []string{"t"},
						Value:   1,
						Usage:   "Number of tests to run per provider",
					},
					&cli.DurationFlag{
						Name:    "duration",
						Aliases: []string{"d"},
						Value:   10 * time.Second,
						Usage:   "Duration of each speed test",
					},
					&cli.BoolFlag{
						Name:  "verbose",
						Usage: "Enable verbose output",
					},
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Output file for benchmark results",
					},
				},
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "provider",
				Aliases: []string{"p"},
				Value:   "auto",
				Usage:   "Speed test provider: auto, fastcom, cloudflare, mlab, librespeed, ookla, all, or comma-separated list",
				EnvVars: []string{"SPEEDTEST_PROVIDER"},
			},
			&cli.DurationFlag{
				Name:    "duration",
				Aliases: []string{"d"},
				Value:   15 * time.Second,
				Usage:   "Duration of the speed test",
				EnvVars: []string{"SPEEDTEST_DURATION"},
			},
			&cli.IntFlag{
				Name:    "urls",
				Aliases: []string{"u"},
				Value:   5,
				Usage:   "Number of URLs to test with",
				EnvVars: []string{"SPEEDTEST_URLS"},
			},
			&cli.StringFlag{
				Name:    "server",
				Aliases: []string{"s"},
				Value:   "auto",
				Usage:   "Server selection mode: auto, random, or specific",
				EnvVars: []string{"SPEEDTEST_SERVER"},
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Usage:   "Enable verbose output during test",
				EnvVars: []string{"SPEEDTEST_VERBOSE"},
			},
			&cli.BoolFlag{
				Name:    "save",
				Aliases: []string{"w"},
				Usage:   "Save results to a file",
				EnvVars: []string{"SPEEDTEST_SAVE"},
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Value:   "speedtest_results.txt",
				Usage:   "Output file name",
				EnvVars: []string{"SPEEDTEST_OUTPUT"},
			},
		},
		Action: cmdRun,
		After: func(c *cli.Context) error {
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func cmdRun(c *cli.Context) error {
	provider := c.String("provider")
	duration := c.Duration("duration")
	urls := c.Int("urls")
	server := c.String("server")
	verbose := c.Bool("verbose")
	save := c.Bool("save")
	output := c.String("output")

	fmt.Println("Internet Speed Test Client")
	fmt.Println("===========================")
	fmt.Println()

	// Create speed test client with options
	opts := []speedtest.Option{
		speedtest.WithVerbose(verbose),
		speedtest.WithProvider(provider),
		speedtest.WithServerMode(server),
		speedtest.WithDuration(duration),
		speedtest.WithURLCount(urls),
	}

	if save {
		opts = append(opts, speedtest.WithOutputFile(output))
	}

	client := speedtest.New(opts...)

	// Run the speed test
	_, err := client.Run()
	if err != nil {
		return fmt.Errorf("error running speed test: %w", err)
	}

	// Get formatted results
	results := client.GetFormattedResult()

	// Create reporter and output results
	rep := reporter.New()
	rep.PrintToConsole(results)

	// Save to file if requested
	if save {
		err = rep.SaveToFile(output, results)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to save results to file: %v\n", err)
		}
	}

	return nil
}

func cmdProviders(c *cli.Context) error {
	fmt.Print(`Available Speed Test Providers
===============================

1. Fast.com (fastcom)
   - Netflix's speed test
   - Accurate for streaming speeds
   - Good for casual users

2. Cloudflare (cloudflare)
   - Cloudflare's speed test
   - Low latency focus
   - Global CDN coverage

3. M-Lab (mlab)
   - Google's open-source measurement lab
   - Research-focused
   - NDT protocol support

4. LibreSpeed (librespeed)
   - Open-source speed test
   - Privacy-focused
   - Self-hosted options available

5. Speedtest.net (ookla)
   - Industry standard
   - Largest server network
   - Most widely used

Connection Grade Meanings:
  A: Excellent (≥ 300 Mbps)
  B: Good (100-300 Mbps)
  C: Fair (10-100 Mbps)
  D: Poor (5-10 Mbps)
  F: Very Poor (< 5 Mbps)

Quality Metrics:
  - Streaming: ≥ 25 Mbps recommended
  - Gaming: ≥ 35 Mbps recommended
  - Video Calls: ≥ 10 Mbps recommended
  - Latency: < 20ms is excellent
  - Jitter: < 5ms is excellent
`)
	return nil
}

func cmdBenchmark(c *cli.Context) error {
	numTests := c.Int("tests")
	duration := c.Duration("duration")
	outputFile := c.String("output")

	providers := []string{"fastcom", "cloudflare", "mlab", "librespeed", "ookla"}
	totalProviders := len(providers)
	
	fmt.Println("Internet Speed Test Client - Provider Benchmark")
	fmt.Println("==============================================")
	fmt.Println()
	fmt.Printf("Testing %d providers with %d test(s) each\n", totalProviders, numTests)
	fmt.Printf("Duration per test: %s\n", duration)
	fmt.Printf("Estimated total time: ~%s\n", time.Duration(int64(duration)*int64(totalProviders)*int64(numTests)*12/10))
	fmt.Println()
	fmt.Println("💡 Tip: Press Ctrl+C to cancel at any time")
	fmt.Println()

	// For now, run a comprehensive test with all providers
	client := speedtest.New(
		speedtest.WithVerbose(true), // Force verbose to show progress
		speedtest.WithProvider("all"),
		speedtest.WithDuration(duration),
	)

	_, err := client.Run()
	if err != nil {
		return fmt.Errorf("error running benchmark: %w", err)
	}

	results := client.GetFormattedResult()
	rep := reporter.New()
	rep.PrintToConsole(results)

	// Save to file if requested
	if outputFile != "" {
		err = rep.SaveToFile(outputFile, results)
		if err != nil {
			return fmt.Errorf("error saving benchmark results: %w", err)
		}
		fmt.Printf("\nBenchmark results saved to %s\n", outputFile)
	}

	return nil
}
