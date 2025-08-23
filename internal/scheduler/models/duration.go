package models

import (
	"encoding/json"
	"time"
)

// Duration is a wrapper around time.Duration that marshals/unmarshals as a string
type Duration time.Duration

// MarshalJSON implements json.Marshaler
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON implements json.Unmarshaler
func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		// Try to unmarshal as number (nanoseconds) for backward compatibility
		var ns int64
		if err := json.Unmarshal(b, &ns); err != nil {
			return err
		}
		*d = Duration(ns)
		return nil
	}
	
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

// String returns the string representation
func (d Duration) String() string {
	return time.Duration(d).String()
}

// ToDuration converts to time.Duration
func (d Duration) ToDuration() time.Duration {
	return time.Duration(d)
}