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
	client.serverURL = server.URL
	client.Init()
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
	client.Init()

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
		w.Write(testData)
	}))
	defer server.Close()

	client := New()
	client.serverURL = server.URL
	client.Init()
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

func TestMeasureDownloadWithLargeFile(t *testing.T) {
	testData := bytes.Repeat([]byte("x"), 10*1024*1024)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", string(rune(len(testData))))
		w.WriteHeader(http.StatusOK)
		w.Write(testData)
	}))
	defer server.Close()

	client := New()
	client.serverURL = server.URL
	client.Init()
	speedChan := make(chan float64, 200)

	go func() {
		for range speedChan {
		}
	}()

	err := client.MeasureDownload(speedChan)
	if err != nil {
		t.Logf("Large download error: %v", err)
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
	client.serverURL = server.URL
	client.Init()
	speedChan := make(chan float64, 100)

	measurements := 0
	go func() {
		for range speedChan {
			measurements++
		}
	}()

	err := client.MeasureUpload(1*time.Second, speedChan)
	if err != nil {
		t.Logf("MeasureUpload error: %v", err)
	}
	t.Logf("Upload test made %d speed measurements", measurements)
}

func TestMeasureUploadWithShortDuration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New()
	client.serverURL = server.URL
	client.Init()
	speedChan := make(chan float64, 50)

	go func() {
		for range speedChan {
		}
	}()

	err := client.MeasureUpload(100*time.Millisecond, speedChan)
	if err != nil {
		t.Logf("Short upload error: %v", err)
	}
}
