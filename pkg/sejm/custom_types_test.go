package sejm

import (
	"encoding/json"
	"testing"
	"time"
)

// TestCustomTimeUnmarshalJSON tests CustomTime JSON unmarshaling
func TestCustomTimeUnmarshalJSON(t *testing.T) {
	testCases := []struct {
		name         string
		input        string
		expectError  bool
		expectedTime string
		description  string
	}{
		{
			name:         "RFC3339 format",
			input:        `"2023-12-28T22:00:40Z"`,
			expectError:  false,
			expectedTime: "2023-12-28T22:00:40Z",
			description:  "Should parse RFC3339 format",
		},
		{
			name:         "RFC3339 with timezone",
			input:        `"2023-12-28T22:00:40+01:00"`,
			expectError:  false,
			expectedTime: "2023-12-28T21:00:40Z", // Converted to UTC
			description:  "Should parse RFC3339 with timezone",
		},
		{
			name:         "Sejm API format without timezone",
			input:        `"2023-12-28T22:00:40"`,
			expectError:  false,
			expectedTime: "2023-12-28T22:00:40Z",
			description:  "Should parse Sejm API datetime format",
		},
		{
			name:         "date only format",
			input:        `"2023-11-13"`,
			expectError:  false,
			expectedTime: "2023-11-13T00:00:00Z",
			description:  "Should parse date-only format",
		},
		{
			name:        "invalid format",
			input:       `"invalid-datetime"`,
			expectError: true,
			description: "Should error on invalid format",
		},
		{
			name:        "wrong date format",
			input:       `"28/12/2023 22:00:40"`,
			expectError: true,
			description: "Should error on wrong date format",
		},
		{
			name:        "empty string",
			input:       `""`,
			expectError: true,
			description: "Should error on empty string",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var customTime CustomTime
			err := json.Unmarshal([]byte(tc.input), &customTime)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tc.description)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tc.description, err)
				}

				if tc.expectedTime != "" {
					actual := customTime.Time.UTC().Format(time.RFC3339)
					if actual != tc.expectedTime {
						t.Errorf("Expected time %s for %s, but got %s", tc.expectedTime, tc.description, actual)
					}
				}
			}
		})
	}
}

// TestCustomTimeMarshalJSON tests CustomTime JSON marshaling
func TestCustomTimeMarshalJSON(t *testing.T) {
	testCases := []struct {
		name        string
		customTime  CustomTime
		expected    string
		description string
	}{
		{
			name:        "valid datetime",
			customTime:  CustomTime{Time: time.Date(2023, 12, 28, 22, 0, 40, 0, time.UTC)},
			expected:    `"2023-12-28T22:00:40Z"`,
			description: "Should marshal datetime in RFC3339 format",
		},
		{
			name:        "zero time",
			customTime:  CustomTime{},
			expected:    `"0001-01-01T00:00:00Z"`,
			description: "Should marshal zero time in RFC3339 format",
		},
		{
			name:        "with timezone",
			customTime:  CustomTime{Time: time.Date(2023, 12, 28, 22, 0, 40, 0, time.FixedZone("CET", 3600))},
			expected:    `"2023-12-28T22:00:40+01:00"`, // RFC3339 with timezone preserved
			description: "Should marshal datetime with timezone in RFC3339 format",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := json.Marshal(tc.customTime)
			if err != nil {
				t.Errorf("Unexpected error for %s: %v", tc.description, err)
			}

			if string(result) != tc.expected {
				t.Errorf("Expected %s for %s, but got %s", tc.expected, tc.description, string(result))
			}
		})
	}
}

// TestCustomTimeRoundTrip tests round-trip marshaling/unmarshaling
func TestCustomTimeRoundTrip(t *testing.T) {
	testCases := []struct {
		name        string
		original    CustomTime
		description string
	}{
		{
			name:        "UTC time",
			original:    CustomTime{Time: time.Date(2023, 12, 28, 22, 0, 40, 0, time.UTC)},
			description: "Should preserve UTC time through round-trip",
		},
		{
			name:        "time with timezone",
			original:    CustomTime{Time: time.Date(2023, 12, 28, 22, 0, 40, 0, time.FixedZone("CET", 3600))},
			description: "Should handle timezone through round-trip",
		},
		{
			name:        "date only equivalent",
			original:    CustomTime{Time: time.Date(2023, 11, 13, 0, 0, 0, 0, time.UTC)},
			description: "Should preserve date-only time through round-trip",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Marshal to JSON
			jsonData, err := json.Marshal(tc.original)
			if err != nil {
				t.Fatalf("Failed to marshal CustomTime for %s: %v", tc.description, err)
			}

			// Unmarshal back
			var restored CustomTime
			err = json.Unmarshal(jsonData, &restored)
			if err != nil {
				t.Fatalf("Failed to unmarshal CustomTime for %s: %v", tc.description, err)
			}

			// Compare (both should be in UTC after round-trip)
			originalUTC := tc.original.Time.UTC().Truncate(time.Second)
			restoredUTC := restored.Time.UTC().Truncate(time.Second)

			if !originalUTC.Equal(restoredUTC) {
				t.Errorf("Round-trip failed for %s: original %v, restored %v", tc.description, originalUTC, restoredUTC)
			}
		})
	}
}

// TestCustomTimeParsingPriority tests the parsing priority order
func TestCustomTimeParsingPriority(t *testing.T) {
	// Test that RFC3339 is tried first and succeeds
	input := `"2023-12-28T22:00:40Z"`

	var customTime CustomTime
	err := json.Unmarshal([]byte(input), &customTime)
	if err != nil {
		t.Fatalf("Unexpected error parsing RFC3339: %v", err)
	}

	expected := time.Date(2023, 12, 28, 22, 0, 40, 0, time.UTC)
	if !customTime.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, customTime)
	}
}

