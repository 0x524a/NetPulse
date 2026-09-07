// Package ookla wraps Ookla's own official, closed-source "speedtest" CLI.
//
// Ookla's terms of service for speedtest.net contain an explicit
// anti-automation / anti-reverse-engineering clause covering their
// measurement infrastructure and wire protocol. There is no sanctioned
// public API for third-party native Ookla measurement, so this package does
// not implement, reconstruct, or reverse-engineer that protocol.
//
// Instead, if the user has installed Ookla's own official CLI themselves
// (https://www.speedtest.net/apps/cli), this provider shells out to it and
// parses its JSON output. If the binary is missing, or if what's on PATH is
// not actually the official Ookla build (many distros ship "speedtest" as
// speedtest-cli, an unrelated unofficial Python client), the provider
// reports itself unavailable rather than guessing or driving the wrong tool.
package ookla

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/0x524a/netpulse/internal/providers/provider"
)

const (
	// officialBinaryName is the executable this provider looks for on PATH.
	officialBinaryName = "speedtest"

	// ooklaVersionMarker is the substring the official Ookla CLI prints in its
	// `--version` output. speedtest-cli (the unofficial Python client that
	// many distros install under the same "speedtest" name) does not print
	// this, which is how the two are told apart.
	ooklaVersionMarker = "Speedtest by Ookla"

	// versionCheckTimeout bounds the local `--version` invocation used to
	// verify the binary's identity during Init(). This never touches the
	// network.
	versionCheckTimeout = 5 * time.Second

	// measureTimeout bounds a single `speedtest --format=json` run (ping +
	// download + upload). exec.CommandContext ensures a hung binary is
	// killed rather than wedging the caller indefinitely.
	measureTimeout = 2 * time.Minute
)

// Client wraps the official Ookla Speedtest CLI. It never talks to any
// Ookla host itself; all network activity happens inside the external
// binary that the user installed and licensed on their own.
type Client struct {
	mu sync.Mutex

	// binaryPath is set by Init() once the official binary has been located
	// and its identity verified. Empty means the provider is unavailable.
	binaryPath string

	// result and runErr cache the outcome of the single `speedtest
	// --format=json` invocation. One run yields ping+download+upload
	// together, but the Provider interface exposes them as separate
	// methods, so the first method that needs data runs the binary once and
	// every other method (and every later call) is served from this cache.
	result *ooklaResult
	runErr error
}

var _ provider.Provider = (*Client)(nil)

// New creates a new Ookla CLI wrapper. Call Init() before use.
func New() *Client {
	return &Client{}
}

// Name returns the provider name, naming the underlying mechanism so it's
// clear to users this is not a NetPulse-native implementation.
func (c *Client) Name() string {
	return "Speedtest.net (Ookla official CLI)"
}

// IsAvailable reports whether Init() has located and verified the official
// binary. It performs no network measurement -- it only reflects whether
// Init() previously succeeded.
func (c *Client) IsAvailable() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.binaryPath != ""
}

// Init locates Ookla's official "speedtest" CLI on PATH and verifies that it
// really is the official Ookla build rather than the unrelated, unofficial
// speedtest-cli that many Linux distributions install under the same name.
//
// It deliberately does not run a measurement (which would be needed to
// discover whether the CLI's license/GDPR terms have been accepted) and it
// never passes --accept-license or --accept-gdpr on the user's behalf: this
// package will not accept those terms silently. If acceptance is still
// needed, that surfaces as a clear error from the first Measure* call
// instead.
func (c *Client) Init() error {
	path, err := exec.LookPath(officialBinaryName)
	if err != nil {
		return fmt.Errorf(
			"official Ookla Speedtest CLI (%q) was not found on PATH; install it from "+
				"https://www.speedtest.net/apps/cli -- NetPulse cannot measure Ookla speeds "+
				"without it, because Ookla's terms of service do not permit reimplementing "+
				"their protocol: %w", officialBinaryName, provider.ErrProviderUnavailable)
	}

	ctx, cancel := context.WithTimeout(context.Background(), versionCheckTimeout)
	defer cancel()

	// #nosec G204 -- path comes from exec.LookPath (not user input) and the
	// argument list is a fixed literal slice; nothing here is interpolated
	// through a shell.
	cmd := exec.CommandContext(ctx, path, "--version")
	out, verErr := cmd.CombinedOutput()
	if verErr != nil {
		return fmt.Errorf(
			"found %q on PATH but could not run %q --version (%v): %w",
			path, path, verErr, provider.ErrProviderUnavailable)
	}

	if !bytes.Contains(out, []byte(ooklaVersionMarker)) {
		return fmt.Errorf(
			"found %q on PATH, but it does not identify itself as %q; this is likely "+
				"speedtest-cli (an unofficial Python client with different flags and its own "+
				"terms-of-service exposure), not Ookla's official CLI -- install the official "+
				"CLI from https://www.speedtest.net/apps/cli: %w",
			path, ooklaVersionMarker, provider.ErrProviderUnavailable)
	}

	c.mu.Lock()
	c.binaryPath = path
	c.result = nil
	c.runErr = nil
	c.mu.Unlock()

	return nil
}

