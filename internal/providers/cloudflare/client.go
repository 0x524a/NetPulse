package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/0x524a/netpulse/internal/providers/provider"
)

const (
	// defaultBaseURL is Cloudflare's public speed test edge. It is kept
	// separate from measurement/upload logic so tests can point a Client at
	// an httptest server instead.
	defaultBaseURL = "https://speed.cloudflare.com"

	userAgent = "netpulse/1.0"

	// estimatedHeaderFraction mirrors the ESTIMATED_HEADER_FRACTION constant
	// used by Cloudflare's own browser-based speed test client. A browser can
	// read exact byte counts off PerformanceResourceTiming, including
	// transport framing; a plain net/http client can only see payload bytes,
	// so this factor approximates the TCP/TLS/HTTP overhead that would
	// otherwise be invisible to us and understate real throughput.
	estimatedHeaderFraction = 1.005

	// concurrency is the number of simultaneous connections used for both
	// download and upload measurement, matching the original implementation.
	concurrency = 4
)

// downloadTestDuration bounds how long MeasureDownload's sampling loop
// runs.
// TODO(wave2): this should be sourced from config; MeasureDownload's
// signature (shared across all providers) has no duration parameter yet.
// It is a var rather than a const so tests can shorten it instead of
// spending real wall-clock seconds on every run.
var downloadTestDuration = 15 * time.Second

// downloadSizes is the payload-size schedule used by Cloudflare's official
// speed test client for downloads. The leading 0-byte entry is not a
// throughput sample -- it is used as a latency probe (see probe below); the
// remaining sizes are used to characterize download throughput.
var downloadSizes = []int{0, 100_000, 1_000_000, 10_000_000, 25_000_000, 100_000_000}

// uploadSizes is a deliberately smaller schedule than downloadSizes. Upload
// measurement is bounded by a caller-supplied duration rather than a fixed
// sample count, so scheduling the largest download sizes (up to 100MB) here
// could dominate a short test window on a slow link. 10MB is already enough
// to saturate most consumer connections for a useful sample.
var uploadSizes = []int{100_000, 1_000_000, 10_000_000}

var (
	// ErrNotAvailable indicates the Cloudflare speed test edge could not be
	// reached.
	ErrNotAvailable = errors.New("cloudflare speed test not available")
)

// reqDurRe matches the request-duration metric Cloudflare's edge reports in
// the Server-Timing header, e.g. "cfRequestDuration;dur=12.3" or the
// shortened "cfReqDur;dur=12.3".
var reqDurRe = regexp.MustCompile(`cfReq(?:uest)?Dur(?:ation)?;\s*dur=([0-9.]+)`)

// speedDurRe matches the family of cfSpeed* metrics Cloudflare's edge
// reports in the Server-Timing header when a request-duration metric is not
// present, e.g. "cfSpeedBrew;dur=1.2".
var speedDurRe = regexp.MustCompile(`cfSpeed[a-zA-Z]*;\s*dur=([0-9.]+)`)

// Client implements provider.Provider against Cloudflare's public speed
// test edge (speed.cloudflare.com).
type Client struct {
	client  *http.Client
	baseURL string
}

var _ provider.Provider = (*Client)(nil)

func New() *Client {
	return &Client{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		baseURL: defaultBaseURL,
	}
}

// newWithBaseURL builds a Client pointed at an arbitrary base URL. It exists
// so tests can exercise the full request/response handling against an
// httptest.Server instead of the live Cloudflare edge.
func newWithBaseURL(base string) *Client {
	c := New()
	c.baseURL = base
	return c
}

func (c *Client) Name() string {
	return "Cloudflare"
}

func (c *Client) IsAvailable() bool {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/cdn-cgi/trace", nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode == http.StatusOK
}

func (c *Client) Init() error {
	return nil
}

// downloadURL builds a GET /__down request URL for n bytes.
func (c *Client) downloadURL(n int) string {
	return fmt.Sprintf("%s/__down?bytes=%d", c.baseURL, n)
}

// uploadURL builds a POST /__up request URL for n bytes.
func (c *Client) uploadURL(n int) string {
	return fmt.Sprintf("%s/__up?bytes=%d", c.baseURL, n)
}

// serverProcessingTime extracts the server-side processing duration from a
// Cloudflare Server-Timing header, mirroring the parsing used by
// Cloudflare's own speed test client: prefer the explicit request-duration
// metric and, failing that, sum every cfSpeed* metric present. The second
// return value reports whether any duration could be extracted.
func serverProcessingTime(header string) (time.Duration, bool) {
	if header == "" {
		return 0, false
	}

	if m := reqDurRe.FindStringSubmatch(header); m != nil {
		if ms, err := strconv.ParseFloat(m[1], 64); err == nil {
			return time.Duration(ms * float64(time.Millisecond)), true
		}
	}

	matches := speedDurRe.FindAllStringSubmatch(header, -1)
	if len(matches) == 0 {
		return 0, false
	}

	var totalMs float64
	found := false
	for _, m := range matches {
		ms, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			continue
		}
		totalMs += ms
		found = true
	}
	if !found {
		return 0, false
	}
	return time.Duration(totalMs * float64(time.Millisecond)), true
}

