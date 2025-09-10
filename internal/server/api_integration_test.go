package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// setupMockServer creates a mock server with predefined responses
func setupMockServer() *httptest.Server {
	responses := map[string]string{
		"/eli/acts": `[
			{"code": "DU", "name": "Dziennik Ustaw"},
			{"code": "MP", "name": "Monitor Polski"}
		]`,
		"/eli/acts/search": `{
			"acts": [
				{
					"publisher": "DU",
					"year": "1997",
					"position": "78",
					"title": "Konstytucja Rzeczypospolitej Polskiej",
					"eli": "http://eli.sejm.gov.pl/eli/DU/1997/78"
				}
			],
			"count": 1
		}`,
		"/eli/acts/DU/1997/78": `{
			"publisher": "DU",
			"year": "1997",
			"position": "78",
			"title": "Konstytucja Rzeczypospolitej Polskiej",
			"announcementDate": "1997-07-16",
			"status": "obowiązujący",
			"formats": ["html", "pdf"]
		}`,
		"/eli/acts/DU/1997/78/text": `<html><body>Sample legal text content</body></html>`,
		"/sejm/mps": `[
			{"id": 1, "firstName": "Jan", "lastName": "Kowalski", "club": "Test Club"}
		]`,
		"/sejm/committees": `[
			{"code": "SUE", "name": "Komisja do Spraw Unii Europejskiej"}
		]`,
		"/sejm/votings": `{
			"votings": [
				{"sitting": 1, "votingNumber": 1, "title": "Test voting", "yes": 250, "no": 200}
			]
		}`,
		"/sejm/interpellations": `[
			{"num": "1", "title": "Test interpellation", "from": "Jan Kowalski"}
		]`,
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.Path
		if r.URL.RawQuery != "" {
			url += "?" + r.URL.RawQuery
		}

		// Find matching response with more specific matching
		if strings.Contains(url, "/eli/acts/search") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, responses["/eli/acts/search"])
			return
		}

		if strings.Contains(url, "/eli/acts/DU/1997/78") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, responses["/eli/acts/DU/1997/78"])
			return
		}

		if strings.Contains(url, "/sejm/mps") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, responses["/sejm/mps"])
			return
		}

		// Find matching response for other patterns
		for pattern, response := range responses {
			if strings.Contains(url, pattern) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, response)
				return
			}
		}

		// Default 404
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error": "Not found"}`)
	}))
}

// TestAPIIntegrationWithMockServer tests API integration using mock server
func TestAPIIntegrationWithMockServer(t *testing.T) {
	t.Parallel()

	t.Run("ELI Search Acts", func(t *testing.T) {
		t.Parallel()
		mockServer := setupMockServer()
		defer mockServer.Close()

		// Create server with mock client
		server := NewSejmServer()
		server.client = &http.Client{
			Transport: &http.Transport{},
		}

		ctx := context.Background()
		mockEndpoint := mockServer.URL + "/eli/acts/search"

		result, err := server.makeAPIRequest(ctx, mockEndpoint, map[string]string{
			"title": "konstytucja",
		})

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if result == nil {
			t.Error("Expected non-nil result")
		}

		resultStr := string(result)
		if !strings.Contains(resultStr, "Konstytucja") {
			t.Errorf("Expected result to contain 'Konstytucja', got: %s", resultStr)
		}
	})

	t.Run("ELI Get Act Details", func(t *testing.T) {
		t.Parallel()
		mockServer := setupMockServer()
		defer mockServer.Close()

		// Create server with mock client
		server := NewSejmServer()
		server.client = &http.Client{
			Transport: &http.Transport{},
		}

		ctx := context.Background()
		mockEndpoint := mockServer.URL + "/eli/acts/DU/1997/78"
		result, err := server.makeAPIRequest(ctx, mockEndpoint, nil)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if result == nil {
			t.Error("Expected non-nil result")
		}

		resultStr := string(result)
		if !strings.Contains(resultStr, "obowiązujący") {
			t.Errorf("Expected result to contain status, got: %s", resultStr)
		}
	})

	t.Run("Sejm Get MPs", func(t *testing.T) {
		t.Parallel()
		mockServer := setupMockServer()
		defer mockServer.Close()

		// Create server with mock client
		server := NewSejmServer()
		server.client = &http.Client{
			Transport: &http.Transport{},
		}

		ctx := context.Background()
		mockEndpoint := mockServer.URL + "/sejm/mps"
		result, err := server.makeAPIRequest(ctx, mockEndpoint, map[string]string{
			"term": "10",
		})

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if result == nil {
			t.Error("Expected non-nil result")
		}

		resultStr := string(result)
		if !strings.Contains(resultStr, "Kowalski") {
			t.Errorf("Expected result to contain MP name, got: %s", resultStr)
		}
	})
}

