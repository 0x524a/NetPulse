package librespeed

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	defaultServer = "https://speedtest-ams.librefiber.net"
	bufferSize    = 8192
	userAgent     = "netpulse/1.0"
)

var (
	ErrNoServers = errors.New("no servers available")
	ErrInternet  = errors.New("internet error. Please try again later")
)

// Server represents a LibreSpeed server
type Server struct {
	URL  string  `json:"url"`
	Name string  `json:"name"`
	Ping float64 `json:"ping"`
}

// Client represents a LibreSpeed speed test client
type Client struct {
	server *Server
	client *http.Client
}

// New creates a new LibreSpeed client
func New() *Client {
	return &Client{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the provider name
func (c *Client) Name() string {
	return "LibreSpeed"
}

// IsAvailable checks if LibreSpeed is accessible
func (c *Client) IsAvailable() bool {
	resp, err := c.client.Get(defaultServer)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// Init initializes the client by finding the best server
func (c *Client) Init() error {
	// Use a default server for simplicity
	// In production, you'd fetch a list of servers and choose the best one
	c.server = &Server{
		URL:  defaultServer,
		Name: "LibreSpeed Default",
	}
	return nil
}

// MeasureDownload measures download speed (implements Provider interface)
func (c *Client) MeasureDownload(speedChan chan<- float64) error {
	if c.server == nil {
		return ErrNoServers
	}

	done := make(chan struct{})
	var once sync.Once
	stop := func() {
		close(done)
		close(speedChan)
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

	// Start download test
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

// downloadChunk downloads a chunk of random data
func (c *Client) downloadChunk() int64 {
	// Use a test file endpoint that serves larger data for accurate speed measurement
	url := "https://cachefly.cachefly.net/100mb.test"

	resp, err := c.client.Get(url)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

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
	if c.server == nil {
		return ErrNoServers
	}

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
	close(speedChan)

	return nil
}

// uploadChunk uploads a chunk of data
func (c *Client) uploadChunk(data []byte) int64 {
	// Use httpbin.org for upload testing
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
	defer resp.Body.Close()

	io.Copy(io.Discard, resp.Body)

	return int64(len(data))
}

// MeasureLatency measures the round-trip time (implements Provider interface)
func (c *Client) MeasureLatency() (time.Duration, error) {
	if c.server == nil {
		return 0, ErrNoServers
	}

	// Measure ping to empty.php endpoint
	url := fmt.Sprintf("%s/backend/empty.php", c.server.URL)

	start := time.Now()
	resp, err := c.client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	return time.Since(start), nil
}

func (c *Client) MeasureJitter(samples int) (time.Duration, error) {
	if c.server == nil {
		return 0, ErrNoServers
	}

	if samples <= 1 {
		samples = 10
	}

	var measurements []time.Duration
	url := fmt.Sprintf("%s/backend/empty.php", c.server.URL)

	for i := 0; i < samples; i++ {
		start := time.Now()
		resp, err := c.client.Get(url)
		if err != nil {
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
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
