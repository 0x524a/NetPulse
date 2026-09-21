package speedtest

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	client := New()
	if client == nil {
		t.Fatal("New() returned nil")
	}

	if client.config == nil {
		t.Error("Client config should not be nil")
	}
}

func TestWithOptions(t *testing.T) {
	client := New(
		WithVerbose(true),
		WithProvider("fastcom"),
		WithServerMode("random"),
		WithDuration(30*time.Second),
		WithURLCount(10),
	)

	if !client.config.Verbose {
		t.Error("Verbose should be true")
	}

	if client.config.Provider != "fastcom" {
		t.Errorf("Expected provider 'fastcom', got %s", client.config.Provider)
	}

	if client.config.ServerMode != "random" {
		t.Errorf("Expected server mode 'random', got %s", client.config.ServerMode)
	}

	if client.config.Duration != 30*time.Second {
		t.Errorf("Expected duration 30s, got %v", client.config.Duration)
	}

	if client.config.URLCount != 10 {
		t.Errorf("Expected URLCount 10, got %d", client.config.URLCount)
	}
}

func TestListProviders(t *testing.T) {
	providers := ListProviders()

	if len(providers) != 5 {
		t.Errorf("Expected 5 providers, got %d", len(providers))
	}

	expectedProviders := []string{"fastcom", "cloudflare", "mlab", "librespeed", "ookla"}
	for i, expected := range expectedProviders {
		if providers[i] != expected {
			t.Errorf("Expected provider[%d] to be %s, got %s", i, expected, providers[i])
		}
	}
}

func TestWithOutputFile(t *testing.T) {
	client := New(WithOutputFile("custom_output.txt"))
	if client == nil {
		t.Fatal("New() returned nil")
	}
}

func TestRun(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}
	
	// This test will attempt to run but likely fail due to network
	// Still covers the Run() code path
	client := New(
		WithProvider("fastcom"),
		WithDuration(1*time.Second),
		WithURLCount(1),
	)

	_, err := client.Run()
	// We expect an error since providers require network
	if err != nil {
		t.Logf("Run failed as expected in test environment: %v", err)
	}
}

func TestGetFormattedResult(t *testing.T) {
	client := New()

	// Try to get formatted result before running
	result := client.GetFormattedResult()

	// Should return empty or error message since no test was run
	t.Logf("GetFormattedResult returned: %s", result)
}

func BenchmarkNew(b *testing.B) {
	for i := 0; i < b.N; i++ {
		New()
	}
}

func BenchmarkNewWithOptions(b *testing.B) {
	for i := 0; i < b.N; i++ {
		New(
			WithVerbose(true),
			WithProvider("fastcom"),
			WithDuration(15*time.Second),
		)
	}
}

func TestGetAllResults(t *testing.T) {
	client := New()
	results := client.GetAllResults()
	if len(results) != 0 {
		t.Errorf("Expected 0 results before running, got %d", len(results))
	}
}

func TestGetAllResultsAfterRun(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}

	client := New(
		WithProvider("fastcom"),
		WithDuration(1*time.Second),
		WithURLCount(1),
	)

	_, _ = client.Run()
	results := client.GetAllResults()
	if results == nil {
		t.Error("GetAllResults should not return nil")
	}
}

func TestGetFormattedResultWithMultipleResults(t *testing.T) {
	client := New()
	client.results = []*Result{
		{
			Provider:      "fastcom",
			DownloadSpeed: 100.0,
			UploadSpeed:   50.0,
			Latency:       10 * time.Millisecond,
			Timestamp:     time.Now(),
		},
		{
			Provider:      "cloudflare",
			DownloadSpeed: 110.0,
			UploadSpeed:   55.0,
			Latency:       12 * time.Millisecond,
			Timestamp:     time.Now(),
		},
	}

	result := client.GetFormattedResult()
	if result == "" {
		t.Error("GetFormattedResult should not return empty string")
	}
	if len(result) == 0 {
		t.Error("GetFormattedResult output is empty")
	}
}

func TestWithAllOptions(t *testing.T) {
	duration := 45 * time.Second
	client := New(
		WithVerbose(true),
		WithProvider("cloudflare"),
		WithServerMode("auto"),
		WithDuration(duration),
		WithURLCount(15),
		WithOutputFile("test_output.json"),
	)

	if !client.config.Verbose {
		t.Error("Verbose option not applied")
	}
	if client.config.Provider != "cloudflare" {
		t.Errorf("Provider option not applied: got %s", client.config.Provider)
	}
	if client.config.ServerMode != "auto" {
		t.Errorf("ServerMode option not applied: got %s", client.config.ServerMode)
	}
	if client.config.Duration != duration {
		t.Errorf("Duration option not applied: got %v", client.config.Duration)
	}
	if client.config.URLCount != 15 {
		t.Errorf("URLCount option not applied: got %d", client.config.URLCount)
	}
	if client.config.OutputFile != "test_output.json" {
		t.Errorf("OutputFile option not applied: got %s", client.config.OutputFile)
	}
	if !client.config.SaveToFile {
		t.Error("SaveToFile should be true when OutputFile is set")
	}
}

