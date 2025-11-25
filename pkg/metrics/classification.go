package metrics

import (
	"fmt"
	"time"
)

// ConnectionType represents the type of internet connection
type ConnectionType string

const (
	ConnectionTypeSlowDSL        ConnectionType = "Slow DSL"
	ConnectionTypeStandardDSL    ConnectionType = "Standard DSL"
	ConnectionTypeModernDSL      ConnectionType = "Modern DSL"
	ConnectionTypeSlowCable      ConnectionType = "Slow Cable"
	ConnectionTypeStandardCable  ConnectionType = "Standard Cable"
	ConnectionTypeHighSpeedCable ConnectionType = "High-Speed Cable"
	ConnectionTypeFiber          ConnectionType = "Fiber"
	ConnectionTypeMobile4G       ConnectionType = "Mobile 4G"
	ConnectionTypeMobile5G       ConnectionType = "Mobile 5G"
)

// SpeedGrade represents the quality grade (A-F)
type SpeedGrade string

const (
	GradeA SpeedGrade = "A"
	GradeB SpeedGrade = "B"
	GradeC SpeedGrade = "C"
	GradeD SpeedGrade = "D"
	GradeF SpeedGrade = "F"
)

// ConnectionMetrics holds classified connection metrics
type ConnectionMetrics struct {
	Grade              SpeedGrade
	Classification     ConnectionType
	QualityScore       float64 // 0-100
	IsGoodForStreaming bool
	IsGoodForGaming    bool
	IsGoodForWork      bool
}

// ClassifySpeed determines connection type and quality based on download speed
func ClassifySpeed(downloadMbps float64) ConnectionMetrics {
	metrics := ConnectionMetrics{
		QualityScore:       calculateQualityScore(downloadMbps),
		IsGoodForStreaming: downloadMbps >= 25,
		IsGoodForGaming:    downloadMbps >= 35,
		IsGoodForWork:      downloadMbps >= 10,
	}

	// Classify connection type
	switch {
	case downloadMbps < 5:
		metrics.Classification = ConnectionTypeSlowDSL
		metrics.Grade = GradeF
	case downloadMbps < 10:
		metrics.Classification = ConnectionTypeStandardDSL
		metrics.Grade = GradeD
	case downloadMbps < 25:
		metrics.Classification = ConnectionTypeModernDSL
		metrics.Grade = GradeC
	case downloadMbps < 50:
		metrics.Classification = ConnectionTypeSlowCable
		metrics.Grade = GradeC
	case downloadMbps < 100:
		metrics.Classification = ConnectionTypeStandardCable
		metrics.Grade = GradeB
	case downloadMbps < 300:
		metrics.Classification = ConnectionTypeHighSpeedCable
		metrics.Grade = GradeA
	default:
		metrics.Classification = ConnectionTypeFiber
		metrics.Grade = GradeA
	}

	return metrics
}

// AnalyzeLatency provides feedback on latency quality
func AnalyzeLatency(latency time.Duration) string {
	ms := float64(latency.Milliseconds())
	switch {
	case ms < 20:
		return "Excellent (< 20ms)"
	case ms < 50:
		return "Very Good (20-50ms)"
	case ms < 100:
		return "Good (50-100ms)"
	case ms < 150:
		return "Fair (100-150ms)"
	default:
		return "Poor (> 150ms)"
	}
}

// AnalyzeJitter provides feedback on jitter quality
func AnalyzeJitter(jitter time.Duration) string {
	ms := float64(jitter.Milliseconds())
	switch {
	case ms < 5:
		return "Excellent (< 5ms)"
	case ms < 10:
		return "Very Good (5-10ms)"
	case ms < 20:
		return "Good (10-20ms)"
	case ms < 50:
		return "Fair (20-50ms)"
	default:
		return "Poor (> 50ms)"
	}
}

// calculateQualityScore calculates a 0-100 quality score
func calculateQualityScore(downloadMbps float64) float64 {
	switch {
	case downloadMbps >= 500:
		return 100
	case downloadMbps >= 300:
		return 95
	case downloadMbps >= 150:
		return 85
	case downloadMbps >= 100:
		return 75
	case downloadMbps >= 50:
		return 65
	case downloadMbps >= 25:
		return 50
	case downloadMbps >= 10:
		return 35
	case downloadMbps >= 5:
		return 20
	default:
		return 10
	}
}

// GetSpeedComparison returns comparison text with average speeds
func GetSpeedComparison(downloadMbps float64) string {
	// US average broadband (2024): ~200 Mbps
	// Global average: ~80 Mbps
	uSAverage := 200.0
	globalAverage := 80.0

	if downloadMbps > uSAverage {
		percent := ((downloadMbps - uSAverage) / uSAverage) * 100
		return fmt.Sprintf("%.0f%% faster than US average (%.0f Mbps)", percent, uSAverage)
	} else if downloadMbps > globalAverage {
		percent := ((downloadMbps - globalAverage) / globalAverage) * 100
		return fmt.Sprintf("%.0f%% faster than global average (%.0f Mbps)", percent, globalAverage)
	} else {
		percent := ((globalAverage - downloadMbps) / globalAverage) * 100
		return fmt.Sprintf("%.0f%% slower than global average (%.0f Mbps)", percent, globalAverage)
	}
}
