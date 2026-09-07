// Package librespeed implements the provider.Provider interface against the
// LibreSpeed (https://librespeed.org) speed test protocol.
//
// LibreSpeed does not have a single fixed server: a public list of
// community-run backends is published at serversListURL and clients are
// expected to discover servers, probe them for liveness/latency, and pick
// the best one for a given run. This file reimplements that wire protocol
// (server discovery, liveness ping, chunked download/upload, ping/jitter)
// against public LibreSpeed backends -- it does not vendor or copy any
// LibreSpeed source code (LibreSpeed is LGPL-3.0 licensed).
package librespeed

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/0x524a/netpulse/internal/providers/provider"
)

// Compile-time assertion that Client satisfies the shared provider interface.
var _ provider.Provider = (*Client)(nil)

const (
	userAgent = "netpulse/1.0"

	// pingWorkers matches the concurrency the official librespeed-cli uses
	// when probing candidate servers during selection.
	pingWorkers = 10
	pingTimeout = 5 * time.Second
	// initTimeout bounds server discovery plus the full liveness/latency
	// probe pass across every candidate server.
	initTimeout = 30 * time.Second

	// downloadStreams/uploadStreams mirror the official client's defaults:
	// several concurrent HTTP streams, each immediately reopened when it
	// completes, running for a fixed wall-clock duration.
	downloadStreams = 3
	uploadStreams   = 3

	// downloadChunkSizeMB is the ckSize (in MB) requested from dlURL. Valid
	// range per the LibreSpeed backend protocol is 4-1024.
	downloadChunkSizeMB = 20
	minChunkSizeMB      = 4
	maxChunkSizeMB      = 1024

	// uploadChunkKiB is the size (in KiB) of each generated upload payload,
	// matching the official client's default of 1024 KiB per request.
	uploadChunkKiB = 1024

	// streamStaggerDelay is the delay between starting successive
	// concurrent streams, matching the official client's ~200ms stagger.
	streamStaggerDelay = 200 * time.Millisecond
)

// downloadTestDuration is the fixed wall-clock duration the download test
// runs for, matching the official client's default of 15s. It's a var
// (rather than a const) purely so tests can shrink it; production code
// never mutates it.
//
// TODO(wave2): this should be sourced from user/config-supplied test
// duration rather than a fixed internal value.
var downloadTestDuration = 15 * time.Second

// serversListURL is the public LibreSpeed server directory. It is a var
// (not a const) so tests can point it at an httptest server.
var serversListURL = "https://librespeed.org/backend-servers/servers.php"

var (
	// ErrNoServers is returned when no LibreSpeed server could be
	// discovered, or none of the discovered servers responded as live.
	ErrNoServers = errors.New("no servers available")
	// ErrInternet is returned when a network-level error prevents talking
	// to LibreSpeed's infrastructure at all.
	ErrInternet = errors.New("internet error. Please try again later")
)

// Server represents a single LibreSpeed backend, as returned by the public
// server list (servers.php) or its .well-known fallback.
//
// The list returns dlURL/ulURL/pingURL/getIpURL as paths *relative* to the
// server's base URL, not absolute URLs -- resolve() joins them.
type Server struct {
	Name        string `json:"name"`
	Base        string `json:"server"`
	DLPath      string `json:"dlURL"`
	ULPath      string `json:"ulURL"`
	PingPath    string `json:"pingURL"`
	GetIPPath   string `json:"getIpURL"`
	SponsorName string `json:"sponsorName"`
	SponsorURL  string `json:"sponsorURL"`
	ID          int    `json:"id"`

	// Resolved absolute URLs, populated by resolve() after JSON decoding.
	DownloadURL string `json:"-"`
	UploadURL   string `json:"-"`
	PingURL     string `json:"-"`
	GetIPURL    string `json:"-"`

	// Ping is the measured round-trip latency to this server, populated
	// during server selection in Init.
	Ping time.Duration `json:"-"`
}

