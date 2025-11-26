package ookla

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
		t.Logf("Init() error: %v", err)
	}
}

func TestMeasureLatency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			t.Logf("Write error: %v", err)
		}
	}))
	defer server.Close()

	client := New()
	client.serverURL = server.URL
	if err := client.Init(); err != nil {
		t.Logf("Init error: %v", err)
	}
	latency, err := client.MeasureLatency()
	if err != nil {
		t.Logf("MeasureLatency() error: %v", err)
	}
	if latency < 0 {
		t.Errorf("MeasureLatency() returned negative latency: %v", latency)
	}
	t.Logf("Measured latency: %v", latency)
}

func TestMeasureLatencyMultipleTimes(t *testing.T) {
	client := New()
	if err := client.Init(); err != nil {
		t.Logf("Init error: %v", err)
	}

	latency1, err1 := client.MeasureLatency()
	latency2, err2 := client.MeasureLatency()

	if err1 == nil && err2 == nil {
		t.Logf("Latency measurements: %v and %v", latency1, latency2)
	} else {
		t.Logf("Errors: %v, %v", err1, err2)
	}
}

func TestMeasureDownloadWithMock(t *testing.T) {
	testData := bytes.Repeat([]byte("b"), 6*1024*1024)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", string(rune(len(testData))))
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(testData); err != nil {
			t.Logf("Write error: %v", err)
		}
	}))
	defer server.Close()

	client := New()
	client.serverURL = server.URL
	if err := client.Init(); err != nil {
		t.Logf("Init error: %v", err)
	}
	speedChan := make(chan float64, 100)

	measurements := 0
	go func() {
		for range speedChan {
			measurements++
		}
	}()

	err := client.MeasureDownload(speedChan)
	if err != nil {
		t.Logf("MeasureDownload error: %v", err)
	}
	t.Logf("Download test made %d speed measurements", measurements)
}

func TestMeasureUploadWithMock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		t.Logf("Received upload data: %d bytes", len(body))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New()
	client.serverURL = server.URL
	if err := client.Init(); err != nil {
		t.Logf("Init error: %v", err)
	}
	speedChan := make(chan float64, 100)

	measurements := 0
	go func() {
		for range speedChan {
			measurements++
		}
	}()

	err := client.MeasureUpload(200*time.Millisecond, speedChan)
	if err != nil {
		t.Logf("MeasureUpload error: %v", err)
	}
	t.Logf("Upload test made %d speed measurements", measurements)
}
