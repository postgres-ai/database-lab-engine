/*
2022 © Postgres.ai
*/

package models

import (
	"bytes"
	"fmt"
	"time"
)

const legacyFormat = "2006-01-02 15:04:05 UTC"

// LocalTime defines a type of time with custom marshalling depends on locale.
type LocalTime struct {
	time.Time
}

// NewLocalTime creates a new instance of LocalTime.
func NewLocalTime(t time.Time) *LocalTime {
	return &LocalTime{Time: t}
}

// UnmarshalJSON un-marshals LocalTime.
func (t *LocalTime) UnmarshalJSON(data []byte) error {
	localTime := bytes.Trim(data, "\"")

	if len(localTime) == 0 {
		return nil
	}

	parsedTime, err := time.Parse(time.RFC3339, string(localTime))
	if err != nil {
		// Try to parse the legacy format to keep backward compatibility when restore clone sessions.
		parsedTime, err = time.Parse(legacyFormat, string(localTime))
		if err != nil {
			return err
		}
	}

	t.Time = parsedTime

	return nil
}

// MarshalJSON marshals LocalTime.
func (t *LocalTime) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte(`""`), nil
	}

	return []byte(fmt.Sprintf("%q", t.Local().Format(time.RFC3339))), nil
}
