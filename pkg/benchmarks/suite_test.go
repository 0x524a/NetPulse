package benchmarks

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestNewBenchmarkSuite(t *testing.T) {
	suite := NewBenchmarkSuite()

	if suite.Benchmarks == nil {
		t.Error("Benchmarks map is nil")
	}
	if suite.ProviderScores == nil {
		t.Error("ProviderScores map is nil")
	}
	if suite.DownloadRankings == nil {
		t.Error("DownloadRankings is nil")
	}
	if suite.UploadRankings == nil {
		t.Error("UploadRankings is nil")
	}
	if suite.LatencyRankings == nil {
		t.Error("LatencyRankings is nil")
	}
	if suite.ReliabilityRankings == nil {
		t.Error("ReliabilityRankings is nil")
	}
	if suite.ConsistencyRankings == nil {
		t.Error("ConsistencyRankings is nil")
	}
	if suite.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}
}

func TestAddResult(t *testing.T) {
	suite := NewBenchmarkSuite()

	result := &ProviderResult{
		ProviderName: "Provider1",
		DownloadMbps: 100,
		UploadMbps:   50,
		Latency:      50 * time.Millisecond,
		Jitter:       10 * time.Millisecond,
		Success:      true,
	}

	suite.AddResult(result)

	if len(suite.Benchmarks) != 1 {
		t.Errorf("expected 1 benchmark, got %d", len(suite.Benchmarks))
	}

	benchmark := suite.Benchmarks["Provider1"]
	if benchmark == nil {
		t.Fatal("benchmark for Provider1 is nil")
	}
	if benchmark.ProviderName != "Provider1" {
		t.Errorf("expected Provider1, got %s", benchmark.ProviderName)
	}
	if benchmark.NumTests != 1 {
		t.Errorf("expected 1 test, got %d", benchmark.NumTests)
	}
	if benchmark.SuccessfulTests != 1 {
		t.Errorf("expected 1 successful test, got %d", benchmark.SuccessfulTests)
	}
	if len(benchmark.TestResults) != 1 {
		t.Errorf("expected 1 result, got %d", len(benchmark.TestResults))
	}
}

func TestAddResultMultipleProviders(t *testing.T) {
	suite := NewBenchmarkSuite()

	results := []*ProviderResult{
		{ProviderName: "Provider1", DownloadMbps: 100, UploadMbps: 50, Latency: 50 * time.Millisecond, Jitter: 10 * time.Millisecond, Success: true},
		{ProviderName: "Provider2", DownloadMbps: 150, UploadMbps: 75, Latency: 30 * time.Millisecond, Jitter: 5 * time.Millisecond, Success: true},
		{ProviderName: "Provider1", DownloadMbps: 110, UploadMbps: 55, Latency: 45 * time.Millisecond, Jitter: 12 * time.Millisecond, Success: true},
	}

	for _, r := range results {
		suite.AddResult(r)
	}

	if len(suite.Benchmarks) != 2 {
		t.Errorf("expected 2 providers, got %d", len(suite.Benchmarks))
	}

	p1 := suite.Benchmarks["Provider1"]
	if p1.NumTests != 2 {
		t.Errorf("expected Provider1 to have 2 tests, got %d", p1.NumTests)
	}

	p2 := suite.Benchmarks["Provider2"]
	if p2.NumTests != 1 {
		t.Errorf("expected Provider2 to have 1 test, got %d", p2.NumTests)
	}
}

func TestAddResultWithFailures(t *testing.T) {
	suite := NewBenchmarkSuite()

	results := []*ProviderResult{
		{ProviderName: "Provider1", Success: true},
		{ProviderName: "Provider1", Success: false, Error: "timeout"},
		{ProviderName: "Provider1", Success: true},
	}

	for _, r := range results {
		suite.AddResult(r)
	}

	benchmark := suite.Benchmarks["Provider1"]
	if benchmark.NumTests != 3 {
		t.Errorf("expected 3 tests, got %d", benchmark.NumTests)
	}
	if benchmark.SuccessfulTests != 2 {
		t.Errorf("expected 2 successful tests, got %d", benchmark.SuccessfulTests)
	}
	if benchmark.FailedTests != 1 {
		t.Errorf("expected 1 failed test, got %d", benchmark.FailedTests)
	}
}

