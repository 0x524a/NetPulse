package benchmarks

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// NewBenchmarkSuite creates a new benchmark suite
func NewBenchmarkSuite() *BenchmarkSuite {
	return &BenchmarkSuite{
		Benchmarks:          make(map[string]*ProviderBenchmark),
		ProviderScores:      make(map[string]float64),
		DownloadRankings:    []string{},
		UploadRankings:      []string{},
		LatencyRankings:     []string{},
		ReliabilityRankings: []string{},
		ConsistencyRankings: []string{},
		StartTime:           time.Now(),
	}
}

// AddResult adds a test result to the benchmark suite
func (bs *BenchmarkSuite) AddResult(result *ProviderResult) {
	providerName := result.ProviderName

	if bs.Benchmarks[providerName] == nil {
		bs.Benchmarks[providerName] = &ProviderBenchmark{
			ProviderName:    providerName,
			TestResults:     []ProviderResult{},
			MinDownloadMbps: math.MaxFloat64,
			MinUploadMbps:   math.MaxFloat64,
			MinLatencyMs:    math.MaxFloat64,
		}
	}

	benchmark := bs.Benchmarks[providerName]
	benchmark.TestResults = append(benchmark.TestResults, *result)
	benchmark.NumTests++

	if result.Success {
		benchmark.SuccessfulTests++
	} else {
		benchmark.FailedTests++
	}
}

// Finalize computes aggregate statistics and rankings
func (bs *BenchmarkSuite) Finalize() error {
	bs.EndTime = time.Now()
	bs.TotalDuration = bs.EndTime.Sub(bs.StartTime)

	// Calculate aggregates for each provider
	for providerName, benchmark := range bs.Benchmarks {
		if benchmark.NumTests == 0 {
			continue
		}

		// Calculate averages and min/max
		var (
			sumDownload, sumUpload, sumLatency, sumJitter float64
			successResults                                []ProviderResult
		)

		for _, result := range benchmark.TestResults {
			if result.Success {
				successResults = append(successResults, result)
				sumDownload += result.DownloadMbps
				sumUpload += result.UploadMbps
				sumLatency += result.Latency.Seconds() * 1000
				sumJitter += float64(result.Jitter.Milliseconds())

				if result.DownloadMbps < benchmark.MinDownloadMbps {
					benchmark.MinDownloadMbps = result.DownloadMbps
				}
				if result.DownloadMbps > benchmark.MaxDownloadMbps {
					benchmark.MaxDownloadMbps = result.DownloadMbps
				}
				if result.UploadMbps < benchmark.MinUploadMbps {
					benchmark.MinUploadMbps = result.UploadMbps
				}
				if result.UploadMbps > benchmark.MaxUploadMbps {
					benchmark.MaxUploadMbps = result.UploadMbps
				}
				if result.Latency.Milliseconds() < int64(benchmark.MinLatencyMs) {
					benchmark.MinLatencyMs = float64(result.Latency.Milliseconds())
				}
				if result.Latency.Milliseconds() > int64(benchmark.MaxLatencyMs) {
					benchmark.MaxLatencyMs = float64(result.Latency.Milliseconds())
				}
			}
		}

		if len(successResults) == 0 {
			benchmark.Reliability = 0
			continue
		}

		benchmark.AvgDownloadMbps = sumDownload / float64(len(successResults))
		benchmark.AvgUploadMbps = sumUpload / float64(len(successResults))
		benchmark.AvgLatencyMs = sumLatency / float64(len(successResults))
		benchmark.AvgJitterMs = sumJitter / float64(len(successResults))
		benchmark.Reliability = float64(benchmark.SuccessfulTests) / float64(benchmark.NumTests) * 100

		// Calculate consistency (inverse of coefficient of variation)
		benchmark.Consistency = calculateConsistency(successResults)

		// Calculate overall performance score
		score := calculatePerformanceScore(benchmark)
		bs.ProviderScores[providerName] = score
	}

	// Generate rankings
	bs.generateRankings()

	return nil
}

