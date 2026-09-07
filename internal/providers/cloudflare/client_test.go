package cloudflare

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	client := New()
	if client == nil {
		t.Fatal("New() returned nil")
	}
	if client.baseURL != defaultBaseURL {
		t.Errorf("New() baseURL = %q, want %q", client.baseURL, defaultBaseURL)
	}
}

func TestName(t *testing.T) {
	client := New()
	name := client.Name()
	if name == "" {
		t.Error("Name() should not return empty string")
	}
}

func TestInit(t *testing.T) {
	client := New()
	if err := client.Init(); err != nil {
		t.Errorf("Init() returned error: %v", err)
	}
}

// --- deterministic tests against httptest servers ---

// cloudflareHandler emulates just enough of speed.cloudflare.com for the
// client's request/response handling to be exercised: GET /__down?bytes=N
// returns exactly N bytes, POST /__up?bytes=N reads and discards the body
// and reports how many bytes it read via a response header.
func cloudflareHandler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/__down"):
			n := 0
			if v := r.URL.Query().Get("bytes"); v != "" {
				var err error
				n, err = parseIntForTest(v)
				if err != nil {
					http.Error(w, "bad bytes param", http.StatusBadRequest)
					return
				}
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(strings.Repeat("d", n)))
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/__up"):
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			t.Logf("upload handler received %d bytes", len(body))
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == "/cdn-cgi/trace":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("fl=1f1\nh=speed.cloudflare.com\nip=127.0.0.1\nts=1700000000.000\ncolo=TEST\n"))
		default:
			http.NotFound(w, r)
		}
	}
}

func parseIntForTest(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, io.ErrUnexpectedEOF
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

func TestFetchDownloadSuccess(t *testing.T) {
	server := httptest.NewServer(cloudflareHandler(t))
	defer server.Close()

	client := newWithBaseURL(server.URL)
	n, err := client.fetchDownload(context.Background(), 12345)
	if err != nil {
		t.Fatalf("fetchDownload() error: %v", err)
	}
	if n != 12345 {
		t.Errorf("fetchDownload() bytesRead = %d, want 12345", n)
	}
}

func TestFetchDownloadNonSuccessStatusCountsZeroBytes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a server that rejects the request but still writes a body.
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(strings.Repeat("x", 5000)))
	}))
	defer server.Close()

	client := newWithBaseURL(server.URL)
	n, err := client.fetchDownload(context.Background(), 5000)
	if err == nil {
		t.Fatal("fetchDownload() expected an error for a 404 response, got nil")
	}
	if n != 0 {
		t.Errorf("fetchDownload() on non-2xx status returned bytesRead = %d, want 0", n)
	}
}

func TestFetchUploadSuccess(t *testing.T) {
	var receivedBody []byte
	var receivedContentLength int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentLength = r.ContentLength
		body, _ := io.ReadAll(r.Body)
		receivedBody = body
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newWithBaseURL(server.URL)
	n, err := client.fetchUpload(context.Background(), 2048)
	if err != nil {
		t.Fatalf("fetchUpload() error: %v", err)
	}
	if n != 2048 {
		t.Errorf("fetchUpload() bytesSent = %d, want 2048", n)
	}
	if receivedContentLength != 2048 {
		t.Errorf("server observed Content-Length = %d, want 2048 (want explicit length, not chunked)", receivedContentLength)
	}
	if len(receivedBody) != 2048 {
		t.Errorf("server received %d bytes, want 2048", len(receivedBody))
	}
	if strings.Trim(string(receivedBody), "0") != "" {
		t.Errorf("upload body should be all ASCII '0' characters, got %q", string(receivedBody[:min(len(receivedBody), 32)]))
	}
}

// TestFetchUploadNonSuccessStatusCountsZeroBytes is the key regression test
// for the bug this rewrite fixes: the original client counted len(data) as
// bytes sent unconditionally, even when the server rejected the upload
// (404/405/413/500/...). A non-2xx response must contribute exactly zero
// bytes to the throughput measurement.
func TestFetchUploadNonSuccessStatusCountsZeroBytes(t *testing.T) {
	statuses := []int{http.StatusNotFound, http.StatusMethodNotAllowed, http.StatusRequestEntityTooLarge, http.StatusInternalServerError}

	for _, status := range statuses {
		status := status
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Drain the body like a real server would, then reject.
				_, _ = io.Copy(io.Discard, r.Body)
				w.WriteHeader(status)
			}))
			defer server.Close()

			client := newWithBaseURL(server.URL)
			n, err := client.fetchUpload(context.Background(), 4096)
			if err == nil {
				t.Fatalf("fetchUpload() expected an error for status %d, got nil", status)
			}
			if n != 0 {
				t.Errorf("fetchUpload() on status %d returned bytesSent = %d, want 0", status, n)
			}
		})
	}
}

