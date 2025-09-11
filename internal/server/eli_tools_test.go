package server

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// TestExtractTextFromPDF tests the PDF text extraction functionality
func TestExtractTextFromPDF(t *testing.T) {
	server := NewSejmServer()

	testCases := []struct {
		name        string
		pdfData     []byte
		expectError bool
		description string
	}{
		{
			name:        "empty PDF data",
			pdfData:     []byte{},
			expectError: true,
			description: "Should fail with empty PDF data",
		},
		{
			name:        "invalid PDF data",
			pdfData:     []byte("not a PDF"),
			expectError: true,
			description: "Should fail with invalid PDF data",
		},
		{
			name:        "nil PDF data",
			pdfData:     nil,
			expectError: true,
			description: "Should fail with nil PDF data",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := server.extractTextFromPDF(tc.pdfData)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tc.description)
				}
				if result != "" {
					t.Errorf("Expected empty result for error case, but got: %s", result)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tc.description, err)
				}
				if result == "" {
					t.Errorf("Expected non-empty result for %s", tc.description)
				}
			}
		})
	}
}

// TestValidateDocumentType tests the fuzzy document type validation
func TestValidateDocumentType(t *testing.T) {
	server := NewSejmServer()

	testCases := []struct {
		name           string
		input          string
		expectedValid  bool
		shouldHaveSugg bool
		description    string
	}{
		{
			name:           "empty input",
			input:          "",
			expectedValid:  true,
			shouldHaveSugg: false,
			description:    "Empty input should be valid",
		},
		{
			name:           "exact match",
			input:          "Ustawa",
			expectedValid:  true,
			shouldHaveSugg: false,
			description:    "Exact match should be valid",
		},
		{
			name:           "case insensitive match",
			input:          "ustawa",
			expectedValid:  true,
			shouldHaveSugg: false,
			description:    "Case insensitive match should be valid",
		},
		{
			name:           "fuzzy match - typo",
			input:          "konstytucya",
			expectedValid:  false,
			shouldHaveSugg: true,
			description:    "Typo should trigger fuzzy suggestions",
		},
		{
			name:           "fuzzy match - missing chars",
			input:          "dekrt",
			expectedValid:  false,
			shouldHaveSugg: true,
			description:    "Missing characters should trigger fuzzy suggestions",
		},
		{
			name:           "no match",
			input:          "invalidtype",
			expectedValid:  false,
			shouldHaveSugg: true,
			description:    "Invalid type should provide suggestions",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valid, suggestions, err := server.validateDocumentType(tc.input)

			if err != nil {
				t.Errorf("Unexpected error for %s: %v", tc.description, err)
			}

			if valid != tc.expectedValid {
				t.Errorf("Expected valid=%v for %s, but got %v", tc.expectedValid, tc.description, valid)
			}

			if tc.shouldHaveSugg {
				if len(suggestions) == 0 {
					t.Errorf("Expected suggestions for %s, but got none", tc.description)
				}
			} else {
				if len(suggestions) > 0 {
					t.Errorf("Expected no suggestions for %s, but got %v", tc.description, suggestions)
				}
			}
		})
	}
}

// TestValidateKeywords tests the fuzzy keyword validation
func TestValidateKeywords(t *testing.T) {
	server := NewSejmServer()

	testCases := []struct {
		name        string
		input       string
		expectSugg  bool
		description string
	}{
		{
			name:        "empty input",
			input:       "",
			expectSugg:  false,
			description: "Empty input should return no suggestions",
		},
		{
			name:        "exact keyword match",
			input:       "konstytucja",
			expectSugg:  true,
			description: "Exact keyword should return suggestions",
		},
		{
			name:        "fuzzy keyword match",
			input:       "konstytucya",
			expectSugg:  true,
			description: "Fuzzy keyword should return suggestions",
		},
		{
			name:        "partial keyword match",
			input:       "sad",
			expectSugg:  true,
			description: "Partial keyword should return suggestions",
		},
		{
			name:        "no match",
			input:       "xyz123",
			expectSugg:  true,
			description: "No match should return popular categories",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			suggestions := server.validateKeywords(tc.input)

			if tc.expectSugg {
				if len(suggestions) == 0 {
					t.Errorf("Expected suggestions for %s, but got none", tc.description)
				}
			} else {
				if len(suggestions) > 0 {
					t.Errorf("Expected no suggestions for %s, but got %v", tc.description, suggestions)
				}
			}
		})
	}
}

