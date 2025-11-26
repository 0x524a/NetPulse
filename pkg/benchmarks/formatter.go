package benchmarks

import (
	"fmt"
	"strings"
)

// FormatBenchmarkReport generates a formatted report of benchmark results
func FormatBenchmarkReport(suite *BenchmarkSuite) string {
	var report strings.Builder

	report.WriteString("\n")
	report.WriteString("╔════════════════════════════════════════════════════════════════╗\n")
	report.WriteString("║          SPEED TEST PROVIDER BENCHMARK RESULTS                  ║\n")
	report.WriteString("╚════════════════════════════════════════════════════════════════╝\n")
	report.WriteString("\n")

	// Overall Rankings
	report.WriteString("OVERALL PERFORMANCE RANKINGS\n")
	report.WriteString("────────────────────────────────────────────────────────────────\n")
	for i, provider := range suite.OrderedProviders {
		score := suite.ProviderScores[provider]
		grade := getScoreGrade(score)
		report.WriteString(fmt.Sprintf("%d. %-20s Score: %6.2f/100  Grade: %s\n", i+1, provider, score, grade))
	}
	report.WriteString("\n")

	// Download Speed Rankings
	report.WriteString("DOWNLOAD SPEED RANKINGS\n")
	report.WriteString("────────────────────────────────────────────────────────────────\n")
	downloadComp := suite.GetComparison("download")
	for i, point := range downloadComp.Providers {
		report.WriteString(fmt.Sprintf("%d. %-20s %8.2f Mbps  [%5.1f%%]\n", i+1, point.ProviderName, point.Value, point.Percentage))
	}
	report.WriteString("\n")

	// Upload Speed Rankings
	report.WriteString("UPLOAD SPEED RANKINGS\n")
	report.WriteString("────────────────────────────────────────────────────────────────\n")
	uploadComp := suite.GetComparison("upload")
	for i, point := range uploadComp.Providers {
		report.WriteString(fmt.Sprintf("%d. %-20s %8.2f Mbps  [%5.1f%%]\n", i+1, point.ProviderName, point.Value, point.Percentage))
	}
	report.WriteString("\n")

	// Latency Rankings
	report.WriteString("LATENCY RANKINGS (Lower is Better)\n")
	report.WriteString("────────────────────────────────────────────────────────────────\n")
	latencyComp := suite.GetComparison("latency")
	for i, point := range latencyComp.Providers {
		report.WriteString(fmt.Sprintf("%d. %-20s %8.2f ms   [%5.1f%%]\n", i+1, point.ProviderName, point.Value, point.Percentage))
	}
	report.WriteString("\n")

	// Jitter Rankings
	report.WriteString("JITTER RANKINGS (Lower is Better)\n")
	report.WriteString("────────────────────────────────────────────────────────────────\n")
	jitterComp := suite.GetComparison("jitter")
	for i, point := range jitterComp.Providers {
		report.WriteString(fmt.Sprintf("%d. %-20s %8.2f ms   [%5.1f%%]\n", i+1, point.ProviderName, point.Value, point.Percentage))
	}
	report.WriteString("\n")

	// Reliability Rankings
	report.WriteString("RELIABILITY RANKINGS\n")
	report.WriteString("────────────────────────────────────────────────────────────────\n")
	reliabilityComp := suite.GetComparison("reliability")
	for i, point := range reliabilityComp.Providers {
		report.WriteString(fmt.Sprintf("%d. %-20s %6.2f%%\n", i+1, point.ProviderName, point.Value))
	}
	report.WriteString("\n")

	// Consistency Rankings
	report.WriteString("CONSISTENCY RANKINGS (Higher is Better)\n")
	report.WriteString("────────────────────────────────────────────────────────────────\n")
	consistencyComp := suite.GetComparison("consistency")
	for i, point := range consistencyComp.Providers {
		report.WriteString(fmt.Sprintf("%d. %-20s %6.2f/100\n", i+1, point.ProviderName, point.Value))
	}
	report.WriteString("\n")

	// Detailed Provider Stats
	report.WriteString("DETAILED PROVIDER STATISTICS\n")
	report.WriteString("════════════════════════════════════════════════════════════════\n\n")
	for _, provider := range suite.OrderedProviders {
		benchmark := suite.Benchmarks[provider]
		report.WriteString(fmt.Sprintf("Provider: %s\n", provider))
		report.WriteString("────────────────────────────────────────────────────────────────\n")
		report.WriteString(fmt.Sprintf("  Tests Run:            %d\n", benchmark.NumTests))
		report.WriteString(fmt.Sprintf("  Successful:           %d\n", benchmark.SuccessfulTests))
		report.WriteString(fmt.Sprintf("  Failed:               %d\n", benchmark.FailedTests))
		report.WriteString(fmt.Sprintf("  Reliability:          %.2f%%\n", benchmark.Reliability))
		report.WriteString(fmt.Sprintf("  Consistency Score:    %.2f/100\n", benchmark.Consistency))
		report.WriteString("\n")
		report.WriteString("  Download Speed:\n")
		report.WriteString(fmt.Sprintf("    Average:            %.2f Mbps\n", benchmark.AvgDownloadMbps))
		report.WriteString(fmt.Sprintf("    Min:                %.2f Mbps\n", benchmark.MinDownloadMbps))
		report.WriteString(fmt.Sprintf("    Max:                %.2f Mbps\n", benchmark.MaxDownloadMbps))
		report.WriteString("\n")
		report.WriteString("  Upload Speed:\n")
		report.WriteString(fmt.Sprintf("    Average:            %.2f Mbps\n", benchmark.AvgUploadMbps))
		report.WriteString(fmt.Sprintf("    Min:                %.2f Mbps\n", benchmark.MinUploadMbps))
		report.WriteString(fmt.Sprintf("    Max:                %.2f Mbps\n", benchmark.MaxUploadMbps))
		report.WriteString("\n")
		report.WriteString("  Latency:\n")
		report.WriteString(fmt.Sprintf("    Average:            %.2f ms\n", benchmark.AvgLatencyMs))
		report.WriteString(fmt.Sprintf("    Min:                %.2f ms\n", benchmark.MinLatencyMs))
		report.WriteString(fmt.Sprintf("    Max:                %.2f ms\n", benchmark.MaxLatencyMs))
		report.WriteString("\n")
		report.WriteString("  Jitter:\n")
		report.WriteString(fmt.Sprintf("    Average:            %.2f ms\n", benchmark.AvgJitterMs))
		report.WriteString("\n")
		report.WriteString(fmt.Sprintf("  Overall Score:       %.2f/100\n", suite.ProviderScores[provider]))
		report.WriteString("\n\n")
	}

	report.WriteString("SCORING METHODOLOGY\n")
	report.WriteString("════════════════════════════════════════════════════════════════\n")
	report.WriteString("Overall Score = (Download×0.30) + (Upload×0.20) + (Latency×0.20)\n")
	report.WriteString("              + (Jitter×0.10) + (Reliability×0.15) + (Consistency×0.05)\n")
	report.WriteString("\n")
	report.WriteString("Where:\n")
	report.WriteString("  - Download:    300 Mbps = 100 points\n")
	report.WriteString("  - Upload:      150 Mbps = 100 points\n")
	report.WriteString("  - Latency:     0 ms = 100 points, 100 ms = 0 points\n")
	report.WriteString("  - Jitter:      0 ms = 100 points, 50 ms = 0 points\n")
	report.WriteString("  - Reliability: % of successful tests\n")
	report.WriteString("  - Consistency: Based on result variance (lower stddev = higher score)\n")
	report.WriteString("\n")

	report.WriteString("Benchmark Duration: " + suite.TotalDuration.String() + "\n")

	return report.String()
}

