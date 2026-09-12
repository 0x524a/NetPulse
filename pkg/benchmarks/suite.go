package benchmarks

import (
	"errors"
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

	if len(bs.Benchmarks) == 0 {
		return errors.New("cannot finalize benchmark suite: no benchmarks recorded")
	}

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
				latencyMs := float64(result.Latency.Milliseconds())
				if latencyMs < benchmark.MinLatencyMs {
					benchmark.MinLatencyMs = latencyMs
				}
				if latencyMs > benchmark.MaxLatencyMs {
					benchmark.MaxLatencyMs = latencyMs
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

	// Sort the base slice by name so that the (non-stable) metric sorts below
	// start from a deterministic order. Without this, Go's randomized map
	// iteration makes the relative order of providers tied on a metric vary
	// between runs of identical data.
	sort.Strings(providers)

	// Each ranking below gets its own copy of the base slice. sort.Slice
	// mutates in place, so without a copy every ranking field would end up
	// sharing one backing array and all six fields would collapse onto
	// whichever sort ran last (see BENCHMARK_FEATURES.md's "Seven Separate
	// Rankings" feature).
	newCopy := func() []string {
		c := make([]string, len(providers))
		copy(c, providers)
		return c
	}

	// Sort by download speed
	downloadRankings := newCopy()
	sort.SliceStable(downloadRankings, func(i, j int) bool {
		return bs.Benchmarks[downloadRankings[i]].AvgDownloadMbps > bs.Benchmarks[downloadRankings[j]].AvgDownloadMbps
	})
	bs.DownloadRankings = downloadRankings

	// Sort by upload speed
	uploadRankings := newCopy()
	sort.SliceStable(uploadRankings, func(i, j int) bool {
		return bs.Benchmarks[uploadRankings[i]].AvgUploadMbps > bs.Benchmarks[uploadRankings[j]].AvgUploadMbps
	})
	bs.UploadRankings = uploadRankings

	// Sort by latency (lower is better)
	latencyRankings := newCopy()
	sort.SliceStable(latencyRankings, func(i, j int) bool {
		return bs.Benchmarks[latencyRankings[i]].AvgLatencyMs < bs.Benchmarks[latencyRankings[j]].AvgLatencyMs
	})
	bs.LatencyRankings = latencyRankings

	// Sort by reliability
	reliabilityRankings := newCopy()
	sort.SliceStable(reliabilityRankings, func(i, j int) bool {
		return bs.Benchmarks[reliabilityRankings[i]].Reliability > bs.Benchmarks[reliabilityRankings[j]].Reliability
	})
	bs.ReliabilityRankings = reliabilityRankings

	// Sort by consistency
	consistencyRankings := newCopy()
	sort.SliceStable(consistencyRankings, func(i, j int) bool {
		return bs.Benchmarks[consistencyRankings[i]].Consistency > bs.Benchmarks[consistencyRankings[j]].Consistency
	})
	bs.ConsistencyRankings = consistencyRankings

	// Sort by overall performance score
	orderedProviders := newCopy()
	sort.SliceStable(orderedProviders, func(i, j int) bool {
		return bs.ProviderScores[orderedProviders[i]] > bs.ProviderScores[orderedProviders[j]]
	})
	bs.OrderedProviders = orderedProviders
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
	minValue := math.MaxFloat64

	// Iterate in sorted name order rather than map order: Go randomizes map
	// iteration per call, and the sorts below are not stable, so providers
	// tied on a metric would otherwise be ranked differently between runs of
	// the same data.
	names := make([]string, 0, len(bs.Benchmarks))
	for name := range bs.Benchmarks {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		benchmark := bs.Benchmarks[name]
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
		if value < minValue {
			minValue = value
		}

		comp.Providers = append(comp.Providers, ProviderComparisonPoint{
			ProviderName: benchmark.ProviderName,
			Value:        value,
		})
	}

	// Calculate rank and percentage
	// SliceStable so that providers tied on this metric keep the sorted-name
	// order established above, making the ranking reproducible.
	if metric == "latency" || metric == "jitter" {
		// For latency/jitter, lower is better
		sort.SliceStable(comp.Providers, func(i, j int) bool {
			return comp.Providers[i].Value < comp.Providers[j].Value
		})
	} else {
		// For other metrics, higher is better
		sort.SliceStable(comp.Providers, func(i, j int) bool {
			return comp.Providers[i].Value > comp.Providers[j].Value
		})
	}

	for i := range comp.Providers {
		comp.Providers[i].Rank = i + 1

		if metric == "latency" || metric == "jitter" {
			// Lower is better for these metrics, so the percentage must be
			// relative to the best (lowest) value: the best provider gets
			// 100%, and slower providers get proportionally less - matching
			// BENCHMARK_FEATURES.md's documented semantics and sample
			// output. Dividing by maxValue (the worst provider) as before
			// inverted this, giving the best provider the lowest percentage.
			v := comp.Providers[i].Value
			switch {
			case v > 0:
				comp.Providers[i].Percentage = (minValue / v) * 100
			case minValue == 0:
				// All providers tie at 0ms - all are equally best.
				comp.Providers[i].Percentage = 100
			}
		} else if maxValue > 0 {
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
