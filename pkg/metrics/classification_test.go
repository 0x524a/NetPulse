package metrics

import (
	"strings"
	"testing"
	"time"
)

func TestClassifySpeed(t *testing.T) {
	tests := []struct {
		name           string
		downloadMbps   float64
		expectedGrade  SpeedGrade
		expectedType   ConnectionType
		expectStreaming bool
		expectGaming   bool
		expectWork     bool
		expectedScore  float64
	}{
		// Boundary values and normal cases
		{
			name:             "0 Mbps - minimal",
			downloadMbps:     0,
			expectedGrade:    GradeF,
			expectedType:     ConnectionTypeSlowDSL,
			expectStreaming:  false,
			expectGaming:     false,
			expectWork:       false,
			expectedScore:    10,
		},
		{
			name:             "4.9 Mbps - just below slow DSL boundary",
			downloadMbps:     4.9,
			expectedGrade:    GradeF,
			expectedType:     ConnectionTypeSlowDSL,
			expectStreaming:  false,
			expectGaming:     false,
			expectWork:       false,
			expectedScore:    10,
		},
		{
			name:             "5 Mbps - slow DSL boundary",
			downloadMbps:     5,
			expectedGrade:    GradeD,
			expectedType:     ConnectionTypeStandardDSL,
			expectStreaming:  false,
			expectGaming:     false,
			expectWork:       false,
			expectedScore:    20,
		},
		{
			name:             "7.5 Mbps - between slow and standard DSL",
			downloadMbps:     7.5,
			expectedGrade:    GradeD,
			expectedType:     ConnectionTypeStandardDSL,
			expectStreaming:  false,
			expectGaming:     false,
			expectWork:       false,
			expectedScore:    20,
		},
		{
			name:             "10 Mbps - standard DSL to modern DSL boundary",
			downloadMbps:     10,
			expectedGrade:    GradeC,
			expectedType:     ConnectionTypeModernDSL,
			expectStreaming:  false,
			expectGaming:     false,
			expectWork:       true,
			expectedScore:    35,
		},
		{
			name:             "15 Mbps - modern DSL",
			downloadMbps:     15,
			expectedGrade:    GradeC,
			expectedType:     ConnectionTypeModernDSL,
			expectStreaming:  false,
			expectGaming:     false,
			expectWork:       true,
			expectedScore:    35,
		},
		{
			name:             "25 Mbps - modern DSL to slow cable boundary",
			downloadMbps:     25,
			expectedGrade:    GradeC,
			expectedType:     ConnectionTypeSlowCable,
			expectStreaming:  true,
			expectGaming:     false,
			expectWork:       true,
			expectedScore:    50,
		},
		{
			name:             "35 Mbps - slow cable, streaming + gaming ready",
			downloadMbps:     35,
			expectedGrade:    GradeC,
			expectedType:     ConnectionTypeSlowCable,
			expectStreaming:  true,
			expectGaming:     true,
			expectWork:       true,
			expectedScore:    50,
		},
		{
			name:             "50 Mbps - standard cable boundary",
			downloadMbps:     50,
			expectedGrade:    GradeB,
			expectedType:     ConnectionTypeStandardCable,
			expectStreaming:  true,
			expectGaming:     true,
			expectWork:       true,
			expectedScore:    65,
		},
		{
			name:             "100 Mbps - high-speed cable boundary",
			downloadMbps:     100,
			expectedGrade:    GradeA,
			expectedType:     ConnectionTypeHighSpeedCable,
			expectStreaming:  true,
			expectGaming:     true,
			expectWork:       true,
			expectedScore:    75,
		},
		{
			name:             "150 Mbps - between high-speed cable and fiber",
			downloadMbps:     150,
			expectedGrade:    GradeA,
			expectedType:     ConnectionTypeHighSpeedCable,
			expectStreaming:  true,
			expectGaming:     true,
			expectWork:       true,
			expectedScore:    85,
		},
		{
			name:             "300 Mbps - fiber boundary",
			downloadMbps:     300,
			expectedGrade:    GradeA,
			expectedType:     ConnectionTypeFiber,
			expectStreaming:  true,
			expectGaming:     true,
			expectWork:       true,
			expectedScore:    95,
		},
		{
			name:             "500 Mbps - excellent fiber",
			downloadMbps:     500,
			expectedGrade:    GradeA,
			expectedType:     ConnectionTypeFiber,
			expectStreaming:  true,
			expectGaming:     true,
			expectWork:       true,
			expectedScore:    100,
		},
		{
			name:             "1000 Mbps - ultra high speed",
			downloadMbps:     1000,
			expectedGrade:    GradeA,
			expectedType:     ConnectionTypeFiber,
			expectStreaming:  true,
			expectGaming:     true,
			expectWork:       true,
			expectedScore:    100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := ClassifySpeed(tt.downloadMbps)

			if metrics.Grade != tt.expectedGrade {
				t.Errorf("Grade: got %q, want %q", metrics.Grade, tt.expectedGrade)
			}
			if metrics.Classification != tt.expectedType {
				t.Errorf("Classification: got %q, want %q", metrics.Classification, tt.expectedType)
			}
			if metrics.IsGoodForStreaming != tt.expectStreaming {
				t.Errorf("IsGoodForStreaming: got %v, want %v", metrics.IsGoodForStreaming, tt.expectStreaming)
			}
			if metrics.IsGoodForGaming != tt.expectGaming {
				t.Errorf("IsGoodForGaming: got %v, want %v", metrics.IsGoodForGaming, tt.expectGaming)
			}
			if metrics.IsGoodForWork != tt.expectWork {
				t.Errorf("IsGoodForWork: got %v, want %v", metrics.IsGoodForWork, tt.expectWork)
			}
			if metrics.QualityScore != tt.expectedScore {
				t.Errorf("QualityScore: got %f, want %f", metrics.QualityScore, tt.expectedScore)
			}
		})
	}
}

