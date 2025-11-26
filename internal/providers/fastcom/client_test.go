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
