package main

import (
	"flag"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestCmdProviders(t *testing.T) {
	app := newApp()

	// Capture stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	oldStdout := os.Stdout
	os.Stdout = w

	defer func() {
		os.Stdout = oldStdout
	}()

	// Run the providers command
	err = app.Run([]string{"speedtest", "providers"})

	// Close the write end and read the output
	_ = w.Close()
	output, _ := io.ReadAll(r)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	outputStr := string(output)

	// Check for expected provider names in output
	expectedProviders := []string{
		"Fast.com",
		"fastcom",
		"Cloudflare",
		"cloudflare",
		"M-Lab",
		"mlab",
		"LibreSpeed",
		"librespeed",
		"Speedtest.net",
		"ookla",
	}

	for _, provider := range expectedProviders {
		if !strings.Contains(outputStr, provider) {
			t.Errorf("Expected output to contain %q, got: %s", provider, outputStr)
		}
	}
}

func TestCmdProvidersWithAlias(t *testing.T) {
	app := newApp()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	oldStdout := os.Stdout
	os.Stdout = w

	defer func() {
		os.Stdout = oldStdout
	}()

	// Run the providers command via alias "list"
	err = app.Run([]string{"speedtest", "list"})

	_ = w.Close()
	output, _ := io.ReadAll(r)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Fast.com") {
		t.Errorf("Expected output to contain provider info, got: %s", outputStr)
	}
}

// TestCmdRunErrorPaths tests error handling for various invalid configurations
// These tests verify that config validation errors are caught before attempting network calls
func TestCmdRunErrorPaths(t *testing.T) {
	// Clear environment variables to prevent interference
	envVars := []string{
		"SPEEDTEST_PROVIDER",
		"SPEEDTEST_DURATION",
		"SPEEDTEST_URLS",
		"SPEEDTEST_SERVER",
		"SPEEDTEST_VERBOSE",
		"SPEEDTEST_SAVE",
		"SPEEDTEST_OUTPUT",
	}
	for _, v := range envVars {
		_ = os.Unsetenv(v)
	}

	tests := []struct {
		name      string
		args      []string
		wantError bool
		errMsg    string
	}{
		{
			name:      "unknown provider",
			args:      []string{"speedtest", "--provider=bogusprovider"},
			wantError: true,
			errMsg:    "unknown provider",
		},
		{
			name:      "negative duration",
			args:      []string{"speedtest", "--duration=-5s"},
			wantError: true,
			errMsg:    "duration must be positive",
		},
		{
			name:      "zero duration",
			args:      []string{"speedtest", "--duration=0s"},
			wantError: true,
			errMsg:    "duration must be positive",
		},
		{
			name:      "negative urls",
			args:      []string{"speedtest", "--urls=-5"},
			wantError: true,
			errMsg:    "urlCount must be positive",
		},
		{
			name:      "zero urls",
			args:      []string{"speedtest", "--urls=0"},
			wantError: true,
			errMsg:    "urlCount must be positive",
		},
		{
			name:      "invalid server mode",
			args:      []string{"speedtest", "--server=unknownmode"},
			wantError: true,
			errMsg:    "unknown server mode",
		},
		{
			name:      "multiple providers with invalid entry",
			args:      []string{"speedtest", "--provider=fastcom,invalid,cloudflare"},
			wantError: true,
			errMsg:    "unknown provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newApp()
			err := app.Run(tt.args)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got: %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestCmdRunMalformedDurationFlag(t *testing.T) {
	_ = os.Unsetenv("SPEEDTEST_DURATION")
	app := newApp()
	err := app.Run([]string{"speedtest", "--duration=notaduration"})

	if err == nil {
		t.Error("Expected error for malformed duration, got nil")
	}
}

func TestCmdRunMalformedURLsFlag(t *testing.T) {
	_ = os.Unsetenv("SPEEDTEST_URLS")
	app := newApp()
	err := app.Run([]string{"speedtest", "--urls=notanumber"})

	if err == nil {
		t.Error("Expected error for malformed urls, got nil")
	}
}

func TestCmdRunValidServerModes(t *testing.T) {
	_ = os.Unsetenv("SPEEDTEST_PROVIDER")
	_ = os.Unsetenv("SPEEDTEST_DURATION")

	// Test that valid server modes pass flag parsing
	validModes := []string{"auto", "random", "specific"}

	for _, mode := range validModes {
		t.Run("mode="+mode, func(t *testing.T) {
			app := newApp()
			// Errors expected (network/provider execution), but not from flag parsing
			err := app.Run([]string{"speedtest", "--server=" + mode, "--provider=auto", "--duration=0s"})

			// The error should be about duration, not server mode
			if err != nil && strings.Contains(err.Error(), "unknown server mode") {
				t.Errorf("Should accept valid server mode %q, got: %v", mode, err)
			}
		})
	}
}

func TestCmdBenchmarkErrorPaths(t *testing.T) {
	_ = os.Unsetenv("SPEEDTEST_DURATION")

	tests := []struct {
		name      string
		args      []string
		wantError bool
		errMsg    string
	}{
		{
			name:      "negative duration",
			args:      []string{"speedtest", "benchmark", "--duration=-1s"},
			wantError: true,
			errMsg:    "duration must be positive",
		},
		{
			name:      "zero duration",
			args:      []string{"speedtest", "benchmark", "--duration=0s"},
			wantError: true,
			errMsg:    "duration must be positive",
		},
		{
			name:      "malformed duration",
			args:      []string{"speedtest", "benchmark", "--duration=invalid"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newApp()
			err := app.Run(tt.args)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got: %v", tt.errMsg, err)
				}
			}
		})
	}
}

func TestCmdBenchmarkMalformedFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "malformed tests value",
			args: []string{"speedtest", "benchmark", "--tests=notanumber"},
		},
		{
			name: "malformed duration value",
			args: []string{"speedtest", "benchmark", "--duration=notduration"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newApp()
			err := app.Run(tt.args)

			if err == nil {
				t.Errorf("Expected error for %s", tt.name)
			}
		})
	}
}

