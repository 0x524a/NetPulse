package cloudflare

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/0x524a/netpulse/internal/providers/provider"
)

const (
	measurementURL = "https://cachefly.cachefly.net/100mb.test"
	uploadURL      = "https://httpbin.org/post"
	bufferSize     = 1024 * 1024
	userAgent      = "netpulse/1.0"
)

var (
	ErrNotAvailable = errors.New("cloudflare speed test not available")
)

type Client struct {
	client *http.Client
}

func New() *Client {
	return &Client{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Name() string {
	return "Cloudflare"
}

func (c *Client) IsAvailable() bool {
	req, err := http.NewRequest("GET", "https://speed.cloudflare.com", nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (c *Client) Init() error {
	return nil
}

func (c *Client) MeasureLatency() (time.Duration, error) {
	const attempts = 3
	var totalLatency time.Duration

	for i := 0; i < attempts; i++ {
		start := time.Now()

		req, err := http.NewRequest("GET", "https://speed.cloudflare.com", nil)
		if err != nil {
			return 0, err
		}
		req.Header.Set("User-Agent", userAgent)

		resp, err := c.client.Do(req)
		if err != nil {
			return 0, err
		}
		resp.Body.Close()

		latency := time.Since(start)
		totalLatency += latency

		time.Sleep(100 * time.Millisecond)
	}

	return totalLatency / attempts, nil
}

func (c *Client) MeasureJitter(samples int) (time.Duration, error) {
	if samples <= 1 {
		samples = 10
	}

	var measurements []time.Duration
	for i := 0; i < samples; i++ {
		start := time.Now()

		req, err := http.NewRequest("GET", "https://speed.cloudflare.com", nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", userAgent)

		resp, err := c.client.Do(req)
		if err != nil {
			continue
		}
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

func (c *Client) MeasureDownload(speedChan chan<- float64) error {
	done := make(chan struct{})
	var once sync.Once
	stop := func() {
		close(done)
		close(speedChan)
	}

	var totalBytes int64
	var byteMux sync.Mutex
	var lastBytes int64
	var lastTime time.Time
	lastTime = time.Now()

	go func() {
		time.Sleep(15 * time.Second)
		once.Do(stop)
	}()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	go func() {
		defer func() { recover() }()
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
					speedKbps := float64(bytesDiff*8) / 1000.0 / elapsed
					select {
					case speedChan <- speedKbps:
					case <-done:
						return
					}
				}

				lastBytes = currentBytes
				lastTime = now
			}
		}
	}()

	var wg sync.WaitGroup
	errors := make(chan error, 4)

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-done:
					return
				default:
					req, err := http.NewRequest("GET", measurementURL, nil)
					if err != nil {
						errors <- err
						return
					}
					req.Header.Set("User-Agent", userAgent)

					resp, err := c.client.Do(req)
					if err != nil {
						errors <- err
						return
					}

					buffer := make([]byte, bufferSize)
					for {
						select {
						case <-done:
							resp.Body.Close()
							return
						default:
							n, err := resp.Body.Read(buffer)
							if n > 0 {
								byteMux.Lock()
								totalBytes += int64(n)
								byteMux.Unlock()
							}
							if err != nil {
								resp.Body.Close()
								if err != io.EOF {
									errors <- err
								}
								return
							}
						}
					}
				}
			}
		}()
	}

	wg.Wait()
	once.Do(stop)

	select {
	case err := <-errors:
		return err
	default:
		return nil
	}
}

func (c *Client) MeasureUpload(duration time.Duration, speedChan chan<- float64) error {
	done := make(chan struct{})
	var once sync.Once
	stop := func() {
		close(done)
	}

	var totalBytes int64
	var byteMux sync.Mutex
	startTime := time.Now()

	go func() {
		time.Sleep(duration)
		once.Do(stop)
	}()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	go func() {
		defer func() { recover() }()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				byteMux.Lock()
				bytes := totalBytes
				byteMux.Unlock()

				elapsed := time.Since(startTime).Seconds()
				if elapsed > 0 {
					speedKbps := float64(bytes*8) / elapsed / 1000.0
					select {
					case speedChan <- speedKbps:
					case <-done:
						return
					}
				}
			}
		}
	}()

	var wg sync.WaitGroup
	errors := make(chan error, 4)

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			data := make([]byte, bufferSize)
			rand.Read(data)

			for {
				select {
				case <-done:
					return
				default:
					req, err := http.NewRequest("POST", uploadURL, bytes.NewReader(data))
					if err != nil {
						errors <- err
						return
					}
					req.Header.Set("User-Agent", userAgent)
					req.Header.Set("Content-Type", "application/octet-stream")

					resp, err := c.client.Do(req)
					if err != nil {
						errors <- err
						return
					}

					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()

					byteMux.Lock()
					totalBytes += int64(len(data))
					byteMux.Unlock()
				}
			}
		}()
	}

	wg.Wait()
	once.Do(stop)

	select {
	case err := <-errors:
		return err
	default:
		return nil
	}
}

func (c *Client) GetResult() *provider.Result {
	return nil
}
