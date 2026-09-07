package benchmarks

import (
	"math"
	"reflect"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Bug 1: ranking aliasing
// ---------------------------------------------------------------------------

// addSuccess is a small helper for building deterministic ProviderResult
// entries for the ranking / scoring tests below.
func addSuccess(bs *BenchmarkSuite, provider string, downloadMbps, uploadMbps float64, latencyMs int64) {
	bs.AddResult(&ProviderResult{
		ProviderName: provider,
		DownloadMbps: downloadMbps,
		UploadMbps:   uploadMbps,
		Latency:      time.Duration(latencyMs) * time.Millisecond,
		Jitter:       time.Duration(0),
		Success:      true,
	})
}

func TestGenerateRankings_AreIndependentSlices(t *testing.T) {
	bs := NewBenchmarkSuite()

	// Deliberately construct three providers whose download / upload /
	// latency orderings all differ from one another (and from whatever
	// the overall weighted score ends up ranking them).
	//
	//   download (best first): A(300) > B(200) > C(100)
	//   upload   (best first): C(150) > B(50)  > A(10)
	//   latency  (best first): B(10)  < A(50)  < C(90)
	addSuccess(bs, "A", 300, 10, 50)
	addSuccess(bs, "B", 200, 50, 10)
	addSuccess(bs, "C", 100, 150, 90)

	if err := bs.Finalize(); err != nil {
		t.Fatalf("Finalize() returned unexpected error: %v", err)
	}

	wantDownload := []string{"A", "B", "C"}
	wantUpload := []string{"C", "B", "A"}
	wantLatency := []string{"B", "A", "C"}

	if !reflect.DeepEqual(bs.DownloadRankings, wantDownload) {
		t.Errorf("DownloadRankings = %v, want %v", bs.DownloadRankings, wantDownload)
	}
	if !reflect.DeepEqual(bs.UploadRankings, wantUpload) {
		t.Errorf("UploadRankings = %v, want %v", bs.UploadRankings, wantUpload)
	}
	if !reflect.DeepEqual(bs.LatencyRankings, wantLatency) {
		t.Errorf("LatencyRankings = %v, want %v", bs.LatencyRankings, wantLatency)
	}

	// The three rankings must be genuinely independent slices/orderings,
	// not aliases of one another (or of OrderedProviders) that all ended
	// up holding the final sort's order.
	if reflect.DeepEqual(bs.DownloadRankings, bs.UploadRankings) {
		t.Errorf("DownloadRankings and UploadRankings are identical (%v); they should differ given the constructed data", bs.DownloadRankings)
	}
	if reflect.DeepEqual(bs.UploadRankings, bs.LatencyRankings) {
		t.Errorf("UploadRankings and LatencyRankings are identical (%v); they should differ given the constructed data", bs.UploadRankings)
	}
	if reflect.DeepEqual(bs.DownloadRankings, bs.LatencyRankings) {
		t.Errorf("DownloadRankings and LatencyRankings are identical (%v); they should differ given the constructed data", bs.DownloadRankings)
	}

	// Mutating one ranking slice must never affect another - they must
	// not share a backing array.
	bs.DownloadRankings[0] = "MUTATED"
	if bs.UploadRankings[0] == "MUTATED" || bs.LatencyRankings[0] == "MUTATED" || bs.OrderedProviders[0] == "MUTATED" {
		t.Errorf("mutating DownloadRankings leaked into other ranking slices - they share a backing array")
	}
}

// ---------------------------------------------------------------------------
// Bug 2: inverted latency / jitter percentages
// ---------------------------------------------------------------------------

func TestGetComparison_LatencyPercentageFavorsLowestLatency(t *testing.T) {
	bs := NewBenchmarkSuite()

	// fastcom has the best (lowest) latency, cloudflare the worst.
	addSuccess(bs, "fastcom", 100, 50, 10)
	addSuccess(bs, "mlab", 100, 50, 20)
	addSuccess(bs, "cloudflare", 100, 50, 40)

	if err := bs.Finalize(); err != nil {
		t.Fatalf("Finalize() returned unexpected error: %v", err)
	}

	comp := bs.GetComparison("latency")

	var pct = map[string]float64{}
	var rank = map[string]int{}
	for _, p := range comp.Providers {
		pct[p.ProviderName] = p.Percentage
		rank[p.ProviderName] = p.Rank
	}

	if rank["fastcom"] != 1 {
		t.Errorf("expected fastcom (lowest latency) to be rank 1, got %d", rank["fastcom"])
	}

	// The best (lowest-latency) provider must have the HIGHEST percentage
	// (100%), matching BENCHMARK_FEATURES.md's documented semantics and
	// sample output, not the lowest.
	if pct["fastcom"] <= pct["cloudflare"] {
		t.Errorf("best-latency provider fastcom should have a higher percentage than worst-latency provider cloudflare; got fastcom=%.2f cloudflare=%.2f", pct["fastcom"], pct["cloudflare"])
	}
	if math.Abs(pct["fastcom"]-100) > 0.01 {
		t.Errorf("best-latency provider should be at 100%%, got %.2f", pct["fastcom"])
	}
	// mlab (20ms) relative to fastcom (10ms) should be 10/20*100 = 50%.
	if math.Abs(pct["mlab"]-50) > 0.01 {
		t.Errorf("mlab percentage = %.2f, want 50.00 (10ms best / 20ms mlab * 100)", pct["mlab"])
	}
	// cloudflare (40ms) relative to fastcom (10ms) should be 10/40*100 = 25%.
	if math.Abs(pct["cloudflare"]-25) > 0.01 {
		t.Errorf("cloudflare percentage = %.2f, want 25.00 (10ms best / 40ms cloudflare * 100)", pct["cloudflare"])
	}
}

func TestGetComparison_JitterPercentageFavorsLowestJitter(t *testing.T) {
	bs := NewBenchmarkSuite()
	bs.AddResult(&ProviderResult{ProviderName: "low", Success: true, DownloadMbps: 10, UploadMbps: 10, Latency: 10 * time.Millisecond, Jitter: 2 * time.Millisecond})
	bs.AddResult(&ProviderResult{ProviderName: "high", Success: true, DownloadMbps: 10, UploadMbps: 10, Latency: 10 * time.Millisecond, Jitter: 8 * time.Millisecond})

	if err := bs.Finalize(); err != nil {
		t.Fatalf("Finalize() returned unexpected error: %v", err)
	}

	comp := bs.GetComparison("jitter")
	pct := map[string]float64{}
	for _, p := range comp.Providers {
		pct[p.ProviderName] = p.Percentage
	}

	if pct["low"] <= pct["high"] {
		t.Errorf("lowest-jitter provider should have higher percentage than highest-jitter provider; got low=%.2f high=%.2f", pct["low"], pct["high"])
	}
}

func TestGetComparison_DownloadPercentageFavorsHighestValue(t *testing.T) {
	// Sanity check that the "higher is better" metrics were not broken by
	// the latency/jitter fix.
	bs := NewBenchmarkSuite()
	addSuccess(bs, "fast", 200, 50, 20)
	addSuccess(bs, "slow", 100, 50, 20)

	if err := bs.Finalize(); err != nil {
		t.Fatalf("Finalize() returned unexpected error: %v", err)
	}

	comp := bs.GetComparison("download")
	pct := map[string]float64{}
	for _, p := range comp.Providers {
		pct[p.ProviderName] = p.Percentage
	}

	if math.Abs(pct["fast"]-100) > 0.01 {
		t.Errorf("fast provider (best download) should be at 100%%, got %.2f", pct["fast"])
	}
	if math.Abs(pct["slow"]-50) > 0.01 {
		t.Errorf("slow provider percentage = %.2f, want 50.00 (100/200*100)", pct["slow"])
	}
}

// ---------------------------------------------------------------------------
// Bug 3: int64(math.MaxFloat64) undefined behavior on first measurement
// ---------------------------------------------------------------------------

func TestFinalize_FirstLatencyMeasurementIsRecordedCorrectly(t *testing.T) {
	bs := NewBenchmarkSuite()

	// A single result exercises the "still at the math.MaxFloat64 sentinel"
	// path for MinLatencyMs/MaxLatencyMs.
	addSuccess(bs, "solo", 100, 50, 42)

	if err := bs.Finalize(); err != nil {
		t.Fatalf("Finalize() returned unexpected error: %v", err)
	}

	benchmark := bs.Benchmarks["solo"]
	if benchmark == nil {
		t.Fatal("expected benchmark for provider 'solo'")
	}

	if benchmark.MinLatencyMs != 42 {
		t.Errorf("MinLatencyMs = %v, want 42 (first-measurement path must not be corrupted by converting math.MaxFloat64 to int64)", benchmark.MinLatencyMs)
	}
	if benchmark.MaxLatencyMs != 42 {
		t.Errorf("MaxLatencyMs = %v, want 42", benchmark.MaxLatencyMs)
	}
}

func TestFinalize_MinMaxLatencyAcrossMultipleMeasurements(t *testing.T) {
	bs := NewBenchmarkSuite()
	addSuccess(bs, "p", 100, 50, 42)
	addSuccess(bs, "p", 100, 50, 5)
	addSuccess(bs, "p", 100, 50, 99)

	if err := bs.Finalize(); err != nil {
		t.Fatalf("Finalize() returned unexpected error: %v", err)
	}

	benchmark := bs.Benchmarks["p"]
	if benchmark.MinLatencyMs != 5 {
		t.Errorf("MinLatencyMs = %v, want 5", benchmark.MinLatencyMs)
	}
	if benchmark.MaxLatencyMs != 99 {
		t.Errorf("MaxLatencyMs = %v, want 99", benchmark.MaxLatencyMs)
	}
}

// ---------------------------------------------------------------------------
// Bug 4: dead error path in Finalize()
// ---------------------------------------------------------------------------

func TestFinalize_ReturnsErrorWhenNoBenchmarksRecorded(t *testing.T) {
	bs := NewBenchmarkSuite()

	err := bs.Finalize()
	if err == nil {
		t.Fatal("expected Finalize() to return an error when no benchmarks were recorded, got nil")
	}
}

func TestFinalize_NoErrorWithAtLeastOneBenchmark(t *testing.T) {
	bs := NewBenchmarkSuite()
	addSuccess(bs, "p", 100, 50, 20)

	if err := bs.Finalize(); err != nil {
		t.Fatalf("Finalize() returned unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// General coverage: AddResult / NewBenchmarkSuite
// ---------------------------------------------------------------------------

func TestNewBenchmarkSuite_InitialState(t *testing.T) {
	bs := NewBenchmarkSuite()
	if bs.Benchmarks == nil {
		t.Error("Benchmarks map should be initialized")
	}
	if bs.ProviderScores == nil {
		t.Error("ProviderScores map should be initialized")
	}
	if len(bs.DownloadRankings) != 0 {
		t.Error("DownloadRankings should start empty")
	}
	if bs.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}
}

func TestAddResult_TracksSuccessAndFailureCounts(t *testing.T) {
	bs := NewBenchmarkSuite()
	bs.AddResult(&ProviderResult{ProviderName: "p", Success: true, DownloadMbps: 10, UploadMbps: 5, Latency: 10 * time.Millisecond})
	bs.AddResult(&ProviderResult{ProviderName: "p", Success: false, Error: "boom"})

	b := bs.Benchmarks["p"]
	if b == nil {
		t.Fatal("expected benchmark for provider 'p'")
	}
	if b.NumTests != 2 {
		t.Errorf("NumTests = %d, want 2", b.NumTests)
	}
	if b.SuccessfulTests != 1 {
		t.Errorf("SuccessfulTests = %d, want 1", b.SuccessfulTests)
	}
	if b.FailedTests != 1 {
		t.Errorf("FailedTests = %d, want 1", b.FailedTests)
	}
	if len(b.TestResults) != 2 {
		t.Errorf("TestResults length = %d, want 2", len(b.TestResults))
	}
}

func TestFinalize_AllFailedResultsYieldsZeroReliability(t *testing.T) {
	bs := NewBenchmarkSuite()
	bs.AddResult(&ProviderResult{ProviderName: "p", Success: false, Error: "boom"})

	if err := bs.Finalize(); err != nil {
		t.Fatalf("Finalize() returned unexpected error: %v", err)
	}

	b := bs.Benchmarks["p"]
	if b.Reliability != 0 {
		t.Errorf("Reliability = %v, want 0 for a provider with only failed tests", b.Reliability)
	}
}

func TestAddResult_EmptyProviderNameIsIgnoredByFinalize(t *testing.T) {
	// Regression-style guard: Finalize must not panic when a provider has
	// zero recorded tests somehow (defensive; NumTests==0 branch).
	bs := NewBenchmarkSuite()
	bs.Benchmarks["ghost"] = &ProviderBenchmark{ProviderName: "ghost"}

	if err := bs.Finalize(); err != nil {
		t.Fatalf("Finalize() returned unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Scoring helpers
// ---------------------------------------------------------------------------

func TestNormalizeDownload(t *testing.T) {
	cases := []struct {
		mbps float64
		want float64
	}{
		{-10, 0},
		{0, 0},
		{150, 50},
		{300, 100},
		{600, 100},
	}
	for _, c := range cases {
		if got := normalizeDownload(c.mbps); math.Abs(got-c.want) > 0.001 {
			t.Errorf("normalizeDownload(%v) = %v, want %v", c.mbps, got, c.want)
		}
	}
}

func TestNormalizeUpload(t *testing.T) {
	cases := []struct {
		mbps float64
		want float64
	}{
		{-10, 0},
		{0, 0},
		{75, 50},
		{150, 100},
		{300, 100},
	}
	for _, c := range cases {
		if got := normalizeUpload(c.mbps); math.Abs(got-c.want) > 0.001 {
			t.Errorf("normalizeUpload(%v) = %v, want %v", c.mbps, got, c.want)
		}
	}
}

func TestNormalizeLatency(t *testing.T) {
	cases := []struct {
		ms   float64
		want float64
	}{
		{-5, 100},
		{0, 100},
		{50, 50},
		{100, 0},
		{200, 0},
	}
	for _, c := range cases {
		if got := normalizeLatency(c.ms); math.Abs(got-c.want) > 0.001 {
			t.Errorf("normalizeLatency(%v) = %v, want %v", c.ms, got, c.want)
		}
	}
}

func TestNormalizeJitter(t *testing.T) {
	cases := []struct {
		ms   float64
		want float64
	}{
		{-5, 100},
		{0, 100},
		{25, 50},
		{50, 0},
		{100, 0},
	}
	for _, c := range cases {
		if got := normalizeJitter(c.ms); math.Abs(got-c.want) > 0.001 {
			t.Errorf("normalizeJitter(%v) = %v, want %v", c.ms, got, c.want)
		}
	}
}

func TestCalculateConsistency_EmptyResults(t *testing.T) {
	if got := calculateConsistency(nil); got != 0 {
		t.Errorf("calculateConsistency(nil) = %v, want 0", got)
	}
}

func TestCalculateConsistency_IdenticalResultsIsMaximallyConsistent(t *testing.T) {
	results := []ProviderResult{
		{DownloadMbps: 100},
		{DownloadMbps: 100},
		{DownloadMbps: 100},
	}
	got := calculateConsistency(results)
	if math.Abs(got-100) > 0.001 {
		t.Errorf("calculateConsistency(identical) = %v, want 100", got)
	}
}

func TestCalculateConsistency_VariedResultsIsLessConsistent(t *testing.T) {
	steady := []ProviderResult{{DownloadMbps: 100}, {DownloadMbps: 100}, {DownloadMbps: 100}}
	varied := []ProviderResult{{DownloadMbps: 10}, {DownloadMbps: 100}, {DownloadMbps: 200}}

	steadyScore := calculateConsistency(steady)
	variedScore := calculateConsistency(varied)

	if variedScore >= steadyScore {
		t.Errorf("varied results should have lower consistency than steady results; steady=%v varied=%v", steadyScore, variedScore)
	}
}

func TestCalculatePerformanceScore_NoSuccessfulTestsIsZero(t *testing.T) {
	b := &ProviderBenchmark{SuccessfulTests: 0}
	if got := calculatePerformanceScore(b); got != 0 {
		t.Errorf("calculatePerformanceScore(no successes) = %v, want 0", got)
	}
}

func TestCalculatePerformanceScore_PerfectProviderScoresHigh(t *testing.T) {
	b := &ProviderBenchmark{
		SuccessfulTests: 10,
		AvgDownloadMbps: 300,
		AvgUploadMbps:   150,
		AvgLatencyMs:    0,
		AvgJitterMs:     0,
		Reliability:     100,
		Consistency:     100,
	}
	got := calculatePerformanceScore(b)
	if math.Abs(got-100) > 0.001 {
		t.Errorf("calculatePerformanceScore(perfect) = %v, want 100", got)
	}
}

// ---------------------------------------------------------------------------
// GetBestProvider / GetProviderScore / Summary
// ---------------------------------------------------------------------------

func TestGetBestProvider_EmptyReturnsEmptyString(t *testing.T) {
	bs := NewBenchmarkSuite()
	if got := bs.GetBestProvider(); got != "" {
		t.Errorf("GetBestProvider() on empty suite = %q, want empty string", got)
	}
}

func TestGetBestProvider_ReturnsHighestScoringProvider(t *testing.T) {
	bs := NewBenchmarkSuite()
	addSuccess(bs, "weak", 10, 5, 90)
	addSuccess(bs, "strong", 300, 150, 1)

	if err := bs.Finalize(); err != nil {
		t.Fatalf("Finalize() returned unexpected error: %v", err)
	}

	if got := bs.GetBestProvider(); got != "strong" {
		t.Errorf("GetBestProvider() = %q, want %q", got, "strong")
	}
}

func TestGetProviderScore_UnknownProviderIsZero(t *testing.T) {
	bs := NewBenchmarkSuite()
	if got := bs.GetProviderScore("nope"); got != 0 {
		t.Errorf("GetProviderScore(unknown) = %v, want 0", got)
	}
}

func TestSummary_EmptySuite(t *testing.T) {
	bs := NewBenchmarkSuite()
	got := bs.Summary()
	want := "No benchmark results available"
	if got != want {
		t.Errorf("Summary() = %q, want %q", got, want)
	}
}

func TestSummary_WithResultsMentionsWinner(t *testing.T) {
	bs := NewBenchmarkSuite()
	addSuccess(bs, "winner", 300, 150, 1)

	if err := bs.Finalize(); err != nil {
		t.Fatalf("Finalize() returned unexpected error: %v", err)
	}

	got := bs.Summary()
	if !contains(got, "winner") {
		t.Errorf("Summary() = %q, want it to mention the winning provider", got)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}

// ---------------------------------------------------------------------------
// GetComparison: unknown metric and rank assignment
// ---------------------------------------------------------------------------

func TestGetComparison_UnknownMetricReturnsEmptyProviders(t *testing.T) {
	bs := NewBenchmarkSuite()
	addSuccess(bs, "p", 100, 50, 20)
	if err := bs.Finalize(); err != nil {
		t.Fatalf("Finalize() returned unexpected error: %v", err)
	}

	comp := bs.GetComparison("not-a-real-metric")
	if len(comp.Providers) != 0 {
		t.Errorf("GetComparison(unknown metric) Providers = %v, want empty", comp.Providers)
	}
}

func TestGetComparison_RanksAreSequential(t *testing.T) {
	bs := NewBenchmarkSuite()
	addSuccess(bs, "first", 300, 50, 20)
	addSuccess(bs, "second", 200, 50, 20)
	addSuccess(bs, "third", 100, 50, 20)

	if err := bs.Finalize(); err != nil {
		t.Fatalf("Finalize() returned unexpected error: %v", err)
	}

	comp := bs.GetComparison("download")
	for i, p := range comp.Providers {
		if p.Rank != i+1 {
			t.Errorf("Providers[%d].Rank = %d, want %d", i, p.Rank, i+1)
		}
	}
}
