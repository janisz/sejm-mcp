package server

import (
	"fmt"
	"testing"
)

func TestValidateTerm(t *testing.T) {
	server := NewSejmServer()

	testCases := []struct {
		name     string
		input    string
		expected int
		hasError bool
	}{
		{
			name:     "empty string defaults to current term",
			input:    "",
			expected: 10,
			hasError: false,
		},
		{
			name:     "valid term 10",
			input:    "10",
			expected: 10,
			hasError: false,
		},
		{
			name:     "valid term 1",
			input:    "1",
			expected: 1,
			hasError: false,
		},
		{
			name:     "valid term 5",
			input:    "5",
			expected: 5,
			hasError: false,
		},
		{
			name:     "invalid term 0",
			input:    "0",
			expected: 0,
			hasError: true,
		},
		{
			name:     "invalid term 11",
			input:    "11",
			expected: 0,
			hasError: true,
		},
		{
			name:     "invalid term -1",
			input:    "-1",
			expected: 0,
			hasError: true,
		},
		{
			name:     "invalid term text",
			input:    "invalid",
			expected: 0,
			hasError: true,
		},
		{
			name:     "invalid term decimal",
			input:    "10.5",
			expected: 0,
			hasError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := server.validateTerm(tc.input)

			if tc.hasError {
				if err == nil {
					t.Errorf("Expected error for input '%s', but got none", tc.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input '%s': %v", tc.input, err)
				}
				if result != tc.expected {
					t.Errorf("Expected %d for input '%s', but got %d", tc.expected, tc.input, result)
				}
			}
		})
	}
}

func TestNewSejmServer(t *testing.T) {
	server := NewSejmServer()

	if server == nil {
		t.Fatal("NewSejmServer should return a non-nil server")
	}

	if server.server == nil {
		t.Error("Server should have a non-nil MCP server")
	}

	if server.client == nil {
		t.Error("Server should have a non-nil HTTP client")
	}
}

func BenchmarkValidateTerm(b *testing.B) {
	server := &SejmServer{}

	b.Run("ValidTerm", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = server.validateTerm("10")
		}
	})

	b.Run("InvalidTerm", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = server.validateTerm("invalid")
		}
	})

	b.Run("EmptyTerm", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = server.validateTerm("")
		}
	})
}

// TestFuzzySearch tests the fuzzy search algorithms
func TestFuzzySearch(t *testing.T) {
	server := NewSejmServer()

	testCases := []struct {
		name          string
		query         string
		candidates    []string
		threshold     float64
		expectMatches bool
		description   string
	}{
		{
			name:          "exact match",
			query:         "konstytucja",
			candidates:    []string{"Konstytucja", "Ustawa", "Rozporządzenie"},
			threshold:     0.8,
			expectMatches: true,
			description:   "Should find exact match",
		},
		{
			name:          "fuzzy match - typo",
			query:         "konstytucya",
			candidates:    []string{"Konstytucja", "Ustawa", "Rozporządzenie"},
			threshold:     0.6,
			expectMatches: true,
			description:   "Should find fuzzy match for typo",
		},
		{
			name:          "partial match",
			query:         "konst",
			candidates:    []string{"Konstytucja", "Ustawa", "Rozporządzenie"},
			threshold:     0.5,
			expectMatches: true,
			description:   "Should find partial match",
		},
		{
			name:          "no match - high threshold",
			query:         "xyz",
			candidates:    []string{"Konstytucja", "Ustawa", "Rozporządzenie"},
			threshold:     0.9,
			expectMatches: false,
			description:   "Should not match with high threshold",
		},
		{
			name:          "Polish characters normalization",
			query:         "rozporzadzenie",
			candidates:    []string{"Rozporządzenie", "Ustawa", "Konstytucja"},
			threshold:     0.7,
			expectMatches: true,
			description:   "Should match after Polish character normalization",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matches := server.fuzzyMatchText(tc.query, tc.candidates, tc.threshold)

			if tc.expectMatches {
				if len(matches) == 0 {
					t.Errorf("Expected matches for %s, but got none", tc.description)
				}
			} else {
				if len(matches) > 0 {
					t.Errorf("Expected no matches for %s, but got %d matches", tc.description, len(matches))
				}
			}

			// Verify that matches are sorted by score
			for i := 1; i < len(matches); i++ {
				if matches[i-1].Score < matches[i].Score {
					t.Errorf("Matches should be sorted by score (descending), but found %f > %f", matches[i].Score, matches[i-1].Score)
				}
			}
		})
	}
}

