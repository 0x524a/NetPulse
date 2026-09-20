package benchmarks

import (
	"math"
	"testing"
	"time"
)

func TestNormalizeDownload(t *testing.T) {
	tests := []struct {
		name     string
		mbps     float64
		expected float64
	}{
		{"negative", -10, 0},
		{"zero", 0, 0},
		{"small value", 30, (30.0 / 300) * 100},
		{"150 mbps", 150, 50},
		{"300 mbps", 300, 100},
		{"over 300 mbps", 400, 100},
		{"large value", 1000, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeDownload(tt.mbps)
			if math.Abs(result-tt.expected) > 0.01 {
				t.Errorf("got %f, want %f", result, tt.expected)
			}
		})
	}
}

func TestNormalizeUpload(t *testing.T) {
	tests := []struct {
		name     string
		mbps     float64
		expected float64
	}{
		{"negative", -10, 0},
		{"zero", 0, 0},
		{"small value", 15, (15.0 / 150) * 100},
		{"75 mbps", 75, 50},
		{"150 mbps", 150, 100},
		{"over 150 mbps", 200, 100},
		{"large value", 500, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeUpload(tt.mbps)
			if math.Abs(result-tt.expected) > 0.01 {
				t.Errorf("got %f, want %f", result, tt.expected)
			}
		})
	}
}

func TestNormalizeLatency(t *testing.T) {
	tests := []struct {
		name     string
		ms       float64
		expected float64
	}{
		{"negative", -10, 100},
		{"zero ms", 0, 100},
		{"50 ms", 50, 50},
		{"100 ms", 100, 0},
		{"over 100 ms", 150, 0},
		{"small value", 10, 90},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeLatency(tt.ms)
			if math.Abs(result-tt.expected) > 0.01 {
				t.Errorf("got %f, want %f", result, tt.expected)
			}
		})
	}
}

func TestNormalizeJitter(t *testing.T) {
	tests := []struct {
		name     string
		ms       float64
		expected float64
	}{
		{"negative", -10, 100},
		{"zero ms", 0, 100},
		{"25 ms", 25, 50},
		{"50 ms", 50, 0},
		{"over 50 ms", 100, 0},
		{"small value", 5, 90},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeJitter(tt.ms)
			if math.Abs(result-tt.expected) > 0.01 {
				t.Errorf("got %f, want %f", result, tt.expected)
			}
		})
	}
}

