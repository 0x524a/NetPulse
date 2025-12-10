package speedtest

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/0x524a/netpulse/internal/config"
	"github.com/0x524a/netpulse/internal/providers/cloudflare"
	"github.com/0x524a/netpulse/internal/providers/fastcom"
	"github.com/0x524a/netpulse/internal/providers/librespeed"
	"github.com/0x524a/netpulse/internal/providers/mlab"
	"github.com/0x524a/netpulse/internal/providers/ookla"
	"github.com/0x524a/netpulse/internal/providers/provider"
	"github.com/0x524a/netpulse/pkg/metrics"
)

// SpeedTest manages the speed testing operations
type SpeedTest struct {
	config         *config.Config
	result         *provider.Result
	resultMu       sync.RWMutex
	providers      []provider.Provider
	activeProvider provider.Provider
}

// New creates a new SpeedTest instance
func New(cfg *config.Config) *SpeedTest {
	return &SpeedTest{
		config: cfg,
		providers: []provider.Provider{
			fastcom.New(),
			cloudflare.New(),
			mlab.New(),
			librespeed.New(),
			ookla.New(),
		},
	}
}

// Run executes the speed test with provider fallback
func (st *SpeedTest) Run() error {
	if st.config.Verbose {
		fmt.Println("Starting speed test...")
		if st.config.Provider == "auto" {
			if st.config.ServerMode == "random" {
				fmt.Printf("Using random provider from %d available\n", len(st.providers))
			} else {
				fmt.Printf("Testing with fallback across %d providers\n", len(st.providers))
			}
		} else {
			fmt.Printf("Using provider: %s\n", st.config.Provider)
		}
		fmt.Println("---")
	}

	// Handle random provider selection
	if st.config.Provider == "auto" && st.config.ServerMode == "random" {
		return st.runWithRandomProvider()
	}

	// If specific provider requested, use only that one
	if st.config.Provider != "auto" {
		for _, p := range st.providers {
			providerName := ""
			switch st.config.Provider {
			case "fastcom":
				providerName = "Fast.com"
			case "cloudflare":
				providerName = "Cloudflare"
			case "mlab":
				providerName = "M-Lab"
			case "librespeed":
				providerName = "LibreSpeed"
			case "ookla":
				providerName = "Speedtest.net (Ookla)"
			}

			if p.Name() == providerName {
				// Check availability
				if !p.IsAvailable() {
					if st.config.Verbose {
						fmt.Printf("%s is not currently available\n", providerName)
					}
					return fmt.Errorf("provider '%s' is not available", st.config.Provider)
				}

				// Initialize provider
				if err := p.Init(); err != nil {
					return fmt.Errorf("failed to initialize provider '%s': %w", st.config.Provider, err)
				}

				st.activeProvider = p
				return st.runWithProvider(p)
			}
		}
		return fmt.Errorf("provider '%s' not found", st.config.Provider)
	}

	// Try each provider until one succeeds
	var lastErr error
	for _, p := range st.providers {
		if st.config.Verbose {
			fmt.Printf("Trying provider: %s...\n", p.Name())
		}

		// Check if provider is available
		if !p.IsAvailable() {
			if st.config.Verbose {
				fmt.Printf("%s is not available, trying next provider\n", p.Name())
			}
			continue
		}

		// Initialize provider
		err := p.Init()
		if err != nil {
			if st.config.Verbose {
				fmt.Printf("Failed to initialize %s: %v\n", p.Name(), err)
			}
			lastErr = err
			continue
		}

		// Run the test with this provider
		st.activeProvider = p
		err = st.runWithProvider(p)
		if err == nil {
			return nil
		}

		if st.config.Verbose {
			fmt.Printf("Test failed with %s: %v\n", p.Name(), err)
		}
		lastErr = err
	}

	if lastErr != nil {
		return fmt.Errorf("all providers failed, last error: %w", lastErr)
	}
	return fmt.Errorf("no providers available")
}

