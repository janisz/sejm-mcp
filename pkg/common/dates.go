// Package common provides shared utilities and types used across the sejm-mcp application.
package common

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// SejmDate handles date parsing for the Sejm API
// Supports formats: "2023-11-13" and "2023-11-13T00:00:00"
type SejmDate struct {
	time.Time
}

// SejmDateTime handles datetime parsing for the Sejm API
// Supports formats: "2023-12-28T22:00:40" (no timezone)
type SejmDateTime struct {
	time.Time
}

const nullString = "null"

// UnmarshalJSON implements json.Unmarshaler for SejmDate
func (d *SejmDate) UnmarshalJSON(data []byte) error {
	// Remove quotes
	str := strings.Trim(string(data), `"`)

	if str == nullString || str == "" {
		return nil
	}

	// Try different date formats
	formats := []string{
		"2006-01-02",          // "2023-11-13"
		"2006-01-02T15:04:05", // "2023-11-13T00:00:00"
		time.RFC3339,          // "2023-11-13T00:00:00Z"
	}

	for _, format := range formats {
		if t, err := time.Parse(format, str); err == nil {
			d.Time = t
			return nil
		}
	}

	return fmt.Errorf("unable to parse date %q", str)
}

// MarshalJSON implements json.Marshaler for SejmDate
func (d SejmDate) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(d.Format("2006-01-02"))
}

// UnmarshalJSON implements json.Unmarshaler for SejmDateTime
func (dt *SejmDateTime) UnmarshalJSON(data []byte) error {
	// Remove quotes
	str := strings.Trim(string(data), `"`)

	if str == "null" || str == "" {
		return nil
	}

	// Try different datetime formats
	formats := []string{
		"2006-01-02T15:04:05",       // "2023-12-28T22:00:40" (no timezone)
		time.RFC3339,                // "2023-12-28T22:00:40Z"
		"2006-01-02T15:04:05Z07:00", // Full RFC3339
		"2006-01-02T15:04:05.000Z",  // With milliseconds
	}

	for _, format := range formats {
		if t, err := time.Parse(format, str); err == nil {
			dt.Time = t
			return nil
		}
	}

	return fmt.Errorf("unable to parse datetime %q", str)
}

// MarshalJSON implements json.Marshaler for SejmDateTime
func (dt SejmDateTime) MarshalJSON() ([]byte, error) {
	if dt.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(dt.Format("2006-01-02T15:04:05"))
}
