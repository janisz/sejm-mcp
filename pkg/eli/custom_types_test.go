package eli

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCustomTime_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
		expected  time.Time
	}{
		{
			name:      "RFC3339 format",
			input:     `"2024-03-15T11:32:00Z"`,
			expectErr: false,
			expected:  time.Date(2024, 3, 15, 11, 32, 0, 0, time.UTC),
		},
		{
			name:      "ISO format without timezone",
			input:     `"2024-03-15T11:32:00"`,
			expectErr: false,
			expected:  time.Date(2024, 3, 15, 11, 32, 0, 0, time.UTC),
		},
		{
			name:      "Space format with minutes only",
			input:     `"2024-03-15 11:32"`,
			expectErr: false,
			expected:  time.Date(2024, 3, 15, 11, 32, 0, 0, time.UTC),
		},
		{
			name:      "Date only format",
			input:     `"2024-03-15"`,
			expectErr: false,
			expected:  time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "Invalid format should fail",
			input:     `"2024/03/15 11:32"`,
			expectErr: true,
		},
		{
			name:      "Empty string",
			input:     `""`,
			expectErr: false,
			expected:  time.Time{},
		},
		{
			name:      "null value",
			input:     `"null"`,
			expectErr: false,
			expected:  time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ct CustomTime
			err := json.Unmarshal([]byte(tt.input), &ct)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error for input %s, but got none", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for input %s: %v", tt.input, err)
				return
			}

			if !tt.expected.IsZero() && !ct.Time.Equal(tt.expected) {
				t.Errorf("Expected time %v, got %v for input %s", tt.expected, ct.Time, tt.input)
			}

			if tt.expected.IsZero() && !ct.Time.IsZero() {
				t.Errorf("Expected zero time, got %v for input %s", ct.Time, tt.input)
			}
		})
	}
}

func TestCustomDate_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
		expected  time.Time
	}{
		{
			name:      "Date only format",
			input:     `"2024-03-15"`,
			expectErr: false,
			expected:  time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "RFC3339 format",
			input:     `"2024-03-15T11:32:00Z"`,
			expectErr: false,
			expected:  time.Date(2024, 3, 15, 11, 32, 0, 0, time.UTC),
		},
		{
			name:      "ISO format without timezone",
			input:     `"2024-03-15T11:32:00"`,
			expectErr: false,
			expected:  time.Date(2024, 3, 15, 11, 32, 0, 0, time.UTC),
		},
		{
			name:      "Space format with minutes only",
			input:     `"2024-03-15 11:32"`,
			expectErr: false,
			expected:  time.Date(2024, 3, 15, 11, 32, 0, 0, time.UTC),
		},
		{
			name:      "Invalid format should fail",
			input:     `"2024/03/15"`,
			expectErr: true,
		},
		{
			name:      "Empty string",
			input:     `""`,
			expectErr: false,
			expected:  time.Time{},
		},
		{
			name:      "null value",
			input:     `"null"`,
			expectErr: false,
			expected:  time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cd CustomDate
			err := json.Unmarshal([]byte(tt.input), &cd)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error for input %s, but got none", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for input %s: %v", tt.input, err)
				return
			}

			if !tt.expected.IsZero() && !cd.Time.Equal(tt.expected) {
				t.Errorf("Expected time %v, got %v for input %s", tt.expected, cd.Time, tt.input)
			}

			if tt.expected.IsZero() && !cd.Time.IsZero() {
				t.Errorf("Expected zero time, got %v for input %s", cd.Time, tt.input)
			}
		})
	}
}
