package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// Helper function to extract text content from MCP result
func extractTextContent(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}

	var content strings.Builder
	for _, c := range result.Content {
		if textContent, ok := mcp.AsTextContent(c); ok {
			content.WriteString(textContent.Text)
		}
	}
	return content.String()
}

// Create a mock CallToolRequest for testing
func createMockRequest(params map[string]interface{}) mcp.CallToolRequest {
	// Create a proper CallToolRequest struct with the params
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: params,
		},
	}
	return request
}

// Simple mock server for testing
func setupWorkingMockServer(responses map[string]string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.String()

		// Find the matching response pattern
		for pattern, response := range responses {
			if strings.Contains(url, pattern) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, response)
				return
			}
		}

		// Default 404 response
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error": "Not found"}`)
	}))
}

// Test validation logic for search acts
func TestValidationLogic(t *testing.T) {
	t.Parallel()
	server := NewSejmServer()

	t.Run("Empty search parameters should fail", func(t *testing.T) {
		t.Parallel()
		request := createMockRequest(map[string]interface{}{})

		result, err := server.handleSearchActs(context.Background(), request)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if result == nil || !result.IsError {
			t.Error("Expected error for empty search parameters")
		}

		content := extractTextContent(result)
		if !strings.Contains(content, "search parameter") {
			t.Errorf("Expected error message about search parameters, got: %s", content)
		}
	})

	t.Run("Invalid term should fail", func(t *testing.T) {
		t.Parallel()
		request := createMockRequest(map[string]interface{}{
			"term": "15", // Invalid term
		})

		result, err := server.handleGetMPs(context.Background(), request)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if result == nil || !result.IsError {
			t.Error("Expected error for invalid term")
		}

		content := extractTextContent(result)
		if !strings.Contains(content, "invalid term") {
			t.Errorf("Expected error message about invalid term, got: %s", content)
		}
	})
}

// Test parameter validation
func TestParameterValidation(t *testing.T) {
	t.Parallel()
	server := NewSejmServer()

	t.Run("ELI act details requires all parameters", func(t *testing.T) {
		t.Parallel()
		request := createMockRequest(map[string]interface{}{
			"publisher": "DU",
			// Missing year and position
		})

		result, err := server.handleGetActDetails(context.Background(), request)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if result == nil || !result.IsError {
			t.Error("Expected error for missing parameters")
		}

		content := extractTextContent(result)
		if !strings.Contains(content, "required") {
			t.Errorf("Expected error message about required parameters, got: %s", content)
		}
	})

	t.Run("Year validation", func(t *testing.T) {
		t.Parallel()
		request := createMockRequest(map[string]interface{}{
			"publisher": "DU",
			"year":      "97", // Invalid year format
			"position":  "78",
		})

		result, err := server.handleGetActDetails(context.Background(), request)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if result == nil || !result.IsError {
			t.Error("Expected error for invalid year format")
		}

		content := extractTextContent(result)
		if !strings.Contains(content, "4-digit") {
			t.Errorf("Expected error message about year format, got: %s", content)
		}
	})
}

// Test fuzzy validation functions
func TestFuzzyValidation(t *testing.T) {
	t.Parallel()
	// Use mocked server to avoid real HTTP requests
	server := NewMockedSejmServer()
	ctx := context.Background()

	t.Run("Document type validation", func(t *testing.T) {
		t.Parallel()
		// Test valid type
		valid, suggestions, err := server.validateDocumentType("Ustawa")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !valid {
			t.Error("Expected 'Ustawa' to be valid")
		}
		if len(suggestions) > 0 {
			t.Errorf("Expected no suggestions for valid type, got: %v", suggestions)
		}

		// Test invalid type with suggestions
		valid, suggestions, err = server.validateDocumentType("konstytucya")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if valid {
			t.Error("Expected 'konstytucya' to be invalid")
		}
		if len(suggestions) == 0 {
			t.Error("Expected suggestions for invalid type")
		}
	})

	t.Run("Keyword validation", func(t *testing.T) {
		t.Parallel()
		suggestions := server.validateKeywords("konstytucja")
		if len(suggestions) == 0 {
			t.Error("Expected keyword suggestions")
		}

		// Test empty input
		suggestions = server.validateKeywords("")
		if len(suggestions) > 0 {
			t.Errorf("Expected no suggestions for empty input, got: %v", suggestions)
		}
	})

	t.Run("Institution validation", func(t *testing.T) {
		t.Parallel()
		// Test valid institution
		valid, suggestions, err := server.validateInstitution(ctx, "Sejm")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !valid {
			t.Error("Expected 'Sejm' to be valid")
		}
		if len(suggestions) > 0 {
			t.Errorf("Expected no suggestions for valid institution, got: %v", suggestions)
		}
	})
}

