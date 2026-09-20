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
	if err := flag.CommandLine.Parse([]string{}); err != nil {
		t.Fatalf("Failed to parse flags: %v", err)
	}

	// After loading from flags with defaults, values should match NewConfig defaults
	if cfg.URLCount != 5 {
		t.Errorf("Expected URLCount 5 after LoadFromFlags, got %d", cfg.URLCount)
	}

	if cfg.Duration != 15*time.Second {
		t.Errorf("Expected Duration 15s after LoadFromFlags, got %v", cfg.Duration)
	}
}

func TestValidate(t *testing.T) {
	valid := func() *Config {
		cfg := NewConfig()
		cfg.URLCount = 5
		cfg.Duration = 15 * time.Second
		cfg.Provider = "auto"
		cfg.ServerMode = "auto"
		return cfg
	}

	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{"defaults are valid", func(c *Config) {}, false},
		{"zero urlCount", func(c *Config) { c.URLCount = 0 }, true},
		{"negative urlCount", func(c *Config) { c.URLCount = -1 }, true},
		{"zero duration", func(c *Config) { c.Duration = 0 }, true},
		{"negative duration", func(c *Config) { c.Duration = -time.Second }, true},
		{"unknown server mode", func(c *Config) { c.ServerMode = "bogus" }, true},
		{"server mode random", func(c *Config) { c.ServerMode = "random" }, false},
		{"server mode specific", func(c *Config) { c.ServerMode = "specific" }, false},
		{"provider all", func(c *Config) { c.Provider = "all" }, false},
		{"provider known single", func(c *Config) { c.Provider = "cloudflare" }, false},
		{"provider unknown", func(c *Config) { c.Provider = "bogus" }, true},
		{"provider multiple with known list", func(c *Config) {
			c.Provider = "multiple"
			c.Providers = []string{"fastcom", "mlab"}
		}, false},
		{"provider multiple with unknown entry", func(c *Config) {
			c.Provider = "multiple"
			c.Providers = []string{"fastcom", "bogus"}
		}, true},
		{"provider multiple with no providers", func(c *Config) {
			c.Provider = "multiple"
			c.Providers = nil
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := valid()
			tt.mutate(cfg)

			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("Validate() expected an error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

func BenchmarkNewConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewConfig()
	}
}