// TestPolishNormalization tests Polish character normalization
func TestPolishNormalization(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"konstytucja", "konstytucja"}, // no change
		{"konstytucyą", "konstytuca"},  // ą -> a, then cy -> c
		{"sąd", "sad"},                 // ą -> a
		{"więcej", "wiecej"},           // ę -> e
		{"ław", "law"},                 // ł -> l
		{"żółw", "zolw"},               // ż -> z, ó -> o
		{"Konstytucja", "konstytucja"}, // uppercase
		{"KONSTYTUCJA", "konstytucja"}, // all uppercase
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("normalize_%s", tc.input), func(t *testing.T) {
			result := normalizePolish(tc.input)
			if result != tc.expected {
				t.Errorf("normalizePolish(%s) = %s, expected %s", tc.input, result, tc.expected)
			}
		})
	}
}

// TestSimilarityFunctions tests the similarity calculation functions
func TestSimilarityFunctions(t *testing.T) {
	testCases := []struct {
		s1             string
		s2             string
		minLevenshtein float64
		minJaroWinkler float64
		description    string
	}{
		{
			s1: "konstytucja", s2: "konstytucja",
			minLevenshtein: 1.0, minJaroWinkler: 1.0,
			description: "identical strings should have similarity 1.0",
		},
		{
			s1: "konstytucja", s2: "konstytucya",
			minLevenshtein: 0.8, minJaroWinkler: 0.9,
			description: "similar strings should have high similarity",
		},
		{
			s1: "hello", s2: "world",
			minLevenshtein: 0.0, minJaroWinkler: 0.0,
			description: "different strings should have low similarity",
		},
		{
			s1: "sejm", s2: "sejmowy",
			minLevenshtein: 0.5, minJaroWinkler: 0.8,
			description: "prefix matches should score well with Jaro-Winkler",
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s_vs_%s", tc.s1, tc.s2), func(t *testing.T) {
			levSim := similarity(tc.s1, tc.s2)
			jaroSim := jaroWinklerSimilarity(tc.s1, tc.s2)

			if levSim < tc.minLevenshtein {
				t.Errorf("Levenshtein similarity for %s: expected >= %f, got %f", tc.description, tc.minLevenshtein, levSim)
			}

			if jaroSim < tc.minJaroWinkler {
				t.Errorf("Jaro-Winkler similarity for %s: expected >= %f, got %f", tc.description, tc.minJaroWinkler, jaroSim)
			}

			// Test that similarities are between 0 and 1
			if levSim < 0 || levSim > 1 {
				t.Errorf("Levenshtein similarity should be between 0 and 1, got %f", levSim)
			}

			if jaroSim < 0 || jaroSim > 1 {
				t.Errorf("Jaro-Winkler similarity should be between 0 and 1, got %f", jaroSim)
			}
		})
	}
}

// TestCacheOperations tests cache functionality
func TestCacheOperations(t *testing.T) {
	server := NewSejmServer()

	// Test that cache starts empty
	status := server.getCacheStatus()
	for key, value := range status {
		if key == "httpCache" {
			continue // Different structure for HTTP cache
		}

		valueMap, ok := value.(map[string]interface{})
		if !ok {
			t.Errorf("Cache status entry '%s' should be a map", key)
			continue
		}

		if cachedVal, exists := valueMap["cached"]; exists {
			if cached, ok := cachedVal.(bool); ok && cached {
				t.Errorf("Cache key '%s' should start as not cached", key)
			}
		}
	}

	// Test cache clearing
	server.clearAllCache()
	server.clearExpiredCache()

	// These shouldn't panic or error
}

// Benchmark fuzzy search functions
func BenchmarkFuzzySearch(b *testing.B) {
	server := NewSejmServer()
	candidates := []string{
		"Konstytucja", "Ustawa", "Rozporządzenie", "Dekret", "Zarządzenie",
		"Obwieszczenie", "Komunikat", "Uchwała", "Decyzja", "Rozstrzygnięcie",
	}

	b.Run("ExactMatch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = server.fuzzyMatchText("Ustawa", candidates, 0.8)
		}
	})

	b.Run("FuzzyMatch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = server.fuzzyMatchText("konstytucya", candidates, 0.6)
		}
	})

	b.Run("NoMatch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = server.fuzzyMatchText("xyz123", candidates, 0.8)
		}
	})
}

func BenchmarkPolishNormalization(b *testing.B) {
	testStrings := []string{
		"konstytucja",
		"rozporządzenie",
		"administracja",
		"sąd najwyższy",
		"trybunał konstytucyjny",
	}

	b.Run("NormalizationBench", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, s := range testStrings {
				_ = normalizePolish(s)
			}
		}
	})
}

func BenchmarkSimilarityFunctions(b *testing.B) {
	s1 := "konstytucja"
	s2 := "konstytucya"

	b.Run("LevenshteinSimilarity", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = similarity(s1, s2)
		}
	})

	b.Run("JaroWinklerSimilarity", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = jaroWinklerSimilarity(s1, s2)
		}
	})

	b.Run("LevenshteinDistance", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = levenshteinDistance(s1, s2)
		}
	})
}
