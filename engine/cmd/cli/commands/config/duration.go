/*
2021 © Postgres.ai
*/

package config

import (
	"bytes"
	"fmt"
	"time"
)

// Duration defines a custom duration type.
type Duration time.Duration

// String returns formatted duration.
func (d Duration) String() string {
	return time.Duration(d).String()
}

// UnmarshalJSON un-marshals json duration.
func (d *Duration) UnmarshalJSON(b []byte) error {
	timeDuration, err := time.ParseDuration(string(bytes.Trim(b, `"`)))
	if err != nil {
		return err
	}

	*d = Duration(timeDuration)

	return err
}

// MarshalJSON marshals json duration.
func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", d.String())), nil
}

// MarshalYAML marshals yaml duration.
func (d Duration) MarshalYAML() (interface{}, error) {
	return d.String(), nil
}

// UnmarshalYAML marshals yaml duration.
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var durationString string
	if err := unmarshal(&durationString); err != nil {
		return err
	}

	timeDuration, err := time.ParseDuration(durationString)
	if err != nil {
		*d = 0
		return err
	}

	*d = Duration(timeDuration)

	return nil
}

// Duration retrieves duration from a custom type.
func (d *Duration) Duration() time.Duration {
	return time.Duration(*d)
}
