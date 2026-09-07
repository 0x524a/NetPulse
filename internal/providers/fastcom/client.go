// Package fastcom implements a speedtest provider backed by fast.com.
//
// fast.com genuinely performs token/endpoint discovery against Netflix's
// undocumented speedtest API (see Init and extractToken below) and measures
// real download throughput against the returned targets. It does NOT
// support upload measurement: fast.com's public API is download-only, and
// MeasureUpload reports that honestly via provider.ErrUploadNotSupported
// instead of measuring an unrelated third-party host and mislabeling the
// result.
package fastcom

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"golang.org/x/net/html"

	"github.com/0x524a/netpulse/internal/providers/provider"
)

const (
	endpoint   = "https://fast.com"
	bufferSize = 8192
	userAgent  = "netpulse/1.0"

	// downloadMeasureDuration is how long MeasureDownload keeps pulling data
	// before it stops and reports. This used to be two separate hardcoded
	// values (a 10s "minimum" and a 30s "hard cap"); consolidated to one.
	//
	// TODO(wave2): this should be a configurable value passed in by the
	// caller (like MeasureUpload's duration parameter), not a fixed
	// per-provider constant.
	downloadMeasureDuration = 10 * time.Second
)

var (
	ErrAPI      = errors.New("fast.com API error. Please try again later")
	ErrInternet = errors.New("internet error. Please try again later")

	// ErrTokenNotFound is returned when the speedtest auth token can't be
	// located in fast.com's served JS bundle. See extractToken.
	ErrTokenNotFound = errors.New("fastcom: could not locate speedtest token in fast.com bundle (site may have changed)")

	// ErrEndpointNotFound is returned when the API endpoint can't be located
	// in fast.com's served JS bundle.
	ErrEndpointNotFound = errors.New("fastcom: could not locate api endpoint in fast.com bundle (site may have changed)")

	reEndpoint = regexp.MustCompile(`apiEndpoint="([\w|\/|\.]*)"`)
	// reToken matches the `token:"<value>"` key/value pair as it appears in
	// fast.com's minified JS bundle. See extractToken for details on why
	// this is fragile.
	reToken = regexp.MustCompile(`token:"(\w*)"`)
	reCount = regexp.MustCompile(`urlCount:(\d*)`)
)

// Client represents a fast.com speed test client
type Client struct {
	token    string
	apiURL   string
	urlCount int
	client   *http.Client
}

var _ provider.Provider = (*Client)(nil)

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

// extractToken scrapes fast.com's speedtest auth token out of its served JS
// bundle.
//
// This token is NOT available from any documented endpoint or public API
// listing -- Netflix embeds it as a literal string inside their minified,
// client-side JS bundle (the file referenced by the <script src="/app-*.js">
// tag on https://fast.com). As last verified against the live bundle, it
// appears as an object literal member shaped like:
//
//	DEFAULT_PARAMS={https:!0,token:"YXNkZmFzZGxmbnNkYWZoYXNkZmhrYWxm",urlCount:3,...}
//
// i.e. the object key is literally `token`, followed by a double-quoted
// alphanumeric value: the pattern `token:"(\w*)"` (see reToken). Netflix has
// reshaped this bundle before with no notice and no changelog, and there is
// nothing to validate the extracted value against ahead of time -- if this
// stops matching, that means the bundle's shape changed, not that the
// network is down, so we return a specific error rather than a generic one.
func (c *Client) extractToken(jsData []byte) (string, error) {
	matches := reToken.FindSubmatch(jsData)
	if len(matches) < 2 || len(matches[1]) == 0 {
		return "", fmt.Errorf("%w: pattern `token:\"(\\w*)\"` had no match", ErrTokenNotFound)
	}
	return string(matches[1]), nil
}

// parseJSData extracts token, API endpoint, and URL count from JavaScript
func (c *Client) parseJSData(jsData []byte) error {
	// Extract API endpoint
	matches := reEndpoint.FindSubmatch(jsData)
	if len(matches) < 2 || len(matches[1]) == 0 {
		return fmt.Errorf("%w: pattern `apiEndpoint=\"...\"` had no match", ErrEndpointNotFound)
	}
	c.apiURL = "https://" + string(matches[1])

	// Extract token (isolated: see extractToken doc comment on why this is
	// the fragile part of discovery)
	token, err := c.extractToken(jsData)
	if err != nil {
		return err
	}
	c.token = token

	// Extract URL count. This one is not critical -- fall back to the
	// client's existing default if the pattern isn't found.
	matches = reCount.FindSubmatch(jsData)
	if len(matches) < 2 || len(matches[1]) == 0 {
		if c.urlCount <= 0 {
			c.urlCount = 5 // default
		}
	} else {
		count, err := strconv.Atoi(string(matches[1]))
		if err == nil && count > 0 {
			c.urlCount = count
		}
	}

	return nil
}