func TestMeasureUploadWithMockServer(t *testing.T) {
	server := httptest.NewServer(cloudflareHandler(t))
	defer server.Close()

	client := newWithBaseURL(server.URL)
	speedChan := make(chan float64, 100)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range speedChan {
		}
	}()

	err := client.MeasureUpload(2*time.Second, speedChan)
	close(speedChan)
	<-done

	if err != nil {
		t.Errorf("MeasureUpload() error: %v", err)
	}
}

func TestMeasureUploadNonSuccessStatusReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := newWithBaseURL(server.URL)
	speedChan := make(chan float64, 100)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range speedChan {
		}
	}()

	err := client.MeasureUpload(1*time.Second, speedChan)
	close(speedChan)
	<-done

	if err == nil {
		t.Error("MeasureUpload() expected an error when every upload is rejected, got nil")
	}
}

func TestMeasureDownloadWithMockServer(t *testing.T) {
	server := httptest.NewServer(cloudflareHandler(t))
	defer server.Close()

	orig := downloadTestDuration
	downloadTestDuration = 2 * time.Second
	defer func() { downloadTestDuration = orig }()

	client := newWithBaseURL(server.URL)
	speedChan := make(chan float64, 100)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range speedChan {
		}
	}()

	err := client.MeasureDownload(speedChan)
	close(speedChan)
	<-done

	if err != nil {
		t.Errorf("MeasureDownload() error: %v", err)
	}
}

func TestProbeUsesServerTimingSubtraction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server-Timing", "cfRequestDuration;dur=5.0")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newWithBaseURL(server.URL)
	rtt, err := client.probe(context.Background())
	if err != nil {
		t.Fatalf("probe() error: %v", err)
	}
	if rtt < 0 {
		t.Errorf("probe() returned negative rtt: %v", rtt)
	}
}

func TestProbeNonSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	client := newWithBaseURL(server.URL)
	if _, err := client.probe(context.Background()); err == nil {
		t.Error("probe() expected an error for a 502 response, got nil")
	}
}

func TestFetchEdgeMetadataFallsBackToTrace(t *testing.T) {
	server := httptest.NewServer(cloudflareHandler(t))
	defer server.Close()

	client := newWithBaseURL(server.URL)
	meta, err := client.FetchEdgeMetadata(context.Background())
	if err != nil {
		t.Fatalf("FetchEdgeMetadata() error: %v", err)
	}
	if meta.Colo != "TEST" {
		t.Errorf("FetchEdgeMetadata() colo = %q, want %q", meta.Colo, "TEST")
	}
	if meta.IP != "127.0.0.1" {
		t.Errorf("FetchEdgeMetadata() ip = %q, want %q", meta.IP, "127.0.0.1")
	}
}

// --- unit tests with no network dependency ---

func TestServerProcessingTime(t *testing.T) {
	cases := []struct {
		name    string
		header  string
		wantOK  bool
		wantSec float64
	}{
		{"empty", "", false, 0},
		{"request duration long form", "cfRequestDuration;dur=12.5", true, 0.0125},
		{"request duration short form", "cfReqDur;dur=3.0", true, 0.003},
		{"speed metrics summed", "cfSpeedBrew;dur=1.0, cfSpeedKettle;dur=2.0", true, 0.003},
		{"unrelated metric", "cache;dur=1.0", false, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, ok := serverProcessingTime(tc.header)
			if ok != tc.wantOK {
				t.Fatalf("serverProcessingTime(%q) ok = %v, want %v", tc.header, ok, tc.wantOK)
			}
			if !ok {
				return
			}
			wantDur := time.Duration(tc.wantSec * float64(time.Second))
			if d != wantDur {
				t.Errorf("serverProcessingTime(%q) = %v, want %v", tc.header, d, wantDur)
			}
		})
	}
}

func TestBitsPerSecond(t *testing.T) {
	got := bitsPerSecond(1_000_000, 1.0)
	want := 8 * 1_000_000 * estimatedHeaderFraction
	if diff := got - want; diff > 1e-6 || diff < -1e-6 {
		t.Errorf("bitsPerSecond(1e6, 1.0) = %v, want %v", got, want)
	}

	if got := bitsPerSecond(1000, 0); got != 0 {
		t.Errorf("bitsPerSecond(1000, 0) = %v, want 0", got)
	}
}

func TestDownloadURLAndUploadURL(t *testing.T) {
	client := newWithBaseURL("https://example.test")
	if got, want := client.downloadURL(100), "https://example.test/__down?bytes=100"; got != want {
		t.Errorf("downloadURL(100) = %q, want %q", got, want)
	}
	if got, want := client.uploadURL(200), "https://example.test/__up?bytes=200"; got != want {
		t.Errorf("uploadURL(200) = %q, want %q", got, want)
	}
}

