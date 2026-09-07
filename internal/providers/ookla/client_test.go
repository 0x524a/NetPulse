package ookla

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0x524a/netpulse/internal/providers/provider"
)

// fakeOfficialScript is a stand-in for Ookla's real "speedtest" binary. It
// answers --version the way the real CLI does, and answers --format=json by
// echoing $FAKE_JSON_FILE's contents to stdout (recording the call in
// $FAKE_COUNTER_FILE) or, if $FAKE_EXIT_NONZERO is set, by writing
// $FAKE_STDERR_MSG to stderr and exiting 1.
const fakeOfficialScript = `#!/bin/sh
case "$1" in
  --version)
    echo "Speedtest by Ookla 1.2.0.84 (Linux/x86_64), Runtime: OoklaCLI"
    exit 0
    ;;
  --format=json)
    if [ -n "$FAKE_EXIT_NONZERO" ]; then
      echo "$FAKE_STDERR_MSG" 1>&2
      exit 1
    fi
    if [ -n "$FAKE_COUNTER_FILE" ]; then
      echo run >> "$FAKE_COUNTER_FILE"
    fi
    cat "$FAKE_JSON_FILE"
    exit 0
    ;;
esac
exit 1
`

// fakeUnofficialScript stands in for speedtest-cli, the unrelated unofficial
// Python client that many Linux distros install as "speedtest".
const fakeUnofficialScript = `#!/bin/sh
case "$1" in
  --version)
    echo "speedtest-cli 2.1.3"
    exit 0
    ;;
esac
exit 1
`

const validOoklaJSON = `{
  "type": "result",
  "ping": {"jitter": 1.234, "latency": 12.5},
  "download": {"bandwidth": 12500000, "bytes": 125000000, "elapsed": 10000},
  "upload": {"bandwidth": 1250000, "bytes": 6250000, "elapsed": 5000},
  "server": {"id": 1234, "host": "speedtest.example.net", "name": "Example ISP"},
  "result": {"id": "abcd-1234", "url": "https://www.speedtest.net/result/c/abcd-1234"}
}`

// installFakeBinary writes the given script as an executable file named
// "speedtest" in a fresh temp directory and puts that directory first on
// PATH, so exec.LookPath("speedtest") in the code under test resolves to it
// deterministically regardless of what (if anything) is really installed.
func installFakeBinary(t *testing.T, script string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "speedtest")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write fake binary: %v", err)
	}

	// Put dir first so exec.LookPath("speedtest") finds our fake binary, but
	// keep the real PATH behind it so the fake script's own shebang and its
	// use of ordinary tools (cat, echo, sh) keep working.
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return path
}

func TestName(t *testing.T) {
	client := New()
	if got := client.Name(); got == "" || !strings.Contains(got, "Ookla") {
		t.Errorf("Name() = %q, want it to mention Ookla", got)
	}
}

func TestIsAvailable_BeforeInit(t *testing.T) {
	client := New()
	if client.IsAvailable() {
		t.Error("IsAvailable() = true before Init(), want false")
	}
}

func TestInit_BinaryMissing(t *testing.T) {
	// Point PATH at an empty directory so exec.LookPath("speedtest") fails
	// regardless of what's really installed on the host running the tests.
	t.Setenv("PATH", t.TempDir())

	client := New()
	err := client.Init()
	if err == nil {
		t.Fatal("Init() = nil error, want an error when the binary is missing")
	}
	if !errors.Is(err, provider.ErrProviderUnavailable) {
		t.Errorf("Init() error = %v, want it to wrap provider.ErrProviderUnavailable", err)
	}
	if !strings.Contains(err.Error(), "speedtest.net/apps/cli") {
		t.Errorf("Init() error = %v, want it to point the user at https://www.speedtest.net/apps/cli", err)
	}
	if client.IsAvailable() {
		t.Error("IsAvailable() = true after failed Init(), want false")
	}
}

func TestInit_UnofficialBinaryTreatedAsUnavailable(t *testing.T) {
	installFakeBinary(t, fakeUnofficialScript)

	client := New()
	err := client.Init()
	if err == nil {
		t.Fatal("Init() = nil error, want an error for a non-official 'speedtest' binary")
	}
	if !errors.Is(err, provider.ErrProviderUnavailable) {
		t.Errorf("Init() error = %v, want it to wrap provider.ErrProviderUnavailable", err)
	}
	if !strings.Contains(err.Error(), "speedtest-cli") {
		t.Errorf("Init() error = %v, want it to name speedtest-cli as the likely culprit", err)
	}
	if client.IsAvailable() {
		t.Error("IsAvailable() = true after Init() rejected an unofficial binary, want false")
	}
}

func TestInit_OfficialBinarySucceeds(t *testing.T) {
	installFakeBinary(t, fakeOfficialScript)

	client := New()
	if err := client.Init(); err != nil {
		t.Fatalf("Init() error = %v, want nil for a binary that identifies as official", err)
	}
	if !client.IsAvailable() {
		t.Error("IsAvailable() = false after successful Init(), want true")
	}
}

