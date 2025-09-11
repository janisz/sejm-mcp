// Package eli provides types and utilities for working with the Polish Legal Information System (ELI) API.
package eli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// CustomTime handles both RFC3339 format and the format returned by ELI API
type CustomTime struct {
	time.Time
}

// UnmarshalJSON implements json.Unmarshaler for CustomTime
func (ct *CustomTime) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	// Handle empty or null values
	if s == "" || s == "null" {
		ct.Time = time.Time{}
		return nil
	}

	// Try RFC3339 format first
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		ct.Time = t
		return nil
	}

	// Try the format returned by ELI API: "2025-09-09T09:21:22"
	if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
		ct.Time = t
		return nil
	}

	// Try the format with space instead of T and no seconds: "2024-03-15 11:32"
	if t, err := time.Parse("2006-01-02 15:04", s); err == nil {
		ct.Time = t
		return nil
	}

	// Try date only format: "2025-09-09"
	if t, err := time.Parse("2006-01-02", s); err == nil {
		ct.Time = t
		return nil
	}

	// If all parsing attempts fail, return the last error
	return &time.ParseError{
		Layout: "RFC3339 or 2006-01-02T15:04:05 or 2006-01-02 15:04 or 2006-01-02",
		Value:  s,
	}
}

// MarshalJSON implements json.Marshaler for CustomTime
func (ct CustomTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(ct.Format(time.RFC3339))
}

// CustomDate handles various date formats returned by ELI API
type CustomDate struct {
	time.Time
}

// UnmarshalJSON implements json.Unmarshaler for CustomDate
func (cd *CustomDate) UnmarshalJSON(data []byte) error {
	// Remove quotes
	str := strings.Trim(string(data), `"`)

	if str == "null" || str == "" {
		return nil
	}

	// Try different date formats
	formats := []string{
		"2006-01-02",          // "2023-11-13"
		"2006-01-02T15:04:05", // "2023-11-13T00:00:00"
		time.RFC3339,          // "2023-11-13T00:00:00Z"
		"2006-01-02 15:04",    // "2024-03-15 11:32"
	}

	for _, format := range formats {
		if t, err := time.Parse(format, str); err == nil {
			cd.Time = t
			return nil
		}
	}

	return fmt.Errorf("unable to parse date %q", str)
}

// MarshalJSON implements json.Marshaler for CustomDate
func (cd CustomDate) MarshalJSON() ([]byte, error) {
	if cd.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(cd.Format("2006-01-02"))
}

// CustomReferenceDetailsInfo is a custom version of ReferenceDetailsInfo with proper date parsing
type CustomReferenceDetailsInfo struct {
	// Act a referenced act
	Act *ActInfo `json:"act,omitempty"`

	// Art referenced article (optional)
	Art *string `json:"art,omitempty"`

	// Date a date (optional, e.g. for repeal) - using CustomDate for better parsing
	Date *CustomDate `json:"date,omitempty"`
}

// CustomReferencesDetailsInfo a map of type -> list of references with custom date parsing
type CustomReferencesDetailsInfo map[string][]CustomReferenceDetailsInfo