// getScoreGrade returns a letter grade for a score
func getScoreGrade(score float64) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 80:
		return "B"
	case score >= 70:
		return "C"
	case score >= 60:
		return "D"
	case score >= 50:
		return "E"
	default:
		return "F"
	}
}

// FormatComparisonTable returns a formatted comparison table
func FormatComparisonTable(suite *BenchmarkSuite) string {
	var table strings.Builder

	table.WriteString("\nQUICK COMPARISON TABLE\n")
	table.WriteString("════════════════════════════════════════════════════════════════\n")
	table.WriteString(fmt.Sprintf("%-20s | %10s | %8s | %8s | %8s | %8s\n",
		"Provider", "Download", "Upload", "Latency", "Score", "Grade"))
	table.WriteString("────────────────────────────────────────────────────────────────\n")

	for _, provider := range suite.OrderedProviders {
		benchmark := suite.Benchmarks[provider]
		score := suite.ProviderScores[provider]
		grade := getScoreGrade(score)

		table.WriteString(fmt.Sprintf("%-20s | %8.2f M | %6.2f M | %6.2f ms | %6.2f | %s\n",
			provider,
			benchmark.AvgDownloadMbps,
			benchmark.AvgUploadMbps,
			benchmark.AvgLatencyMs,
			score,
			grade))
	}
	table.WriteString("════════════════════════════════════════════════════════════════\n\n")

	return table.String()
}