func TestParseOoklaResult(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		wantErr     bool
		wantLatency time.Duration
		wantJitter  time.Duration
		wantDLKbps  float64
		wantULKbps  float64
	}{
		{
			name:        "valid full document",
			json:        validOoklaJSON,
			wantLatency: 12*time.Millisecond + 500*time.Microsecond,
			wantJitter:  1*time.Millisecond + 234*time.Microsecond,
			// 12,500,000 bytes/sec * 8 / 1000 = 100,000 Kbps (100 Mbps)
			wantDLKbps: 100000,
			// 1,250,000 bytes/sec * 8 / 1000 = 10,000 Kbps (10 Mbps)
			wantULKbps: 10000,
		},
		{
			name:    "empty output",
			json:    "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			json:    "   \n  ",
			wantErr: true,
		},
		{
			name:    "malformed json",
			json:    `{"ping": {"latency": `,
			wantErr: true,
		},
		{
			name:    "not an object",
			json:    `[1, 2, 3]`,
			wantErr: true,
		},
		{
			name:        "partial document -- only ping present",
			json:        `{"ping": {"latency": 5, "jitter": 0.5}}`,
			wantLatency: 5 * time.Millisecond,
			wantJitter:  500 * time.Microsecond,
			wantDLKbps:  0,
			wantULKbps:  0,
		},
		{
			name:        "zero bandwidth",
			json:        `{"download": {"bandwidth": 0}, "upload": {"bandwidth": 0}}`,
			wantDLKbps:  0,
			wantULKbps:  0,
			wantLatency: 0,
			wantJitter:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseOoklaResult([]byte(tt.json))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseOoklaResult() error = nil, want an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseOoklaResult() error = %v, want nil", err)
			}
			if result.latency != tt.wantLatency {
				t.Errorf("latency = %v, want %v", result.latency, tt.wantLatency)
			}
			if result.jitter != tt.wantJitter {
				t.Errorf("jitter = %v, want %v", result.jitter, tt.wantJitter)
			}
			if result.downloadKbps != tt.wantDLKbps {
				t.Errorf("downloadKbps = %v, want %v", result.downloadKbps, tt.wantDLKbps)
			}
			if result.uploadKbps != tt.wantULKbps {
				t.Errorf("uploadKbps = %v, want %v", result.uploadKbps, tt.wantULKbps)
			}
		})
	}
}

func TestBandwidthConversions(t *testing.T) {
	tests := []struct {
		bytesPerSec float64
		wantMbps    float64
		wantKbps    float64
	}{
		{bytesPerSec: 0, wantMbps: 0, wantKbps: 0},
		{bytesPerSec: 125000, wantMbps: 1, wantKbps: 1000},
		{bytesPerSec: 12500000, wantMbps: 100, wantKbps: 100000},
		{bytesPerSec: 1250000, wantMbps: 10, wantKbps: 10000},
	}

	for _, tt := range tests {
		if got := bandwidthToMbps(tt.bytesPerSec); got != tt.wantMbps {
			t.Errorf("bandwidthToMbps(%v) = %v, want %v", tt.bytesPerSec, got, tt.wantMbps)
		}
		if got := bandwidthToKbps(tt.bytesPerSec); got != tt.wantKbps {
			t.Errorf("bandwidthToKbps(%v) = %v, want %v", tt.bytesPerSec, got, tt.wantKbps)
		}
	}
}

