package speedtest

import (
	"fmt"
	"testing"
	"time"

	"github.com/0x524a/netpulse/internal/config"
	"github.com/0x524a/netpulse/internal/providers/provider"
)

type fakeProvider struct {
	name             string
	available        bool
	initErr          error
	latency          time.Duration
	latencyErr       error
	jitter           time.Duration
	jitterErr        error
	downloadErr      error
	uploadErr        error
	downloadSpeeds   []float64
	uploadSpeeds     []float64
}

func (fp *fakeProvider) Name() string {
	return fp.name
}

func (fp *fakeProvider) IsAvailable() bool {
	return fp.available
}

func (fp *fakeProvider) Init() error {
	return fp.initErr
}

func (fp *fakeProvider) MeasureLatency() (time.Duration, error) {
	return fp.latency, fp.latencyErr
}

func (fp *fakeProvider) MeasureJitter(samples int) (time.Duration, error) {
	return fp.jitter, fp.jitterErr
}

func (fp *fakeProvider) MeasureDownload(speedChan chan<- float64) error {
	if fp.downloadErr != nil {
		return fp.downloadErr
	}
	for _, speed := range fp.downloadSpeeds {
		speedChan <- speed
	}
	return nil
}

func (fp *fakeProvider) MeasureUpload(duration time.Duration, speedChan chan<- float64) error {
	if fp.uploadErr != nil {
		return fp.uploadErr
	}
	for _, speed := range fp.uploadSpeeds {
		speedChan <- speed
	}
	return nil
}

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

			for _, p := range st.providers {
				if p.Name() == tt.expectedName {
					return
				}
			}
		})
	}
}

func TestRunWithProvider_Success(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Verbose = false
	st := New(cfg)

	fp := &fakeProvider{
		name:           "FakeProvider",
		available:      true,
		latency:        10 * time.Millisecond,
		jitter:         2 * time.Millisecond,
		downloadSpeeds: []float64{1000, 1100, 1050},
		uploadSpeeds:   []float64{500, 510, 495},
	}

	err := st.runWithProvider(fp)
	if err != nil {
		t.Fatalf("runWithProvider failed: %v", err)
	}

	result := st.GetResult()
	if result == nil {
		t.Fatal("Expected result after successful run")
	}

	if result.ProviderName != "FakeProvider" {
		t.Errorf("Expected provider name 'FakeProvider', got %q", result.ProviderName)
	}

	if result.DownloadSpeed != 1050 {
		t.Errorf("Expected final download speed 1050, got %v", result.DownloadSpeed)
	}

	if result.UploadSpeed != 495 {
		t.Errorf("Expected final upload speed 495, got %v", result.UploadSpeed)
	}

	if result.Latency != 10*time.Millisecond {
		t.Errorf("Expected latency 10ms, got %v", result.Latency)
	}

	if result.Jitter != 2*time.Millisecond {
		t.Errorf("Expected jitter 2ms, got %v", result.Jitter)
	}

	if !result.Success {
		t.Error("Expected Success to be true")
	}
}

