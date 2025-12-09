package ookla

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	defaultServer = "https://speedtest.net"
	bufferSize    = 8192
	userAgent     = "netpulse/1.0"
)

var (
	ErrNoServers = errors.New("no servers available")
	ErrInternet  = errors.New("internet error. Please try again later")
)

// Client represents an Ookla/Speedtest.net client
type Client struct {
	serverURL string
	client    *http.Client
}

// New creates a new Ookla client
func New() *Client {
	return &Client{
		serverURL: defaultServer,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the provider name
func (c *Client) Name() string {
	return "Speedtest.net (Ookla)"
}

// IsAvailable checks if the service is accessible
func (c *Client) IsAvailable() bool {
	resp, err := c.client.Get(c.serverURL)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode == http.StatusOK
}

// Init initializes the client
func (c *Client) Init() error {
	// For simplicity, we use a default server
	// Production code would fetch server list and choose best one
	return nil
}

// MeasureDownload measures download speed (implements Provider interface)
func (c *Client) MeasureDownload(speedChan chan<- float64) error {
	done := make(chan struct{})
	var once sync.Once
	stop := func() {
		close(done)
	}

	// Track bytes downloaded
	var totalBytes int64
	var byteMux sync.Mutex
	var lastBytes int64
	var lastTime time.Time
	lastTime = time.Now()

	// Timeout after 15 seconds
	go func() {
		time.Sleep(15 * time.Second)
		once.Do(stop)
	}()

	// Report speed every second
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	go func() {
		defer func() {
			// Ensure we don't panic if channel is closed
			_ = recover()
		}()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				byteMux.Lock()
				currentBytes := totalBytes
				byteMux.Unlock()

				now := time.Now()
				elapsed := now.Sub(lastTime).Seconds()
				bytesDiff := currentBytes - lastBytes

				if elapsed > 0 && bytesDiff > 0 {
					kbps := float64(bytesDiff*8) / 1000 / elapsed
					select {
					case speedChan <- kbps:
					case <-done:
						return
					}
				}

				lastBytes = currentBytes
				lastTime = now
			}
		}
	}()

	// Download from multiple connections
	var wg sync.WaitGroup
	numConnections := 4

	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-done:
					return
				default:
					downloaded := c.downloadChunk()
					if downloaded > 0 {
						byteMux.Lock()
						totalBytes += downloaded
						byteMux.Unlock()
					}
				}
			}
		}()
	}

	wg.Wait()
	once.Do(stop)

	return nil
}

// downloadChunk downloads test data
func (c *Client) downloadChunk() int64 {
	// Use a test file endpoint that serves larger data for accurate speed measurement
	url := "https://cachefly.cachefly.net/100mb.test"

	resp, err := c.client.Get(url)
	if err != nil {
		return 0
	}
	defer func() { _ = resp.Body.Close() }()

	buf := make([]byte, bufferSize)
	var total int64

	for {
		n, err := resp.Body.Read(buf)
		total += int64(n)
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
	}

	return total
}

// MeasureUpload measures upload speed (implements Provider interface)
func (c *Client) MeasureUpload(duration time.Duration, speedChan chan<- float64) error {
	done := make(chan struct{})
	var wg sync.WaitGroup

	// Track total bytes uploaded
	var totalBytes int64
	var byteMux sync.Mutex
	startTime := time.Now()

	// Stop after duration
	go func() {
		time.Sleep(duration)
		close(done)
	}()

	// Report speed every second
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	go func() {
		defer func() {
			// Ensure we don't panic if channel is closed
			_ = recover()
		}()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				elapsed := time.Since(startTime).Seconds()
				if elapsed > 0 {
					byteMux.Lock()
					kbps := (float64(totalBytes) * 8) / 1000 / elapsed
					byteMux.Unlock()

					select {
					case speedChan <- kbps:
					case <-done:
						return
					}
				}
			}
		}
	}()

	// Create random data (1MB chunks)
	chunkSize := 1024 * 1024
	data := make([]byte, chunkSize)
	if _, err := rand.Read(data); err != nil {
		return err
	}

	// Upload from multiple connections
	numConnections := 4
	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-done:
					return
				default:
					uploaded := c.uploadChunk(data)
					if uploaded > 0 {
						byteMux.Lock()
						totalBytes += uploaded
						byteMux.Unlock()
					}
				}
			}
		}()
	}

	wg.Wait()

	return nil
}

// uploadChunk uploads a chunk of data
func (c *Client) uploadChunk(data []byte) int64 {
	url := "https://httpbin.org/post"

	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return 0
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("User-Agent", userAgent)
	req.ContentLength = int64(len(data))

	resp, err := c.client.Do(req)
	if err != nil {
		return 0
	}
	defer func() { _ = resp.Body.Close() }()

	_, _ = io.Copy(io.Discard, resp.Body)

	return int64(len(data))
}

// MeasureLatency measures the round-trip time (implements Provider interface)
func (c *Client) MeasureLatency() (time.Duration, error) {
	start := time.Now()
	resp, err := c.client.Get(c.serverURL)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)

	return time.Since(start), nil
}

// MeasureJitter measures latency variation
func (c *Client) MeasureJitter(samples int) (time.Duration, error) {
	if samples <= 1 {
		samples = 10
	}

	var measurements []time.Duration
	for i := 0; i < samples; i++ {
		start := time.Now()
		resp, err := c.client.Get(c.serverURL)
		if err != nil {
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		measurements = append(measurements, time.Since(start))
	}

	if len(measurements) <= 1 {
		return 0, nil
	}

	// Calculate jitter as mean absolute deviation
	var sumLatency int64
	for _, m := range measurements {
		sumLatency += m.Milliseconds()
	}
	meanLatency := sumLatency / int64(len(measurements))

	var totalDeviation int64
	for _, m := range measurements {
		diff := m.Milliseconds() - meanLatency
		if diff < 0 {
			diff = -diff
		}
		totalDeviation += diff
	}

	jitterMs := totalDeviation / int64(len(measurements))
	return time.Duration(jitterMs * 1000000), nil // Convert back to nanoseconds
}
