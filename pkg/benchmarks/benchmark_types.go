package benchmarks

import (
	"time"
)

// ProviderResult holds a single test result from a provider
type ProviderResult struct {
	ProviderName        string
	DownloadMbps        float64
	UploadMbps          float64
	Latency             time.Duration
	Jitter              time.Duration
	MinDownload         float64
	MaxDownload         float64
	AvgDownload         float64
	MinUpload           float64
	MaxUpload           float64
	AvgUpload           float64
	Success             bool
	Error               string
	DownloadTime        time.Duration
	UploadTime          time.Duration
	Timestamp           time.Time
	LatencyMeasurements []time.Duration
}

// ProviderBenchmark holds performance metrics for a single provider
type ProviderBenchmark struct {
	ProviderName    string
	TestResults     []ProviderResult // All test results for this provider
	AvgDownloadMbps float64
	AvgUploadMbps   float64
	AvgLatencyMs    float64
	AvgJitterMs     float64
	Consistency     float64 // 0-100 score based on stddev of results
	Reliability     float64 // Percentage of successful tests
	TestDuration    time.Duration
	NumTests        int
	SuccessfulTests int
	FailedTests     int
	MinDownloadMbps float64
	MaxDownloadMbps float64
	MinUploadMbps   float64
	MaxUploadMbps   float64
	MinLatencyMs    float64
	MaxLatencyMs    float64
}

// BenchmarkSuite holds results for all providers
type BenchmarkSuite struct {
	Benchmarks          map[string]*ProviderBenchmark
	OrderedProviders    []string // Sorted by performance score
	StartTime           time.Time
	EndTime             time.Time
	TotalDuration       time.Duration
	ProviderScores      map[string]float64 // Overall performance score per provider
	DownloadRankings    []string           // Ranked by download speed
	UploadRankings      []string           // Ranked by upload speed
	LatencyRankings     []string           // Ranked by latency (best first)
	ReliabilityRankings []string           // Ranked by reliability
	ConsistencyRankings []string           // Ranked by consistency
}

// DetailedComparison provides rich comparison data
type DetailedComparison struct {
	MetricName string
	Providers  []ProviderComparisonPoint
}

type ProviderComparisonPoint struct {
	ProviderName string
	Value        float64
	Rank         int
	Percentage   float64 // Percentage relative to best performer
}

// PerformanceScore holds the overall performance rating
type PerformanceScore struct {
	ProviderName     string
	DownloadScore    float64 // 0-100
	UploadScore      float64 // 0-100
	LatencyScore     float64 // 0-100 (inverted: lower latency = higher score)
	JitterScore      float64 // 0-100 (inverted: lower jitter = higher score)
	ReliabilityScore float64 // 0-100
	ConsistencyScore float64 // 0-100
	OverallScore     float64 // 0-100 (weighted average)
}