// TestMeasure_RunsBinaryExactlyOnce verifies that a single cached speedtest
// invocation serves MeasureDownload, MeasureUpload, MeasureLatency and
// MeasureJitter, even when they're called concurrently -- and does so
// cleanly under -race, since a sibling provider package's cache is known to
// fail the race detector.
func TestMeasure_RunsBinaryExactlyOnce(t *testing.T) {
	installFakeBinary(t, fakeOfficialScript)

	dir := t.TempDir()
	jsonFile := filepath.Join(dir, "result.json")
	if err := os.WriteFile(jsonFile, []byte(validOoklaJSON), 0o644); err != nil {
		t.Fatalf("failed to write fixture JSON: %v", err)
	}
	counterFile := filepath.Join(dir, "counter")

	t.Setenv("FAKE_JSON_FILE", jsonFile)
	t.Setenv("FAKE_COUNTER_FILE", counterFile)

	client := New()
	if err := client.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	var wg sync.WaitGroup
	errs := make(chan error, 4)
	downloadChan := make(chan float64, 1)
	uploadChan := make(chan float64, 1)

	wg.Add(4)
	go func() {
		defer wg.Done()
		errs <- client.MeasureDownload(downloadChan)
	}()
	go func() {
		defer wg.Done()
		errs <- client.MeasureUpload(5*time.Second, uploadChan)
	}()
	go func() {
		defer wg.Done()
		_, err := client.MeasureLatency()
		errs <- err
	}()
	go func() {
		defer wg.Done()
		_, err := client.MeasureJitter(5)
		errs <- err
	}()
	wg.Wait()
	close(errs)

	anyErr := false
	for err := range errs {
		if err != nil {
			anyErr = true
			t.Errorf("unexpected error from concurrent Measure* call: %v", err)
		}
	}
	if anyErr {
		t.Fatal("aborting: a concurrent Measure* call failed, so speedChan will never receive a value")
	}

	var gotDL, gotUL float64
	select {
	case gotDL = <-downloadChan:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for MeasureDownload's speedChan value")
	}
	select {
	case gotUL = <-uploadChan:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for MeasureUpload's speedChan value")
	}
	if gotDL != 100000 {
		t.Errorf("MeasureDownload sent %v Kbps, want 100000", gotDL)
	}
	if gotUL != 10000 {
		t.Errorf("MeasureUpload sent %v Kbps, want 10000", gotUL)
	}

	latency, err := client.MeasureLatency()
	if err != nil {
		t.Fatalf("MeasureLatency() error = %v", err)
	}
	if want := 12*time.Millisecond + 500*time.Microsecond; latency != want {
		t.Errorf("MeasureLatency() = %v, want %v", latency, want)
	}

	jitter, err := client.MeasureJitter(1)
	if err != nil {
		t.Fatalf("MeasureJitter() error = %v", err)
	}
	if want := 1*time.Millisecond + 234*time.Microsecond; jitter != want {
		t.Errorf("MeasureJitter() = %v, want %v", jitter, want)
	}

	counterData, err := os.ReadFile(counterFile)
	if err != nil {
		t.Fatalf("failed to read counter file: %v", err)
	}
	invocations := len(strings.Fields(string(counterData)))
	if invocations != 1 {
		t.Errorf("speedtest binary was invoked %d times, want exactly 1 (cache should serve every method)", invocations)
	}
}

// TestMeasure_NonZeroExitSurfacesError verifies that a failing binary run
// produces an error carrying its stderr, and never a fabricated
// measurement.
func TestMeasure_NonZeroExitSurfacesError(t *testing.T) {
	installFakeBinary(t, fakeOfficialScript)
	t.Setenv("FAKE_EXIT_NONZERO", "1")
	t.Setenv("FAKE_STDERR_MSG", "boom: network unreachable")

	client := New()
	if err := client.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	speedChan := make(chan float64, 1)
	err := client.MeasureDownload(speedChan)
	if err == nil {
		t.Fatal("MeasureDownload() error = nil, want an error when the binary exits non-zero")
	}
	if !strings.Contains(err.Error(), "boom: network unreachable") {
		t.Errorf("MeasureDownload() error = %v, want it to include the binary's stderr", err)
	}
	select {
	case v := <-speedChan:
		t.Errorf("speedChan received %v after a failed run, want no fabricated value", v)
	default:
	}
}

// TestMeasure_LicenseNotAcceptedSurfacesActionableError verifies that a
// license/GDPR refusal from the binary is surfaced as an actionable error,
// and that this package never passes --accept-license/--accept-gdpr itself.
func TestMeasure_LicenseNotAcceptedSurfacesActionableError(t *testing.T) {
	installFakeBinary(t, fakeOfficialScript)
	t.Setenv("FAKE_EXIT_NONZERO", "1")
	t.Setenv("FAKE_STDERR_MSG", "You must accept the license and privacy policy (GDPR) to continue")

	client := New()
	if err := client.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	_, err := client.MeasureLatency()
	if err == nil {
		t.Fatal("MeasureLatency() error = nil, want an error when the license has not been accepted")
	}
	if !strings.Contains(err.Error(), "license") {
		t.Errorf("MeasureLatency() error = %v, want it to mention the license", err)
	}
	if !strings.Contains(err.Error(), "will not accept them on your behalf") {
		t.Errorf("MeasureLatency() error = %v, want it to make clear NetPulse won't auto-accept", err)
	}
}

func TestMeasure_BeforeInitReturnsUnavailable(t *testing.T) {
	client := New()
	_, err := client.MeasureLatency()
	if err == nil {
		t.Fatal("MeasureLatency() error = nil, want an error before Init()")
	}
	if !errors.Is(err, provider.ErrProviderUnavailable) {
		t.Errorf("MeasureLatency() error = %v, want it to wrap provider.ErrProviderUnavailable", err)
	}
}

// TestRealBinaryIntegration only runs against a genuinely installed,
// official Ookla CLI. It's skipped in short mode and skipped whenever no
// such binary is actually available, which is expected to be every CI run.
func TestRealBinaryIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real-binary integration test in short mode")
	}

	client := New()
	if err := client.Init(); err != nil {
		t.Skipf("no official Ookla speedtest CLI available: %v", err)
	}
	if !client.IsAvailable() {
		t.Skip("provider reports unavailable after Init()")
	}

	latency, err := client.MeasureLatency()
	if err != nil {
		t.Fatalf("MeasureLatency() error = %v", err)
	}
	t.Logf("real binary latency: %v", latency)
}
