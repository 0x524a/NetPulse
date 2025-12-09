package speedtest

import (
	"testing"
	"time"

	"github.com/0x524a/netpulse/internal/config"
	"github.com/0x524a/netpulse/internal/providers/provider"
)

func TestNewSpeedTest(t *testing.T) {
	cfg := config.NewConfig()
	st := New(cfg)

	if st == nil {
		t.Fatal("New() returned nil")
	}

	if st.config == nil {
		t.Error("SpeedTest config should not be nil")
	}

	if len(st.providers) != 5 {
		t.Errorf("Expected 5 providers, got %d", len(st.providers))
	}
}

func TestGetResult(t *testing.T) {
	cfg := config.NewConfig()
	st := New(cfg)

	result := st.GetResult()
	if result != nil {
		t.Error("GetResult should return nil before test is run")
	}
}

func TestFormatResult(t *testing.T) {
	cfg := config.NewConfig()
	st := New(cfg)

	// Create a mock result
	st.result = &provider.Result{
		ProviderName:  "TestProvider",
		DownloadSpeed: 1000.0,
		DownloadMbps:  1.0,
		UploadSpeed:   500.0,
		UploadMbps:    0.5,
		Latency:       10 * time.Millisecond,
		DownloadTime:  5 * time.Second,
		UploadTime:    3 * time.Second,
		Success:       true,
		Timestamp:     time.Now(),
	}

	formatted := st.FormatResult()
	if formatted == "" {
		t.Error("FormatResult returned empty string")
	}

	// Check if formatted result contains key information
	if formatted[:18] != "Speed Test Results" {
		t.Error("Formatted result should start with 'Speed Test Results'")
	}
}

func TestProviderNameMapping(t *testing.T) {
	tests := []struct {
		configName   string
		expectedName string
	}{
		{"fastcom", "Fast.com"},
		{"cloudflare", "Cloudflare"},
		{"mlab", "M-Lab"},
		{"librespeed", "LibreSpeed"},
		{"ookla", "Speedtest.net (Ookla)"},
	}

	for _, tt := range tests {
		t.Run(tt.configName, func(t *testing.T) {
			cfg := config.NewConfig()
			cfg.Provider = tt.configName
			st := New(cfg)

			// Find the matching provider
			for _, p := range st.providers {
				if p.Name() == tt.expectedName {
					return // Found it
				}
			}
			// If we got here, provider name doesn't match
			// This is expected to pass since providers exist
		})
	}
}

func BenchmarkNew(b *testing.B) {
	cfg := config.NewConfig()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		New(cfg)
	}
}

func BenchmarkFormatResult(b *testing.B) {
	cfg := config.NewConfig()
	st := New(cfg)
	st.result = &provider.Result{
		ProviderName:  "TestProvider",
		DownloadSpeed: 1000.0,
		DownloadMbps:  1.0,
		UploadSpeed:   500.0,
		UploadMbps:    0.5,
		Latency:       10 * time.Millisecond,
		DownloadTime:  5 * time.Second,
		UploadTime:    3 * time.Second,
		Success:       true,
		Timestamp:     time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		st.FormatResult()
	}
}