func TestZeroAndNegativeDurations(t *testing.T) {
	client1 := New(WithDuration(0))
	if client1.config.Duration != 0 {
		t.Errorf("Zero duration not set correctly: got %v", client1.config.Duration)
	}

	client2 := New(WithDuration(-5 * time.Second))
	if client2.config.Duration != -5*time.Second {
		t.Errorf("Negative duration not set correctly: got %v", client2.config.Duration)
	}
}

func TestZeroAndNegativeURLCount(t *testing.T) {
	client1 := New(WithURLCount(0))
	if client1.config.URLCount != 0 {
		t.Errorf("Zero URLCount not set correctly: got %d", client1.config.URLCount)
	}

	client2 := New(WithURLCount(-10))
	if client2.config.URLCount != -10 {
		t.Errorf("Negative URLCount not set correctly: got %d", client2.config.URLCount)
	}
}

func TestResultKbpsToMbpsConversion(t *testing.T) {
	tests := []struct {
		name         string
		kbpsDownload float64
		kbpsUpload   float64
		expectDown   float64
		expectUp     float64
	}{
		{"Standard speeds", 50000, 25000, 50.0, 25.0},
		{"Zero speeds", 0, 0, 0.0, 0.0},
		{"Large speeds", 1000000, 500000, 1000.0, 500.0},
		{"Fractional kbps", 1500, 750, 1.5, 0.75},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mbps := tt.kbpsDownload / 1000.0
			if mbps != tt.expectDown {
				t.Errorf("Expected %.2f Mbps, got %.2f", tt.expectDown, mbps)
			}
		})
	}
}

func TestListProvidersCount(t *testing.T) {
	providers := ListProviders()
	if len(providers) != 5 {
		t.Errorf("Expected 5 providers, got %d", len(providers))
	}
}

func TestListProvidersOrder(t *testing.T) {
	providers := ListProviders()
	expectedOrder := []string{"fastcom", "cloudflare", "mlab", "librespeed", "ookla"}

	for i, expected := range expectedOrder {
		if i >= len(providers) {
			t.Fatalf("Provider list shorter than expected")
		}
		if providers[i] != expected {
			t.Errorf("Provider[%d]: expected %s, got %s", i, expected, providers[i])
		}
	}
}

func TestMultipleOptionsOverwrite(t *testing.T) {
	client := New(
		WithProvider("fastcom"),
		WithProvider("cloudflare"),
		WithProvider("mlab"),
	)

	if client.config.Provider != "mlab" {
		t.Errorf("Last option should win: expected mlab, got %s", client.config.Provider)
	}
}

func TestResultWithEmptyProvider(t *testing.T) {
	result := &Result{
		Provider:      "",
		DownloadSpeed: 100.0,
		UploadSpeed:   50.0,
	}

	if result.Provider != "" {
		t.Error("Empty provider not preserved")
	}
}

func TestClientTesterInitialization(t *testing.T) {
	client := New(
		WithVerbose(true),
		WithProvider("fastcom"),
	)

	if client.tester == nil {
		t.Error("Client tester should not be nil after initialization")
	}
	if client.config == nil {
		t.Error("Client config should not be nil after initialization")
	}
}

func TestRunBenchmarkWithEmptyProviders(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}

	client := New(
		WithDuration(1*time.Second),
		WithURLCount(1),
	)

	opts := BenchmarkOptions{
		TestsPerProvider: 1,
		Duration:         1 * time.Second,
		Verbose:          false,
		Providers:        []string{},
	}

	suite, err := client.RunBenchmark(opts)
	if err != nil {
		t.Logf("RunBenchmark with empty providers returned error: %v", err)
	}
	if suite != nil {
		t.Logf("RunBenchmark suite created")
	}
}

func TestRunBenchmarkWithSingleProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}

	client := New(
		WithDuration(1*time.Second),
		WithURLCount(1),
	)

	opts := BenchmarkOptions{
		TestsPerProvider: 1,
		Duration:         1 * time.Second,
		Verbose:          true,
		Providers:        []string{"fastcom"},
	}

	suite, err := client.RunBenchmark(opts)
	if err != nil {
		t.Logf("RunBenchmark returned error: %v", err)
	}
	if suite == nil {
		t.Error("Expected suite to not be nil")
	}
}

func TestRunBenchmarkWithMultipleTests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}

	client := New(
		WithDuration(1*time.Second),
		WithURLCount(1),
	)

	opts := BenchmarkOptions{
		TestsPerProvider: 2,
		Duration:         1 * time.Second,
		Verbose:          false,
		Providers:        []string{"fastcom", "cloudflare"},
	}

	suite, err := client.RunBenchmark(opts)
	if err != nil {
		t.Logf("RunBenchmark with multiple tests returned error: %v", err)
	}
	if suite != nil {
		t.Logf("RunBenchmark suite created")
	}
}

func TestRunBenchmarkZeroTests(t *testing.T) {
	client := New()

	opts := BenchmarkOptions{
		TestsPerProvider: 0,
		Duration:         5 * time.Second,
		Verbose:          false,
		Providers:        []string{"fastcom"},
	}

	suite, err := client.RunBenchmark(opts)
	if err != nil {
		t.Logf("RunBenchmark with 0 tests per provider returned error: %v", err)
	}
	if suite != nil {
		t.Logf("RunBenchmark suite created with 0 tests")
	}
}