func TestAppConstruction(t *testing.T) {
	app := newApp()

	if app.Name != "speedtest" {
		t.Errorf("Expected app name 'speedtest', got %q", app.Name)
	}

	if len(app.Commands) != 2 {
		t.Errorf("Expected 2 commands, got %d", len(app.Commands))
	}

	// Check providers command exists
	providersCmd := app.Commands[0]
	if providersCmd.Name != "providers" {
		t.Errorf("Expected first command 'providers', got %q", providersCmd.Name)
	}

	// Check benchmark command exists
	benchmarkCmd := app.Commands[1]
	if benchmarkCmd.Name != "benchmark" {
		t.Errorf("Expected second command 'benchmark', got %q", benchmarkCmd.Name)
	}

	// Verify command aliases
	if len(providersCmd.Aliases) != 2 {
		t.Errorf("Expected 2 aliases for providers, got %d", len(providersCmd.Aliases))
	}

	if len(benchmarkCmd.Aliases) != 2 {
		t.Errorf("Expected 2 aliases for benchmark, got %d", len(benchmarkCmd.Aliases))
	}
}

func TestAppGlobalFlags(t *testing.T) {
	app := newApp()

	// Check that global flags exist
	flagNames := make(map[string]bool)
	for _, flag := range app.Flags {
		flagNames[flag.Names()[0]] = true
	}

	expectedFlags := []string{"provider", "duration", "urls", "server", "verbose", "save", "output"}
	for _, expected := range expectedFlags {
		if !flagNames[expected] {
			t.Errorf("Expected global flag %q not found", expected)
		}
	}
}