func TestFinalizeBasic(t *testing.T) {
	suite := NewBenchmarkSuite()

	result := &ProviderResult{
		ProviderName: "Provider1",
		DownloadMbps: 100,
		UploadMbps:   50,
		Latency:      50 * time.Millisecond,
		Jitter:       10 * time.Millisecond,
		Success:      true,
	}

	suite.AddResult(result)
	err := suite.Finalize()

	if err != nil {
		t.Errorf("Finalize failed: %v", err)
	}

	if suite.EndTime.IsZero() {
		t.Error("EndTime should be set")
	}
	if suite.TotalDuration == 0 {
		t.Error("TotalDuration should be set")
	}

	benchmark := suite.Benchmarks["Provider1"]
	if benchmark.AvgDownloadMbps != 100 {
		t.Errorf("expected avg download 100, got %f", benchmark.AvgDownloadMbps)
	}
	if benchmark.AvgUploadMbps != 50 {
		t.Errorf("expected avg upload 50, got %f", benchmark.AvgUploadMbps)
	}
	if benchmark.Reliability != 100 {
		t.Errorf("expected 100%% reliability, got %f", benchmark.Reliability)
	}
}

func TestFinalizeMultipleResults(t *testing.T) {
	suite := NewBenchmarkSuite()

	results := []*ProviderResult{
		{ProviderName: "Provider1", DownloadMbps: 100, UploadMbps: 50, Latency: 50 * time.Millisecond, Jitter: 10 * time.Millisecond, Success: true},
		{ProviderName: "Provider1", DownloadMbps: 110, UploadMbps: 55, Latency: 60 * time.Millisecond, Jitter: 12 * time.Millisecond, Success: true},
		{ProviderName: "Provider1", DownloadMbps: 120, UploadMbps: 60, Latency: 40 * time.Millisecond, Jitter: 8 * time.Millisecond, Success: true},
	}

	for _, r := range results {
		suite.AddResult(r)
	}

	_ = suite.Finalize()

	benchmark := suite.Benchmarks["Provider1"]
	expectedAvgDL := (100.0 + 110.0 + 120.0) / 3.0
	if diff := math.Abs(benchmark.AvgDownloadMbps - expectedAvgDL); diff > 0.01 {
		t.Errorf("expected avg download %f, got %f", expectedAvgDL, benchmark.AvgDownloadMbps)
	}

	expectedAvgUL := (50.0 + 55.0 + 60.0) / 3.0
	if diff := math.Abs(benchmark.AvgUploadMbps - expectedAvgUL); diff > 0.01 {
		t.Errorf("expected avg upload %f, got %f", expectedAvgUL, benchmark.AvgUploadMbps)
	}
}

func TestFinalizeMinMaxValues(t *testing.T) {
	suite := NewBenchmarkSuite()

	results := []*ProviderResult{
		{ProviderName: "Provider1", DownloadMbps: 80, UploadMbps: 40, Latency: 30 * time.Millisecond, Jitter: 5 * time.Millisecond, Success: true},
		{ProviderName: "Provider1", DownloadMbps: 150, UploadMbps: 70, Latency: 100 * time.Millisecond, Jitter: 10 * time.Millisecond, Success: true},
	}

	for _, r := range results {
		suite.AddResult(r)
	}

	_ = suite.Finalize()

	benchmark := suite.Benchmarks["Provider1"]
	if benchmark.MinDownloadMbps != 80 {
		t.Errorf("expected min download 80, got %f", benchmark.MinDownloadMbps)
	}
	if benchmark.MaxDownloadMbps != 150 {
		t.Errorf("expected max download 150, got %f", benchmark.MaxDownloadMbps)
	}
	if benchmark.MinUploadMbps != 40 {
		t.Errorf("expected min upload 40, got %f", benchmark.MinUploadMbps)
	}
	if benchmark.MaxUploadMbps != 70 {
		t.Errorf("expected max upload 70, got %f", benchmark.MaxUploadMbps)
	}
	if benchmark.MinLatencyMs != 30 {
		t.Errorf("expected min latency 30, got %f", benchmark.MinLatencyMs)
	}
	if benchmark.MaxLatencyMs != 100 {
		t.Errorf("expected max latency 100, got %f", benchmark.MaxLatencyMs)
	}
}

func TestFinalizeReliability(t *testing.T) {
	suite := NewBenchmarkSuite()

	results := []*ProviderResult{
		{ProviderName: "Provider1", DownloadMbps: 100, UploadMbps: 50, Latency: 50 * time.Millisecond, Success: true},
		{ProviderName: "Provider1", DownloadMbps: 0, UploadMbps: 0, Latency: 0, Success: false},
		{ProviderName: "Provider1", DownloadMbps: 110, UploadMbps: 55, Latency: 40 * time.Millisecond, Success: true},
		{ProviderName: "Provider1", DownloadMbps: 0, UploadMbps: 0, Latency: 0, Success: false},
	}

	for _, r := range results {
		suite.AddResult(r)
	}

	_ = suite.Finalize()

	benchmark := suite.Benchmarks["Provider1"]
	expectedReliability := 50.0
	if diff := math.Abs(benchmark.Reliability - expectedReliability); diff > 0.01 {
		t.Errorf("expected reliability %f, got %f", expectedReliability, benchmark.Reliability)
	}
}

