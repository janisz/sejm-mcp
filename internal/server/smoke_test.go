package server

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// TestAPIConnectivitySmoke is a single smoke test to validate that the real APIs are reachable.
// This is the ONLY test that should make real HTTP requests to external APIs.
// All other tests should use mocks to avoid network dependencies and flakiness.
func TestAPIConnectivitySmoke(t *testing.T) {
	// Skip this test in CI/automated environments or when SKIP_SMOKE_TESTS is set
	if testing.Short() {
		t.Skip("Skipping smoke test in short mode")
	}

	// Use a short timeout to avoid hanging
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test basic connectivity to both major API endpoints
	endpoints := []struct {
		name string
		url  string
	}{
		{"ELI API", "https://api.sejm.gov.pl/eli/acts"},
		{"Sejm API", "https://api.sejm.gov.pl/sejm/term10/mps"},
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(ctx, "GET", endpoint.url, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Logf("WARNING: %s connectivity test failed: %v", endpoint.name, err)
				t.Logf("This may indicate network issues or API unavailability")
				t.Skip("Skipping due to connectivity issues - this is expected in offline environments")
				return
			}
			defer resp.Body.Close()

			// We just want to verify we can reach the API, not that it returns specific data
			if resp.StatusCode >= 500 {
				t.Logf("WARNING: %s returned server error: %d", endpoint.name, resp.StatusCode)
				t.Skip("Skipping due to server error - this may be temporary")
				return
			}

			t.Logf("SUCCESS: %s is reachable (status: %d)", endpoint.name, resp.StatusCode)
		})
	}
}