// calculatePerformanceScore calculates an overall performance score for a provider
func calculatePerformanceScore(benchmark *ProviderBenchmark) float64 {
	if benchmark.SuccessfulTests == 0 {
		return 0
	}

	// Normalize metrics to 0-100 scale
	downloadScore := normalizeDownload(benchmark.AvgDownloadMbps)
	uploadScore := normalizeUpload(benchmark.AvgUploadMbps)
	latencyScore := normalizeLatency(benchmark.AvgLatencyMs)
	jitterScore := normalizeJitter(benchmark.AvgJitterMs)
	reliabilityScore := benchmark.Reliability
	consistencyScore := benchmark.Consistency

	// Weighted average (adjustable weights)
	weights := map[string]float64{
		"download":    0.30,
		"upload":      0.20,
		"latency":     0.20,
		"jitter":      0.10,
		"reliability": 0.15,
		"consistency": 0.05,
	}

	score := (downloadScore * weights["download"]) +
		(uploadScore * weights["upload"]) +
		(latencyScore * weights["latency"]) +
		(jitterScore * weights["jitter"]) +
		(reliabilityScore * weights["reliability"]) +
		(consistencyScore * weights["consistency"])

	return score
}

// normalizeDownload normalizes download speed to 0-100 scale
// Assumes 300 Mbps is excellent (100), 0 Mbps is poor (0)
func normalizeDownload(mbps float64) float64 {
	if mbps < 0 {
		return 0
	}
	if mbps > 300 {
		return 100
	}
	return (mbps / 300) * 100
}

// normalizeUpload normalizes upload speed to 0-100 scale
// Assumes 150 Mbps is excellent (100), 0 Mbps is poor (0)
func normalizeUpload(mbps float64) float64 {
	if mbps < 0 {
		return 0
	}
	if mbps > 150 {
		return 100
	}
	return (mbps / 150) * 100
}

// normalizeLatency normalizes latency to 0-100 scale (inverted)
// 0ms = 100, 100ms = 0
func normalizeLatency(ms float64) float64 {
	if ms < 0 {
		return 100
	}
	if ms > 100 {
		return 0
	}
	return 100 - ms
}

// normalizeJitter normalizes jitter to 0-100 scale (inverted)
// 0ms = 100, 50ms = 0
func normalizeJitter(ms float64) float64 {
	if ms < 0 {
		return 100
	}
	if ms > 50 {
		return 0
	}
	return 100 - (ms * 2)
}

// calculateConsistency calculates consistency score based on standard deviation
func calculateConsistency(results []ProviderResult) float64 {
	if len(results) == 0 {
		return 0
	}

	// Calculate average download speed
	var sum float64
	for _, r := range results {
		sum += r.DownloadMbps
	}
	avg := sum / float64(len(results))

	// Calculate standard deviation
	var sumSq float64
	for _, r := range results {
		diff := r.DownloadMbps - avg
		sumSq += diff * diff
	}
	stdDev := math.Sqrt(sumSq / float64(len(results)))

	// Calculate coefficient of variation (stddev / mean)
	var cv float64
	if avg > 0 {
		cv = stdDev / avg
	}

	// Convert to consistency score (0-100)
	// Lower CV = higher consistency
	// CV of 0.1 (10%) = 90 points, CV of 1.0 = 0 points
	consistency := math.Max(0, 100-(cv*100))
	return math.Min(100, consistency)
}

