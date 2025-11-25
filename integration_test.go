package speedtestclient
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




























































}	}		t.Error("Expected positive download speed")	if result.DownloadMbps <= 0 {	}		return		t.Logf("Random server test failed: %v", err)	if err != nil {	result, err := client.Run()	)		speedtest.WithURLCount(2),		speedtest.WithDuration(5*time.Second),		speedtest.WithServerMode("random"),	client := speedtest.New(	}		t.Skip("Skipping integration test in short mode")	if testing.Short() {func TestIntegrationRandomServer(t *testing.T) {}	t.Logf("Used provider: %s", result.ProviderName)	}		t.Error("Expected positive download speed")	if result.DownloadMbps <= 0 {	}		t.Fatalf("Auto fallback failed: %v", err)	if err != nil {	result, err := client.Run()	)		speedtest.WithURLCount(2),		speedtest.WithDuration(5*time.Second),		speedtest.WithProvider("auto"),	client := speedtest.New(	}		t.Skip("Skipping integration test in short mode")	if testing.Short() {func TestIntegrationAutoFallback(t *testing.T) {}	}		t.Error("Expected positive download speed")	if result.DownloadMbps <= 0 {	}		return		t.Logf("Fast.com test failed (may be unavailable): %v", err)	if err != nil {	result, err := client.Run()	)		speedtest.WithVerbose(true),		speedtest.WithURLCount(2),		speedtest.WithDuration(5*time.Second),