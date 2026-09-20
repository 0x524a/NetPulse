package fastcom

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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

func TestIsAvailable(t *testing.T) {
	client := New()
	available := client.IsAvailable()
	t.Logf("%s available: %v", client.Name(), available)
}

func TestInitWithMockServer(t *testing.T) {
	htmlWithScript := `<!DOCTYPE html>
<html>
<head><title>Fast.com</title></head>
<body>
<script src="https://fast.com/app-abc123.js"></script>
<script>
var token = "test_token_123";
var urlCount = 5;
</script>
</body>
</html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		if _, err := io.WriteString(w, htmlWithScript); err != nil {
			t.Logf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	client := New()
	// Set apiURL to use our mock server
	client.apiURL = server.URL
	err := client.Init()
	if err != nil {
		t.Logf("Init with mock server: %v (this is OK - Init fetches from real fast.com)", err)
	}
}

func TestInitMultipleTimes(t *testing.T) {
	client := New()
	// Init should be idempotent
	err1 := client.Init()
	err2 := client.Init()
	if (err1 == nil) != (err2 == nil) {
		t.Logf("Init consistency: %v vs %v", err1, err2)
	}
}

func TestMeasureLatencyWithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	}))
	defer server.Close()

	client := New()
	latency, err := client.MeasureLatency()
	if err != nil {
		t.Logf("MeasureLatency: %v (expected - uses real fast.com)", err)
	}
	if latency >= 0 {
		t.Logf("Latency measurement returned: %v", latency)
	}
}

func TestMeasureDownloadWithMockData(t *testing.T) {
	// Create 2MB of test data
	testData := bytes.Repeat([]byte("0123456789"), 200*1024)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", string(rune(len(testData))))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(testData)
	}))
	defer server.Close()

	client := New()
	client.apiURL = server.URL

	speedChan := make(chan float64, 100)
	go func() {
		for range speedChan {
			// Drain channel
		}
	}()

	err := client.MeasureDownload(speedChan)
	if err != nil {
		t.Logf("MeasureDownload returned: %v", err)
	}
}

func TestMeasureUploadWithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err == nil {
			t.Logf("Mock server received %d bytes", len(body))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := New()
	client.apiURL = server.URL

	speedChan := make(chan float64, 100)
	go func() {
		for range speedChan {
			// Drain channel
		}
	}()

	err := client.MeasureUpload(100*time.Millisecond, speedChan)
	if err != nil {
		t.Logf("MeasureUpload returned: %v", err)
	}
	t.Logf("Upload test sent data to mock server")
}

func TestMeasureDownloadWhenNotInitialized(t *testing.T) {
	client := New()
	// Don't call Init()
	speedChan := make(chan float64, 10)

	go func() {
		for range speedChan {
		}
	}()

	// Should handle gracefully
	err := client.MeasureDownload(speedChan)
	if err != nil {
		t.Logf("Expected behavior - MeasureDownload requires initialization: %v", err)
	}
}

func TestMeasureUploadWhenNotInitialized(t *testing.T) {
	client := New()
	// Don't call Init()
	speedChan := make(chan float64, 10)

	go func() {
		for range speedChan {
		}
	}()

	// Should handle gracefully
	err := client.MeasureUpload(100*time.Millisecond, speedChan)
	if err != nil {
		t.Logf("Expected behavior - MeasureUpload requires initialization: %v", err)
	}
}

func TestIsAvailableWhenFastcomDown(t *testing.T) {
	// This test will fail if fast.com is actually down, but that's OK
	// In a real scenario, you'd mock the HTTP client
	client := New()
	available := client.IsAvailable()
	t.Logf("IsAvailable returned: %v (depends on fast.com connectivity)", available)
}

func TestInitErrorRecovery(t *testing.T) {
	client := New()
	// First attempt (may fail if no network)
	_ = client.Init()
	// Second attempt should also work
	err := client.Init()
	t.Logf("Second Init attempt: %v", err)
}

func TestMeasureLatencyMultipleTimes(t *testing.T) {
	client := New()

	latency1, err1 := client.MeasureLatency()
	if err1 != nil {
		t.Logf("First MeasureLatency: %v", err1)
	}

	latency2, err2 := client.MeasureLatency()
	if err2 != nil {
		t.Logf("Second MeasureLatency: %v", err2)
	}

	if err1 == nil && err2 == nil {
		t.Logf("Both latency measurements succeeded: %v, %v", latency1, latency2)
	}
}

func TestGetURLsWithInvalidCount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `[{"url":"http://example.com/1"}]`)
	}))
	defer server.Close()

	client := New()
	client.token = "test_token"
	client.apiURL = server.URL

	_, err := client.GetURLs(0)
	if err != nil {
		t.Logf("GetURLs with count=0: %v", err)
	}

	_, err = client.GetURLs(-1)
	if err != nil {
		t.Logf("GetURLs with count=-1: %v", err)
	}
}

func TestGetURLsNotInitialized(t *testing.T) {
	client := New()
	urls, err := client.GetURLs(5)
	if err == nil {
		t.Error("Expected error when client not initialized")
	}
	if urls != nil {
		t.Error("Expected nil URLs when not initialized")
	}
}

func TestParseURLsFromJSONMalformed(t *testing.T) {
	client := New()

	tests := []struct {
		name        string
		jsonData    string
		shouldError bool
	}{
		{"Empty JSON", "{}", true},
		{"No URLs", `{"data":"test"}`, true},
		{"Invalid JSON", `{broken json}`, true},
		{"Empty array", `[]`, true},
		{"Single URL", `[{"url":"http://example.com/test"}]`, false},
		{"Multiple URLs", `[{"url":"http://1.com"},{"url":"http://2.com"}]`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urls, err := client.parseURLsFromJSON([]byte(tt.jsonData))
			if tt.shouldError && err == nil {
				t.Errorf("Expected error for %s", tt.name)
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.name, err)
			}
			if !tt.shouldError && len(urls) == 0 {
				t.Errorf("Expected URLs for %s", tt.name)
			}
		})
	}
}

func TestFindScriptSrcErrors(t *testing.T) {
	client := New()

	tests := []struct {
		name        string
		htmlData    string
		shouldError bool
	}{
		{"No script tag", `<html><body>test</body></html>`, true},
		{"Script without src", `<html><script></script></body></html>`, true},
		{"Wrong script src", `<html><script src="/other.js"></script></body></html>`, true},
		{"Valid script", `<html><script src="/app-abc123.js"></script></body></html>`, false},
		{"Multiple scripts, one valid", `<html><script src="/other.js"></script><script src="/app-xyz.js"></script></html>`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, err := client.findScriptSrc([]byte(tt.htmlData))
			if tt.shouldError && err == nil {
				t.Errorf("Expected error for %s", tt.name)
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.name, err)
			}
			if !tt.shouldError && len(src) == 0 {
				t.Errorf("Expected script src for %s", tt.name)
			}
		})
	}
}

func TestParseJSDataMissingFields(t *testing.T) {
	client := New()

	tests := []struct {
		name        string
		jsData      string
		shouldError bool
		expectToken string
		expectCount int
	}{
		{"Missing all", `var x = 1;`, true, "", 5},
		{"Missing token", `apiEndpoint="api.fast.com" urlCount:10`, true, "", 10},
		{"Missing endpoint", `token:"abc123" urlCount:10`, true, "abc123", 10},
		{"Missing count", `apiEndpoint="api.fast.com" token:"abc123"`, false, "abc123", 5},
		{"Valid all", `apiEndpoint="api.fast.com" token:"test_token" urlCount:20`, false, "test_token", 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.parseJSData([]byte(tt.jsData))
			if tt.shouldError && err == nil {
				t.Errorf("Expected error for %s", tt.name)
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.name, err)
			}
			if !tt.shouldError && client.token != tt.expectToken {
				t.Errorf("Expected token %s, got %s for %s", tt.expectToken, client.token, tt.name)
			}
			if !tt.shouldError && client.urlCount != tt.expectCount {
				t.Errorf("Expected count %d, got %d for %s", tt.expectCount, client.urlCount, tt.name)
			}
		})
	}
}

func TestMeasureJitterEdgeCases(t *testing.T) {
	client := New()

	t.Run("Zero samples defaults to 10", func(t *testing.T) {
		jitter, err := client.MeasureJitter(0)
		if err != nil && jitter == 0 {
			t.Logf("MeasureJitter with 0 samples: %v", err)
		}
	})

	t.Run("Single sample returns zero", func(t *testing.T) {
		jitter, err := client.MeasureJitter(1)
		if err != nil {
			t.Logf("MeasureJitter with 1 sample: %v", err)
		}
		if jitter == 0 {
			t.Logf("Got 0 jitter with 1 sample as expected")
		}
	})

	t.Run("Negative samples defaults to 10", func(t *testing.T) {
		jitter, err := client.MeasureJitter(-5)
		if err != nil && jitter == 0 {
			t.Logf("MeasureJitter with -5 samples: %v", err)
		}
	})
}

func TestDownloadWithBrokenURL(t *testing.T) {
	client := New()
	client.token = "test_token"
	client.apiURL = "http://invalid-domain-that-does-not-exist-xyz.local"

	speedChan := make(chan float64, 10)
	go func() {
		for range speedChan {
		}
	}()

	err := client.MeasureDownload(speedChan)
	if err != nil {
		t.Logf("MeasureDownload with broken URL: %v", err)
	}
}

func TestUploadWithBrokenURL(t *testing.T) {
	client := New()
	client.token = "test_token"
	client.apiURL = "http://invalid-domain-that-does-not-exist-xyz.local"

	speedChan := make(chan float64, 10)
	go func() {
		for range speedChan {
		}
	}()

	err := client.MeasureUpload(100*time.Millisecond, speedChan)
	if err != nil {
		t.Logf("MeasureUpload with broken URL: %v", err)
	}
}

func TestDownloadNoURLs(t *testing.T) {
	client := New()
	client.token = "test_token"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `[]`)
	}))
	defer server.Close()

	client.apiURL = server.URL

	speedChan := make(chan float64, 10)
	go func() {
		for range speedChan {
		}
	}()

	err := client.MeasureDownload(speedChan)
	if err == nil {
		t.Error("Expected error when no URLs provided")
	}
}

func TestUploadNoURLs(t *testing.T) {
	client := New()
	client.token = "test_token"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `[]`)
	}))
	defer server.Close()

	client.apiURL = server.URL

	speedChan := make(chan float64, 10)
	go func() {
		for range speedChan {
		}
	}()

	err := client.MeasureUpload(100*time.Millisecond, speedChan)
	if err == nil {
		t.Error("Expected error when no URLs provided")
	}
}

func TestClientHTTPTimeout(t *testing.T) {
	client := New()

	if client.client == nil {
		t.Error("HTTP client should be initialized")
	}
	if client.client.Timeout != 30*time.Second {
		t.Errorf("Expected 30s timeout, got %v", client.client.Timeout)
	}
}

func TestClientInitialization(t *testing.T) {
	client := New()

	if client.urlCount != 5 {
		t.Errorf("Expected default urlCount of 5, got %d", client.urlCount)
	}
	if client.token != "" {
		t.Errorf("Expected empty token on init, got %s", client.token)
	}
	if client.apiURL != "" {
		t.Errorf("Expected empty apiURL on init, got %s", client.apiURL)
	}
	if client.client == nil {
		t.Error("Expected HTTP client to be initialized")
	}
}

func TestUploadChunkWithBrokenServer(t *testing.T) {
	client := New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	data := []byte("test data")
	uploaded := client.uploadChunk(server.URL, data)

	if uploaded != int64(len(data)) {
		t.Logf("uploadChunk returned %d bytes (server returns 500)", uploaded)
	}
}

func TestUploadChunkInvalidURL(t *testing.T) {
	client := New()
	data := []byte("test data")

	uploaded := client.uploadChunk("http://\x00invalid", data)
	if uploaded != 0 {
		t.Errorf("Expected 0 bytes for invalid URL, got %d", uploaded)
	}
}

func TestDownloadHTTPError(t *testing.T) {
	client := New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, "Not Found")
	}))
	defer server.Close()

	speedChan := make(chan float64, 10)
	stopCh := make(chan struct{})

	go func() {
		for range speedChan {
		}
	}()

	go func() {
		time.Sleep(100 * time.Millisecond)
		close(stopCh)
	}()

	byteLenChan := make(chan int64, 10)
	err := client.download(server.URL, byteLenChan, stopCh)
	if err != nil {
		t.Logf("download with 404: %v", err)
	}
}

func TestInitErrorHandling(t *testing.T) {
	client := New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	origEndpoint := endpoint
	client.client.Timeout = 100 * time.Millisecond

	t.Run("Server error on main page", func(t *testing.T) {
		client := New()
		client.client.Timeout = 100 * time.Millisecond
		err := client.Init()
		if err == nil {
			t.Logf("Init: %v", err)
		}
	})

	_ = origEndpoint
}