// generateRankings generates rankings for each metric
func (bs *BenchmarkSuite) generateRankings() {
	providers := make([]string, 0, len(bs.Benchmarks))
	for name := range bs.Benchmarks {
		providers = append(providers, name)
	}

	// Sort by download speed
	sort.Slice(providers, func(i, j int) bool {
		return bs.Benchmarks[providers[i]].AvgDownloadMbps > bs.Benchmarks[providers[j]].AvgDownloadMbps
	})
	bs.DownloadRankings = providers

	// Sort by upload speed
	sort.Slice(providers, func(i, j int) bool {
		return bs.Benchmarks[providers[i]].AvgUploadMbps > bs.Benchmarks[providers[j]].AvgUploadMbps
	})
	bs.UploadRankings = providers

	// Sort by latency (lower is better)
	sort.Slice(providers, func(i, j int) bool {
		return bs.Benchmarks[providers[i]].AvgLatencyMs < bs.Benchmarks[providers[j]].AvgLatencyMs
	})
	bs.LatencyRankings = providers

	// Sort by reliability
	sort.Slice(providers, func(i, j int) bool {
		return bs.Benchmarks[providers[i]].Reliability > bs.Benchmarks[providers[j]].Reliability
	})
	bs.ReliabilityRankings = providers

	// Sort by consistency
	sort.Slice(providers, func(i, j int) bool {
		return bs.Benchmarks[providers[i]].Consistency > bs.Benchmarks[providers[j]].Consistency
	})
	bs.ConsistencyRankings = providers

	// Sort by overall performance score
	sort.Slice(providers, func(i, j int) bool {
		return bs.ProviderScores[providers[i]] > bs.ProviderScores[providers[j]]
	})
	bs.OrderedProviders = providers
}

// GetBestProvider returns the best performing provider
func (bs *BenchmarkSuite) GetBestProvider() string {
	if len(bs.OrderedProviders) == 0 {
		return ""
	}
	return bs.OrderedProviders[0]
}

// GetProviderScore returns the performance score for a specific provider
func (bs *BenchmarkSuite) GetProviderScore(providerName string) float64 {
	return bs.ProviderScores[providerName]
}

// GetComparison returns a detailed comparison for a specific metric
func (bs *BenchmarkSuite) GetComparison(metric string) *DetailedComparison {
	comp := &DetailedComparison{
		MetricName: metric,
		Providers:  []ProviderComparisonPoint{},
	}

	var maxValue float64
	for _, benchmark := range bs.Benchmarks {
		var value float64
		switch metric {
		case "download":
			value = benchmark.AvgDownloadMbps
		case "upload":
			value = benchmark.AvgUploadMbps
		case "latency":
			value = benchmark.AvgLatencyMs
		case "jitter":
			value = benchmark.AvgJitterMs
		case "reliability":
			value = benchmark.Reliability
		case "consistency":
			value = benchmark.Consistency
		default:
			continue
		}

		if value > maxValue {
			maxValue = value
		}

		comp.Providers = append(comp.Providers, ProviderComparisonPoint{
			ProviderName: benchmark.ProviderName,
			Value:        value,
		})
	}

	// Calculate rank and percentage
	if metric == "latency" || metric == "jitter" {
		// For latency/jitter, lower is better
		sort.Slice(comp.Providers, func(i, j int) bool {
			return comp.Providers[i].Value < comp.Providers[j].Value
		})
	} else {
		// For other metrics, higher is better
		sort.Slice(comp.Providers, func(i, j int) bool {
			return comp.Providers[i].Value > comp.Providers[j].Value
		})
	}

	for i := range comp.Providers {
		comp.Providers[i].Rank = i + 1
		if maxValue > 0 {
			comp.Providers[i].Percentage = (comp.Providers[i].Value / maxValue) * 100
		}
	}

	return comp
}

// Summary returns a string summary of the benchmark results
func (bs *BenchmarkSuite) Summary() string {
	if len(bs.OrderedProviders) == 0 {
		return "No benchmark results available"
	}

	summary := fmt.Sprintf("Benchmark Summary (%s)\n", bs.TotalDuration)
	summary += fmt.Sprintf("Winner: %s (Score: %.2f/100)\n", bs.GetBestProvider(), bs.ProviderScores[bs.GetBestProvider()])
	summary += fmt.Sprintf("Total Tests: %d providers\n", len(bs.Benchmarks))

	return summary
}

// FormatBenchmarkReport generates a formatted report of benchmark results
func (bs *BenchmarkSuite) FormatBenchmarkReport() string {
	return FormatBenchmarkReport(bs)
}
