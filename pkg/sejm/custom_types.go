package sejm

import (
	"encoding/json"
	"time"
)

// CustomTime handles both RFC3339 format and the format returned by Sejm API
type CustomTime struct {
	time.Time
}

// UnmarshalJSON implements json.Unmarshaler for CustomTime
func (ct *CustomTime) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	// Try RFC3339 format first
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		ct.Time = t
		return nil
	}

	// Try the format returned by Sejm API: "2023-12-28T22:00:40"
	if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
		ct.Time = t
		return nil
	}

	// Try date only format: "2023-11-13"
	if t, err := time.Parse("2006-01-02", s); err == nil {
		ct.Time = t
		return nil
	}

	// If all parsing attempts fail, return the last error
	return &time.ParseError{
		Layout: "RFC3339 or 2006-01-02T15:04:05 or 2006-01-02",
		Value:  s,
	}
}

// MarshalJSON implements json.Marshaler for CustomTime
func (ct CustomTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(ct.Time.Format(time.RFC3339))
}