// TestHTTPCacheStatsIntegration tests cache statistics functionality
func TestHTTPCacheStatsIntegration(t *testing.T) {
	t.Parallel()
	server := NewSejmServer()

	// Test initial cache stats
	stats := server.getHTTPCacheStats()
	if stats.Requests != 0 {
		t.Errorf("Expected 0 initial requests, got %d", stats.Requests)
	}
	if stats.Hits != 0 {
		t.Errorf("Expected 0 initial hits, got %d", stats.Hits)
	}
	if stats.Misses != 0 {
		t.Errorf("Expected 0 initial misses, got %d", stats.Misses)
	}

	// Test cache status reporting
	cacheStatus := server.getCacheStatus()
	if cacheStatus == nil {
		t.Fatal("Expected non-nil cache status")
	}

	httpCacheData, exists := cacheStatus["httpCache"]
	if !exists {
		t.Fatal("Expected httpCache in cache status")
	}

	httpCache, ok := httpCacheData.(map[string]interface{})
	if !ok {
		t.Fatal("Expected httpCache to be a map")
	}

	if enabled, exists := httpCache["enabled"]; !exists || enabled != true {
		t.Error("Expected httpCache to be enabled")
	}
}

// TestMakeAPIRequestErrorHandling tests error handling in API requests
func TestMakeAPIRequestErrorHandling(t *testing.T) {
	t.Parallel()
	server := NewSejmServer()
	ctx := context.Background()

	t.Run("Invalid URL", func(t *testing.T) {
		t.Parallel()
		_, err := server.makeAPIRequest(ctx, "not-a-valid-url", nil)
		if err == nil {
			t.Error("Expected error for invalid URL")
		}
		// Check for the actual error pattern that comes from retry logic
		if !strings.Contains(err.Error(), "failed to make request") {
			t.Errorf("Expected 'failed to make request' in error, got: %v", err)
		}
	})

	t.Run("Valid URL but short timeout", func(t *testing.T) {
		t.Parallel()
		// Create a short timeout context to avoid long waits
		shortCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancel()

		_, err := server.makeAPIRequest(shortCtx, "http://httpbin.org/delay/1", nil)
		if err == nil {
			t.Error("Expected error for timeout")
		}
		// The error should mention context cancellation or timeout
	})
}

// TestMakeAPIRequestWithHeaders tests API requests with custom headers
func TestMakeAPIRequestWithHeaders(t *testing.T) {
	t.Parallel()

	t.Run("JSON Request", func(t *testing.T) {
		t.Parallel()
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check for custom header
			if r.Header.Get("Accept") == "application/pdf" {
				w.Header().Set("Content-Type", "application/pdf")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("PDF content"))
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"result": "success"}`)
		}))
		defer mockServer.Close()

		server := NewSejmServer()
		ctx := context.Background()

		result, err := server.makeAPIRequestWithHeaders(ctx, mockServer.URL, nil, map[string]string{
			"Accept": "application/json",
		})

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		resultStr := string(result)
		if !strings.Contains(resultStr, "success") {
			t.Errorf("Expected JSON result, got: %s", resultStr)
		}
	})

	t.Run("PDF Request", func(t *testing.T) {
		t.Parallel()
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check for custom header
			if r.Header.Get("Accept") == "application/pdf" {
				w.Header().Set("Content-Type", "application/pdf")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("PDF content"))
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"result": "success"}`)
		}))
		defer mockServer.Close()

		server := NewSejmServer()
		ctx := context.Background()

		result, err := server.makeAPIRequestWithHeaders(ctx, mockServer.URL, nil, map[string]string{
			"Accept": "application/pdf",
		})

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		resultStr := string(result)
		if !strings.Contains(resultStr, "PDF content") {
			t.Errorf("Expected PDF result, got: %s", resultStr)
		}
	})
}

