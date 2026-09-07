package librespeed

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// mockServerSpec describes one entry to serve from a fake servers.php list,
// all rooted at a single httptest.Server so we don't need to spin up one
// httptest instance per LibreSpeed server.
type mockServerSpec struct {
	name      string
	dlPath    string
	ulPath    string
	pingPath  string
	getIPPath string
}

func rawServerJSON(base string, s mockServerSpec) Server {
	return Server{
		Name:      s.name,
		Base:      base,
		DLPath:    s.dlPath,
		ULPath:    s.ulPath,
		PingPath:  s.pingPath,
		GetIPPath: s.getIPPath,
	}
}

func TestNew(t *testing.T) {
	client := New()
	if client == nil {
		t.Fatal("New() returned nil")
	}
}

func TestName(t *testing.T) {
	client := New()
	if name := client.Name(); name != "LibreSpeed" {
		t.Errorf("Name() = %q, want %q", name, "LibreSpeed")
	}
}

func TestMeasureDownloadWithoutInit(t *testing.T) {
	client := &Client{client: http.DefaultClient}
	speedChan := make(chan float64, 10)
	go func() {
		for range speedChan {
		}
	}()

	if err := client.MeasureDownload(speedChan); err != ErrNoServers {
		t.Errorf("MeasureDownload() without Init() = %v, want %v", err, ErrNoServers)
	}
}

func TestMeasureUploadWithoutInit(t *testing.T) {
	client := &Client{client: http.DefaultClient}
	speedChan := make(chan float64, 10)
	go func() {
		for range speedChan {
		}
	}()

	if err := client.MeasureUpload(100*time.Millisecond, speedChan); err != ErrNoServers {
		t.Errorf("MeasureUpload() without Init() = %v, want %v", err, ErrNoServers)
	}
}

func TestMeasureLatencyWithoutInit(t *testing.T) {
	client := &Client{client: http.DefaultClient}
	if _, err := client.MeasureLatency(); err != ErrNoServers {
		t.Errorf("MeasureLatency() without Init() = %v, want %v", err, ErrNoServers)
	}
}

func TestMeasureJitterWithoutInit(t *testing.T) {
	client := &Client{client: http.DefaultClient}
	if _, err := client.MeasureJitter(5); err != ErrNoServers {
		t.Errorf("MeasureJitter() without Init() = %v, want %v", err, ErrNoServers)
	}
}

// --- Server.resolve() URL joining ---

func TestServerResolveJoinsRelativeURLs(t *testing.T) {
	cases := []struct {
		name     string
		base     string
		dl       string
		wantBase string
	}{
		{"no trailing slash on base", "https://host.example/backend", "garbage.php", "https://host.example/backend/garbage.php"},
		{"trailing slash on base", "https://host.example/", "backend/garbage.php", "https://host.example/backend/garbage.php"},
		{"no path on base", "https://host.example", "garbage.php", "https://host.example/garbage.php"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &Server{Name: "test", Base: tc.base, DLPath: tc.dl, ULPath: "empty.php", PingPath: "empty.php"}
			if err := s.resolve(); err != nil {
				t.Fatalf("resolve() error: %v", err)
			}
			if s.DownloadURL != tc.wantBase {
				t.Errorf("DownloadURL = %q, want %q", s.DownloadURL, tc.wantBase)
			}
		})
	}
}

func TestServerResolveRejectsEmptyBase(t *testing.T) {
	s := &Server{Name: "bad", Base: "", DLPath: "garbage.php"}
	if err := s.resolve(); err == nil {
		t.Error("resolve() with empty base should return an error")
	}
}

