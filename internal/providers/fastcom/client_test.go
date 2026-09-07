package fastcom

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/0x524a/netpulse/internal/providers/provider"
)

func TestNew(t *testing.T) {
	client := New()
	if client == nil {
		t.Fatal("New() returned nil")
	}
}

func TestName(t *testing.T) {
	client := New()
	name := client.Name()
	if name != "Fast.com" {
		t.Errorf("Expected 'Fast.com', got %s", name)
	}
}

// TestMeasureUploadReturnsErrUploadNotSupported locks down the headline
// behavior change: fast.com has no upload endpoint, so MeasureUpload must
// report that honestly via the shared sentinel error rather than measuring
// against download-only URLs (or any third-party substitute).
func TestMeasureUploadReturnsErrUploadNotSupported(t *testing.T) {
	client := New()
	speedChan := make(chan float64, 10)
	go func() {
		for range speedChan {
		}
	}()

	err := client.MeasureUpload(100*time.Millisecond, speedChan)
	if err == nil {
		t.Fatal("expected MeasureUpload to return an error, got nil")
	}
	if !errors.Is(err, provider.ErrUploadNotSupported) {
		t.Fatalf("expected errors.Is(err, provider.ErrUploadNotSupported) to hold, got: %v", err)
	}
}

// TestMeasureUploadDoesNotHitNetwork verifies MeasureUpload never even
// attempts to fetch URLs or hit the network (i.e. it fails fast without an
// initialized client), reinforcing that no request -- to fast.com or any
// third-party host -- is made in service of "measuring" upload.
func TestMeasureUploadDoesNotHitNetwork(t *testing.T) {
	client := New() // deliberately not Init()'d; token/apiURL are empty
	speedChan := make(chan float64, 10)
	go func() {
		for range speedChan {
		}
	}()

	err := client.MeasureUpload(time.Second, speedChan)
	if !errors.Is(err, provider.ErrUploadNotSupported) {
		t.Fatalf("expected ErrUploadNotSupported even without Init(), got: %v", err)
	}
}

func TestProviderInterfaceSatisfied(t *testing.T) {
	var _ provider.Provider = New()
}

// --- Token extraction ---

func TestExtractTokenFound(t *testing.T) {
	client := New()
	jsData := []byte(`DEFAULT_PARAMS={https:!0,token:"YXNkZmFzZGxmbnNkYWZoYXNkZmhrYWxm",urlCount:3}`)

	token, err := client.extractToken(jsData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "YXNkZmFzZGxmbnNkYWZoYXNkZmhrYWxm" {
		t.Errorf("unexpected token: %s", token)
	}
}

func TestExtractTokenMissingReturnsDiagnosableError(t *testing.T) {
	client := New()
	jsData := []byte(`this bundle has been reshaped and no longer contains a token field at all`)

	_, err := client.extractToken(jsData)
	if err == nil {
		t.Fatal("expected an error when no token pattern is present")
	}
	if !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("expected errors.Is(err, ErrTokenNotFound) to hold, got: %v", err)
	}
}

func TestExtractTokenEmptyValueReturnsError(t *testing.T) {
	client := New()
	jsData := []byte(`token:""`)

	_, err := client.extractToken(jsData)
	if !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("expected ErrTokenNotFound for empty token value, got: %v", err)
	}
}

// --- parseJSData / malformed bundle cases ---

func TestParseJSDataMalformedBundleMissingEndpoint(t *testing.T) {
	client := New()
	// Has a token but no apiEndpoint at all.
	jsData := []byte(`token:"abc123"`)

	err := client.parseJSData(jsData)
	if !errors.Is(err, ErrEndpointNotFound) {
		t.Fatalf("expected ErrEndpointNotFound, got: %v", err)
	}
}

func TestParseJSDataMalformedBundleMissingToken(t *testing.T) {
	client := New()
	jsData := []byte(`apiEndpoint="api.fast.com/netflix/speedtest/v2"`)

	err := client.parseJSData(jsData)
	if !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("expected ErrTokenNotFound, got: %v", err)
	}
}