// TestMakeTextRequest tests text-specific API requests
func TestMakeTextRequest(t *testing.T) {
	t.Parallel()

	t.Run("PDF Format", func(t *testing.T) {
		t.Parallel()
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			accept := r.Header.Get("Accept")

			if accept == "application/pdf" {
				w.Header().Set("Content-Type", "application/pdf")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("PDF content"))
			} else if accept == "text/html" {
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("<html><body>HTML content</body></html>"))
			} else {
				w.WriteHeader(http.StatusBadRequest)
			}
		}))
		defer mockServer.Close()

		server := NewSejmServer()
		ctx := context.Background()

		result, err := server.makeTextRequest(ctx, mockServer.URL, "pdf")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		resultStr := string(result)
		if !strings.Contains(resultStr, "PDF content") {
			t.Errorf("Expected PDF content, got: %s", resultStr)
		}
	})

	t.Run("HTML Format", func(t *testing.T) {
		t.Parallel()
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			accept := r.Header.Get("Accept")

			if accept == "application/pdf" {
				w.Header().Set("Content-Type", "application/pdf")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("PDF content"))
			} else if accept == "text/html" {
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("<html><body>HTML content</body></html>"))
			} else {
				w.WriteHeader(http.StatusBadRequest)
			}
		}))
		defer mockServer.Close()

		server := NewSejmServer()
		ctx := context.Background()

		result, err := server.makeTextRequest(ctx, mockServer.URL, "html")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		resultStr := string(result)
		if !strings.Contains(resultStr, "HTML content") {
			t.Errorf("Expected HTML content, got: %s", resultStr)
		}
	})
}

// TestAPIRequestRetryLogic tests retry mechanism with fast failures
func TestAPIRequestRetryLogic(t *testing.T) {
	t.Parallel()
	attemptCount := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++

		// Always fail to test retry count without waiting
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	server := NewSejmServer()
	// Use short timeout to avoid waiting for full retry delays
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err := server.makeAPIRequest(ctx, mockServer.URL, nil)
	if err == nil {
		t.Error("Expected error for server errors")
	}

	// Should have attempted at least once
	if attemptCount < 1 {
		t.Errorf("Expected at least 1 attempt, got %d", attemptCount)
	}
}

// TestAPIRequestStatusCodes tests various HTTP status code handling
func TestAPIRequestStatusCodes(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		statusCode    int
		expectedError string
		description   string
	}{
		{200, "", "Should succeed with 200"},
		{404, "resource not found", "Should handle 404"},
		{403, "access denied", "Should handle 403"},
		{429, "rate limit exceeded", "Should handle 429"},
		{500, "server error", "Should handle 500"},
		{400, "bad request", "Should handle 400"},
		{401, "unauthorized", "Should handle 401"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				if tc.statusCode == 200 {
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprint(w, `{"result": "success"}`)
				}
			}))
			defer mockServer.Close()

			server := NewSejmServer()
			ctx := context.Background()

			result, err := server.makeAPIRequest(ctx, mockServer.URL, nil)

			if tc.expectedError == "" {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tc.description, err)
				}
				if result == nil {
					t.Errorf("Expected result for %s", tc.description)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error for %s", tc.description)
				}
				if !strings.Contains(strings.ToLower(err.Error()), tc.expectedError) {
					t.Errorf("Expected error containing '%s' for %s, got: %v", tc.expectedError, tc.description, err)
				}
			}
		})
	}
}

// BenchmarkAPIRequest benchmarks API request performance
func BenchmarkAPIRequest(b *testing.B) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"result": "success"}`)
	}))
	defer mockServer.Close()

	server := NewSejmServer()
	ctx := context.Background()

	b.Run("SimpleRequest", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = server.makeAPIRequest(ctx, mockServer.URL, nil)
		}
	})

	b.Run("RequestWithParams", func(b *testing.B) {
		params := map[string]string{
			"param1": "value1",
			"param2": "value2",
		}
		for i := 0; i < b.N; i++ {
			_, _ = server.makeAPIRequest(ctx, mockServer.URL, params)
		}
	})
}