func TestClampChunkSizeMB(t *testing.T) {
	cases := []struct{ in, want int }{
		{0, minChunkSizeMB},
		{1, minChunkSizeMB},
		{4, 4},
		{500, 500},
		{1024, 1024},
		{5000, maxChunkSizeMB},
	}
	for _, tc := range cases {
		if got := clampChunkSizeMB(tc.in); got != tc.want {
			t.Errorf("clampChunkSizeMB(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

// --- checkServer liveness semantics ---

func TestCheckServerLiveness(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
		body       string
		wantUp     bool
	}{
		{"empty body and 200 is up", http.StatusOK, "", true},
		{"non-empty body and 200 is down", http.StatusOK, "unexpected", false},
		{"empty body but non-200 is down", http.StatusInternalServerError, "", false},
		{"non-empty body and non-200 is down", http.StatusNotFound, "not found", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer ts.Close()

			client := New()
			s := &Server{Name: "t", PingURL: ts.URL}
			_, up := client.checkServer(context.Background(), s)
			if up != tc.wantUp {
				t.Errorf("checkServer() up = %v, want %v", up, tc.wantUp)
			}
		})
	}
}

// --- downloadOnce / uploadOnce unit behavior ---

func TestDownloadOnceCountsBytesAndSendsIdentityHeader(t *testing.T) {
	payload := strings.Repeat("d", 256*1024)
	var gotAcceptEncoding string
	var gotCkSize string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAcceptEncoding = r.Header.Get("Accept-Encoding")
		gotCkSize = r.URL.Query().Get("ckSize")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(payload))
	}))
	defer ts.Close()

	client := New()
	client.server = &Server{Name: "t", DownloadURL: ts.URL}

	n := client.downloadOnce(context.Background())
	if n != int64(len(payload)) {
		t.Errorf("downloadOnce() = %d bytes, want %d", n, len(payload))
	}
	if gotAcceptEncoding != "identity" {
		t.Errorf("Accept-Encoding header = %q, want %q", gotAcceptEncoding, "identity")
	}
	if gotCkSize == "" {
		t.Error("expected ckSize query parameter to be set")
	}
}

func TestDownloadOnceNonSuccessStatusContributesZeroBytes(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(strings.Repeat("x", 1024)))
	}))
	defer ts.Close()

	client := New()
	client.server = &Server{Name: "t", DownloadURL: ts.URL}

	if n := client.downloadOnce(context.Background()); n != 0 {
		t.Errorf("downloadOnce() with 503 response = %d bytes, want 0", n)
	}
}

func TestUploadOnceReturnsBlobLengthOnSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := New()
	client.server = &Server{Name: "t", UploadURL: ts.URL}
	blob := make([]byte, 4096)

	if n := client.uploadOnce(context.Background(), blob); n != int64(len(blob)) {
		t.Errorf("uploadOnce() = %d, want %d", n, len(blob))
	}
}

// TestUploadOnceNonSuccessContributesZeroBytes proves that a failed upload
// (non-2xx response) is recorded as zero bytes sent, not as
// len(data) -- this was the original bug: the provider counted bytes
// written to the request regardless of what the server said back.
func TestUploadOnceNonSuccessContributesZeroBytes(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := New()
	client.server = &Server{Name: "t", UploadURL: ts.URL}
	blob := make([]byte, 8192)

	if n := client.uploadOnce(context.Background(), blob); n != 0 {
		t.Errorf("uploadOnce() with 500 response = %d bytes, want 0", n)
	}
}

// TestUploadOnceSetsContentLengthAndAvoidsChunkedEncoding guards against
// the librespeed/speedtest-cli#122-style hang: if the client fails to set
// ContentLength explicitly, Go falls back to Transfer-Encoding: chunked,
// which some real LibreSpeed backends never decode.
func TestUploadOnceSetsContentLengthAndAvoidsChunkedEncoding(t *testing.T) {
	var gotContentLength int64
	var gotTransferEncoding []string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentLength = r.ContentLength
		gotTransferEncoding = r.TransferEncoding
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := New()
	client.server = &Server{Name: "t", UploadURL: ts.URL}
	blob := make([]byte, 1024*1024)

	client.uploadOnce(context.Background(), blob)

	if gotContentLength <= 0 {
		t.Errorf("server observed ContentLength = %d, want > 0", gotContentLength)
	}
	if gotContentLength != int64(len(blob)) {
		t.Errorf("server observed ContentLength = %d, want %d", gotContentLength, len(blob))
	}
	for _, te := range gotTransferEncoding {
		if strings.EqualFold(te, "chunked") {
			t.Errorf("server observed Transfer-Encoding: chunked, upload must use a known Content-Length")
		}
	}
}

// --- MeasureLatency / MeasureJitter ---

func TestMeasureLatency(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := New()
	client.server = &Server{Name: "t", PingURL: ts.URL}

	latency, err := client.MeasureLatency()
	if err != nil {
		t.Fatalf("MeasureLatency() error: %v", err)
	}
	if latency < 0 {
		t.Errorf("MeasureLatency() = %v, want non-negative", latency)
	}
}

func TestMeasureLatencyServerDown(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not empty"))
	}))
	defer ts.Close()

	client := New()
	client.server = &Server{Name: "t", PingURL: ts.URL}

	if _, err := client.MeasureLatency(); err == nil {
		t.Error("MeasureLatency() against a non-conforming ping response should error")
	}
}