func TestCalculateConsistency(t *testing.T) {
	tests := []struct {
		name     string
		results  []ProviderResult
		minScore float64
		maxScore float64
	}{
		{
			name:     "empty results",
			results:  []ProviderResult{},
			minScore: 0,
			maxScore: 0,
		},
		{
			name: "single result",
			results: []ProviderResult{
				{DownloadMbps: 100},
			},
			minScore: 100,
			maxScore: 100,
		},
		{
			name: "consistent results",
			results: []ProviderResult{
				{DownloadMbps: 100},
				{DownloadMbps: 100},
				{DownloadMbps: 100},
			},
			minScore: 100,
			maxScore: 100,
		},
		{
			name: "variable results",
			results: []ProviderResult{
				{DownloadMbps: 50},
				{DownloadMbps: 100},
				{DownloadMbps: 150},
			},
			minScore: 0,
			maxScore: 100,
		},
		{
			name: "highly variable results",
			results: []ProviderResult{
				{DownloadMbps: 10},
				{DownloadMbps: 100},
				{DownloadMbps: 200},
			},
			minScore: 0,
			maxScore: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateConsistency(tt.results)
			if result < tt.minScore || result > tt.maxScore {
				t.Errorf("got %f, expected between %f and %f", result, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestCalculatePerformanceScore(t *testing.T) {
	tests := []struct {
		name      string
		benchmark *ProviderBenchmark
		checkFunc func(float64) bool
	}{
		{
			name: "no successful tests",
			benchmark: &ProviderBenchmark{
				SuccessfulTests: 0,
			},
			checkFunc: func(score float64) bool { return score == 0 },
		},
		{
			name: "perfect metrics",
			benchmark: &ProviderBenchmark{
				SuccessfulTests: 1,
				AvgDownloadMbps: 300,
				AvgUploadMbps:   150,
				AvgLatencyMs:    0,
				AvgJitterMs:     0,
				Reliability:     100,
				Consistency:     100,
			},
			checkFunc: func(score float64) bool { return score > 90 && score <= 100 },
		},
		{
			name: "poor metrics",
			benchmark: &ProviderBenchmark{
				SuccessfulTests: 1,
				AvgDownloadMbps: 10,
				AvgUploadMbps:   5,
				AvgLatencyMs:    200,
				AvgJitterMs:     100,
				Reliability:     50,
				Consistency:     20,
			},
			checkFunc: func(score float64) bool { return score >= 0 && score < 30 },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculatePerformanceScore(tt.benchmark)
			if !tt.checkFunc(score) {
				t.Errorf("score %f did not match expected range", score)
			}
		})
	}
}

func TestProviderResultFields(t *testing.T) {
	result := &ProviderResult{
		ProviderName:        "TestProvider",
		DownloadMbps:        100,
		UploadMbps:          50,
		Latency:             50 * time.Millisecond,
		Jitter:              10 * time.Millisecond,
		MinDownload:         90,
		MaxDownload:         110,
		AvgDownload:         100,
		MinUpload:           45,
		MaxUpload:           55,
		AvgUpload:           50,
		Success:             true,
		DownloadTime:        5 * time.Second,
		UploadTime:          10 * time.Second,
		Timestamp:           time.Now(),
		LatencyMeasurements: []time.Duration{40 * time.Millisecond, 50 * time.Millisecond, 60 * time.Millisecond},
	}

	if result.ProviderName != "TestProvider" {
		t.Errorf("expected TestProvider, got %s", result.ProviderName)
	}
	if result.DownloadMbps != 100 {
		t.Errorf("expected download 100, got %f", result.DownloadMbps)
	}
	if !result.Success {
		t.Error("expected success true")
	}
	if len(result.LatencyMeasurements) != 3 {
		t.Errorf("expected 3 latency measurements, got %d", len(result.LatencyMeasurements))
	}
}

func TestProviderBenchmarkFields(t *testing.T) {
	benchmark := &ProviderBenchmark{
		ProviderName:    "TestProvider",
		TestResults:     []ProviderResult{},
		AvgDownloadMbps: 100,
		AvgUploadMbps:   50,
		AvgLatencyMs:    50,
		AvgJitterMs:     10,
		Consistency:     85,
		Reliability:     95,
		NumTests:        10,
		SuccessfulTests: 9,
		FailedTests:     1,
		MinDownloadMbps: 90,
		MaxDownloadMbps: 110,
		MinUploadMbps:   45,
		MaxUploadMbps:   55,
		MinLatencyMs:    40,
		MaxLatencyMs:    60,
	}

	if benchmark.ProviderName != "TestProvider" {
		t.Errorf("expected TestProvider, got %s", benchmark.ProviderName)
	}
	if benchmark.NumTests != 10 {
		t.Errorf("expected 10 tests, got %d", benchmark.NumTests)
	}
	if benchmark.SuccessfulTests != 9 {
		t.Errorf("expected 9 successful, got %d", benchmark.SuccessfulTests)
	}
	if benchmark.Reliability != 95 {
		t.Errorf("expected 95%% reliability, got %f", benchmark.Reliability)
	}
}

func TestBenchmarkSuiteFields(t *testing.T) {
	suite := &BenchmarkSuite{
		Benchmarks:          make(map[string]*ProviderBenchmark),
		OrderedProviders:    []string{"Provider1", "Provider2"},
		StartTime:           time.Now(),
		EndTime:             time.Now().Add(10 * time.Second),
		TotalDuration:       10 * time.Second,
		ProviderScores:      make(map[string]float64),
		DownloadRankings:    []string{"Provider2", "Provider1"},
		UploadRankings:      []string{"Provider2", "Provider1"},
		LatencyRankings:     []string{"Provider1", "Provider2"},
		ReliabilityRankings: []string{"Provider2", "Provider1"},
		ConsistencyRankings: []string{"Provider1", "Provider2"},
	}

	if len(suite.OrderedProviders) != 2 {
		t.Errorf("expected 2 ordered providers, got %d", len(suite.OrderedProviders))
	}
	if suite.TotalDuration != 10*time.Second {
		t.Errorf("expected 10s duration, got %v", suite.TotalDuration)
	}
}

func TestDetailedComparisonFields(t *testing.T) {
	comp := &DetailedComparison{
		MetricName: "download",
		Providers: []ProviderComparisonPoint{
			{ProviderName: "Provider1", Value: 100, Rank: 1, Percentage: 100},
			{ProviderName: "Provider2", Value: 50, Rank: 2, Percentage: 50},
		},
	}

	if comp.MetricName != "download" {
		t.Errorf("expected download metric, got %s", comp.MetricName)
	}
	if len(comp.Providers) != 2 {
		t.Errorf("expected 2 providers, got %d", len(comp.Providers))
	}
	if comp.Providers[0].Rank != 1 {
		t.Errorf("expected rank 1, got %d", comp.Providers[0].Rank)
	}
	if comp.Providers[0].Percentage != 100 {
		t.Errorf("expected 100%%, got %f", comp.Providers[0].Percentage)
	}
}

func TestPerformanceScoreFields(t *testing.T) {
	score := &PerformanceScore{
		ProviderName:     "TestProvider",
		DownloadScore:    85,
		UploadScore:      75,
		LatencyScore:     90,
		JitterScore:      85,
		ReliabilityScore: 95,
		ConsistencyScore: 80,
		OverallScore:     85.5,
	}

	if score.ProviderName != "TestProvider" {
		t.Errorf("expected TestProvider, got %s", score.ProviderName)
	}
	if score.DownloadScore != 85 {
		t.Errorf("expected 85, got %f", score.DownloadScore)
	}
	if score.OverallScore != 85.5 {
		t.Errorf("expected 85.5, got %f", score.OverallScore)
	}
}

func TestProviderComparisonPointFields(t *testing.T) {
	point := ProviderComparisonPoint{
		ProviderName: "Provider1",
		Value:        150,
		Rank:         1,
		Percentage:   100,
	}

	if point.ProviderName != "Provider1" {
		t.Errorf("expected Provider1, got %s", point.ProviderName)
	}
	if point.Value != 150 {
		t.Errorf("expected 150, got %f", point.Value)
	}
	if point.Rank != 1 {
		t.Errorf("expected rank 1, got %d", point.Rank)
	}
	if point.Percentage != 100 {
		t.Errorf("expected 100%%, got %f", point.Percentage)
	}
}
