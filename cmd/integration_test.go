// +build integration

package main

import (
	"testing"
	"time"

	"github.com/0x524a/netpulse/pkg/speedtest"
)

// Integration tests that require network access
// Run with: go test -tags=integration

func TestIntegrationFastcom(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := speedtest.New(
		speedtest.WithProvider("fastcom"),
		speedtest.WithVerbose(true),
		speedtest.WithURLCount(2),
		speedtest.WithDuration(5*time.Second),
	)

	result, err := client.Run()
	if err != nil {
		t.Logf("Fast.com test failed (may be unavailable): %v", err)
		return
	}

	if result.DownloadMbps <= 0 {
		t.Error("Expected positive download speed")
	}
	t.Logf("Used provider: %s", result.ProviderName)
}

func TestIntegrationAutoFallback(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := speedtest.New(
		speedtest.WithProvider("auto"),
		speedtest.WithURLCount(2),
		speedtest.WithDuration(5*time.Second),
	)

	result, err := client.Run()
	if err != nil {
		t.Fatalf("Auto fallback failed: %v", err)
	}

	if result.DownloadMbps <= 0 {
		t.Error("Expected positive download speed")
	}
}

func TestIntegrationRandomServer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := speedtest.New(
		speedtest.WithServerMode("random"),
		speedtest.WithURLCount(2),
		speedtest.WithDuration(5*time.Second),
	)

	result, err := client.Run()
	if err != nil {
		t.Logf("Random server test failed: %v", err)
		return
	}

	if result.DownloadMbps <= 0 {
		t.Error("Expected positive download speed")
	}
}