// runWithProvider runs the test using a specific provider
func (st *SpeedTest) runWithProvider(p provider.Provider) error {
	if st.config.Verbose {
		fmt.Printf("Using provider: %s\n", p.Name())
		fmt.Println("---")
	}

	start := time.Now()

	// Measure latency
	latency, err := p.MeasureLatency()
	if err != nil {
		return fmt.Errorf("latency measurement failed: %w", err)
	}

	if st.config.Verbose {
		fmt.Printf("Latency: %.2f ms\n", float64(latency.Microseconds())/1000.0)
		fmt.Println("---")
	}

	// Measure jitter (latency variation)
	jitter, err := p.MeasureJitter(10) // Measure with 10 samples
	if err != nil {
		jitter = 0 // If jitter measurement fails, set to 0
	}

	if st.config.Verbose && jitter > 0 {
		fmt.Printf("Jitter: %.2f ms\n", float64(jitter.Microseconds())/1000.0)
		fmt.Println("---")
	}

	// Measure download speed
	speedChan := make(chan float64, 100)
	var finalSpeed float64
	var minSpeed, maxSpeed, totalSpeed float64
	var speedCount int
	done := make(chan bool)

	go func() {
		for speed := range speedChan {
			finalSpeed = speed
			speedMbps := speed / 1000.0

			if minSpeed == 0 || speed < minSpeed {
				minSpeed = speed
			}
			if speed > maxSpeed {
				maxSpeed = speed
			}
			totalSpeed += speed
			speedCount++

			if st.config.Verbose {
				fmt.Printf("\rDownload speed: %.2f Mbps", speedMbps)
			}
		}
		done <- true
	}()

	err = p.MeasureDownload(speedChan)
	if err != nil {
		return fmt.Errorf("download measurement failed: %w", err)
	}
	close(speedChan)

	<-done

	downloadDuration := time.Since(start)

	if st.config.Verbose {
		fmt.Println()
		fmt.Println("---")
		fmt.Println("Testing upload speed...")
	}

	var avgDownload float64
	if speedCount > 0 {
		avgDownload = totalSpeed / float64(speedCount)
	}

	// Measure upload speed
	uploadStart := time.Now()
	uploadChan := make(chan float64, 100)
	var finalUpload float64
	var minUpload, maxUpload, totalUpload float64
	var uploadCount int
	uploadDone := make(chan bool)

	go func() {
		for speed := range uploadChan {
			finalUpload = speed
			speedMbps := speed / 1000.0

			if minUpload == 0 || speed < minUpload {
				minUpload = speed
			}
			if speed > maxUpload {
				maxUpload = speed
			}
			totalUpload += speed
			uploadCount++

			if st.config.Verbose {
				fmt.Printf("\rUpload speed: %.2f Mbps", speedMbps)
			}
		}
		uploadDone <- true
	}()

	testDuration := 5 * time.Second
	if st.config.Duration < testDuration {
		testDuration = st.config.Duration
	}

	err = p.MeasureUpload(testDuration, uploadChan)
	if err != nil {
		return fmt.Errorf("upload measurement failed: %w", err)
	}
	close(uploadChan)

	<-uploadDone

	uploadDuration := time.Since(uploadStart)

	if st.config.Verbose {
		fmt.Println()
	}

	var avgUpload float64
	if uploadCount > 0 {
		avgUpload = totalUpload / float64(uploadCount)
	}

	// Store results
	st.resultMu.Lock()
	st.result = &provider.Result{
		ProviderName:  p.Name(),
		DownloadSpeed: finalSpeed,
		DownloadMbps:  finalSpeed / 1000.0,
		MinDownload:   minSpeed / 1000.0,
		MaxDownload:   maxSpeed / 1000.0,
		AvgDownload:   avgDownload / 1000.0,
		UploadSpeed:   finalUpload,
		UploadMbps:    finalUpload / 1000.0,
		MinUpload:     minUpload / 1000.0,
		MaxUpload:     maxUpload / 1000.0,
		AvgUpload:     avgUpload / 1000.0,
		Latency:       latency,
		Jitter:        jitter,
		DownloadTime:  downloadDuration,
		UploadTime:    uploadDuration,
		Timestamp:     time.Now(),
		Success:       true,
	}
	st.resultMu.Unlock()

	return nil
}

