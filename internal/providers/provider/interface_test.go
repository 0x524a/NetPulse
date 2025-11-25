package provider

import (
	"testing"
	"time"
)

// MockProvider is a test implementation of the Provider interface
type MockProvider struct {
	name      string
	available bool
	initError error
}

func NewMockProvider(name string, available bool) *MockProvider {
	return &MockProvider{
		name:      name,
		available: available,
	}
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) Init() error {
	return m.initError
}

func (m *MockProvider) MeasureDownload(speedChan chan<- float64) error {
	// Simulate some download measurements
	go func() {
		speedChan <- 1000.0
		speedChan <- 2000.0
		speedChan <- 3000.0
		close(speedChan)
	}()
	return nil
}

func (m *MockProvider) MeasureUpload(duration time.Duration, speedChan chan<- float64) error {
	// Simulate some upload measurements
	go func() {
		speedChan <- 500.0
		speedChan <- 750.0
		speedChan <- 1000.0
		close(speedChan)
	}()
	return nil
}

func (m *MockProvider) MeasureLatency() (time.Duration, error) {
	return 10 * time.Millisecond, nil
}

func (m *MockProvider) IsAvailable() bool {
	return m.available
}

func TestMockProvider(t *testing.T) {
	provider := NewMockProvider("TestProvider", true)

	if provider.Name() != "TestProvider" {
		t.Errorf("Expected name 'TestProvider', got %s", provider.Name())
	}

	if !provider.IsAvailable() {
		t.Error("Expected provider to be available")
	}

	if err := provider.Init(); err != nil {
		t.Errorf("Init failed: %v", err)
	}

	latency, err := provider.MeasureLatency()
	if err != nil {
		t.Errorf("MeasureLatency failed: %v", err)
	}
	if latency != 10*time.Millisecond {
		t.Errorf("Expected latency 10ms, got %v", latency)
	}
}

func TestResult(t *testing.T) {
	result := &Result{
		ProviderName:  "TestProvider",
		DownloadSpeed: 1000.0,
		UploadSpeed:   500.0,
		Latency:       10 * time.Millisecond,
		Success:       true,
	}

	if result.ProviderName != "TestProvider" {
		t.Errorf("Expected ProviderName 'TestProvider', got %s", result.ProviderName)
	}

	if result.DownloadSpeed != 1000.0 {
		t.Errorf("Expected DownloadSpeed 1000.0, got %f", result.DownloadSpeed)
	}

	if !result.Success {
		t.Error("Expected Success to be true")
	}
}