// probe issues a zero-byte GET /__down request and returns the observed
// round-trip time, using httptrace to capture time-to-first-byte (Go has no
// equivalent of the browser's PerformanceResourceTiming). When the response
// carries a Server-Timing header, the server-side processing component is
// subtracted so the result approximates network time rather than
// network+compute time, the same refinement Cloudflare's own client applies.
func (c *Client) probe(ctx context.Context) (time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.downloadURL(0), nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", userAgent)

	var firstByte time.Time
	start := time.Now()
	trace := &httptrace.ClientTrace{
		GotFirstResponseByte: func() { firstByte = time.Now() },
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("cloudflare: latency probe returned unexpected status %s", resp.Status)
	}

	rtt := time.Since(start)
	if !firstByte.IsZero() {
		rtt = firstByte.Sub(start)
	}

	if serverDur, ok := serverProcessingTime(resp.Header.Get("Server-Timing")); ok && serverDur > 0 && serverDur < rtt {
		rtt -= serverDur
	}

	return rtt, nil
}

func (c *Client) MeasureLatency() (time.Duration, error) {
	const attempts = 3

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var total time.Duration
	var ok int
	var lastErr error

	for i := 0; i < attempts; i++ {
		rtt, err := c.probe(ctx)
		if err != nil {
			lastErr = err
			continue
		}
		total += rtt
		ok++

		select {
		case <-time.After(100 * time.Millisecond):
		case <-ctx.Done():
		}
	}

	if ok == 0 {
		if lastErr != nil {
			return 0, lastErr
		}
		return 0, ErrNotAvailable
	}

	return total / time.Duration(ok), nil
}

func (c *Client) MeasureJitter(samples int) (time.Duration, error) {
	if samples <= 1 {
		samples = 10
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(samples)*2*time.Second+5*time.Second)
	defer cancel()

	var measurements []time.Duration
	for i := 0; i < samples; i++ {
		rtt, err := c.probe(ctx)
		if err != nil {
			continue
		}
		measurements = append(measurements, rtt)
	}

	if len(measurements) <= 1 {
		return 0, nil
	}

	var sum time.Duration
	for _, m := range measurements {
		sum += m
	}
	mean := sum / time.Duration(len(measurements))

	var totalDeviation time.Duration
	for _, m := range measurements {
		diff := m - mean
		if diff < 0 {
			diff = -diff
		}
		totalDeviation += diff
	}

	return totalDeviation / time.Duration(len(measurements)), nil
}

// fetchDownload issues a single GET /__down?bytes=n request, fully draining
// the body before returning. It validates the response status explicitly:
// a non-2xx response contributes zero bytes to the caller's throughput
// accounting rather than being counted as a successful transfer.
func (c *Client) fetchDownload(ctx context.Context, n int) (bytesRead int64, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.downloadURL(n), nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return 0, fmt.Errorf("cloudflare: download returned unexpected status %s", resp.Status)
	}

	read, copyErr := io.Copy(io.Discard, resp.Body)
	if copyErr != nil {
		return read, copyErr
	}
	return read, nil
}

// fetchUpload issues a single POST /__up?bytes=n request whose body is n
// ASCII '0' characters, matching Cloudflare's own client rather than random
// bytes. The body is a *strings.Reader of fixed, known length and
// ContentLength is also set explicitly: net/http only omits
// Transfer-Encoding: chunked when it can determine the body length up
// front, and some backends (this one included, historically) handle a
// chunked upload badly. The response status is validated explicitly -- a
// non-2xx response contributes zero bytes to the caller's throughput
// accounting, which is the behavior this whole rewrite exists to fix.
//
// http.Client.Do does not return until the entire request (headers + body)
// has been written and the response headers have been read, so bracketing
// elapsed time around the Do call satisfies "fully write before stopping
// the clock" without needing a custom body-writing wrapper.
func (c *Client) fetchUpload(ctx context.Context, n int) (bytesSent int64, err error) {
	body := strings.NewReader(strings.Repeat("0", n))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.uploadURL(n), body)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "text/plain")
	req.ContentLength = int64(n)

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("cloudflare: upload returned unexpected status %s", resp.Status)
	}

	return int64(n), nil
}

