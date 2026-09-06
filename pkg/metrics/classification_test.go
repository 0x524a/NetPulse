package metrics

import (
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// ClassifySpeed: connection-type / grade boundaries
// ---------------------------------------------------------------------------

func TestClassifySpeed_ConnectionTypeAndGradeBoundaries(t *testing.T) {
	cases := []struct {
		mbps     float64
		wantType ConnectionType
		wantGrd  SpeedGrade
	}{
		{0, ConnectionTypeSlowDSL, GradeF},
		{4.99, ConnectionTypeSlowDSL, GradeF},
		{5, ConnectionTypeStandardDSL, GradeD},
		{9.99, ConnectionTypeStandardDSL, GradeD},
		{10, ConnectionTypeModernDSL, GradeC},
		{24.99, ConnectionTypeModernDSL, GradeC},
		{25, ConnectionTypeSlowCable, GradeC},
		{49.99, ConnectionTypeSlowCable, GradeC},
		{50, ConnectionTypeStandardCable, GradeB},
		{99.99, ConnectionTypeStandardCable, GradeB},
		{100, ConnectionTypeHighSpeedCable, GradeA},
		{299.99, ConnectionTypeHighSpeedCable, GradeA},
		{300, ConnectionTypeFiber, GradeA},
		{1000, ConnectionTypeFiber, GradeA},
	}

	for _, c := range cases {
		got := ClassifySpeed(c.mbps)
		if got.Classification != c.wantType {
			t.Errorf("ClassifySpeed(%v).Classification = %v, want %v", c.mbps, got.Classification, c.wantType)
		}
		if got.Grade != c.wantGrd {
			t.Errorf("ClassifySpeed(%v).Grade = %v, want %v", c.mbps, got.Grade, c.wantGrd)
		}
	}
}

// ---------------------------------------------------------------------------
// ClassifySpeed: usage suitability boundaries
// ---------------------------------------------------------------------------

func TestClassifySpeed_IsGoodForWorkBoundary(t *testing.T) {
	if ClassifySpeed(9.99).IsGoodForWork {
		t.Error("9.99 Mbps should not be good for work (threshold is 10)")
	}
	if !ClassifySpeed(10).IsGoodForWork {
		t.Error("10 Mbps should be good for work (threshold is 10)")
	}
}

func TestClassifySpeed_IsGoodForStreamingBoundary(t *testing.T) {
	if ClassifySpeed(24.99).IsGoodForStreaming {
		t.Error("24.99 Mbps should not be good for streaming (threshold is 25)")
	}
	if !ClassifySpeed(25).IsGoodForStreaming {
		t.Error("25 Mbps should be good for streaming (threshold is 25)")
	}
}

func TestClassifySpeed_IsGoodForGamingBoundary(t *testing.T) {
	if ClassifySpeed(34.99).IsGoodForGaming {
		t.Error("34.99 Mbps should not be good for gaming (threshold is 35)")
	}
	if !ClassifySpeed(35).IsGoodForGaming {
		t.Error("35 Mbps should be good for gaming (threshold is 35)")
	}
}

// ---------------------------------------------------------------------------
// ClassifySpeed: QualityScore boundaries (calculateQualityScore, exercised
// through the exported ClassifySpeed entry point)
// ---------------------------------------------------------------------------

func TestClassifySpeed_QualityScoreBoundaries(t *testing.T) {
	cases := []struct {
		mbps float64
		want float64
	}{
		{0, 10},
		{4.99, 10},
		{5, 20},
		{9.99, 20},
		{10, 35},
		{24.99, 35},
		{25, 50},
		{49.99, 50},
		{50, 65},
		{99.99, 65},
		{100, 75},
		{149.99, 75},
		{150, 85},
		{299.99, 85},
		{300, 95},
		{499.99, 95},
		{500, 100},
		{1000, 100},
	}

	for _, c := range cases {
		got := ClassifySpeed(c.mbps).QualityScore
		if got != c.want {
			t.Errorf("ClassifySpeed(%v).QualityScore = %v, want %v", c.mbps, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// AnalyzeLatency boundaries
// ---------------------------------------------------------------------------

func TestAnalyzeLatency_Boundaries(t *testing.T) {
	cases := []struct {
		ms   int
		want string
	}{
		{0, "Excellent (< 20ms)"},
		{19, "Excellent (< 20ms)"},
		{20, "Very Good (20-50ms)"},
		{49, "Very Good (20-50ms)"},
		{50, "Good (50-100ms)"},
		{99, "Good (50-100ms)"},
		{100, "Fair (100-150ms)"},
		{149, "Fair (100-150ms)"},
		{150, "Poor (> 150ms)"},
		{500, "Poor (> 150ms)"},
	}

	for _, c := range cases {
		got := AnalyzeLatency(time.Duration(c.ms) * time.Millisecond)
		if got != c.want {
			t.Errorf("AnalyzeLatency(%dms) = %q, want %q", c.ms, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// AnalyzeJitter boundaries
// ---------------------------------------------------------------------------

func TestAnalyzeJitter_Boundaries(t *testing.T) {
	cases := []struct {
		ms   int
		want string
	}{
		{0, "Excellent (< 5ms)"},
		{4, "Excellent (< 5ms)"},
		{5, "Very Good (5-10ms)"},
		{9, "Very Good (5-10ms)"},
		{10, "Good (10-20ms)"},
		{19, "Good (10-20ms)"},
		{20, "Fair (20-50ms)"},
		{49, "Fair (20-50ms)"},
		{50, "Poor (> 50ms)"},
		{200, "Poor (> 50ms)"},
	}

	for _, c := range cases {
		got := AnalyzeJitter(time.Duration(c.ms) * time.Millisecond)
		if got != c.want {
			t.Errorf("AnalyzeJitter(%dms) = %q, want %q", c.ms, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// GetSpeedComparison
// ---------------------------------------------------------------------------

func TestGetSpeedComparison_AboveUSAverage(t *testing.T) {
	got := GetSpeedComparison(250) // (250-200)/200*100 = 25%
	if !strings.Contains(got, "25%") || !strings.Contains(got, "faster than US average") {
		t.Errorf("GetSpeedComparison(250) = %q, want it to mention 25%% faster than US average", got)
	}
}

func TestGetSpeedComparison_BetweenGlobalAndUSAverage(t *testing.T) {
	got := GetSpeedComparison(120) // (120-80)/80*100 = 50%
	if !strings.Contains(got, "50%") || !strings.Contains(got, "faster than global average") {
		t.Errorf("GetSpeedComparison(120) = %q, want it to mention 50%% faster than global average", got)
	}
}

func TestGetSpeedComparison_BelowGlobalAverage(t *testing.T) {
	got := GetSpeedComparison(40) // (80-40)/80*100 = 50%
	if !strings.Contains(got, "50%") || !strings.Contains(got, "slower than global average") {
		t.Errorf("GetSpeedComparison(40) = %q, want it to mention 50%% slower than global average", got)
	}
}

func TestGetSpeedComparison_ExactlyUSAverage(t *testing.T) {
	// downloadMbps == uSAverage is not "> uSAverage", so it falls into the
	// "faster than global average" branch (200 > 80).
	got := GetSpeedComparison(200)
	if !strings.Contains(got, "faster than global average") {
		t.Errorf("GetSpeedComparison(200) = %q, want it to fall into the global-average-relative branch at the US-average boundary", got)
	}
}

func TestGetSpeedComparison_ExactlyGlobalAverage(t *testing.T) {
	// downloadMbps == globalAverage is not "> globalAverage", so it falls
	// into the "slower than global average" branch with 0%.
	got := GetSpeedComparison(80)
	if !strings.Contains(got, "0%") || !strings.Contains(got, "slower than global average") {
		t.Errorf("GetSpeedComparison(80) = %q, want 0%% slower than global average at the boundary", got)
	}
}
