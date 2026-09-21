package benchmarks

import (
	"strings"
	"testing"
	"time"
)

func TestFormatBenchmarkReport(t *testing.T) {
	suite := NewBenchmarkSuite()

	results := []*ProviderResult{
		{ProviderName: "Provider1", DownloadMbps: 100, UploadMbps: 50, Latency: 50 * time.Millisecond, Jitter: 10 * time.Millisecond, Success: true},
		{ProviderName: "Provider2", DownloadMbps: 200, UploadMbps: 100, Latency: 20 * time.Millisecond, Jitter: 5 * time.Millisecond, Success: true},
	}

	for _, r := range results {
		suite.AddResult(r)
	}

	_ = suite.Finalize()

	report := FormatBenchmarkReport(suite)

	tests := []struct {
		name      string
		shouldHave string
	}{
		{"header", "SPEED TEST PROVIDER BENCHMARK RESULTS"},
		{"overall rankings", "OVERALL PERFORMANCE RANKINGS"},
		{"download section", "DOWNLOAD SPEED RANKINGS"},
		{"upload section", "UPLOAD SPEED RANKINGS"},
		{"latency section", "LATENCY RANKINGS"},
		{"jitter section", "JITTER RANKINGS"},
		{"reliability section", "RELIABILITY RANKINGS"},
		{"consistency section", "CONSISTENCY RANKINGS"},
		{"detailed stats", "DETAILED PROVIDER STATISTICS"},
		{"methodology", "SCORING METHODOLOGY"},
		{"provider1", "Provider1"},
		{"provider2", "Provider2"},
		{"mbps unit", "Mbps"},
		{"score format", "Score:"},
		{"grade label", "Grade:"},
	}

	for _, tt := range tests {
		if !strings.Contains(report, tt.shouldHave) {
			t.Errorf("report missing %q: %s", tt.name, tt.shouldHave)
		}
	}
}

func TestFormatBenchmarkReportStructure(t *testing.T) {
	suite := NewBenchmarkSuite()

	result := &ProviderResult{
		ProviderName: "TestProvider",
		DownloadMbps: 150,
		UploadMbps:   75,
		Latency:      35 * time.Millisecond,
		Jitter:       8 * time.Millisecond,
		Success:      true,
	}

	suite.AddResult(result)
	_ = suite.Finalize()

	report := FormatBenchmarkReport(suite)

	if !strings.Contains(report, "TestProvider") {
		t.Error("report should contain provider name")
	}

	if !strings.Contains(report, "150.00") {
		t.Error("report should contain download speed")
	}

	if !strings.Contains(report, "75.00") {
		t.Error("report should contain upload speed")
	}

	if !strings.Contains(report, "35.00") {
		t.Error("report should contain latency")
	}

	if !strings.Contains(report, "Benchmark Duration:") {
		t.Error("report should contain duration info")
	}
}

func TestFormatBenchmarkReportWithFailures(t *testing.T) {
	suite := NewBenchmarkSuite()

	results := []*ProviderResult{
		{ProviderName: "Provider1", DownloadMbps: 100, UploadMbps: 50, Latency: 50 * time.Millisecond, Jitter: 10 * time.Millisecond, Success: true},
		{ProviderName: "Provider1", DownloadMbps: 0, UploadMbps: 0, Latency: 0, Success: false, Error: "timeout"},
	}

	for _, r := range results {
		suite.AddResult(r)
	}

	_ = suite.Finalize()

	report := FormatBenchmarkReport(suite)

	if !strings.Contains(report, "Tests Run:") {
		t.Error("report should show test count")
	}
	if !strings.Contains(report, "Successful:") {
		t.Error("report should show successful count")
	}
	if !strings.Contains(report, "Failed:") {
		t.Error("report should show failed count")
	}
	if !strings.Contains(report, "Reliability:") {
		t.Error("report should show reliability")
	}
}

func TestFormatComparisonTable(t *testing.T) {
	suite := NewBenchmarkSuite()

	results := []*ProviderResult{
		{ProviderName: "Provider1", DownloadMbps: 100, UploadMbps: 50, Latency: 50 * time.Millisecond, Jitter: 10 * time.Millisecond, Success: true},
		{ProviderName: "Provider2", DownloadMbps: 200, UploadMbps: 100, Latency: 20 * time.Millisecond, Jitter: 5 * time.Millisecond, Success: true},
	}

	for _, r := range results {
		suite.AddResult(r)
	}

	_ = suite.Finalize()

	table := FormatComparisonTable(suite)

	tests := []struct {
		name      string
		shouldHave string
	}{
		{"title", "QUICK COMPARISON TABLE"},
		{"provider col", "Provider"},
		{"download col", "Download"},
		{"upload col", "Upload"},
		{"latency col", "Latency"},
		{"score col", "Score"},
		{"grade col", "Grade"},
		{"provider1", "Provider1"},
		{"provider2", "Provider2"},
	}

	for _, tt := range tests {
		if !strings.Contains(table, tt.shouldHave) {
			t.Errorf("table missing %q: %s", tt.name, tt.shouldHave)
		}
	}
}

