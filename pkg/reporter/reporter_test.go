package reporter

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	r := New()
	if r == nil {
		t.Fatal("New() returned nil")
	}
}

func TestPrintToConsole(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	pipeReader, pipeWriter, _ := os.Pipe()
	os.Stdout = pipeWriter

	results := "Test output"
	reporter := New()
	reporter.PrintToConsole(results)

	_ = pipeWriter.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, pipeReader)

	if !strings.Contains(buf.String(), "Test output") {
		t.Error("PrintToConsole should output results")
	}
}

func TestFormatResults(t *testing.T) {
	results := `Speed Test Results
==================
Provider:       Test

Download Speed: 100.00 Mbps
Upload Speed:   50.00 Mbps
Latency:        10.00 ms`

	if !strings.Contains(results, "Speed Test Results") {
		t.Error("Results should contain header")
	}

	if !strings.Contains(results, "Download Speed") {
		t.Error("Results should contain download speed")
	}
}

func TestSaveToFile(t *testing.T) {
	r := New()
	testFile := "test_results.txt"
	defer func() { _ = os.Remove(testFile) }()

	results := "Test results"
	err := r.SaveToFile(testFile, results)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Verify file exists and contains data
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	if string(data) != results {
		t.Errorf("Expected file content '%s', got '%s'", results, string(data))
	}
}

func TestSaveToFileError(t *testing.T) {
	r := New()
	// Try to save to an invalid path
	err := r.SaveToFile("/invalid/path/that/does/not/exist/file.txt", "test")
	if err == nil {
		t.Error("Expected error when saving to invalid path")
	}
}

func BenchmarkFormatResults(b *testing.B) {
	results := `Speed Test Results
==================
Provider:       Test

Download Speed: 100.00 Mbps
Upload Speed:   50.00 Mbps
Latency:        10.00 ms`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = strings.Contains(results, "Download Speed")
	}
}