// GetResult returns the speed test result
func (st *SpeedTest) GetResult() *provider.Result {
	st.resultMu.RLock()
	defer st.resultMu.RUnlock()
	return st.result
}

// FormatResult returns a formatted string of the results
func (st *SpeedTest) FormatResult() string {
	st.resultMu.RLock()
	defer st.resultMu.RUnlock()
	
	if st.result == nil {
		return "No results available. Please run the test first."
	}

	// Get connection metrics
	connMetrics := metrics.ClassifySpeed(st.result.DownloadMbps)
	latencyAnalysis := metrics.AnalyzeLatency(st.result.Latency)
	jitterAnalysis := metrics.AnalyzeJitter(st.result.Jitter)
	speedComparison := metrics.GetSpeedComparison(st.result.DownloadMbps)

	output := fmt.Sprintf(`Speed Test Results
==================
Provider:       %s
Grade:          %s (%s)
Quality Score:  %.0f/100

Download Speed: %.2f Mbps (%.2f Kbps)
  - Average:    %.2f Mbps
  - Maximum:    %.2f Mbps
  - Minimum:    %.2f Mbps

Upload Speed:   %.2f Mbps (%.2f Kbps)
  - Average:    %.2f Mbps
  - Maximum:    %.2f Mbps
  - Minimum:    %.2f Mbps

Latency:        %.2f ms (%s)
Jitter:         %.2f ms (%s)

Connection Analysis:
  - Classification: %s
  - Streaming:      %v
  - Gaming:         %v
  - Work/Video Call: %v
  - Comparison:     %s

Download Time:  %v
Upload Time:    %v
Timestamp:      %s
`,
		st.result.ProviderName,
		connMetrics.Grade,
		connMetrics.Classification,
		connMetrics.QualityScore,
		st.result.DownloadMbps,
		st.result.DownloadSpeed,
		st.result.AvgDownload,
		st.result.MaxDownload,
		st.result.MinDownload,
		st.result.UploadMbps,
		st.result.UploadSpeed,
		st.result.AvgUpload,
		st.result.MaxUpload,
		st.result.MinUpload,
		float64(st.result.Latency.Microseconds())/1000.0,
		latencyAnalysis,
		float64(st.result.Jitter.Microseconds())/1000.0,
		jitterAnalysis,
		connMetrics.Classification,
		connMetrics.IsGoodForStreaming,
		connMetrics.IsGoodForGaming,
		connMetrics.IsGoodForWork,
		speedComparison,
		st.result.DownloadTime.Round(time.Millisecond),
		st.result.UploadTime.Round(time.Millisecond),
		st.result.Timestamp.Format(time.RFC3339),
	)

	return output
}

// runWithRandomProvider selects a random provider and runs the test
func (st *SpeedTest) runWithRandomProvider() error {
	// Shuffle providers to get random selection
	shuffled := make([]provider.Provider, len(st.providers))
	copy(shuffled, st.providers)

	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	// Try to find an available provider
	for _, p := range shuffled {
		if st.config.Verbose {
			fmt.Printf("Trying random provider: %s...\n", p.Name())
		}

		if !p.IsAvailable() {
			if st.config.Verbose {
				fmt.Printf("%s is not available, trying next random provider\n", p.Name())
			}
			continue
		}

		if err := p.Init(); err != nil {
			if st.config.Verbose {
				fmt.Printf("Failed to initialize %s: %v\n", p.Name(), err)
			}
			continue
		}

		st.activeProvider = p
		return st.runWithProvider(p)
	}

	return fmt.Errorf("no providers available")
}