// resolve computes the absolute Download/Upload/Ping/GetIP URLs from the
// server's base URL and the relative paths reported by the server list.
//
// The join is deliberately a plain string concatenation (base + "/" + path)
// rather than RFC 3986 relative-reference resolution: server list entries
// use inconsistent trailing slashes on "server" (some end in "/", some
// don't), and RFC 3986 resolution would treat a base like
// "https://host/backend" (no trailing slash) as a *file* named "backend",
// dropping it when resolving "garbage.php" against it -- which breaks
// against real-world entries that expect "backend" to remain part of the
// path.
func (s *Server) resolve() error {
	base := strings.TrimRight(strings.TrimSpace(s.Base), "/")
	if base == "" {
		return fmt.Errorf("librespeed: server %q has empty base URL", s.Name)
	}
	if _, err := url.Parse(base); err != nil {
		return fmt.Errorf("librespeed: server %q has invalid base URL: %w", s.Name, err)
	}

	join := func(p string) string {
		p = strings.TrimSpace(p)
		if p == "" {
			return ""
		}
		return base + "/" + strings.TrimLeft(p, "/")
	}

	s.DownloadURL = join(s.DLPath)
	s.UploadURL = join(s.ULPath)
	s.PingURL = join(s.PingPath)
	s.GetIPURL = join(s.GetIPPath)

	if s.DownloadURL == "" || s.UploadURL == "" || s.PingURL == "" {
		return fmt.Errorf("librespeed: server %q is missing a required URL", s.Name)
	}
	return nil
}

// ISPInfo holds the (best-effort) result of a server's optional getIpURL
// lookup.
type ISPInfo struct {
	ProcessedString string
	RawISPInfo      string
}

// Client represents a LibreSpeed speed test client.
type Client struct {
	server *Server
	client *http.Client
}

// New creates a new LibreSpeed client.
func New() *Client {
	return &Client{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Name returns the provider name.
func (c *Client) Name() string {
	return "LibreSpeed"
}

// IsAvailable checks if the LibreSpeed server directory is reachable.
func (c *Client) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serversListURL, nil)
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

// Init initializes the client by discovering LibreSpeed servers and
// selecting the one with the lowest latency.
func (c *Client) Init() error {
	ctx, cancel := context.WithTimeout(context.Background(), initTimeout)
	defer cancel()

	servers, err := c.discoverServers(ctx)
	if err != nil {
		return err
	}
	if len(servers) == 0 {
		return ErrNoServers
	}

	best, err := c.selectBestServer(ctx, servers)
	if err != nil {
		return err
	}

	c.server = best
	return nil
}

// discoverServers fetches the public LibreSpeed server list, falling back
// to the documented .well-known/librespeed endpoint at the list URL's
// origin if the primary list can't be fetched.
func (c *Client) discoverServers(ctx context.Context) ([]*Server, error) {
	servers, primaryErr := c.fetchServerList(ctx, serversListURL)
	if primaryErr == nil && len(servers) > 0 {
		return servers, nil
	}

	fallbackURL, err := wellKnownFallbackURL(serversListURL)
	if err != nil {
		if primaryErr != nil {
			return nil, primaryErr
		}
		return nil, ErrNoServers
	}

	servers, fallbackErr := c.fetchServerList(ctx, fallbackURL)
	if fallbackErr != nil {
		if primaryErr != nil {
			return nil, fmt.Errorf("librespeed: primary server list failed (%v), fallback also failed: %w", primaryErr, fallbackErr)
		}
		return nil, fallbackErr
	}
	return servers, nil
}

// wellKnownFallbackURL builds the documented fallback discovery URL:
// <scheme>://<host>/.well-known/librespeed, relative to the origin of the
// primary list URL.
func wellKnownFallbackURL(listURL string) (string, error) {
	u, err := url.Parse(listURL)
	if err != nil {
		return "", err
	}
	fallback := &url.URL{Scheme: u.Scheme, Host: u.Host, Path: "/.well-known/librespeed"}
	return fallback.String(), nil
}

// fetchServerList fetches and decodes a LibreSpeed server list from the
// given URL, resolving each entry's relative URLs to absolute ones.
// Individual malformed entries are skipped rather than failing the whole
// fetch.
func (c *Client) fetchServerList(ctx context.Context, listURL string) ([]*Server, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInternet, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("librespeed: server list request to %s returned status %d", listURL, resp.StatusCode)
	}

	var raw []Server
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("librespeed: decoding server list from %s: %w", listURL, err)
	}

	servers := make([]*Server, 0, len(raw))
	for i := range raw {
		s := raw[i]
		if err := s.resolve(); err != nil {
			continue
		}
		servers = append(servers, &s)
	}
	return servers, nil
}