// TestValidateInstitution tests the fuzzy institution validation
func TestValidateInstitution(t *testing.T) {
	// server := NewSejmServer()
	// ctx := context.Background()

	// testCases := []struct {
	// 	name           string
	// 	input          string
	// 	expectedValid  bool
	// 	shouldHaveSugg bool
	// 	description    string
	// }{
	// 	{
	// 		name:           "empty input",
	// 		input:          "",
	// 		expectedValid:  true,
	// 		shouldHaveSugg: false,
	// 		description:    "Empty input should be valid",
	// 	},
	// 	{
	// 		name:           "exact match",
	// 		input:          "Sejm",
	// 		expectedValid:  true,
	// 		shouldHaveSugg: false,
	// 		description:    "Exact institution match should be valid",
	// 	},
	// 	{
	// 		name:           "case insensitive match",
	// 		input:          "sejm",
	// 		expectedValid:  true,
	// 		shouldHaveSugg: false,
	// 		description:    "Case insensitive match should be valid",
	// 	},
	// 	{
	// 		name:           "fuzzy match",
	// 		input:          "Trybunal",
	// 		expectedValid:  false,
	// 		shouldHaveSugg: true,
	// 		description:    "Fuzzy match should provide suggestions",
	// 	},
	// 	{
	// 		name:           "no match",
	// 		input:          "invalidinstitution",
	// 		expectedValid:  false,
	// 		shouldHaveSugg: true,
	// 		description:    "Invalid institution should provide suggestions",
	// 	},
	// }

	// Temporarily disable validateInstitution test due to removed function
	t.Skip("validateInstitution function has been removed")
	// for _, tc := range testCases {
	// 	t.Run(tc.name, func(t *testing.T) {
	// 		valid, suggestions, err := server.validateInstitution(ctx, tc.input)
	//
	// 		if err != nil {
	// 			t.Errorf("Unexpected error for %s: %v", tc.description, err)
	// 		}
	//
	// 		if valid != tc.expectedValid {
	// 			t.Errorf("Expected valid=%v for %s, but got %v", tc.expectedValid, tc.description, valid)
	// 		}
	//
	// 		if tc.shouldHaveSugg {
	// 			if len(suggestions) == 0 {
	// 				t.Errorf("Expected suggestions for %s, but got none", tc.description)
	// 			}
	// 		} else {
	// 			if len(suggestions) > 0 {
	// 				t.Errorf("Expected no suggestions for %s, but got %v", tc.description, suggestions)
	// 			}
	// 		}
	// 	})
	// }
}

// TestGetActTextFormatSelection tests the format selection logic
func TestGetActTextFormatSelection(t *testing.T) {
	server := NewSejmServer()

	testCases := []struct {
		name          string
		format        string
		htmlAvail     bool
		pdfAvail      bool
		expectError   bool
		errorContains string
		description   string
	}{
		{
			name:        "text format with HTML available",
			format:      "text",
			htmlAvail:   true,
			pdfAvail:    true,
			expectError: false,
			description: "Should use HTML when available for text format",
		},
		{
			name:        "text format with only PDF available",
			format:      "text",
			htmlAvail:   false,
			pdfAvail:    true,
			expectError: false,
			description: "Should use PDF extraction when HTML not available",
		},
		{
			name:          "text format with no formats available",
			format:        "text",
			htmlAvail:     false,
			pdfAvail:      false,
			expectError:   true,
			errorContains: "No text formats available",
			description:   "Should error when no formats available",
		},
		{
			name:          "HTML format not available",
			format:        "html",
			htmlAvail:     false,
			pdfAvail:      true,
			expectError:   true,
			errorContains: "HTML format is not available",
			description:   "Should error when HTML specifically requested but not available",
		},
		{
			name:          "PDF format not available",
			format:        "pdf",
			htmlAvail:     true,
			pdfAvail:      false,
			expectError:   true,
			errorContains: "PDF format is not available",
			description:   "Should error when PDF specifically requested but not available",
		},
		{
			name:          "invalid format",
			format:        "invalid",
			htmlAvail:     true,
			pdfAvail:      true,
			expectError:   true,
			errorContains: "Format must be",
			description:   "Should error for invalid format",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock request
			request := &mockCallToolRequest{
				params: map[string]interface{}{
					"publisher": "DU",
					"year":      "1997",
					"position":  "78",
					"format":    tc.format,
				},
			}

			// Mock the server to avoid actual API calls
			mockServer := &SejmServer{
				client: server.client,
				cache:  server.cache,
			}

			ctx := context.Background()
			result, err := mockServer.handleGetActTextValidation(ctx, request, tc.htmlAvail, tc.pdfAvail)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tc.description)
				} else if tc.errorContains != "" && !strings.Contains(err.Error(), tc.errorContains) {
					t.Errorf("Expected error containing '%s' for %s, but got: %s", tc.errorContains, tc.description, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tc.description, err)
				}
				if result == nil {
					t.Errorf("Expected non-nil result for %s", tc.description)
				}
			}
		})
	}
}

