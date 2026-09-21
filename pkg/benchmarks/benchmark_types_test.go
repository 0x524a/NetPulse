package benchmarks

import (
	"testing"
	"time"
)

// These tests exercise the plain data types in benchmark_types.go: their
// zero values and basic field wiring. There are no methods/helpers on these
// types to unit test directly (see types.go, which is a backwards-compat
// placeholder with no types at all), so this focuses on making sure the
// structs hold the values assigned to them and that the zero values used
// elsewhere in the package (e.g. NewBenchmarkSuite) behave as expected.

func TestProviderResult_FieldAssignment(t *testing.T) {
	now := time.Now()
	r := ProviderResult{
		ProviderName:        "fastcom",
		DownloadMbps:        123.45,
		UploadMbps:          67.89,
		Latency:             15 * time.Millisecond,
		Jitter:              2 * time.Millisecond,
		Success:             true,
		Timestamp:           now,
		LatencyMeasurements: []time.Duration{10 * time.Millisecond, 20 * time.Millisecond},
	}

	if r.ProviderName != "fastcom" {
		t.Errorf("ProviderName = %q, want %q", r.ProviderName, "fastcom")
	}
	if !r.Success {
		t.Error("Success = false, want true")
	}
	if len(r.LatencyMeasurements) != 2 {
		t.Errorf("LatencyMeasurements length = %d, want 2", len(r.LatencyMeasurements))
	}
	if !r.Timestamp.Equal(now) {
		t.Errorf("Timestamp = %v, want %v", r.Timestamp, now)
	}
}

func TestProviderBenchmark_ZeroValue(t *testing.T) {
	var b ProviderBenchmark
	if b.NumTests != 0 || b.SuccessfulTests != 0 || b.FailedTests != 0 {
		t.Error("zero-value ProviderBenchmark should have all counts at 0")
	}
	if b.TestResults != nil {
		t.Error("zero-value ProviderBenchmark should have a nil TestResults slice")
	}
}

func TestBenchmarkSuite_ZeroValueMapsAreNil(t *testing.T) {
	// Documents the difference from NewBenchmarkSuite(): a bare zero-value
	// BenchmarkSuite does NOT have initialized maps, so callers must use
	// the constructor rather than a struct literal for a usable suite.
	var bs BenchmarkSuite
	if bs.Benchmarks != nil {
		t.Error("zero-value BenchmarkSuite.Benchmarks should be nil")
	}
	if bs.ProviderScores != nil {
		t.Error("zero-value BenchmarkSuite.ProviderScores should be nil")
	}
}

func TestDetailedComparison_FieldAssignment(t *testing.T) {
	dc := DetailedComparison{
		MetricName: "download",
		Providers: []ProviderComparisonPoint{
			{ProviderName: "a", Value: 100, Rank: 1, Percentage: 100},
			{ProviderName: "b", Value: 50, Rank: 2, Percentage: 50},
		},
	}

	if dc.MetricName != "download" {
		t.Errorf("MetricName = %q, want %q", dc.MetricName, "download")
	}
	if len(dc.Providers) != 2 {
		t.Fatalf("Providers length = %d, want 2", len(dc.Providers))
	}
	if dc.Providers[0].Rank != 1 || dc.Providers[1].Rank != 2 {
		t.Errorf("unexpected ranks: %+v", dc.Providers)
	}
}

func TestPerformanceScore_FieldAssignment(t *testing.T) {
	// PerformanceScore is not currently constructed anywhere in the
	// production code (see the coverage report) - this test only pins
	// down its field semantics in case something starts using it.
	ps := PerformanceScore{
		ProviderName:     "fastcom",
		DownloadScore:    90,
		UploadScore:      80,
		LatencyScore:     95,
		JitterScore:      85,
		ReliabilityScore: 100,
		ConsistencyScore: 70,
		OverallScore:     88.5,
	}

	if ps.ProviderName != "fastcom" {
		t.Errorf("ProviderName = %q, want %q", ps.ProviderName, "fastcom")
	}
	if ps.OverallScore != 88.5 {
		t.Errorf("OverallScore = %v, want 88.5", ps.OverallScore)
	}
}