// GetURLs fetches the test URLs from the API. If count <= 0, the client's
// own discovered/default urlCount is used instead.
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: unexpected status %d from speedtest API", ErrAPI, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ErrInternet
	}

	// Parse JSON response to extract URLs
	urls, err := c.parseURLsFromJSON(body)
	if err != nil {
		return nil, err
	}

	return urls, nil
}

// parseURLsFromJSON extracts URLs from the JSON response
func (c *Client) parseURLsFromJSON(data []byte) ([]string, error) {
	// Simple JSON parsing for "url" fields
	reURL := regexp.MustCompile(`"url"\s*:\s*"([^"]*)"`)
	matches := reURL.FindAllSubmatch(data, -1)

	if len(matches) == 0 {
		return nil, fmt.Errorf("%w: no targets in response", ErrAPI)
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
	// Use the client's own discovered/default urlCount explicitly, rather
	// than relying on GetURLs's count<=0 "use the client's default" fallback
	// implicitly.
	urls, err := c.GetURLs(c.urlCount)
	if err != nil {
		return err
	}
	if len(urls) == 0 {
		return errors.New("no URLs provided")
	}

	done := make(chan struct{})
	var stopOnce sync.Once
	stop := func() { stopOnce.Do(func() { close(done) }) }

	// Single hard timeout for the whole measurement window.
	timer := time.AfterFunc(downloadMeasureDuration, stop)
	defer timer.Stop()

	byteLenChan := make(chan int64, 100)

	// Collect bytes
	var byteLen int64
	var byteMux sync.Mutex

	collectDone := make(chan struct{})
	go func() {
		defer close(collectDone)
		for length := range byteLenChan {
			byteMux.Lock()
			byteLen += length
			byteMux.Unlock()
		}
	}()

	// Measure per second
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var secondsPassed float64
	reportDone := make(chan struct{})
	go func() {
		defer close(reportDone)
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				secondsPassed++
				byteMux.Lock()
				avgKbps := float64(byteLen) * 8 / 1000 / secondsPassed
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
	for _, url := range urls {
		wg.Add(1)
		go func(downloadURL string) {
			defer wg.Done()

			for {
				select {
				case <-done:
					return
				default:
				}

				if err := c.download(downloadURL, byteLenChan, done); err != nil {
					return
				}
			}
		}(url)
	}

	wg.Wait()
	stop()

	// No more writers remain (all download goroutines have returned), so
	// it's safe to close byteLenChan and wait for the collector/reporter
	// goroutines to observe done and exit cleanly rather than leaking them.
	close(byteLenChan)
	<-collectDone
	<-reportDone

	return nil
}

// download downloads data from a URL and reports byte counts. Only bytes
// from a successful (2xx) response are counted; a non-2xx response (e.g. a
// 404/405 error page body) contributes zero bytes.
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w: download request returned status %d", ErrAPI, resp.StatusCode)
	}

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

// MeasureUpload measures upload speed (implements Provider interface).
//
// fast.com's public API is download-only: every URL returned by the
// speedtest API accepts GET and nothing else. There is no endpoint,
// documented or otherwise, that accepts uploaded data -- this was checked
// against several independent unofficial fast.com clients before concluding
// there simply isn't one. Rather than substitute an unrelated third-party
// host and mislabel its throughput as "Fast.com upload", this reports the
// absence honestly via provider.ErrUploadNotSupported.
func (c *Client) MeasureUpload(_ time.Duration, _ chan<- float64) error {
	return fmt.Errorf("fastcom: %w", provider.ErrUploadNotSupported)
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

// MeasureJitter measures latency variation as the mean absolute deviation
// of round-trip times, in full nanosecond precision (no truncation to
// whole milliseconds, so sub-millisecond jitter is still reported). Returns
// an error -- not a silent (0, nil) -- if fewer than 2 samples succeed,
// since that's "we couldn't measure this", not "jitter is zero".
func (c *Client) MeasureJitter(samples int) (time.Duration, error) {
	if samples <= 1 {
		samples = 10
	}

	measurements := make([]time.Duration, 0, samples)
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

	if len(measurements) < 2 {
		return 0, fmt.Errorf("fastcom: jitter requires at least 2 successful latency samples, got %d", len(measurements))
	}

	var sumLatency time.Duration
	for _, m := range measurements {
		sumLatency += m
	}
	meanLatency := sumLatency / time.Duration(len(measurements))

	var totalDeviation time.Duration
	for _, m := range measurements {
		diff := m - meanLatency
		if diff < 0 {
			diff = -diff
		}
		totalDeviation += diff
	}

	return totalDeviation / time.Duration(len(measurements)), nil
}