func TestRunWithProvider_LatencyError(t *testing.T) {
	cfg := config.NewConfig()
	st := New(cfg)

	fp := &fakeProvider{
		name:       "FakeProvider",
		available:  true,
		latencyErr: fmt.Errorf("latency test failed"),
	}

	err := st.runWithProvider(fp)
	if err == nil {
		t.Fatal("Expected error from latency measurement")
	}

	if err.Error() != "latency measurement failed: latency test failed" {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRunWithProvider_DownloadError(t *testing.T) {
	cfg := config.NewConfig()
	st := New(cfg)

	fp := &fakeProvider{
		name:        "FakeProvider",
		available:   true,
		latency:     10 * time.Millisecond,
		downloadErr: fmt.Errorf("download failed"),
	}

	err := st.runWithProvider(fp)
	if err == nil {
		t.Fatal("Expected error from download measurement")
	}

	if err.Error() != "download measurement failed: download failed" {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRunWithProvider_UploadError(t *testing.T) {
	cfg := config.NewConfig()
	st := New(cfg)

	fp := &fakeProvider{
		name:           "FakeProvider",
		available:      true,
		latency:        10 * time.Millisecond,
		downloadSpeeds: []float64{1000},
		uploadErr:      fmt.Errorf("upload failed"),
	}

	err := st.runWithProvider(fp)
	if err == nil {
		t.Fatal("Expected error from upload measurement")
	}

	if err.Error() != "upload measurement failed: upload failed" {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRunWithProvider_JitterError(t *testing.T) {
	cfg := config.NewConfig()
	st := New(cfg)

	fp := &fakeProvider{
		name:           "FakeProvider",
		available:      true,
		latency:        10 * time.Millisecond,
		jitterErr:      fmt.Errorf("jitter failed"),
		downloadSpeeds: []float64{1000},
		uploadSpeeds:   []float64{500},
	}

	err := st.runWithProvider(fp)
	if err != nil {
		t.Fatalf("runWithProvider should not fail on jitter error: %v", err)
	}

	result := st.GetResult()
	if result.Jitter != 0 {
		t.Errorf("Expected jitter to be 0 on error, got %v", result.Jitter)
	}
}

func TestRunWithProvider_EmptyDownloadSpeeds(t *testing.T) {
	cfg := config.NewConfig()
	st := New(cfg)

	fp := &fakeProvider{
		name:           "FakeProvider",
		available:      true,
		latency:        10 * time.Millisecond,
		downloadSpeeds: []float64{},
		uploadSpeeds:   []float64{500},
	}

	err := st.runWithProvider(fp)
	if err != nil {
		t.Fatalf("runWithProvider failed: %v", err)
	}

	result := st.GetResult()
	if result.AvgDownload != 0 {
		t.Errorf("Expected avgDownload 0 for empty speeds, got %v", result.AvgDownload)
	}
	if result.MinDownload != 0 {
		t.Errorf("Expected minDownload 0 for empty speeds, got %v", result.MinDownload)
	}
	if result.MaxDownload != 0 {
		t.Errorf("Expected maxDownload 0 for empty speeds, got %v", result.MaxDownload)
	}
}

func TestRunWithProvider_EmptyUploadSpeeds(t *testing.T) {
	cfg := config.NewConfig()
	st := New(cfg)

	fp := &fakeProvider{
		name:           "FakeProvider",
		available:      true,
		latency:        10 * time.Millisecond,
		downloadSpeeds: []float64{1000},
		uploadSpeeds:   []float64{},
	}

	err := st.runWithProvider(fp)
	if err != nil {
		t.Fatalf("runWithProvider failed: %v", err)
	}

	result := st.GetResult()
	if result.AvgUpload != 0 {
		t.Errorf("Expected avgUpload 0 for empty speeds, got %v", result.AvgUpload)
	}
}

func TestRunWithProvider_SpeedAggregation(t *testing.T) {
	cfg := config.NewConfig()
	st := New(cfg)

	fp := &fakeProvider{
		name:           "FakeProvider",
		available:      true,
		latency:        10 * time.Millisecond,
		downloadSpeeds: []float64{1000, 2000, 3000},
		uploadSpeeds:   []float64{400, 500, 600},
	}

	err := st.runWithProvider(fp)
	if err != nil {
		t.Fatalf("runWithProvider failed: %v", err)
	}

	result := st.GetResult()

	expectedAvgDown := (1000 + 2000 + 3000) / 3.0 / 1000.0
	if result.AvgDownload != expectedAvgDown {
		t.Errorf("Expected avgDownload %.2f, got %.2f", expectedAvgDown, result.AvgDownload)
	}

	if result.MinDownload != 1.0 {
		t.Errorf("Expected minDownload 1.0 Mbps, got %.2f", result.MinDownload)
	}

	if result.MaxDownload != 3.0 {
		t.Errorf("Expected maxDownload 3.0 Mbps, got %.2f", result.MaxDownload)
	}

	expectedAvgUp := (400 + 500 + 600) / 3.0 / 1000.0
	if result.AvgUpload != expectedAvgUp {
		t.Errorf("Expected avgUpload %.2f, got %.2f", expectedAvgUp, result.AvgUpload)
	}

	if result.MinUpload != 0.4 {
		t.Errorf("Expected minUpload 0.4 Mbps, got %.2f", result.MinUpload)
	}

	if result.MaxUpload != 0.6 {
		t.Errorf("Expected maxUpload 0.6 Mbps, got %.2f", result.MaxUpload)
	}
}

func TestRun_SpecificProviderSuccess(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Provider = "fastcom"
	cfg.Verbose = false
	st := New(cfg)

	st.providers = []provider.Provider{
		&fakeProvider{
			name:           "Fast.com",
			available:      true,
			latency:        15 * time.Millisecond,
			downloadSpeeds: []float64{5000},
			uploadSpeeds:   []float64{2000},
		},
	}

	err := st.Run()
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	result := st.GetResult()
	if result == nil {
		t.Fatal("Expected result")
	}

	if result.ProviderName != "Fast.com" {
		t.Errorf("Expected provider Fast.com, got %q", result.ProviderName)
	}
}

func TestRun_SpecificProviderUnavailable(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Provider = "cloudflare"
	cfg.Verbose = false
	st := New(cfg)

	st.providers = []provider.Provider{
		&fakeProvider{
			name:      "Cloudflare",
			available: false,
		},
	}

	err := st.Run()
	if err == nil {
		t.Fatal("Expected error for unavailable provider")
	}

	if err.Error() != "provider 'cloudflare' is not available" {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRun_SpecificProviderInitError(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Provider = "mlab"
	cfg.Verbose = false
	st := New(cfg)

	st.providers = []provider.Provider{
		&fakeProvider{
			name:      "M-Lab",
			available: true,
			initErr:   fmt.Errorf("init failed"),
		},
	}

	err := st.Run()
	if err == nil {
		t.Fatal("Expected error for init failure")
	}

	if err.Error() != "failed to initialize provider 'mlab': init failed" {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRun_SpecificProviderNotFound(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Provider = "nonexistent"
	cfg.Verbose = false
	st := New(cfg)

	st.providers = []provider.Provider{
		&fakeProvider{name: "Fast.com"},
	}

	err := st.Run()
	if err == nil {
		t.Fatal("Expected error for nonexistent provider")
	}

	if err.Error() != "provider 'nonexistent' not found" {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRun_FallbackThroughProviders(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Provider = "auto"
	cfg.ServerMode = "auto"
	cfg.Verbose = false
	st := New(cfg)

	st.providers = []provider.Provider{
		&fakeProvider{
			name:      "Provider1",
			available: true,
			latencyErr: fmt.Errorf("offline"),
		},
		&fakeProvider{
			name:           "Provider2",
			available:      true,
			latency:        10 * time.Millisecond,
			downloadSpeeds: []float64{1000},
			uploadSpeeds:   []float64{500},
		},
	}

	err := st.Run()
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	result := st.GetResult()
	if result.ProviderName != "Provider2" {
		t.Errorf("Expected Provider2, got %q", result.ProviderName)
	}
}

func TestRun_ProviderUnavailableSkipped(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Provider = "auto"
	cfg.ServerMode = "auto"
	cfg.Verbose = false
	st := New(cfg)

	st.providers = []provider.Provider{
		&fakeProvider{
			name:      "Provider1",
			available: false,
		},
		&fakeProvider{
			name:           "Provider2",
			available:      true,
			latency:        10 * time.Millisecond,
			downloadSpeeds: []float64{1000},
			uploadSpeeds:   []float64{500},
		},
	}

	err := st.Run()
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	result := st.GetResult()
	if result.ProviderName != "Provider2" {
		t.Errorf("Expected Provider2, got %q", result.ProviderName)
	}
}

func TestRun_AllProvidersFail(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Provider = "auto"
	cfg.ServerMode = "auto"
	cfg.Verbose = false
	st := New(cfg)

	st.providers = []provider.Provider{
		&fakeProvider{
			name:      "Provider1",
			available: true,
			latencyErr: fmt.Errorf("error1"),
		},
		&fakeProvider{
			name:      "Provider2",
			available: true,
			latencyErr: fmt.Errorf("error2"),
		},
	}

	err := st.Run()
	if err == nil {
		t.Fatal("Expected error when all providers fail")
	}

	if err.Error() != "all providers failed, last error: latency measurement failed: error2" {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRun_NoProvidersAvailable(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Provider = "auto"
	cfg.ServerMode = "auto"
	cfg.Verbose = false
	st := New(cfg)

	st.providers = []provider.Provider{
		&fakeProvider{name: "Provider1", available: false},
		&fakeProvider{name: "Provider2", available: false},
	}

	err := st.Run()
	if err == nil {
		t.Fatal("Expected error when no providers available")
	}

	if err.Error() != "no providers available" {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRun_RandomProviderSelection(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Provider = "auto"
	cfg.ServerMode = "random"
	cfg.Verbose = false
	st := New(cfg)

	st.providers = []provider.Provider{
		&fakeProvider{
			name:           "Provider1",
			available:      true,
			latency:        10 * time.Millisecond,
			downloadSpeeds: []float64{1000},
			uploadSpeeds:   []float64{500},
		},
		&fakeProvider{
			name:      "Provider2",
			available: false,
		},
	}

	err := st.Run()
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	result := st.GetResult()
	if result.ProviderName != "Provider1" {
		t.Errorf("Expected Provider1, got %q", result.ProviderName)
	}
}

func TestRun_RandomProviderAllUnavailable(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Provider = "auto"
	cfg.ServerMode = "random"
	cfg.Verbose = false
	st := New(cfg)

	st.providers = []provider.Provider{
		&fakeProvider{name: "Provider1", available: false},
		&fakeProvider{name: "Provider2", available: false},
	}

	err := st.Run()
	if err == nil {
		t.Fatal("Expected error when no providers available")
	}

	if err.Error() != "no providers available" {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRunWithProvider_ConfigDuration(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Duration = 2 * time.Second
	st := New(cfg)

	fp := &fakeProvider{
		name:           "FakeProvider",
		available:      true,
		latency:        10 * time.Millisecond,
		downloadSpeeds: []float64{1000},
		uploadSpeeds:   []float64{500},
	}

	err := st.runWithProvider(fp)
	if err != nil {
		t.Fatalf("runWithProvider failed: %v", err)
	}

	result := st.GetResult()
	if result.UploadTime > 3*time.Second {
		t.Errorf("Expected upload time <= 3s, got %v", result.UploadTime)
	}
}

func TestRun_VerboseRandomProvider(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Provider = "auto"
	cfg.ServerMode = "random"
	cfg.Verbose = true
	st := New(cfg)

	st.providers = []provider.Provider{
		&fakeProvider{
			name:           "Provider1",
			available:      true,
			latency:        10 * time.Millisecond,
			downloadSpeeds: []float64{1000},
			uploadSpeeds:   []float64{500},
		},
	}

	err := st.Run()
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
}

func TestRun_VerboseAutoFallback(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Provider = "auto"
	cfg.ServerMode = "auto"
	cfg.Verbose = true
	st := New(cfg)

	st.providers = []provider.Provider{
		&fakeProvider{
			name:           "Provider1",
			available:      true,
			latency:        10 * time.Millisecond,
			downloadSpeeds: []float64{1000},
			uploadSpeeds:   []float64{500},
		},
	}

	err := st.Run()
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
}

func TestRun_VerboseSpecificProvider(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Provider = "fastcom"
	cfg.Verbose = true
	st := New(cfg)

	st.providers = []provider.Provider{
		&fakeProvider{
			name:           "Fast.com",
			available:      true,
			latency:        10 * time.Millisecond,
			downloadSpeeds: []float64{1000},
			uploadSpeeds:   []float64{500},
		},
	}

	err := st.Run()
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
}

func TestRunWithProvider_Verbose(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Verbose = true
	st := New(cfg)

	fp := &fakeProvider{
		name:           "FakeProvider",
		available:      true,
		latency:        10 * time.Millisecond,
		jitter:         2 * time.Millisecond,
		downloadSpeeds: []float64{1000, 1100},
		uploadSpeeds:   []float64{500, 510},
	}

	err := st.runWithProvider(fp)
	if err != nil {
		t.Fatalf("runWithProvider failed: %v", err)
	}
}

func TestRun_ProviderInitErrorSkippedVerbose(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Provider = "auto"
	cfg.ServerMode = "auto"
	cfg.Verbose = true
	st := New(cfg)

	st.providers = []provider.Provider{
		&fakeProvider{
			name:       "Provider1",
			available:  true,
			initErr:    fmt.Errorf("init failed"),
		},
		&fakeProvider{
			name:           "Provider2",
			available:      true,
			latency:        10 * time.Millisecond,
			downloadSpeeds: []float64{1000},
			uploadSpeeds:   []float64{500},
		},
	}

	err := st.Run()
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	result := st.GetResult()
	if result.ProviderName != "Provider2" {
		t.Errorf("Expected Provider2, got %q", result.ProviderName)
	}
}

func TestRunWithRandomProvider_SkipsUnavailable(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Provider = "auto"
	cfg.ServerMode = "random"
	cfg.Verbose = false
	st := New(cfg)

	st.providers = []provider.Provider{
		&fakeProvider{name: "Provider1", available: false},
		&fakeProvider{name: "Provider2", available: false},
		&fakeProvider{
			name:           "Provider3",
			available:      true,
			latency:        10 * time.Millisecond,
			downloadSpeeds: []float64{1000},
			uploadSpeeds:   []float64{500},
		},
	}

	err := st.Run()
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	result := st.GetResult()
	if result.ProviderName != "Provider3" {
		t.Errorf("Expected Provider3, got %q", result.ProviderName)
	}
}

func TestRunWithRandomProvider_SkipsInitErrors(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Provider = "auto"
	cfg.ServerMode = "random"
	cfg.Verbose = false
	st := New(cfg)

	st.providers = []provider.Provider{
		&fakeProvider{
			name:       "Provider1",
			available:  true,
			initErr:    fmt.Errorf("init failed"),
		},
		&fakeProvider{
			name:           "Provider2",
			available:      true,
			latency:        10 * time.Millisecond,
			downloadSpeeds: []float64{1000},
			uploadSpeeds:   []float64{500},
		},
	}

	err := st.Run()
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	result := st.GetResult()
	if result.ProviderName != "Provider2" {
		t.Errorf("Expected Provider2, got %q", result.ProviderName)
	}
}

func TestFormatResult_NoResult(t *testing.T) {
	cfg := config.NewConfig()
	st := New(cfg)

	formatted := st.FormatResult()
	if formatted != "No results available. Please run the test first." {
		t.Errorf("Expected 'No results available' message, got %q", formatted)
	}
}

func TestFormatResult_WithZeroValues(t *testing.T) {
	cfg := config.NewConfig()
	st := New(cfg)

	st.result = &provider.Result{
		ProviderName: "TestProvider",
		Jitter:       0,
		Timestamp:    time.Now(),
	}

	formatted := st.FormatResult()
	if formatted == "" {
		t.Error("FormatResult should handle zero jitter")
	}

	if !contains(formatted, "TestProvider") {
		t.Error("Formatted result should contain provider name")
	}
}

func TestRun_UnknownSpecificProvider(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Provider = "unknown_provider"
	cfg.Verbose = false
	st := New(cfg)

	st.providers = []provider.Provider{
		&fakeProvider{name: "Fast.com"},
		&fakeProvider{name: "Cloudflare"},
	}

	err := st.Run()
	if err == nil {
		t.Fatal("Expected error for unknown provider")
	}

	if err.Error() != "provider 'unknown_provider' not found" {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRun_VerboseProviderUnavailable(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Provider = "mlab"
	cfg.Verbose = true
	st := New(cfg)

	st.providers = []provider.Provider{
		&fakeProvider{
			name:      "M-Lab",
			available: false,
		},
	}

	err := st.Run()
	if err == nil {
		t.Fatal("Expected error for unavailable provider")
	}
}

func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
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
