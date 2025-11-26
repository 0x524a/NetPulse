package fastcom

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"golang.org/x/net/html"
)

const (
	endpoint          = "https://fast.com"
	bufferSize        = 8192
	userAgent         = "netpulse/1.0"
	measureTimeoutMin = 10
	measureTimeoutMax = 30
)

var (
	ErrAPI      = errors.New("fast.com API error. Please try again later")
	ErrInternet = errors.New("internet error. Please try again later")

	reEndpoint = regexp.MustCompile(`apiEndpoint="([\w|\/|\.]*)"`)
	reToken    = regexp.MustCompile(`token:"(\w*)"`)
	reCount    = regexp.MustCompile(`urlCount:(\d*)`)
)

// Client represents a fast.com speed test client
type Client struct {
	token    string
	apiURL   string
	urlCount int
	client   *http.Client
}

// New creates a new fast.com client
func New() *Client {
	return &Client{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		urlCount: 5,
	}
}

// Name returns the provider name
func (c *Client) Name() string {
	return "Fast.com"
}

// IsAvailable checks if fast.com is accessible
func (c *Client) IsAvailable() bool {
	resp, err := c.client.Get(endpoint)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode == http.StatusOK
}

// Init initializes the client by fetching token and API endpoint from fast.com
func (c *Client) Init() error {
	// Fetch the main page
	resp, err := c.client.Get(endpoint)
	if err != nil {
		return ErrInternet
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ErrInternet
	}

	// Parse HTML to find the script source
	scriptSrc, err := c.findScriptSrc(body)
	if err != nil {
		return err
	}

	// Fetch the JavaScript file
	jsURL := endpoint + scriptSrc
	resp, err = c.client.Get(jsURL)
	if err != nil {
		return ErrInternet
	}
	defer func() { _ = resp.Body.Close() }()

	jsData, err := io.ReadAll(resp.Body)
	if err != nil {
		return ErrInternet
	}

	// Extract token, API endpoint, and URL count
	if err := c.parseJSData(jsData); err != nil {
		return err
	}

	return nil
}

// findScriptSrc parses HTML to find the main script source
func (c *Client) findScriptSrc(htmlData []byte) (string, error) {
	doc, err := html.Parse(bytes.NewReader(htmlData))
	if err != nil {
		return "", ErrAPI
	}

	var scriptSrc string
	var findScript func(*html.Node)
	findScript = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "script" {
			for _, attr := range n.Attr {
				if attr.Key == "src" && len(attr.Val) > 0 {
					// Look for the main app script (usually starts with /app-)
					if len(attr.Val) > 4 && attr.Val[:4] == "/app" {
						scriptSrc = attr.Val
						return
					}
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			findScript(child)
			if scriptSrc != "" {
				return
			}
		}
	}

	findScript(doc)
	if scriptSrc == "" {
		return "", ErrAPI
	}

	return scriptSrc, nil
}

// parseJSData extracts token, API endpoint, and URL count from JavaScript
func (c *Client) parseJSData(jsData []byte) error {
	// Extract API endpoint
	matches := reEndpoint.FindSubmatch(jsData)
	if len(matches) < 2 {
		return ErrAPI
	}
	c.apiURL = "https://" + string(matches[1])

	// Extract token
	matches = reToken.FindSubmatch(jsData)
	if len(matches) < 2 {
		return ErrAPI
	}
	c.token = string(matches[1])

	// Extract URL count
	matches = reCount.FindSubmatch(jsData)
	if len(matches) < 2 {
		c.urlCount = 5 // default
	} else {
		count, err := strconv.Atoi(string(matches[1]))
		if err == nil {
			c.urlCount = count
		}
	}

	return nil
}

// GetURLs fetches the test URLs from the API
func (c *Client) GetURLs(count int) ([]string, error) {
	if c.token == "" || c.apiURL == "" {
		return nil, errors.New("client not initialized, call Init() first")
	}

	if count <= 0 {
		count = c.urlCount
	}

	url := fmt.Sprintf("%s?https=true&token=%s&urlCount=%d", c.apiURL, c.token, count)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, ErrInternet
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ErrInternet
	}

	// Parse JSON response to extract URLs
	urls, err := c.parseURLsFromJSON(body)
	if err != nil {
		return nil, ErrAPI
	}

	return urls, nil
}