func TestGetBestProvider(t *testing.T) {
	suite := NewBenchmarkSuite()

	results := []*ProviderResult{
		{ProviderName: "Provider1", DownloadMbps: 100, UploadMbps: 50, Latency: 50 * time.Millisecond, Jitter: 10 * time.Millisecond, Success: true},
		{ProviderName: "Provider2", DownloadMbps: 200, UploadMbps: 100, Latency: 20 * time.Millisecond, Jitter: 5 * time.Millisecond, Success: true},
	}

	for _, r := range results {
		suite.AddResult(r)
	}

	_ = suite.Finalize()

	best := suite.GetBestProvider()
	if best != "Provider2" {
		t.Errorf("expected Provider2 as best, got %s", best)
	}
}

func TestGetBestProviderEmpty(t *testing.T) {
	suite := NewBenchmarkSuite()
	_ = suite.Finalize()

	best := suite.GetBestProvider()
	if best != "" {
		t.Errorf("expected empty string for empty suite, got %s", best)
	}
}

func TestGetProviderScore(t *testing.T) {
	suite := NewBenchmarkSuite()

	result := &ProviderResult{
		ProviderName: "Provider1",
		DownloadMbps: 300,
		UploadMbps:   150,
		Latency:      0 * time.Millisecond,
		Jitter:       0 * time.Millisecond,
		Success:      true,
	}

	suite.AddResult(result)
	_ = suite.Finalize()

	score := suite.GetProviderScore("Provider1")
	if score == 0 {
		t.Error("expected non-zero score")
	}
	if score > 100 {
		t.Errorf("expected score <= 100, got %f", score)
	}
}

func TestGetComparison(t *testing.T) {
	suite := NewBenchmarkSuite()

	results := []*ProviderResult{
		{ProviderName: "Provider1", DownloadMbps: 100, UploadMbps: 50, Latency: 50 * time.Millisecond, Jitter: 10 * time.Millisecond, Success: true},
		{ProviderName: "Provider2", DownloadMbps: 150, UploadMbps: 75, Latency: 30 * time.Millisecond, Jitter: 5 * time.Millisecond, Success: true},
	}

	for _, r := range results {
		suite.AddResult(r)
	}

	_ = suite.Finalize()

	comp := suite.GetComparison("download")

	if comp.MetricName != "download" {
		t.Errorf("expected metric 'download', got %s", comp.MetricName)
	}
	if len(comp.Providers) != 2 {
		t.Errorf("expected 2 providers, got %d", len(comp.Providers))
	}
	if comp.Providers[0].Value < comp.Providers[1].Value {
		t.Error("expected providers sorted by value descending for download")
	}
	if comp.Providers[0].Rank != 1 {
		t.Errorf("expected first provider rank 1, got %d", comp.Providers[0].Rank)
	}
	if comp.Providers[0].Percentage != 100 {
		t.Errorf("expected best provider at 100%%, got %f", comp.Providers[0].Percentage)
	}
}

func TestGetComparisonLatency(t *testing.T) {
	suite := NewBenchmarkSuite()

	results := []*ProviderResult{
		{ProviderName: "Provider1", DownloadMbps: 100, UploadMbps: 50, Latency: 50 * time.Millisecond, Jitter: 10 * time.Millisecond, Success: true},
		{ProviderName: "Provider2", DownloadMbps: 150, UploadMbps: 75, Latency: 30 * time.Millisecond, Jitter: 5 * time.Millisecond, Success: true},
	}

	for _, r := range results {
		suite.AddResult(r)
	}

	_ = suite.Finalize()

	comp := suite.GetComparison("latency")

	if comp.Providers[0].Value > comp.Providers[1].Value {
		t.Error("expected providers sorted by value ascending for latency")
	}
}

func TestSummary(t *testing.T) {
	suite := NewBenchmarkSuite()

	result := &ProviderResult{
		ProviderName: "Provider1",
		DownloadMbps: 100,
		UploadMbps:   50,
		Latency:      50 * time.Millisecond,
		Jitter:       10 * time.Millisecond,
		Success:      true,
	}

	suite.AddResult(result)
	_ = suite.Finalize()

	summary := suite.Summary()

	if !strings.Contains(summary, "Provider1") {
		t.Error("summary should contain provider name")
	}
	if !strings.Contains(summary, "Benchmark Summary") {
		t.Error("summary should contain 'Benchmark Summary'")
	}
	if !strings.Contains(summary, "Winner") {
		t.Error("summary should contain 'Winner'")
	}
}

func TestSummaryEmpty(t *testing.T) {
	suite := NewBenchmarkSuite()
	_ = suite.Finalize()

	summary := suite.Summary()

	if summary != "No benchmark results available" {
		t.Errorf("expected 'No benchmark results available', got %q", summary)
	}
}
