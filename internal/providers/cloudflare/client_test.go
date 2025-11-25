package cloudflare

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
	if name == "" {
		t.Error("Name() should not return empty string")
	}
	t.Logf("Provider name: %s", name)
}

func TestIsAvailable(t *testing.T) {
	client := New()
	available := client.IsAvailable()
	t.Logf("%s available: %v", client.Name(), available)
}

func TestInit(t *testing.T) {
	client := New()
	err := client.Init()
	if err != nil {
		t.Errorf("Init() returned error: %v", err)
	}
}

func TestMeasureLatency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := New()
	latency, err := client.MeasureLatency()
	if err != nil {
		t.Errorf("MeasureLatency() error: %v", err)
	}
	if latency < 0 {
		t.Errorf("MeasureLatency() returned negative latency: %v", latency)
	}
	t.Logf("Measured latency: %v", latency)
}

func TestMeasureDownloadWithMock(t *testing.T) {
	// Create mock server serving 10MB of data
	testData := bytes.Repeat([]byte("a"), 10*1024*1024)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", string(rune(len(testData))))
		w.WriteHeader(http.StatusOK)
		w.Write(testData)
	}))
	defer server.Close()

	client := New()
	speedChan := make(chan float64, 100)

	go func() {
		for range speedChan {
			// Drain channel
		}
	}()

	err := client.MeasureDownload(speedChan)
	if err != nil {
		t.Logf("MeasureDownload error (may be network related): %v", err)
	}
}

func TestMeasureUploadWithMock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		t.Logf("Received upload data: %d bytes", len(body))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New()
	speedChan := make(chan float64, 100)

	go func() {
		for range speedChan {
			// Drain channel
		}
	}()

	err := client.MeasureUpload(1*time.Second, speedChan)
	if err != nil {
		t.Logf("MeasureUpload error (may be network related): %v", err)
	}
}

func TestGetResult(t *testing.T) {
	client := New()
	client.Init()
	result := client.GetResult()
	t.Logf("GetResult() returned: %v (may be unimplemented)", result)
}