// Test working cache functionality
func TestWorkingCacheStatus(t *testing.T) {
	t.Parallel()
	server := NewSejmServer()

	status := server.getCacheStatus()

	if status == nil {
		t.Fatal("getCacheStatus should return a non-nil map")
	}

	expectedKeys := []string{"publishers", "popularActs", "statusTypes", "documentTypes", "keywords", "institutions", "httpCache"}
	for _, key := range expectedKeys {
		if _, exists := status[key]; !exists {
			t.Errorf("Cache status should contain key '%s'", key)
		}
	}

	// Test that each cache entry has the expected structure
	for key, value := range status {
		if key == "httpCache" {
			continue // Different structure for HTTP cache
		}

		valueMap, ok := value.(map[string]interface{})
		if !ok {
			t.Errorf("Cache status entry '%s' should be a map", key)
			continue
		}

		if _, exists := valueMap["cached"]; !exists {
			t.Errorf("Cache status entry '%s' should have 'cached' field", key)
		}
	}
}

// Test term validation
func TestTermValidation(t *testing.T) {
	t.Parallel()
	server := NewSejmServer()

	testCases := []struct {
		termStr      string
		expectedTerm int
		expectError  bool
	}{
		{"", 10, false},   // Default to current term
		{"10", 10, false}, // Valid term
		{"1", 1, false},   // Valid term
		{"0", 0, true},    // Invalid term
		{"11", 0, true},   // Invalid term
		{"abc", 0, true},  // Invalid format
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("term=%s", tc.termStr), func(t *testing.T) {
			t.Parallel()
			term, err := server.validateTerm(tc.termStr)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for term '%s'", tc.termStr)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for term '%s': %v", tc.termStr, err)
				}
				if term != tc.expectedTerm {
					t.Errorf("Expected term %d for input '%s', got %d", tc.expectedTerm, tc.termStr, term)
				}
			}
		})
	}
}

// TestServerConfiguration tests server configuration functionality
func TestServerConfiguration(t *testing.T) {
	t.Parallel()
	t.Run("Default configuration", func(t *testing.T) {
		t.Parallel()
		server := NewSejmServer()
		if server == nil {
			t.Fatal("NewSejmServer should return non-nil server")
		}
		if server.config.DebugMode {
			t.Error("Default config should have DebugMode=false")
		}
	})

	t.Run("Debug configuration", func(t *testing.T) {
		t.Parallel()
		config := Config{DebugMode: true}
		server := NewSejmServerWithConfig(config)
		if server == nil {
			t.Fatal("NewSejmServerWithConfig should return non-nil server")
		}
		if !server.config.DebugMode {
			t.Error("Debug config should have DebugMode=true")
		}
	})
}

// TestPopularActs tests popular acts caching functionality
func TestPopularActs(t *testing.T) {
	t.Parallel()
	// Use mocked server to avoid real HTTP requests
	server := NewMockedSejmServer()

	popularActs := server.getPopularActs()
	if len(popularActs) == 0 {
		t.Error("Expected popular acts to be returned")
	}

	// Verify constitution is in popular acts
	foundConstitution := false
	for _, act := range popularActs {
		if strings.Contains(strings.ToLower(act.Title), "konstytucja") {
			foundConstitution = true
			break
		}
	}
	if !foundConstitution {
		t.Error("Expected constitution to be in popular acts")
	}

	// Test caching - second call should return cached result
	popularActs2 := server.getPopularActs()
	if len(popularActs2) != len(popularActs) {
		t.Error("Cached popular acts should have same length")
	}
}

// TestCacheManagement tests cache management functionality
func TestCacheManagement(t *testing.T) {
	t.Parallel()
	// Use mocked server to avoid real HTTP requests
	server := NewMockedSejmServer()

	t.Run("Clear expired cache", func(t *testing.T) {
		t.Parallel()
		// This should not panic
		server.clearExpiredCache()
	})

	t.Run("Clear all cache", func(t *testing.T) {
		t.Parallel()
		// This should not panic
		server.clearAllCache()

		// Verify cache is cleared
		status := server.getCacheStatus()
		for key, value := range status {
			if key == "httpCache" {
				continue
			}

			valueMap, ok := value.(map[string]interface{})
			if !ok {
				continue
			}

			if cachedVal, exists := valueMap["cached"]; exists {
				if cached, ok := cachedVal.(bool); ok && cached {
					t.Errorf("Cache entry '%s' should be cleared", key)
				}
			}
		}
	})
}

