package common

import (
	"encoding/json"
	"testing"
	"time"
)

// TestSejmDateUnmarshalJSON tests SejmDate JSON unmarshaling
func TestSejmDateUnmarshalJSON(t *testing.T) {
	testCases := []struct {
		name         string
		input        string
		expectError  bool
		expectedDate string
		description  string
	}{
		{
			name:         "valid date format YYYY-MM-DD",
			input:        `"2023-11-13"`,
			expectError:  false,
			expectedDate: "2023-11-13",
			description:  "Should parse standard date format",
		},
		{
			name:         "valid datetime format",
			input:        `"2023-11-13T00:00:00"`,
			expectError:  false,
			expectedDate: "2023-11-13",
			description:  "Should parse datetime format and extract date",
		},
		{
			name:         "valid RFC3339 format",
			input:        `"2023-11-13T00:00:00Z"`,
			expectError:  false,
			expectedDate: "2023-11-13",
			description:  "Should parse RFC3339 format",
		},
		{
			name:        "null value",
			input:       `"null"`,
			expectError: false,
			description: "Should handle null values",
		},
		{
			name:        "empty string",
			input:       `""`,
			expectError: false,
			description: "Should handle empty strings",
		},
		{
			name:        "invalid date format",
			input:       `"invalid-date"`,
			expectError: true,
			description: "Should error on invalid date format",
		},
		{
			name:        "wrong format",
			input:       `"13/11/2023"`,
			expectError: true,
			description: "Should error on wrong date format",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var date SejmDate
			err := json.Unmarshal([]byte(tc.input), &date)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tc.description)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tc.description, err)
				}

				if tc.expectedDate != "" {
					actualDate := date.Time.Format("2006-01-02")
					if actualDate != tc.expectedDate {
						t.Errorf("Expected date %s for %s, but got %s", tc.expectedDate, tc.description, actualDate)
					}
				}
			}
		})
	}
}

// TestSejmDateMarshalJSON tests SejmDate JSON marshaling
func TestSejmDateMarshalJSON(t *testing.T) {
	testCases := []struct {
		name        string
		date        SejmDate
		expected    string
		description string
	}{
		{
			name:        "valid date",
			date:        SejmDate{Time: time.Date(2023, 11, 13, 0, 0, 0, 0, time.UTC)},
			expected:    `"2023-11-13"`,
			description: "Should marshal date in YYYY-MM-DD format",
		},
		{
			name:        "zero date",
			date:        SejmDate{},
			expected:    "null",
			description: "Should marshal zero date as null",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := json.Marshal(tc.date)
			if err != nil {
				t.Errorf("Unexpected error for %s: %v", tc.description, err)
			}

			if string(result) != tc.expected {
				t.Errorf("Expected %s for %s, but got %s", tc.expected, tc.description, string(result))
			}
		})
	}
}

// TestSejmDateTimeUnmarshalJSON tests SejmDateTime JSON unmarshaling
func TestSejmDateTimeUnmarshalJSON(t *testing.T) {
	testCases := []struct {
		name         string
		input        string
		expectError  bool
		expectedTime string
		description  string
	}{
		{
			name:         "valid datetime without timezone",
			input:        `"2023-12-28T22:00:40"`,
			expectError:  false,
			expectedTime: "2023-12-28T22:00:40",
			description:  "Should parse datetime without timezone",
		},
		{
			name:         "valid RFC3339 format",
			input:        `"2023-12-28T22:00:40Z"`,
			expectError:  false,
			expectedTime: "2023-12-28T22:00:40",
			description:  "Should parse RFC3339 format",
		},
		{
			name:         "valid RFC3339 with timezone",
			input:        `"2023-12-28T22:00:40+01:00"`,
			expectError:  false,
			expectedTime: "2023-12-28T21:00:40", // Converted to UTC
			description:  "Should parse RFC3339 with timezone",
		},
		{
			name:         "valid with milliseconds",
			input:        `"2023-12-28T22:00:40.000Z"`,
			expectError:  false,
			expectedTime: "2023-12-28T22:00:40",
			description:  "Should parse datetime with milliseconds",
		},
		{
			name:        "null value",
			input:       `"null"`,
			expectError: false,
			description: "Should handle null values",
		},
		{
			name:        "empty string",
			input:       `""`,
			expectError: false,
			description: "Should handle empty strings",
		},
		{
			name:        "invalid datetime format",
			input:       `"invalid-datetime"`,
			expectError: true,
			description: "Should error on invalid datetime format",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var dateTime SejmDateTime
			err := json.Unmarshal([]byte(tc.input), &dateTime)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tc.description)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tc.description, err)
				}

				if tc.expectedTime != "" {
					actualTime := dateTime.Time.UTC().Format("2006-01-02T15:04:05")
					if actualTime != tc.expectedTime {
						t.Errorf("Expected time %s for %s, but got %s", tc.expectedTime, tc.description, actualTime)
					}
				}
			}
		})
	}
}