// bitsPerSecond converts a byte count and elapsed seconds into a
// bits-per-second figure using the same formula as Cloudflare's official
// client: bits = 8 * bytes * ESTIMATED_HEADER_FRACTION, bps = bits /
// seconds. See estimatedHeaderFraction's doc comment for why the constant
// is applied.
func bitsPerSecond(bytesN int64, seconds float64) float64 {
	if seconds <= 0 {
		return 0
	}
	bits := 8 * float64(bytesN) * estimatedHeaderFraction
	return bits / seconds
}

func (c *Client) MeasureDownload(speedChan chan<- float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), downloadTestDuration)
	defer cancel()

	var totalBytes int64
	var byteMux sync.Mutex

	tickerDone := make(chan struct{})
	go func() {
		defer close(tickerDone)

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		lastBytes := int64(0)
		lastTime := time.Now()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				byteMux.Lock()
				current := totalBytes
				byteMux.Unlock()

				now := time.Now()
				elapsed := now.Sub(lastTime).Seconds()
				diff := current - lastBytes

				if elapsed > 0 && diff > 0 {
					speedKbps := bitsPerSecond(diff, elapsed) / 1000.0
					select {
					case speedChan <- speedKbps:
					case <-ctx.Done():
						return
					}
				}

				lastBytes = current
				lastTime = now
			}
		}
	}()

	sizes := downloadSizes[1:] // skip the 0-byte latency probe entry
	var wg sync.WaitGroup
	errCh := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()

			idx := worker % len(sizes)
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				n, err := c.fetchDownload(ctx, sizes[idx])
				if err != nil {
					if ctx.Err() != nil {
						// Context expired mid-request; this is expected
						// shutdown, not a measurement failure.
						return
					}
					select {
					case errCh <- err:
					default:
					}
					return
				}

				byteMux.Lock()
				totalBytes += n
				byteMux.Unlock()

				idx = (idx + 1) % len(sizes)
			}
		}(i)
	}

	wg.Wait()
	cancel()
	<-tickerDone

	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

func (c *Client) MeasureUpload(duration time.Duration, speedChan chan<- float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	var totalBytes int64
	var byteMux sync.Mutex
	startTime := time.Now()

	tickerDone := make(chan struct{})
	go func() {
		defer close(tickerDone)

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				byteMux.Lock()
				current := totalBytes
				byteMux.Unlock()

				elapsed := time.Since(startTime).Seconds()
				if elapsed > 0 {
					speedKbps := bitsPerSecond(current, elapsed) / 1000.0
					select {
					case speedChan <- speedKbps:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	var wg sync.WaitGroup
	errCh := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()

			idx := worker % len(uploadSizes)
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				n, err := c.fetchUpload(ctx, uploadSizes[idx])
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					select {
					case errCh <- err:
					default:
					}
					return
				}

				byteMux.Lock()
				totalBytes += n
				byteMux.Unlock()

				idx = (idx + 1) % len(uploadSizes)
			}
		}(i)
	}

	wg.Wait()
	cancel()
	<-tickerDone

	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

// EdgeMetadata holds informational fields describing the Cloudflare edge
// that served a request. It is not part of the provider.Provider interface;
// it exists purely to enrich CLI output with details like which colo/city
// handled the test, when available.
type EdgeMetadata struct {
	Colo string
	City string
	IP   string
	Raw  map[string]string
}

// FetchEdgeMetadata retrieves edge metadata for the current connection.
// /meta is undocumented, historically returned JSON with colo/city, and is
// unconfirmed to still be live, so it is only trusted if it actually
// responds 200; otherwise this falls back to the confirmed-live
// /cdn-cgi/trace endpoint, which returns plaintext key=value lines (fl, h,
// ip, ts, colo, ...).
func (c *Client) FetchEdgeMetadata(ctx context.Context) (*EdgeMetadata, error) {
	if meta, err := c.fetchMeta(ctx); err == nil {
		return meta, nil
	}
	return c.fetchTrace(ctx)
}

func (c *Client) fetchMeta(ctx context.Context) (*EdgeMetadata, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/meta", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cloudflare: /meta returned unexpected status %s", resp.Status)
	}

	var payload struct {
		Colo string `json:"colo"`
		City string `json:"city"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	return &EdgeMetadata{Colo: payload.Colo, City: payload.City}, nil
}

func (c *Client) fetchTrace(ctx context.Context) (*EdgeMetadata, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/cdn-cgi/trace", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cloudflare: /cdn-cgi/trace returned unexpected status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	raw := make(map[string]string)
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		k, v, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		raw[k] = v
	}

	return &EdgeMetadata{Colo: raw["colo"], IP: raw["ip"], Raw: raw}, nil
}