func TestMeasureJitter(t *testing.T) {
	var count atomic.Int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := count.Add(1)
		// Vary latency slightly across calls so jitter is non-trivial.
		if n%2 == 0 {
			time.Sleep(2 * time.Millisecond)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := New()
	client.server = &Server{Name: "t", PingURL: ts.URL}

	jitter, err := client.MeasureJitter(6)
	if err != nil {
		t.Fatalf("MeasureJitter() error: %v", err)
	}
	if jitter < 0 {
		t.Errorf("MeasureJitter() = %v, want non-negative", jitter)
	}
	// samples=6 -> 7 pings issued, first discarded.
	if got := count.Load(); got != 7 {
		t.Errorf("ping endpoint called %d times, want 7 (samples+1)", got)
	}
}

func TestMeasureJitterDefaultsWhenSamplesTooSmall(t *testing.T) {
	var count atomic.Int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := New()
	client.server = &Server{Name: "t", PingURL: ts.URL}

	if _, err := client.MeasureJitter(1); err != nil {
		t.Fatalf("MeasureJitter() error: %v", err)
	}
	// samples defaults to 10 -> 11 pings issued.
	if got := count.Load(); got != 11 {
		t.Errorf("ping endpoint called %d times, want 11 (default samples+1)", got)
	}
}

// --- MeasureDownload / MeasureUpload end-to-end against a mock server ---

func TestMeasureDownloadEndToEnd(t *testing.T) {
	origDuration := downloadTestDuration
	downloadTestDuration = 1200 * time.Millisecond
	defer func() { downloadTestDuration = origDuration }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strings.Repeat("d", 512*1024)))
	}))
	defer ts.Close()

	client := New()
	client.server = &Server{Name: "t", DownloadURL: ts.URL}

	speedChan := make(chan float64, 100)
	var samples int
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range speedChan {
			samples++
		}
	}()

	if err := client.MeasureDownload(speedChan); err != nil {
		t.Fatalf("MeasureDownload() error: %v", err)
	}
	close(speedChan)
	<-done

	if samples == 0 {
		t.Error("expected at least one speed sample on speedChan")
	}
}

func TestMeasureUploadEndToEnd(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := New()
	client.server = &Server{Name: "t", UploadURL: ts.URL}

	speedChan := make(chan float64, 100)
	var samples int
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range speedChan {
			samples++
		}
	}()

	if err := client.MeasureUpload(1200*time.Millisecond, speedChan); err != nil {
		t.Fatalf("MeasureUpload() error: %v", err)
	}
	close(speedChan)
	<-done

	if samples == 0 {
		t.Error("expected at least one speed sample on speedChan")
	}
}

// --- Full discovery + selection flow via Init(), replacing the old
// t.Skip("would require extensive mock setup"). ---

func TestInitDiscoversAndSelectsFastestServer(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/fast/ping.php", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/slow/ping.php", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(40 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/down/ping.php", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not a real ping response"))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	mux.HandleFunc("/servers.php", func(w http.ResponseWriter, r *http.Request) {
		list := []Server{
			rawServerJSON(ts.URL, mockServerSpec{name: "fast", dlPath: "fast/dl.php", ulPath: "fast/ul.php", pingPath: "fast/ping.php"}),
			rawServerJSON(ts.URL, mockServerSpec{name: "slow", dlPath: "slow/dl.php", ulPath: "slow/ul.php", pingPath: "slow/ping.php"}),
			rawServerJSON(ts.URL, mockServerSpec{name: "down", dlPath: "down/dl.php", ulPath: "down/ul.php", pingPath: "down/ping.php"}),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(list)
	})

	origURL := serversListURL
	serversListURL = ts.URL + "/servers.php"
	defer func() { serversListURL = origURL }()

	client := New()
	if err := client.Init(); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	if client.server == nil {
		t.Fatal("Init() left server nil")
	}
	if client.server.Name != "fast" {
		t.Errorf("Init() selected server %q, want %q", client.server.Name, "fast")
	}
	if client.server.Ping <= 0 {
		t.Errorf("selected server Ping = %v, want > 0", client.server.Ping)
	}
	wantDL := ts.URL + "/fast/dl.php"
	if client.server.DownloadURL != wantDL {
		t.Errorf("DownloadURL = %q, want %q", client.server.DownloadURL, wantDL)
	}
}