// TestSejmDateTimeMarshalJSON tests SejmDateTime JSON marshaling
func TestSejmDateTimeMarshalJSON(t *testing.T) {
	testCases := []struct {
		name        string
		dateTime    SejmDateTime
		expected    string
		description string
	}{
		{
			name:        "valid datetime",
			dateTime:    SejmDateTime{Time: time.Date(2023, 12, 28, 22, 0, 40, 0, time.UTC)},
			expected:    `"2023-12-28T22:00:40"`,
			description: "Should marshal datetime in YYYY-MM-DDTHH:MM:SS format",
		},
		{
			name:        "zero datetime",
			dateTime:    SejmDateTime{},
			expected:    "null",
			description: "Should marshal zero datetime as null",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := json.Marshal(tc.dateTime)
			if err != nil {
				t.Errorf("Unexpected error for %s: %v", tc.description, err)
			}

			if string(result) != tc.expected {
				t.Errorf("Expected %s for %s, but got %s", tc.expected, tc.description, string(result))
			}
		})
	}
}

// TestSejmDateRoundTrip tests round-trip marshaling/unmarshaling
func TestSejmDateRoundTrip(t *testing.T) {
	original := SejmDate{Time: time.Date(2023, 11, 13, 0, 0, 0, 0, time.UTC)}

	// Marshal to JSON
	jsonData, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal SejmDate: %v", err)
	}

	// Unmarshal back
	var restored SejmDate
	err = json.Unmarshal(jsonData, &restored)
	if err != nil {
		t.Fatalf("Failed to unmarshal SejmDate: %v", err)
	}

	// Compare (only date part, ignoring time)
	if !original.Time.Truncate(24 * time.Hour).Equal(restored.Time.Truncate(24 * time.Hour)) {
		t.Errorf("Round-trip failed: original %v, restored %v", original.Time, restored.Time)
	}
}

// TestSejmDateTimeRoundTrip tests round-trip marshaling/unmarshaling
func TestSejmDateTimeRoundTrip(t *testing.T) {
	original := SejmDateTime{Time: time.Date(2023, 12, 28, 22, 0, 40, 0, time.UTC)}

	// Marshal to JSON
	jsonData, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal SejmDateTime: %v", err)
	}

	// Unmarshal back
	var restored SejmDateTime
	err = json.Unmarshal(jsonData, &restored)
	if err != nil {
		t.Fatalf("Failed to unmarshal SejmDateTime: %v", err)
	}

	// Compare (truncate to seconds to avoid nanosecond differences)
	if !original.Time.Truncate(time.Second).Equal(restored.Time.Truncate(time.Second)) {
		t.Errorf("Round-trip failed: original %v, restored %v", original.Time, restored.Time)
	}
}

// TestEdgeCases tests edge cases and error conditions
func TestEdgeCases(t *testing.T) {
	t.Run("SejmDate with invalid JSON", func(t *testing.T) {
		var date SejmDate
		err := json.Unmarshal([]byte(`{invalid json}`), &date)
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
	})

	t.Run("SejmDateTime with invalid JSON", func(t *testing.T) {
		var dateTime SejmDateTime
		err := json.Unmarshal([]byte(`{invalid json}`), &dateTime)
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
	})

	t.Run("SejmDate with number instead of string", func(t *testing.T) {
		var date SejmDate
		err := json.Unmarshal([]byte(`123456789`), &date)
		if err == nil {
			t.Error("Expected error for number input")
		}
	})

	t.Run("SejmDateTime with number instead of string", func(t *testing.T) {
		var dateTime SejmDateTime
		err := json.Unmarshal([]byte(`123456789`), &dateTime)
		if err == nil {
			t.Error("Expected error for number input")
		}
	})
}

// BenchmarkSejmDateUnmarshal benchmarks SejmDate unmarshaling
func BenchmarkSejmDateUnmarshal(b *testing.B) {
	input := []byte(`"2023-11-13"`)

	b.Run("StandardDate", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var date SejmDate
			_ = json.Unmarshal(input, &date)
		}
	})

	datetimeInput := []byte(`"2023-11-13T00:00:00"`)
	b.Run("DateTimeFormat", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var date SejmDate
			_ = json.Unmarshal(datetimeInput, &date)
		}
	})
}

// BenchmarkSejmDateTimeMarshal benchmarks SejmDateTime marshaling
func BenchmarkSejmDateTimeMarshal(b *testing.B) {
	dateTime := SejmDateTime{Time: time.Date(2023, 12, 28, 22, 0, 40, 0, time.UTC)}

	b.Run("Marshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = json.Marshal(dateTime)
		}
	})
}

// TestTimeZoneHandling tests timezone handling in datetime parsing
func TestTimeZoneHandling(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "UTC timezone",
			input:    `"2023-12-28T22:00:40Z"`,
			expected: "2023-12-28T22:00:40",
		},
		{
			name:     "positive timezone offset",
			input:    `"2023-12-28T22:00:40+01:00"`,
			expected: "2023-12-28T21:00:40", // Converted to UTC
		},
		{
			name:     "negative timezone offset",
			input:    `"2023-12-28T22:00:40-05:00"`,
			expected: "2023-12-29T03:00:40", // Converted to UTC
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var dateTime SejmDateTime
			err := json.Unmarshal([]byte(tc.input), &dateTime)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			actual := dateTime.Time.UTC().Format("2006-01-02T15:04:05")
			if actual != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, actual)
			}
		})
	}
}
