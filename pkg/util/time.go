/*
2019 © Postgres.ai
*/

// Package util provides utility functions. Time and duration processing.
package util

import (
	"fmt"
	"math"
	"strconv"
	"time"
)

// TODO (akartasov): Check if functions are being used.

const (
	// NanosecondsInMillisecond defines a number of nanoseconds in an one millisecond.
	NanosecondsInMillisecond = 1000000.0
	// MillisecondsInSecond defines a number of milliseconds in an one second.
	MillisecondsInSecond = 1000.0

	// MillisecondsInMinute defines a number of milliseconds in an one minute.
	MillisecondsInMinute = 60000.0

	// DataStateAtFormat defines the format of a data state timestamp.
	DataStateAtFormat = "20060102150405"
)

// SecondsAgo returns a number of seconds elapsed from the current time.
func SecondsAgo(ts time.Time) uint {
	now := time.Now()

	seconds := now.Sub(ts).Seconds()
	if seconds < 0 {
		return 0
	}

	return uint(math.Floor(seconds))
}

// DurationToString returns human-readable duration with dimensions.
func DurationToString(value time.Duration) string {
	return MillisecondsToString(float64(value) / NanosecondsInMillisecond)
}

// MillisecondsToString return human-readable duration with dimensions.
func MillisecondsToString(value float64) string {
	switch {
	case value < MillisecondsInSecond:
		return fmt.Sprintf("%.3f ms", value)
	case value < MillisecondsInMinute:
		return fmt.Sprintf("%.3f s", value/MillisecondsInSecond)
	default:
		return fmt.Sprintf("%.3f min", value/MillisecondsInMinute)
	}
}

// FormatTime returns string representing time in UTC in defined format.
func FormatTime(t time.Time) string {
	f := t.UTC()
	return fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d UTC", f.Year(), f.Month(),
		f.Day(), f.Hour(), f.Minute(), f.Second())
}

// ParseUnixTime returns time parsed from unix timestamp integer.
func ParseUnixTime(str string) (time.Time, error) {
	timeInt, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(timeInt, 0), nil
}

// ParseCustomTime returns time parsed from string in defined format.
func ParseCustomTime(str string) (time.Time, error) {
	return time.Parse("20060102150405", str)
}