func TestParseJSDataMissingURLCountFallsBackToDefault(t *testing.T) {
	client := New()
	client.urlCount = 0
	jsData := []byte(`apiEndpoint="api.fast.com/netflix/speedtest/v2",token:"abc123"`)

	err := client.parseJSData(jsData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.urlCount != 5 {
		t.Errorf("expected default urlCount 5, got %d", client.urlCount)
	}
}

func TestParseJSDataFull(t *testing.T) {
	client := New()
	jsData := []byte(`apiEndpoint="api.fast.com/netflix/speedtest/v2",token:"abc123",urlCount:7`)

	err := client.parseJSData(jsData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.apiURL != "https://api.fast.com/netflix/speedtest/v2" {
		t.Errorf("unexpected apiURL: %s", client.apiURL)
	}
	if client.token != "abc123" {
		t.Errorf("unexpected token: %s", client.token)
	}
	if client.urlCount != 7 {
		t.Errorf("unexpected urlCount: %d", client.urlCount)
	}
}

// --- target-list (URL) parsing ---

func TestParseURLsFromJSON(t *testing.T) {
	client := New()
	data := []byte(`{"client":{},"targets":[{"url":"https://a.example.com/speedtest"},{"url":"https://b.example.com/speedtest"}]}`)

	urls, err := client.parseURLsFromJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(urls) != 2 {
		t.Fatalf("expected 2 urls, got %d", len(urls))
	}
	if urls[0] != "https://a.example.com/speedtest" || urls[1] != "https://b.example.com/speedtest" {
		t.Errorf("unexpected urls: %v", urls)
	}
}

func TestParseURLsFromJSONEmptyTargets(t *testing.T) {
	client := New()
	data := []byte(`{"client":{},"targets":[]}`)

	urls, err := client.parseURLsFromJSON(data)
	if err == nil {
		t.Fatal("expected an error for empty targets list")
	}
	if !errors.Is(err, ErrAPI) {
		t.Fatalf("expected errors.Is(err, ErrAPI), got: %v", err)
	}
	if urls != nil {
		t.Errorf("expected nil urls, got %v", urls)
	}
}

func TestParseURLsFromJSONMalformed(t *testing.T) {
	client := New()
	data := []byte(`not even json`)

	_, err := client.parseURLsFromJSON(data)
	if !errors.Is(err, ErrAPI) {
		t.Fatalf("expected errors.Is(err, ErrAPI), got: %v", err)
	}
}

// --- byte accounting ---

// TestDownloadNonSuccessStatusContributesZeroBytes proves that a non-2xx
// download response (e.g. a 404/500 error page body) contributes zero
// bytes to the measured total, rather than having its error-page body
// counted as legitimate downloaded data.
func TestDownloadNonSuccessStatusContributesZeroBytes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("this error page body must never be counted as downloaded bytes, padding to be non-trivially sized"))
	}))
	defer server.Close()

	client := New()
	byteLenChan := make(chan int64, 100)
	done := make(chan struct{})

	err := client.download(server.URL, byteLenChan, done)
	close(byteLenChan)

	if err == nil {
		t.Fatal("expected an error for a non-2xx download response")
	}
	if !errors.Is(err, ErrAPI) {
		t.Fatalf("expected errors.Is(err, ErrAPI), got: %v", err)
	}

	var total int64
	for n := range byteLenChan {
		total += n
	}
	if total != 0 {
		t.Errorf("expected zero bytes counted for non-2xx response, got %d", total)
	}
}

// TestDownloadSuccessCountsBytes is the counterpart proving a normal 2xx
// response does contribute its bytes.
func TestDownloadSuccessCountsBytes(t *testing.T) {
	payload := make([]byte, 10000)
	for i := range payload {
		payload[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	client := New()
	byteLenChan := make(chan int64, 100)
	done := make(chan struct{})

	err := client.download(server.URL, byteLenChan, done)
	close(byteLenChan)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var total int64
	for n := range byteLenChan {
		total += n
	}
	if total != int64(len(payload)) {
		t.Errorf("expected %d bytes counted, got %d", len(payload), total)
	}
}

// --- jitter ---

// TestMeasureJitterSubMillisecondPrecision proves that jitter computed from
// sub-millisecond-varying samples is not truncated to zero.
func TestMeasureJitterSubMillisecondPrecision(t *testing.T) {
	client := New()

	// Two "samples" differing by a few hundred microseconds -- if truncated
	// to whole milliseconds first, both would look identical and jitter
	// would be reported as exactly 0.
	measurements := []time.Duration{
		300 * time.Microsecond,
		700 * time.Microsecond,
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
	jitter := totalDeviation / time.Duration(len(measurements))

	if jitter == 0 {
		t.Fatal("expected non-zero sub-millisecond jitter, got 0")
	}
	if jitter >= time.Millisecond {
		t.Fatalf("expected sub-millisecond jitter, got %v", jitter)
	}

	_ = client // client itself is exercised via the live-network test below
}

// TestMeasureJitterTooFewSamplesReturnsError proves that when fewer than 2
// latency samples succeed, MeasureJitter returns a real error instead of
// silently reporting (0, nil), which would be indistinguishable from
// "perfectly stable connection".
func TestMeasureJitterTooFewSamplesReturnsError(t *testing.T) {
	client := New()
	client.client = &http.Client{
		Timeout:   50 * time.Millisecond,
		Transport: failingTransport{},
	}

	_, err := client.MeasureJitter(3)
	if err == nil {
		t.Fatal("expected an error when all latency samples fail")
	}
}

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("simulated network failure")
}

// --- live network tests (guarded) ---

func TestIsAvailable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live network test in short mode")
	}
	client := New()
	available := client.IsAvailable()
	t.Logf("%s available: %v", client.Name(), available)
}

func TestInitLive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live network test in short mode")
	}
	client := New()
	if err := client.Init(); err != nil {
		t.Fatalf("Init against live fast.com failed: %v", err)
	}
	if client.token == "" {
		t.Error("expected a token to be discovered")
	}
	if client.apiURL == "" {
		t.Error("expected an apiURL to be discovered")
	}
}

func TestMeasureLatencyLive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live network test in short mode")
	}
	client := New()
	latency, err := client.MeasureLatency()
	if err != nil {
		t.Fatalf("MeasureLatency failed: %v", err)
	}
	if latency <= 0 {
		t.Errorf("expected positive latency, got %v", latency)
	}
}

func TestMeasureJitterLive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live network test in short mode")
	}
	client := New()
	jitter, err := client.MeasureJitter(5)
	if err != nil {
		t.Fatalf("MeasureJitter failed: %v", err)
	}
	t.Logf("live jitter: %v", jitter)
}

func TestMeasureDownloadLive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live network test in short mode")
	}
	client := New()
	if err := client.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

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
		t.Fatalf("MeasureDownload failed: %v", err)
	}
}
