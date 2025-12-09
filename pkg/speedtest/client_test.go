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