func TestFormatComparisonTableValues(t *testing.T) {
	suite := NewBenchmarkSuite()

	result := &ProviderResult{
		ProviderName: "TestProvider",
		DownloadMbps: 123.45,
		UploadMbps:   67.89,
		Latency:      25 * time.Millisecond,
		Jitter:       5 * time.Millisecond,
		Success:      true,
	}

	suite.AddResult(result)
	_ = suite.Finalize()

	table := FormatComparisonTable(suite)

	if !strings.Contains(table, "TestProvider") {
		t.Error("table should contain provider name")
	}

	if !strings.Contains(table, "123.45") {
		t.Error("table should contain download value")
	}

	if !strings.Contains(table, "67.89") {
		t.Error("table should contain upload value")
	}

	if !strings.Contains(table, "25.00") {
		t.Error("table should contain latency value")
	}
}

func TestFormatComparisonTableMultipleProviders(t *testing.T) {
	suite := NewBenchmarkSuite()

	results := []*ProviderResult{
		{ProviderName: "Alpha", DownloadMbps: 50, UploadMbps: 25, Latency: 100 * time.Millisecond, Jitter: 20 * time.Millisecond, Success: true},
		{ProviderName: "Beta", DownloadMbps: 150, UploadMbps: 75, Latency: 30 * time.Millisecond, Jitter: 5 * time.Millisecond, Success: true},
		{ProviderName: "Gamma", DownloadMbps: 200, UploadMbps: 100, Latency: 15 * time.Millisecond, Jitter: 2 * time.Millisecond, Success: true},
	}

	for _, r := range results {
		suite.AddResult(r)
	}

	_ = suite.Finalize()

	table := FormatComparisonTable(suite)

	if !strings.Contains(table, "Alpha") {
		t.Error("table should contain Alpha")
	}
	if !strings.Contains(table, "Beta") {
		t.Error("table should contain Beta")
	}
	if !strings.Contains(table, "Gamma") {
		t.Error("table should contain Gamma")
	}
}

func TestGetScoreGrade(t *testing.T) {
	tests := []struct {
		score    float64
		expected string
	}{
		{95, "A"},
		{85, "B"},
		{75, "C"},
		{65, "D"},
		{55, "E"},
		{45, "F"},
		{100, "A"},
		{0, "F"},
		{90, "A"},
		{89.9, "B"},
	}

	for _, tt := range tests {
		t.Run(string(rune(int(tt.score))), func(t *testing.T) {
			grade := getScoreGrade(tt.score)
			if grade != tt.expected {
				t.Errorf("score %f: got %s, want %s", tt.score, grade, tt.expected)
			}
		})
	}
}

func TestFormatBenchmarkReportRankings(t *testing.T) {
	suite := NewBenchmarkSuite()

	results := []*ProviderResult{
		{ProviderName: "Slow", DownloadMbps: 10, UploadMbps: 5, Latency: 200 * time.Millisecond, Jitter: 50 * time.Millisecond, Success: true},
		{ProviderName: "Medium", DownloadMbps: 100, UploadMbps: 50, Latency: 50 * time.Millisecond, Jitter: 10 * time.Millisecond, Success: true},
		{ProviderName: "Fast", DownloadMbps: 300, UploadMbps: 150, Latency: 10 * time.Millisecond, Jitter: 2 * time.Millisecond, Success: true},
	}

	for _, r := range results {
		suite.AddResult(r)
	}

	_ = suite.Finalize()

	report := FormatBenchmarkReport(suite)

	downloadIdx := strings.Index(report, "DOWNLOAD SPEED RANKINGS")
	uploadIdx := strings.Index(report, "UPLOAD SPEED RANKINGS")
	latencyIdx := strings.Index(report, "LATENCY RANKINGS")

	if downloadIdx == -1 {
		t.Fatal("no download rankings section")
	}
	if uploadIdx == -1 {
		t.Fatal("no upload rankings section")
	}
	if latencyIdx == -1 {
		t.Fatal("no latency rankings section")
	}

	downloadSection := report[downloadIdx : uploadIdx]
	if !strings.Contains(downloadSection, "Fast") {
		t.Error("Fast provider should be first in download rankings")
	}

	latencySection := report[latencyIdx:]
	if strings.Index(latencySection, "Fast") > strings.Index(latencySection, "Slow") {
		t.Error("latency should rank by lower values first")
	}
}

func TestFormatBenchmarkReportDetailedStats(t *testing.T) {
	suite := NewBenchmarkSuite()

	results := []*ProviderResult{
		{ProviderName: "Provider1", DownloadMbps: 100, UploadMbps: 50, Latency: 50 * time.Millisecond, Jitter: 10 * time.Millisecond, Success: true},
		{ProviderName: "Provider1", DownloadMbps: 120, UploadMbps: 60, Latency: 40 * time.Millisecond, Jitter: 8 * time.Millisecond, Success: true},
	}

	for _, r := range results {
		suite.AddResult(r)
	}

	_ = suite.Finalize()

	report := FormatBenchmarkReport(suite)

	if !strings.Contains(report, "DETAILED PROVIDER STATISTICS") {
		t.Error("report should have detailed stats section")
	}
	if !strings.Contains(report, "Download Speed:") {
		t.Error("report should show download speed stats")
	}
	if !strings.Contains(report, "Upload Speed:") {
		t.Error("report should show upload speed stats")
	}
	if !strings.Contains(report, "Latency:") {
		t.Error("report should show latency stats")
	}
	if !strings.Contains(report, "Jitter:") {
		t.Error("report should show jitter stats")
	}
	if !strings.Contains(report, "Average:") {
		t.Error("report should show averages")
	}
	if !strings.Contains(report, "Min:") {
		t.Error("report should show minimums")
	}
	if !strings.Contains(report, "Max:") {
		t.Error("report should show maximums")
	}
}