// --- live-network tests, skipped in short mode ---

func TestIsAvailableLive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live network test in short mode")
	}
	client := New()
	t.Logf("%s available: %v", client.Name(), client.IsAvailable())
}

func TestMeasureLatencyLive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live network test in short mode")
	}
	client := New()
	latency, err := client.MeasureLatency()
	if err != nil {
		t.Fatalf("MeasureLatency() error: %v", err)
	}
	if latency < 0 {
		t.Errorf("MeasureLatency() returned negative latency: %v", latency)
	}
	t.Logf("Measured latency: %v", latency)
}

func TestMeasureJitterLive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live network test in short mode")
	}
	client := New()
	jitter, err := client.MeasureJitter(5)
	if err != nil {
		t.Fatalf("MeasureJitter() error: %v", err)
	}
	t.Logf("Measured jitter: %v", jitter)
}

func TestFetchEdgeMetadataLive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live network test in short mode")
	}
	client := New()
	meta, err := client.FetchEdgeMetadata(context.Background())
	if err != nil {
		t.Fatalf("FetchEdgeMetadata() error: %v", err)
	}
	t.Logf("Edge metadata: %+v", meta)
}

// TestDownloadSizesRespectServerCap guards a bug found by running the real
// binary: /__down?bytes=100000000 returns 403 Forbidden from the live edge,
// and Cloudflare's published schedule ends at exactly that value. Any link
// fast enough to reach the last schedule entry failed the whole measurement.
func TestDownloadSizesRespectServerCap(t *testing.T) {
	for _, n := range downloadSizes {
		if n > maxDownloadBytes {
			t.Errorf("downloadSizes contains %d, above the server cap of %d; "+
				"the live endpoint answers 403 for such requests", n, maxDownloadBytes)
		}
	}
}

// TestDownloadURLClampsOversizedRequest ensures the clamp holds even if the
// schedule is later edited past the cap.
func TestDownloadURLClampsOversizedRequest(t *testing.T) {
	c := New()
	got := c.downloadURL(500_000_000)
	want := fmt.Sprintf("%s/__down?bytes=%d", c.baseURL, maxDownloadBytes)
	if got != want {
		t.Errorf("downloadURL(500000000) = %q, want clamped %q", got, want)
	}
}

// TestMeasureDownloadRateLimitedReportsErrorNotSpeed is the regression test
// for a bug introduced while fixing the 403: retrying a 429 inside the
// measurement loop made backoff sleeps count toward elapsed wall-clock while
// adding no bytes, so a rate-limited run reported ~16.88 Mbps (a single
// sample) on a link that measures ~800 Mbps. An understated number that looks
// real is worse than a failure, so a rate limit must surface as ErrRateLimited
// and no measurement at all.
func TestMeasureDownloadRateLimitedReportsErrorNotSpeed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	orig := downloadTestDuration
	downloadTestDuration = 2 * time.Second
	defer func() { downloadTestDuration = orig }()

	c := newWithBaseURL(srv.URL)
	speedChan := make(chan float64, 64)
	var samples []float64
	done := make(chan struct{})
	go func() {
		defer close(done)
		for s := range speedChan {
			samples = append(samples, s)
		}
	}()

	err := c.MeasureDownload(speedChan)
	close(speedChan)
	<-done

	if err == nil {
		t.Fatal("expected an error when every request is rate limited")
	}
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("error %v does not wrap ErrRateLimited", err)
	}
	for _, s := range samples {
		if s > 0 {
			t.Errorf("reported a non-zero speed sample (%v) despite being rate limited", s)
		}
	}
}

// TestClassifyTransferErrorDistinguishesTransientFromPermanent ensures the 403
// returned for an oversized request is not mislabeled as a rate limit.
func TestClassifyTransferErrorDistinguishesTransientFromPermanent(t *testing.T) {
	rateLimited := &statusError{op: "download", status: "429 Too Many Requests", code: 429}
	if got := classifyTransferError(rateLimited); !errors.Is(got, ErrRateLimited) {
		t.Errorf("429 classified as %v, want ErrRateLimited", got)
	}

	forbidden := &statusError{op: "download", status: "403 Forbidden", code: 403}
	if got := classifyTransferError(forbidden); errors.Is(got, ErrRateLimited) {
		t.Errorf("403 was classified as a rate limit; it is a permanent refusal")
	}

	serverErr := &statusError{op: "upload", status: "503 Service Unavailable", code: 503}
	if got := classifyTransferError(serverErr); !errors.Is(got, ErrRateLimited) {
		t.Errorf("503 classified as %v, want ErrRateLimited", got)
	}
}