// selectBestServer pings every candidate server with bounded concurrency
// and returns the live server with the lowest latency.
func (c *Client) selectBestServer(ctx context.Context, servers []*Server) (*Server, error) {
	type probeResult struct {
		server  *Server
		latency time.Duration
		up      bool
	}

	jobs := make(chan *Server)
	results := make(chan probeResult, len(servers))

	workers := pingWorkers
	if workers > len(servers) {
		workers = len(servers)
	}

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for s := range jobs {
				latency, up := c.checkServer(ctx, s)
				results <- probeResult{server: s, latency: latency, up: up}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, s := range servers {
			select {
			case jobs <- s:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	var best *Server
	var bestLatency time.Duration
	for r := range results {
		if !r.up {
			continue
		}
		r.server.Ping = r.latency
		if best == nil || r.latency < bestLatency {
			best = r.server
			bestLatency = r.latency
		}
	}

	if best == nil {
		return nil, ErrNoServers
	}
	return best, nil
}

// checkServer probes a single server's pingURL. Per the LibreSpeed
// protocol, a server counts as "up" only if it responds with an empty body
// and exactly HTTP 200; anything else (including a non-empty body) counts
// as down.
func (c *Client) checkServer(ctx context.Context, s *Server) (time.Duration, bool) {
	reqCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, s.PingURL, nil)
	if err != nil {
		return 0, false
	}
	req.Header.Set("User-Agent", userAgent)

	start := time.Now()
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, false
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	latency := time.Since(start)
	if err != nil {
		return 0, false
	}

	if resp.StatusCode != http.StatusOK || len(body) != 0 {
		return 0, false
	}
	return latency, true
}

// byteCounter is an io.Writer that counts the number of bytes written to
// it, used as the sink side of an io.TeeReader so we know exactly how many
// response/request bytes were actually transferred.
type byteCounter struct {
	n int64
}

func (b *byteCounter) Write(p []byte) (int, error) {
	b.n += int64(len(p))
	return len(p), nil
}

// clampChunkSizeMB clamps a ckSize value (in MB) to the valid LibreSpeed
// backend range of 4-1024.
func clampChunkSizeMB(mb int) int {
	if mb < minChunkSizeMB {
		return minChunkSizeMB
	}
	if mb > maxChunkSizeMB {
		return maxChunkSizeMB
	}
	return mb
}

// runThroughput drives a fixed-duration, multi-stream throughput test
// (shared by MeasureDownload and MeasureUpload). It launches numStreams
// goroutines, staggered by streamStaggerDelay, each of which calls doOnce
// in a tight loop -- reopening a new request as soon as the previous one
// completes -- until ctx is done. Aggregate throughput is reported on
// speedChan roughly once per second.
func (c *Client) runThroughput(ctx context.Context, speedChan chan<- float64, numStreams int, doOnce func(context.Context) int64) {
	var total atomic.Int64
	var lastTotal int64
	lastTime := time.Now()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	reporterDone := make(chan struct{})
	go func() {
		defer close(reporterDone)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				now := time.Now()
				cur := total.Load()
				elapsed := now.Sub(lastTime).Seconds()
				diff := cur - lastTotal
				if elapsed > 0 && diff > 0 {
					kbps := float64(diff*8) / 1000 / elapsed
					select {
					case speedChan <- kbps:
					case <-ctx.Done():
						return
					}
				}
				lastTotal = cur
				lastTime = now
			}
		}
	}()

	var wg sync.WaitGroup
	for i := 0; i < numStreams; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			select {
			case <-time.After(time.Duration(idx) * streamStaggerDelay):
			case <-ctx.Done():
				return
			}

			for ctx.Err() == nil {
				if n := doOnce(ctx); n > 0 {
					total.Add(n)
				}
			}
		}(i)
	}

	wg.Wait()
	<-reporterDone
}

// MeasureDownload measures download speed (implements provider.Provider).
func (c *Client) MeasureDownload(speedChan chan<- float64) error {
	if c.server == nil {
		return ErrNoServers
	}

	// TODO(wave2): downloadTestDuration should be sourced from
	// user/config-supplied test duration rather than a fixed constant.
	ctx, cancel := context.WithTimeout(context.Background(), downloadTestDuration)
	defer cancel()

	c.runThroughput(ctx, speedChan, downloadStreams, c.downloadOnce)
	return nil
}

// downloadOnce performs a single GET against the server's download
// endpoint and returns the number of body bytes actually received.
func (c *Client) downloadOnce(ctx context.Context) int64 {
	ckSize := clampChunkSizeMB(downloadChunkSizeMB)
	reqURL := fmt.Sprintf("%s?ckSize=%d", c.server.DownloadURL, ckSize)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0
	}
	// Required so the byte counts we measure are actual wire bytes, not a
	// smaller compressed transfer that Go's client would otherwise
	// transparently negotiate and inflate for us.
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return 0
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return 0
	}

	counter := &byteCounter{}
	_, _ = io.Copy(io.Discard, io.TeeReader(resp.Body, counter))
	return counter.n
}