// TestCustomTimeErrorHandling tests error handling and error types
func TestCustomTimeErrorHandling(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expectError bool
		description string
	}{
		{
			name:        "non-string JSON",
			input:       `123456789`,
			expectError: true,
			description: "Should error on non-string input",
		},
		{
			name:        "invalid JSON",
			input:       `{invalid}`,
			expectError: true,
			description: "Should error on invalid JSON",
		},
		{
			name:        "completely invalid format",
			input:       `"not-a-date-at-all"`,
			expectError: true,
			description: "Should error when no format matches",
		},
		{
			name:        "partial date",
			input:       `"2023-12"`,
			expectError: true,
			description: "Should error on partial date",
		},
		{
			name:        "invalid month",
			input:       `"2023-13-01"`,
			expectError: true,
			description: "Should error on invalid month",
		},
		{
			name:        "invalid day",
			input:       `"2023-02-30"`,
			expectError: true,
			description: "Should error on invalid day",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var customTime CustomTime
			err := json.Unmarshal([]byte(tc.input), &customTime)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tc.description)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tc.description, err)
				}
			}
		})
	}
}

// TestCustomTimeParseError tests that the custom ParseError is returned correctly
func TestCustomTimeParseError(t *testing.T) {
	var customTime CustomTime
	err := json.Unmarshal([]byte(`"invalid-format"`), &customTime)

	if err == nil {
		t.Fatal("Expected error for invalid format")
	}

	// Check that it's a ParseError
	if parseErr, ok := err.(*time.ParseError); ok {
		if parseErr.Layout != "RFC3339 or 2006-01-02T15:04:05 or 2006-01-02" {
			t.Errorf("Expected specific layout in ParseError, got: %s", parseErr.Layout)
		}
		if parseErr.Value != "invalid-format" {
			t.Errorf("Expected value 'invalid-format' in ParseError, got: %s", parseErr.Value)
		}
	} else {
		t.Errorf("Expected *time.ParseError, got: %T", err)
	}
}

// TestCustomTimeStructUsage tests using CustomTime in a struct
func TestCustomTimeStructUsage(t *testing.T) {
	type TestStruct struct {
		ID   int        `json:"id"`
		Time CustomTime `json:"time"`
		Name string     `json:"name"`
	}

	// Test unmarshaling
	jsonInput := `{"id": 1, "time": "2023-12-28T22:00:40", "name": "test"}`

	var testStruct TestStruct
	err := json.Unmarshal([]byte(jsonInput), &testStruct)
	if err != nil {
		t.Fatalf("Failed to unmarshal struct: %v", err)
	}

	if testStruct.ID != 1 {
		t.Errorf("Expected ID 1, got %d", testStruct.ID)
	}
	if testStruct.Name != "test" {
		t.Errorf("Expected name 'test', got %s", testStruct.Name)
	}

	expectedTime := time.Date(2023, 12, 28, 22, 0, 40, 0, time.UTC)
	if !testStruct.Time.Equal(expectedTime) {
		t.Errorf("Expected time %v, got %v", expectedTime, testStruct.Time)
	}

	// Test marshaling
	result, err := json.Marshal(testStruct)
	if err != nil {
		t.Fatalf("Failed to marshal struct: %v", err)
	}

	expected := `{"id":1,"time":"2023-12-28T22:00:40Z","name":"test"}`
	if string(result) != expected {
		t.Errorf("Expected %s, got %s", expected, string(result))
	}
}

// BenchmarkCustomTimeUnmarshal benchmarks CustomTime unmarshaling
func BenchmarkCustomTimeUnmarshal(b *testing.B) {
	testCases := []struct {
		name  string
		input []byte
	}{
		{"RFC3339", []byte(`"2023-12-28T22:00:40Z"`)},
		{"SejmAPI", []byte(`"2023-12-28T22:00:40"`)},
		{"DateOnly", []byte(`"2023-11-13"`)},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				var customTime CustomTime
				_ = json.Unmarshal(tc.input, &customTime)
			}
		})
	}
}

// BenchmarkCustomTimeMarshal benchmarks CustomTime marshaling
func BenchmarkCustomTimeMarshal(b *testing.B) {
	customTime := CustomTime{Time: time.Date(2023, 12, 28, 22, 0, 40, 0, time.UTC)}

	b.Run("Marshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = json.Marshal(customTime)
		}
	})
}

// TestCustomTimeTimezoneConversion tests timezone conversion behavior
func TestCustomTimeTimezoneConversion(t *testing.T) {
	// Test various timezone inputs and ensure they're properly converted
	testCases := []struct {
		name        string
		input       string
		expectedUTC string
	}{
		{
			name:        "UTC explicit",
			input:       `"2023-12-28T22:00:40Z"`,
			expectedUTC: "2023-12-28T22:00:40Z",
		},
		{
			name:        "Europe/Warsaw winter time",
			input:       `"2023-12-28T22:00:40+01:00"`,
			expectedUTC: "2023-12-28T21:00:40Z",
		},
		{
			name:        "US Eastern time",
			input:       `"2023-12-28T22:00:40-05:00"`,
			expectedUTC: "2023-12-29T03:00:40Z",
		},
		{
			name:        "No timezone (assumed local)",
			input:       `"2023-12-28T22:00:40"`,
			expectedUTC: "2023-12-28T22:00:40Z", // Parsed as UTC when no timezone
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var customTime CustomTime
			err := json.Unmarshal([]byte(tc.input), &customTime)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			actualUTC := customTime.Time.UTC().Format(time.RFC3339)
			if actualUTC != tc.expectedUTC {
				t.Errorf("Expected %s, got %s", tc.expectedUTC, actualUTC)
			}
		})
	}
}
