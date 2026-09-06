package provider

import (
	"errors"
	"time"
)

// ErrUploadNotSupported is returned by MeasureUpload for providers whose
// service genuinely exposes no upload endpoint. Fast.com is the case in point:
// its public API is download-only, so an honest client reports the absence
// rather than measuring an unrelated host and presenting the number as the
// provider's upload speed. Callers should treat this as "no data", not as a
// test failure.
var ErrUploadNotSupported = errors.New("upload measurement is not supported by this provider")

// ErrProviderUnavailable is returned by Init when a provider cannot run in the
// current environment for a reason the user could act on -- for example, the
// Ookla provider requires Ookla's own official speedtest binary to be
// installed, because reimplementing their wire protocol is not permitted by
// their terms of use.
var ErrProviderUnavailable = errors.New("provider is not available in this environment")

// Provider defines the interface that all speed test providers must implement
type Provider interface {
	// Name returns the name of the provider
	Name() string

	// Init initializes the provider (fetch tokens, endpoints, etc.)
	Init() error

	// MeasureDownload performs a download speed test
	// speedChan receives speed updates in Kbps
	MeasureDownload(speedChan chan<- float64) error

	// MeasureUpload performs an upload speed test
	// duration specifies how long to run the test
	// speedChan receives speed updates in Kbps
	MeasureUpload(duration time.Duration, speedChan chan<- float64) error

	// MeasureLatency measures the round-trip time to the test servers
	MeasureLatency() (time.Duration, error)

	// MeasureJitter measures latency variation
	MeasureJitter(samples int) (time.Duration, error)

	// IsAvailable checks if the provider is currently accessible
	IsAvailable() bool
}

// Result holds comprehensive speed test results
type Result struct {
	ProviderName        string
	DownloadSpeed       float64 // Kbps
	DownloadMbps        float64
	MinDownload         float64
	MaxDownload         float64
	AvgDownload         float64
	StdDevDownload      float64 // Standard deviation
	UploadSpeed         float64 // Kbps
	UploadMbps          float64
	MinUpload           float64
	MaxUpload           float64
	AvgUpload           float64
	StdDevUpload        float64       // Standard deviation
	Latency             time.Duration // Average RTT
	Jitter              time.Duration // Latency variation
	IdleLatency         time.Duration // Baseline latency
	PacketLoss          float64       // Percentage
	DownloadTime        time.Duration
	UploadTime          time.Duration
	Timestamp           time.Time
	Success             bool
	Error               string
	LatencyMeasurements []time.Duration
}