func TestAnalyzeLatency(t *testing.T) {
	tests := []struct {
		name     string
		latency  time.Duration
		expected string
	}{
		{"0ms", 0 * time.Millisecond, "Excellent (< 20ms)"},
		{"10ms", 10 * time.Millisecond, "Excellent (< 20ms)"},
		{"19ms", 19 * time.Millisecond, "Excellent (< 20ms)"},
		{"20ms", 20 * time.Millisecond, "Very Good (20-50ms)"},
		{"35ms", 35 * time.Millisecond, "Very Good (20-50ms)"},
		{"50ms", 50 * time.Millisecond, "Good (50-100ms)"},
		{"75ms", 75 * time.Millisecond, "Good (50-100ms)"},
		{"100ms", 100 * time.Millisecond, "Fair (100-150ms)"},
		{"125ms", 125 * time.Millisecond, "Fair (100-150ms)"},
		{"150ms", 150 * time.Millisecond, "Poor (> 150ms)"},
		{"200ms", 200 * time.Millisecond, "Poor (> 150ms)"},
		{"1s", 1 * time.Second, "Poor (> 150ms)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AnalyzeLatency(tt.latency)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestAnalyzeJitter(t *testing.T) {
	tests := []struct {
		name     string
		jitter   time.Duration
		expected string
	}{
		{"0ms", 0 * time.Millisecond, "Excellent (< 5ms)"},
		{"2ms", 2 * time.Millisecond, "Excellent (< 5ms)"},
		{"5ms", 5 * time.Millisecond, "Very Good (5-10ms)"},
		{"7ms", 7 * time.Millisecond, "Very Good (5-10ms)"},
		{"10ms", 10 * time.Millisecond, "Good (10-20ms)"},
		{"15ms", 15 * time.Millisecond, "Good (10-20ms)"},
		{"20ms", 20 * time.Millisecond, "Fair (20-50ms)"},
		{"35ms", 35 * time.Millisecond, "Fair (20-50ms)"},
		{"50ms", 50 * time.Millisecond, "Poor (> 50ms)"},
		{"75ms", 75 * time.Millisecond, "Poor (> 50ms)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AnalyzeJitter(tt.jitter)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetSpeedComparison(t *testing.T) {
	tests := []struct {
		name         string
		downloadMbps float64
		shouldContain string
	}{
		{"1 Mbps - slower than global avg", 1, "slower than global average"},
		{"40 Mbps - slower than global avg", 40, "slower than global average"},
		{"80 Mbps - equal to global avg", 80, "slower than global average"},
		{"100 Mbps - faster than global", 100, "faster than global average"},
		{"200 Mbps - faster than global", 200, "faster than global average"},
		{"300 Mbps - faster than US avg", 300, "faster than US average"},
		{"400 Mbps - well above US avg", 400, "faster than US average"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetSpeedComparison(tt.downloadMbps)
			if !strings.Contains(result, tt.shouldContain) {
				t.Errorf("got %q, should contain %q", result, tt.shouldContain)
			}
		})
	}
}

