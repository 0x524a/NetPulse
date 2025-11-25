package config

import (
	"flag"
	"testing"
	"time"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()

	if cfg == nil {
		t.Fatal("NewConfig returned nil")
	}

	if cfg.URLCount != 5 {
		t.Errorf("Expected URLCount 5, got %d", cfg.URLCount)
	}

	if cfg.Duration != 15*time.Second {
		t.Errorf("Expected Duration 15s, got %v", cfg.Duration)
	}

	if cfg.Verbose != false {
		t.Error("Expected Verbose false")
	}

	if cfg.SaveToFile != false {
		t.Error("Expected SaveToFile false")
	}

	if cfg.Provider != "auto" {
		t.Errorf("Expected Provider 'auto', got %s", cfg.Provider)
	}

	if cfg.ServerMode != "auto" {
		t.Errorf("Expected ServerMode 'auto', got %s", cfg.ServerMode)
	}
}

func TestLoadFromFlags(t *testing.T) {
	// Reset flags for testing
	oldCommandLine := flag.CommandLine
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	defer func() { flag.CommandLine = oldCommandLine }()

	cfg := NewConfig()
	cfg.LoadFromFlags()

	// Parse empty args to use defaults
	flag.CommandLine.Parse([]string{})

	// After loading from flags with defaults, values should match NewConfig defaults
	if cfg.URLCount != 5 {
		t.Errorf("Expected URLCount 5 after LoadFromFlags, got %d", cfg.URLCount)
	}

	if cfg.Duration != 15*time.Second {
		t.Errorf("Expected Duration 15s after LoadFromFlags, got %v", cfg.Duration)
	}
}

func BenchmarkNewConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewConfig()
	}
}