func TestAppBenchmarkFlags(t *testing.T) {
	app := newApp()

	// Find benchmark command
	var benchmarkCmd *cli.Command
	for _, cmd := range app.Commands {
		if cmd.Name == "benchmark" {
			benchmarkCmd = cmd
			break
		}
	}

	if benchmarkCmd == nil {
		t.Fatal("benchmark command not found")
	}

	// Check that benchmark flags exist
	flagNames := make(map[string]bool)
	for _, flag := range benchmarkCmd.Flags {
		flagNames[flag.Names()[0]] = true
	}

	expectedFlags := []string{"tests", "duration", "verbose", "output"}
	for _, expected := range expectedFlags {
		if !flagNames[expected] {
			t.Errorf("Expected benchmark flag %q not found", expected)
		}
	}
}

func TestCmdRunValidAutoProvider(t *testing.T) {
	_ = os.Unsetenv("SPEEDTEST_PROVIDER")
	_ = os.Unsetenv("SPEEDTEST_DURATION")

	app := newApp()
	err := app.Run([]string{"speedtest", "--provider=auto", "--duration=0s"})

	// Should error due to zero duration, not provider
	if err != nil && !strings.Contains(err.Error(), "duration must be positive") {
		t.Errorf("Should fail on duration validation, not provider, got: %v", err)
	}
}

func TestCmdRunValidAllProvider(t *testing.T) {
	_ = os.Unsetenv("SPEEDTEST_PROVIDER")
	_ = os.Unsetenv("SPEEDTEST_DURATION")

	app := newApp()
	err := app.Run([]string{"speedtest", "--provider=all", "--duration=0s"})

	// Should error due to zero duration, not provider
	if err != nil && !strings.Contains(err.Error(), "duration must be positive") {
		t.Errorf("Should fail on duration validation, not provider, got: %v", err)
	}
}

func TestCmdRunValidKnownProvider(t *testing.T) {
	_ = os.Unsetenv("SPEEDTEST_PROVIDER")
	_ = os.Unsetenv("SPEEDTEST_DURATION")

	app := newApp()
	// Test that known provider names pass flag parsing with invalid duration to avoid network call
	err := app.Run([]string{"speedtest", "--provider=fastcom", "--duration=0s"})

	// Should error due to zero duration, not unknown provider
	if err != nil && !strings.Contains(err.Error(), "duration must be positive") {
		t.Errorf("Should accept fastcom provider but fail on duration, got: %v", err)
	}
}

func TestCmdProvidersCommandAction(t *testing.T) {
	// Directly test cmdProviders function
	ctx := cli.NewContext(newApp(), &flag.FlagSet{}, nil)

	// Capture stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	oldStdout := os.Stdout
	os.Stdout = w

	err = cmdProviders(ctx)

	os.Stdout = oldStdout
	_ = w.Close()
	output, _ := io.ReadAll(r)

	if err != nil {
		t.Errorf("cmdProviders returned error: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Available Speed Test Providers") {
		t.Errorf("Expected providers output, got: %s", outputStr)
	}
}

func TestCmdRunHeaderOutput(t *testing.T) {
	_ = os.Unsetenv("SPEEDTEST_DURATION")

	app := newApp()

	// Capture stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	oldStdout := os.Stdout
	os.Stdout = w

	// Run with invalid duration to ensure output before error
	_ = app.Run([]string{"speedtest", "--duration=0s"})

	os.Stdout = oldStdout
	_ = w.Close()
	output, _ := io.ReadAll(r)

	outputStr := string(output)

	// Should see the header
	if !strings.Contains(outputStr, "Internet Speed Test Client") {
		t.Errorf("Expected header output, got: %s", outputStr)
	}
}

func TestCmdBenchmarkHeaderOutput(t *testing.T) {
	_ = os.Unsetenv("SPEEDTEST_DURATION")

	app := newApp()

	// Capture stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	oldStdout := os.Stdout
	os.Stdout = w

	// Run with invalid duration to ensure output before error
	_ = app.Run([]string{"speedtest", "benchmark", "--duration=0s"})

	os.Stdout = oldStdout
	_ = w.Close()
	output, _ := io.ReadAll(r)

	outputStr := string(output)

	// Should see the header
	if !strings.Contains(outputStr, "Internet Speed Test Client - Provider Benchmark") {
		t.Errorf("Expected header output, got: %s", outputStr)
	}
}