// ooklaPing mirrors the "ping" object in the official CLI's JSON output.
// Latency and Jitter are both in milliseconds.
type ooklaPing struct {
	Latency float64 `json:"latency"`
	Jitter  float64 `json:"jitter"`
}

// ooklaTransfer mirrors the "download"/"upload" objects in the official
// CLI's JSON output. Bandwidth is in bytes per second (not bits, and not
// bytes total) -- Bytes is the total bytes transferred, Elapsed is
// milliseconds.
type ooklaTransfer struct {
	Bandwidth float64 `json:"bandwidth"`
	Bytes     int64   `json:"bytes"`
	Elapsed   int64   `json:"elapsed"`
}

// ooklaServer mirrors the "server" object in the official CLI's JSON output.
type ooklaServer struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Host string `json:"host"`
}

// ooklaResultLink mirrors the "result" object in the official CLI's JSON
// output (the shareable results-page URL).
type ooklaResultLink struct {
	URL string `json:"url"`
}

// ooklaJSON is the (documented) shape of `speedtest --format=json` output:
//
//	{
//	  "ping":     {"latency": ..., "jitter": ...},
//	  "download": {"bandwidth": ..., "bytes": ..., "elapsed": ...},
//	  "upload":   {"bandwidth": ..., "bytes": ..., "elapsed": ...},
//	  "server":   {"id": ..., "name": ..., "host": ...},
//	  "result":   {"url": ...}
//	}
type ooklaJSON struct {
	Ping     ooklaPing       `json:"ping"`
	Download ooklaTransfer   `json:"download"`
	Upload   ooklaTransfer   `json:"upload"`
	Server   ooklaServer     `json:"server"`
	Result   ooklaResultLink `json:"result"`
}

// ooklaResult is the parsed, unit-converted outcome of a single speedtest
// run, cached on Client so every Provider method can be served from it.
type ooklaResult struct {
	latency      time.Duration
	jitter       time.Duration
	downloadKbps float64
	uploadKbps   float64
	server       ooklaServer
	resultURL    string
}

// bandwidthToMbps converts the official CLI's "bandwidth" field -- reported
// in bytes per second -- into megabits per second.
func bandwidthToMbps(bytesPerSecond float64) float64 {
	return bytesPerSecond * 8 / 1e6
}

// bandwidthToKbps converts the official CLI's "bandwidth" field -- reported
// in bytes per second -- into kilobits per second, which is the unit the
// Provider interface's speedChan expects.
func bandwidthToKbps(bytesPerSecond float64) float64 {
	return bandwidthToMbps(bytesPerSecond) * 1000
}

// millisToDuration converts a millisecond float (as used by the official
// CLI's "latency"/"jitter" fields) into a time.Duration.
func millisToDuration(ms float64) time.Duration {
	return time.Duration(ms * float64(time.Millisecond))
}

// parseOoklaResult parses `speedtest --format=json` output into an
// ooklaResult, performing all unit conversions up front.
func parseOoklaResult(data []byte) (*ooklaResult, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, errors.New("ookla speedtest command produced no JSON output")
	}

	var parsed ooklaJSON
	if err := json.Unmarshal(trimmed, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse ookla speedtest JSON output: %w", err)
	}

	return &ooklaResult{
		latency:      millisToDuration(parsed.Ping.Latency),
		jitter:       millisToDuration(parsed.Ping.Jitter),
		downloadKbps: bandwidthToKbps(parsed.Download.Bandwidth),
		uploadKbps:   bandwidthToKbps(parsed.Upload.Bandwidth),
		server:       parsed.Server,
		resultURL:    parsed.Result.URL,
	}, nil
}