func TestInitReturnsErrNoServersWhenAllDown(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/down/ping.php", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	mux.HandleFunc("/servers.php", func(w http.ResponseWriter, r *http.Request) {
		list := []Server{
			rawServerJSON(ts.URL, mockServerSpec{name: "down", dlPath: "down/dl.php", ulPath: "down/ul.php", pingPath: "down/ping.php"}),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(list)
	})

	origURL := serversListURL
	serversListURL = ts.URL + "/servers.php"
	defer func() { serversListURL = origURL }()

	client := New()
	if err := client.Init(); err != ErrNoServers {
		t.Errorf("Init() with all servers down = %v, want %v", err, ErrNoServers)
	}
}

func TestInitFallsBackToWellKnown(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping.php", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	// Primary list URL 404s; only the well-known fallback path works.
	mux.HandleFunc("/backend-servers/servers.php", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	mux.HandleFunc("/.well-known/librespeed", func(w http.ResponseWriter, r *http.Request) {
		list := []Server{
			rawServerJSON(ts.URL, mockServerSpec{name: "fallback", dlPath: "dl.php", ulPath: "ul.php", pingPath: "ping.php"}),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(list)
	})

	origURL := serversListURL
	serversListURL = ts.URL + "/backend-servers/servers.php"
	defer func() { serversListURL = origURL }()

	client := New()
	if err := client.Init(); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	if client.server == nil || client.server.Name != "fallback" {
		t.Errorf("Init() did not select the fallback-discovered server: %+v", client.server)
	}
}

// --- IsAvailable ---

func TestIsAvailable(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "[]")
	}))
	defer ts.Close()

	origURL := serversListURL
	serversListURL = ts.URL
	defer func() { serversListURL = origURL }()

	client := New()
	if !client.IsAvailable() {
		t.Error("IsAvailable() = false, want true when server list responds 200")
	}
}

func TestIsAvailableFalseOnError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	origURL := serversListURL
	serversListURL = ts.URL
	defer func() { serversListURL = origURL }()

	client := New()
	if client.IsAvailable() {
		t.Error("IsAvailable() = true, want false when server list responds 500")
	}
}

// --- GetISPInfo graceful degradation ---

func TestGetISPInfoWellFormed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"processedString":"1.2.3.4 - Example ISP","rawIspInfo":{"asn":"AS1234"}}`)
	}))
	defer ts.Close()

	client := New()
	client.server = &Server{Name: "t", GetIPURL: ts.URL}

	info, err := client.GetISPInfo("km")
	if err != nil {
		t.Fatalf("GetISPInfo() error: %v", err)
	}
	if info.ProcessedString != "1.2.3.4 - Example ISP" {
		t.Errorf("ProcessedString = %q, want %q", info.ProcessedString, "1.2.3.4 - Example ISP")
	}
	if info.RawISPInfo == "" {
		t.Error("expected RawISPInfo to be populated")
	}
}

func TestGetISPInfoDegradesOnMalformedResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "not json at all")
	}))
	defer ts.Close()

	client := New()
	client.server = &Server{Name: "t", GetIPURL: ts.URL}

	info, err := client.GetISPInfo("km")
	if err != nil {
		t.Fatalf("GetISPInfo() error: %v", err)
	}
	if info.ProcessedString != "not json at all" {
		t.Errorf("ProcessedString = %q, want raw body fallback", info.ProcessedString)
	}
}

func TestGetISPInfoWithoutInit(t *testing.T) {
	client := &Client{client: http.DefaultClient}
	if _, err := client.GetISPInfo("km"); err != ErrNoServers {
		t.Errorf("GetISPInfo() without Init() = %v, want %v", err, ErrNoServers)
	}
}

// --- Guarded live-network tests. These exercise the real librespeed.org
// server directory and real public LibreSpeed backends, so they are
// skipped whenever -short is passed (see .github/workflows/ci.yml). ---

func TestLiveIsAvailable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live network test in short mode")
	}
	client := New()
	if !client.IsAvailable() {
		t.Error("IsAvailable() = false against the real LibreSpeed server directory")
	}
}

func TestLiveInitAndMeasureLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live network test in short mode")
	}

	client := New()
	if err := client.Init(); err != nil {
		t.Fatalf("Init() against the real LibreSpeed server directory failed: %v", err)
	}
	if client.server == nil {
		t.Fatal("Init() left server nil")
	}
	t.Logf("selected live server: %s (ping=%v)", client.server.Name, client.server.Ping)

	latency, err := client.MeasureLatency()
	if err != nil {
		t.Fatalf("MeasureLatency() against live server failed: %v", err)
	}
	t.Logf("measured live latency: %v", latency)
}