// MeasureUpload measures upload speed (implements provider.Provider).
func (c *Client) MeasureUpload(duration time.Duration, speedChan chan<- float64) error {
	if c.server == nil {
		return ErrNoServers
	}

	blob := make([]byte, uploadChunkKiB*1024)
	if _, err := rand.Read(blob); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	c.runThroughput(ctx, speedChan, uploadStreams, func(ctx context.Context) int64 {
		return c.uploadOnce(ctx, blob)
	})
	return nil
}

// uploadOnce POSTs a single fixed-size payload to the server's upload
// endpoint and returns the number of payload bytes sent, or 0 if the
// request failed or the server responded with a non-2xx status.
func (c *Client) uploadOnce(ctx context.Context, blob []byte) int64 {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.server.UploadURL, bytes.NewReader(blob))
	if err != nil {
		return 0
	}
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("User-Agent", userAgent)
	// MUST be set explicitly and exactly: some real LibreSpeed backends
	// (e.g. librespeed-rs) never decode a chunked request body. If Go
	// can't infer the length up front (e.g. because the body were wrapped
	// in something like a TeeReader), net/http silently falls back to
	// Transfer-Encoding: chunked, and such backends then hang forever
	// waiting for an EOF that never satisfies their fixed-length read --
	// stalling the upload after exactly one payload. See
	// librespeed/speedtest-cli#122.
	req.ContentLength = int64(len(blob))

	resp, err := c.client.Do(req)
	if err != nil {
		return 0
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0
	}
	return int64(len(blob))
}

// MeasureLatency measures the round-trip time to the selected server's
// ping endpoint (implements provider.Provider).
func (c *Client) MeasureLatency() (time.Duration, error) {
	if c.server == nil {
		return 0, ErrNoServers
	}

	latency, up := c.checkServer(context.Background(), c.server)
	if !up {
		return 0, ErrInternet
	}
	return latency, nil
}

// MeasureJitter measures latency variation using LibreSpeed's own
// exponentially-weighted moving average, discarding the first ping sample
// (implements provider.Provider).
func (c *Client) MeasureJitter(samples int) (time.Duration, error) {
	if c.server == nil {
		return 0, ErrNoServers
	}
	if samples <= 1 {
		samples = 10
	}

	latencies := make([]time.Duration, 0, samples)
	for i := 0; i < samples+1; i++ {
		latency, up := c.checkServer(context.Background(), c.server)
		if i == 0 {
			// Discard the first sample: it includes connection setup
			// overhead and isn't representative.
			continue
		}
		if !up {
			continue
		}
		latencies = append(latencies, latency)
	}

	if len(latencies) < 2 {
		return 0, nil
	}

	var jitterMs float64
	prevMs := durationMs(latencies[0])
	for i := 1; i < len(latencies); i++ {
		curMs := durationMs(latencies[i])
		instJitter := math.Abs(curMs - prevMs)

		if jitterMs > instJitter {
			jitterMs = jitterMs*0.7 + instJitter*0.3
		} else {
			jitterMs = instJitter*0.2 + jitterMs*0.8
		}
		prevMs = curMs
	}

	return time.Duration(jitterMs * float64(time.Millisecond)), nil
}

func durationMs(d time.Duration) float64 {
	return float64(d.Microseconds()) / 1000
}

// GetISPInfo performs the optional getIP lookup against the selected
// server, returning best-effort ISP/network info. LibreSpeed backends do
// not guarantee a rigid response schema, so malformed/unexpected JSON is
// degraded to a raw-string result rather than treated as an error.
func (c *Client) GetISPInfo(distanceUnit string) (*ISPInfo, error) {
	if c.server == nil {
		return nil, ErrNoServers
	}
	if c.server.GetIPURL == "" {
		return nil, errors.New("librespeed: selected server does not expose a getIP endpoint")
	}
	if distanceUnit == "" {
		distanceUnit = "km"
	}

	reqURL := fmt.Sprintf("%s?isp=true&distance=%s", c.server.GetIPURL, url.QueryEscape(distanceUnit))
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	info := &ISPInfo{}

	var parsed struct {
		ProcessedString string          `json:"processedString"`
		RawISPInfo      json.RawMessage `json:"rawIspInfo"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil {
		info.ProcessedString = parsed.ProcessedString
		if len(parsed.RawISPInfo) > 0 && string(parsed.RawISPInfo) != "null" {
			info.RawISPInfo = string(parsed.RawISPInfo)
		}
		return info, nil
	}

	// Malformed/unexpected JSON: fall back to the raw body so callers
	// still get something usable instead of a hard failure.
	info.ProcessedString = strings.TrimSpace(string(body))
	return info, nil
}
