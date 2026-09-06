package benchmarks

import (
	"strings"
	"testing"
	"time"
)

func TestGetScoreGrade(t *testing.T) {
	cases := []struct {
		score float64
		want  string
	}{
		{100, "A"},
		{90, "A"},
		{89.99, "B"},
		{80, "B"},
		{79.99, "C"},
		{70, "C"},
		{69.99, "D"},
		{60, "D"},
		{59.99, "E"},
		{50, "E"},
		{49.99, "F"},
		{0, "F"},
		{-10, "F"},
	}
	for _, c := range cases {
		if got := getScoreGrade(c.score); got != c.want {
			t.Errorf("getScoreGrade(%v) = %q, want %q", c.score, got, c.want)
		}
	}
}

func buildSampleSuite(t *testing.T) *BenchmarkSuite {
	t.Helper()
	bs := NewBenchmarkSuite()
	addSuccess(bs, "fastcom", 300, 150, 8)
	addSuccess(bs, "mlab", 200, 100, 12)
	addSuccess(bs, "cloudflare", 100, 50, 16)

	if err := bs.Finalize(); err != nil {
		t.Fatalf("Finalize() returned unexpected error: %v", err)
	}
	return bs
}

func TestFormatBenchmarkReport_ContainsExpectedSections(t *testing.T) {
	bs := buildSampleSuite(t)
	report := FormatBenchmarkReport(bs)

	wantSubstrings := []string{
		"SPEED TEST PROVIDER BENCHMARK RESULTS",
		"OVERALL PERFORMANCE RANKINGS",
		"DOWNLOAD SPEED RANKINGS",
		"UPLOAD SPEED RANKINGS",
		"LATENCY RANKINGS (Lower is Better)",
		"JITTER RANKINGS (Lower is Better)",
		"RELIABILITY RANKINGS",
		"CONSISTENCY RANKINGS (Higher is Better)",
		"DETAILED PROVIDER STATISTICS",
		"SCORING METHODOLOGY",
		"fastcom",
		"mlab",
		"cloudflare",
	}
	for _, sub := range wantSubstrings {
		if !strings.Contains(report, sub) {
			t.Errorf("FormatBenchmarkReport() output missing expected substring %q", sub)
		}
	}
}

func TestFormatBenchmarkReport_BestLatencyProviderRankedFirstInLatencySection(t *testing.T) {
	bs := buildSampleSuite(t)
	report := FormatBenchmarkReport(bs)

	latencySection := report[strings.Index(report, "LATENCY RANKINGS"):strings.Index(report, "JITTER RANKINGS")]
	firstLine := strings.Split(strings.TrimSpace(latencySection), "\n")[2] // skip header + divider
	if !strings.Contains(firstLine, "fastcom") {
		t.Errorf("expected fastcom (lowest latency, 8ms) to be first in latency rankings section, got line: %q", firstLine)
	}
	if !strings.Contains(firstLine, "100.0%") {
		t.Errorf("expected fastcom's latency percentage to read 100.0%%, got line: %q", firstLine)
	}
}

func TestFormatBenchmarkReport_MethodologySectionMentionsWeights(t *testing.T) {
	bs := buildSampleSuite(t)
	report := FormatBenchmarkReport(bs)
	for _, sub := range []string{"Download×0.30", "Upload×0.20", "Latency×0.20", "Jitter×0.10", "Reliability×0.15", "Consistency×0.05"} {
		if !strings.Contains(report, sub) {
			t.Errorf("methodology section missing %q", sub)
		}
	}
}

func TestFormatComparisonTable_ContainsHeaderAndProviders(t *testing.T) {
	bs := buildSampleSuite(t)
	table := FormatComparisonTable(bs)

	for _, sub := range []string{"QUICK COMPARISON TABLE", "Provider", "Download", "Upload", "Latency", "Score", "Grade", "fastcom", "mlab", "cloudflare"} {
		if !strings.Contains(table, sub) {
			t.Errorf("FormatComparisonTable() output missing expected substring %q", sub)
		}
	}
}

func TestFormatComparisonTable_EmptySuiteProducesHeaderOnly(t *testing.T) {
	bs := NewBenchmarkSuite()
	table := FormatComparisonTable(bs)
	if !strings.Contains(table, "QUICK COMPARISON TABLE") {
		t.Errorf("expected header even for empty suite, got: %q", table)
	}
}

func TestBenchmarkSuite_FormatBenchmarkReportMethodMatchesPackageFunc(t *testing.T) {
	// Uses a single-provider suite (rather than buildSampleSuite) so the
	// comparison isn't sensitive to tie-break ordering: with only one
	// result each, Reliability and Consistency are tied across providers
	// in the multi-provider fixture, and GetComparison's provider list is
	// built by ranging over a map (Go randomizes map iteration order), so
	// two independent report renders of a tied suite can legitimately
	// order those tied sections differently. That's pre-existing behavior
	// of GetComparison/generateRankings, not something this delegation
	// test is meant to exercise.
	bs := NewBenchmarkSuite()
	addSuccess(bs, "solo", 100, 50, 10)
	if err := bs.Finalize(); err != nil {
		t.Fatalf("Finalize() returned unexpected error: %v", err)
	}

	if bs.FormatBenchmarkReport() != FormatBenchmarkReport(bs) {
		t.Error("BenchmarkSuite.FormatBenchmarkReport() should delegate to the package-level FormatBenchmarkReport")
	}
}

func TestFormatBenchmarkReport_DoesNotPanicOnEmptySuite(t *testing.T) {
	bs := NewBenchmarkSuite()
	report := FormatBenchmarkReport(bs)
	if !strings.Contains(report, "SCORING METHODOLOGY") {
		t.Error("expected report to still render static sections for an empty suite")
	}
}

func TestFormatBenchmarkReport_IncludesDurationString(t *testing.T) {
	bs := NewBenchmarkSuite()
	addSuccess(bs, "solo", 100, 50, 10)
	bs.StartTime = time.Now().Add(-5 * time.Second)
	if err := bs.Finalize(); err != nil {
		t.Fatalf("Finalize() returned unexpected error: %v", err)
	}

	report := FormatBenchmarkReport(bs)
	if !strings.Contains(report, "Benchmark Duration:") {
		t.Error("expected report to include benchmark duration line")
	}
}