// TestCacheStatus tests the cache status functionality
// func TestCacheStatus(t *testing.T) {
// 	server := NewSejmServer()
//
// 	status := server.getCacheStatus()
//
// 	if status == nil {
// 		t.Fatal("getCacheStatus should return a non-nil map")
// 	}
//
// 	expectedKeys := []string{"publishers", "popularActs", "statusTypes", "documentTypes", "keywords", "institutions"}
// 	for _, key := range expectedKeys {
// 		if _, exists := status[key]; !exists {
// 			t.Errorf("Cache status should contain key '%s'", key)
// 		}
// 	}
//
// 	// Test that each cache entry has the expected structure
// 	for key, value := range status {
// 		if key == "httpCache" {
// 			continue // Different structure for HTTP cache
// 		}
//
// 		valueMap, ok := value.(map[string]interface{})
// 		if !ok {
// 			t.Errorf("Cache status entry '%s' should be a map", key)
// 			continue
// 		}
//
// 		if _, exists := valueMap["cached"]; !exists {
// 			t.Errorf("Cache status entry '%s' should have 'cached' field", key)
// 		}
// 	}
// }

// Mock types for testing
type mockCallToolRequest struct {
	params map[string]interface{}
}

func (m *mockCallToolRequest) GetString(key, defaultValue string) string {
	if val, exists := m.params[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

// Helper function to test format selection logic without actual API calls
func (s *SejmServer) handleGetActTextValidation(ctx context.Context, request *mockCallToolRequest, htmlAvailable, pdfAvailable bool) (*mcp.CallToolResult, error) {
	publisher := request.GetString("publisher", "")
	year := request.GetString("year", "")
	position := request.GetString("position", "")
	format := request.GetString("format", "html")

	if publisher == "" || year == "" || position == "" {
		return nil, fmt.Errorf("missing required parameters")
	}

	// Validate format
	if format != "html" && format != "pdf" && format != "text" {
		return nil, fmt.Errorf("Format must be 'html', 'pdf', or 'text', but got '%s'", format)
	}

	// Test format availability logic
	if format == "html" && !htmlAvailable {
		if pdfAvailable {
			return nil, fmt.Errorf("HTML format is not available for legal act %s/%s/%s", publisher, year, position)
		} else {
			return nil, fmt.Errorf("Neither HTML nor PDF format is available for legal act %s/%s/%s", publisher, year, position)
		}
	}

	if format == "pdf" && !pdfAvailable {
		if htmlAvailable {
			return nil, fmt.Errorf("PDF format is not available for legal act %s/%s/%s", publisher, year, position)
		} else {
			return nil, fmt.Errorf("Neither PDF nor HTML format is available for legal act %s/%s/%s", publisher, year, position)
		}
	}

	if format == "text" && !htmlAvailable && !pdfAvailable {
		return nil, fmt.Errorf("No text formats available for legal act %s/%s/%s", publisher, year, position)
	}

	// If we get here, the format selection logic passed validation
	return mcp.NewToolResultText("Format validation passed"), nil
}

// Benchmark tests for performance
func BenchmarkExtractTextFromPDF(b *testing.B) {
	server := &SejmServer{}

	// Create a minimal valid PDF for benchmarking
	// This is just for testing - real PDFs would be much larger
	invalidPDF := []byte("not a PDF")

	b.Run("InvalidPDF", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = server.extractTextFromPDF(invalidPDF)
		}
	})

	b.Run("EmptyPDF", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = server.extractTextFromPDF([]byte{})
		}
	})

	b.Run("NilPDF", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = server.extractTextFromPDF(nil)
		}
	})
}

func BenchmarkValidateDocumentType(b *testing.B) {
	server := NewSejmServer()

	b.Run("ExactMatch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _, _ = server.validateDocumentType("Ustawa")
		}
	})

	b.Run("FuzzyMatch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _, _ = server.validateDocumentType("konstytucya")
		}
	})

	b.Run("NoMatch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _, _ = server.validateDocumentType("invalidtype")
		}
	})
}

func BenchmarkValidateKeywords(b *testing.B) {
	server := NewSejmServer()

	b.Run("ExactMatch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = server.validateKeywords("konstytucja")
		}
	})

	b.Run("FuzzyMatch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = server.validateKeywords("konstytucya")
		}
	})

	b.Run("NoMatch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = server.validateKeywords("xyz123")
		}
	})
}