// TestGetKeywordContext tests keyword context functionality
func TestGetKeywordContext(t *testing.T) {
	t.Parallel()
	server := NewSejmServer()

	testCases := []struct {
		keyword         string
		expectedContext string
		description     string
	}{
		{"konstytucja", "constitutional law", "Should provide context for constitution"},
		{"sejm", "parliament", "Should provide context for parliament"},
		{"administracja", "administrative law", "Should provide context for administration"},
		{"podatki", "tax law", "Should provide context for taxes"},
		{"unknown-keyword", "", "Should return empty for unknown keyword"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			context := server.getKeywordContext(tc.keyword)
			if context != tc.expectedContext {
				t.Errorf("Expected context '%s' for keyword '%s', got '%s'", tc.expectedContext, tc.keyword, context)
			}
		})
	}
}

// TestGetInstitutionType tests institution type classification
func TestGetInstitutionType(t *testing.T) {
	t.Parallel()
	server := NewSejmServer()

	testCases := []struct {
		institution  string
		expectedType string
		description  string
	}{
		{"Sejm", "parliament", "Should classify Sejm as parliament"},
		{"Senat", "senate", "Should classify Senat as senate"},
		{"Prezydent", "executive", "Should classify President as executive"},
		{"Trybunał Konstytucyjny", "tribunal", "Should classify Constitutional Tribunal as tribunal"},
		{"Sąd Najwyższy", "court", "Should classify Supreme Court as court"},
		{"Najwyższa Izba Kontroli", "oversight", "Should classify NIK as oversight"},
		{"Narodowy Bank Polski", "financial", "Should classify NBP as financial"},
		{"Wojewoda", "local government", "Should classify Voivode as local government"},
		{"Unknown Institution", "", "Should return empty for unknown institution"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			institutionType := server.getInstitutionType(tc.institution)
			if institutionType != tc.expectedType {
				t.Errorf("Expected type '%s' for institution '%s', got '%s'", tc.expectedType, tc.institution, institutionType)
			}
		})
	}
}

// TestCachedDataRetrieval tests cached data retrieval functions
func TestCachedDataRetrieval(t *testing.T) {
	t.Parallel()
	// Use mocked server to avoid real HTTP requests
	server := NewMockedSejmServer()

	t.Run("Status types", func(t *testing.T) {
		t.Parallel()
		statusTypes := server.getCachedStatusTypes()
		if len(statusTypes) == 0 {
			t.Error("Expected status types to be returned")
		}

		// Should contain common statuses
		foundInForce := false
		for _, status := range statusTypes {
			if strings.Contains(strings.ToLower(status), "obowiązujący") {
				foundInForce = true
				break
			}
		}
		if !foundInForce {
			t.Error("Expected 'obowiązujący' status in status types")
		}
	})

	t.Run("Document types", func(t *testing.T) {
		t.Parallel()
		docTypes := server.getCachedDocumentTypes()
		if len(docTypes) == 0 {
			t.Error("Expected document types to be returned")
		}

		// Should contain common document types
		foundUstawa := false
		for _, docType := range docTypes {
			if docType == "Ustawa" {
				foundUstawa = true
				break
			}
		}
		if !foundUstawa {
			t.Error("Expected 'Ustawa' in document types")
		}
	})

	t.Run("Keywords", func(t *testing.T) {
		t.Parallel()
		keywords := server.getCachedKeywords()
		if len(keywords) == 0 {
			t.Error("Expected keywords to be returned")
		}

		// Should contain common keywords
		foundKonstytucja := false
		for _, keyword := range keywords {
			if keyword == "konstytucja" {
				foundKonstytucja = true
				break
			}
		}
		if !foundKonstytucja {
			t.Error("Expected 'konstytucja' in keywords")
		}
	})

	t.Run("Institutions", func(t *testing.T) {
		t.Parallel()
		institutions := server.getCachedInstitutions()
		if len(institutions) == 0 {
			t.Error("Expected institutions to be returned")
		}

		// Should contain common institutions
		foundSejm := false
		for _, institution := range institutions {
			if institution == "Sejm" {
				foundSejm = true
				break
			}
		}
		if !foundSejm {
			t.Error("Expected 'Sejm' in institutions")
		}
	})
}