// parseURLsFromJSON extracts URLs from the JSON response
func (c *Client) parseURLsFromJSON(data []byte) ([]string, error) {
	// Simple JSON parsing for "url" fields
	reURL := regexp.MustCompile(`"url"\s*:\s*"([^"]*)"`)
	matches := reURL.FindAllSubmatch(data, -1)

	if len(matches) == 0 {
		return nil, ErrAPI
	}

	urls := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			urls = append(urls, string(match[1]))
		}
	}

	return urls, nil
}

// MeasureDownload measures download speed (implements Provider interface)
func (c *Client) MeasureDownload(speedChan chan<- float64) error {
	urls, err := c.GetURLs(0)
	if err != nil {
		return err
	}
	if len(urls) == 0 {
		return errors.New("no URLs provided")
	}

	done := make(chan struct{})
	byteLenChan := make(chan int64, 100)

	// Stop function
	var once sync.Once
	stop := func() {
		close(done)
		close(speedChan)
	}

	// Timeout handling
	isTimeout := false
	var timeoutMux sync.Mutex

	// Min timeout
	go func() {
		time.Sleep(measureTimeoutMin * time.Second)
		timeoutMux.Lock()
		isTimeout = true
		timeoutMux.Unlock()
	}()

	// Max timeout
	go func() {
		time.Sleep(measureTimeoutMax * time.Second)
		once.Do(stop)
	}()

	// Collect bytes
	var byteLen int64
	var byteMux sync.Mutex

	go func() {
		for length := range byteLenChan {
			byteMux.Lock()
			byteLen += length
			byteMux.Unlock()
		}
	}()

	// Measure per second
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var secondPass float64
	go func() {
		defer func() { _ = recover() }()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				secondPass++
				byteMux.Lock()
				avgKbps := float64(byteLen) * 8 / 1000 / secondPass
				byteMux.Unlock()

				select {
				case speedChan <- avgKbps:
				case <-done:
					return
				}
			}
		}
	}()

	// Start downloading from URLs
	var wg sync.WaitGroup
	for i, url := range urls {
		wg.Add(1)
		go func(index int, downloadURL string) {
			defer wg.Done()

			for {
				timeoutMux.Lock()
				timeout := isTimeout
				timeoutMux.Unlock()

				if timeout {
					break
				}

				err := c.download(downloadURL, byteLenChan, done)
				if err != nil {
					break
				}
			}
		}(i, url)
	}

	wg.Wait()
	once.Do(stop)

	return nil
}

// download downloads data from a URL and reports byte counts
func (c *Client) download(url string, byteLenChan chan<- int64, done <-chan struct{}) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return ErrInternet
	}
	defer func() { _ = resp.Body.Close() }()

	buf := make([]byte, bufferSize)

	for {
		select {
		case <-done:
			return nil
		default:
			n, err := resp.Body.Read(buf)
			if n > 0 {
				byteLenChan <- int64(n)
			}
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
		}
	}
}

// MeasureUpload measures upload speed (implements Provider interface)
func (c *Client) MeasureUpload(duration time.Duration, speedChan chan<- float64) error {
	urls, err := c.GetURLs(0)
	if err != nil {
		return err
	}
	if len(urls) == 0 {
		return errors.New("no URLs provided")
	}

	// Use a subset of URLs for upload
	uploadURLs := urls
	if len(uploadURLs) > 3 {
		uploadURLs = uploadURLs[:3]
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

	// Create random data to upload (1MB chunks)
	chunkSize := 1024 * 1024
	data := make([]byte, chunkSize)
	if _, err := rand.Read(data); err != nil {
		return err
	}

	// Upload to multiple URLs concurrently
	for _, url := range uploadURLs {
		wg.Add(1)
		go func(uploadURL string) {
			defer wg.Done()

			for {
				select {
				case <-done:
					return
				default:
					uploaded := c.uploadChunk(uploadURL, data)
					if uploaded > 0 {
						byteMux.Lock()
						totalBytes += uploaded
						byteMux.Unlock()
					}
				}
			}
		}(url)
	}

	wg.Wait()
	close(speedChan)

	return nil
}

// uploadChunk uploads a chunk of data to a URL
func (c *Client) uploadChunk(url string, data []byte) int64 {
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

	// Discard response body
	_, _ = io.Copy(io.Discard, resp.Body)

	return int64(len(data))
}

// MeasureLatency measures the round-trip time to fast.com (implements Provider interface)
func (c *Client) MeasureLatency() (time.Duration, error) {
	start := time.Now()
	resp, err := c.client.Get(endpoint)
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
		resp, err := c.client.Get(endpoint)
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