// looksLikeLicenseError reports whether stderr text from the official CLI
// indicates it refused to run because its license/GDPR terms have not been
// accepted yet.
func looksLikeLicenseError(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "license") || strings.Contains(lower, "gdpr")
}

// runSpeedtestLocked invokes the official CLI once and parses its output.
// Callers must hold c.mu; it is called that way deliberately so that a
// second caller arriving while a run is already in flight blocks and then
// reads the same cached result, rather than starting a duplicate run.
func (c *Client) runSpeedtestLocked() (*ooklaResult, error) {
	if c.binaryPath == "" {
		return nil, fmt.Errorf("ookla provider not initialized (call Init() first): %w", provider.ErrProviderUnavailable)
	}

	ctx, cancel := context.WithTimeout(context.Background(), measureTimeout)
	defer cancel()

	// #nosec G204 -- binaryPath was resolved via exec.LookPath and its
	// identity verified in Init(); the argument list is a fixed literal
	// slice with no user-controlled or shell-interpreted content.
	cmd := exec.CommandContext(ctx, c.binaryPath, "--format=json")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrMsg := strings.TrimSpace(stderr.String())
		if looksLikeLicenseError(stderrMsg) {
			return nil, fmt.Errorf(
				"the official Ookla Speedtest CLI at %q has not had its license/GDPR terms "+
					"accepted yet; NetPulse will not accept them on your behalf -- run %q "+
					"yourself once (interactively, or with --accept-license --accept-gdpr) and "+
					"then retry: %s", c.binaryPath, c.binaryPath, stderrMsg)
		}
		if stderrMsg == "" {
			return nil, fmt.Errorf("ookla speedtest command failed: %w", err)
		}
		return nil, fmt.Errorf("ookla speedtest command failed: %w: %s", err, stderrMsg)
	}

	return parseOoklaResult(stdout.Bytes())
}

// ensureResultLocked returns the cached result, running the speedtest binary
// exactly once (on the first call that needs it) if it hasn't run yet.
// Callers must hold c.mu.
func (c *Client) ensureResultLocked() (*ooklaResult, error) {
	if c.result == nil && c.runErr == nil {
		c.result, c.runErr = c.runSpeedtestLocked()
	}
	return c.result, c.runErr
}

// MeasureDownload implements the Provider interface. It runs the official
// CLI (or reuses the cached result from an earlier Measure* call) and
// reports the download speed, in Kbps, on speedChan.
func (c *Client) MeasureDownload(speedChan chan<- float64) error {
	c.mu.Lock()
	result, err := c.ensureResultLocked()
	c.mu.Unlock()
	if err != nil {
		return err
	}

	speedChan <- result.downloadKbps
	return nil
}

// MeasureUpload implements the Provider interface. The official CLI runs
// its own fixed-length upload test as part of a single combined
// measurement, so the requested duration cannot be honored independently
// and is ignored; the cached result (shared with MeasureDownload,
// MeasureLatency, and MeasureJitter) is reused or produced here.
func (c *Client) MeasureUpload(_ time.Duration, speedChan chan<- float64) error {
	c.mu.Lock()
	result, err := c.ensureResultLocked()
	c.mu.Unlock()
	if err != nil {
		return err
	}

	speedChan <- result.uploadKbps
	return nil
}

// MeasureLatency implements the Provider interface, reporting the ping
// latency from the (possibly cached) single speedtest run.
func (c *Client) MeasureLatency() (time.Duration, error) {
	c.mu.Lock()
	result, err := c.ensureResultLocked()
	c.mu.Unlock()
	if err != nil {
		return 0, err
	}

	return result.latency, nil
}

// MeasureJitter implements the Provider interface. The official CLI reports
// a single jitter figure as part of its combined run rather than taking a
// configurable number of samples, so samples is ignored.
func (c *Client) MeasureJitter(_ int) (time.Duration, error) {
	c.mu.Lock()
	result, err := c.ensureResultLocked()
	c.mu.Unlock()
	if err != nil {
		return 0, err
	}

	return result.jitter, nil
}
