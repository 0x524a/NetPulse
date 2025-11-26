package ookla

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
		close(speedChan)
	}

	// Track bytes downloaded
	var totalBytes int64
	var byteMux sync.Mutex
	startTime := time.Now()

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
	// Use a test endpoint (Note: This is simplified - real Ookla uses specific endpoints)
	url := fmt.Sprintf("%s/speedtest/random%dx%d.jpg?r=%d", c.serverURL, 4000, 4000, time.Now().UnixNano())

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
	url := fmt.Sprintf("%s/speedtest/upload.php?r=%d", c.serverURL, time.Now().UnixNano())

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
