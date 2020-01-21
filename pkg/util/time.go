/*
2019 © Postgres.ai
*/

// Package util provides utility functions. Time and duration processing.
package util

import (
	"fmt"
	"math"
	"time"
)

// TODO (akartasov): Check if functions are being used.

const (
	// MillisecondsInSecond defines a number of milliseconds in an one second.
	MillisecondsInSecond = 1000

	// MillisecondsInMinute defines a number of milliseconds in an one minute.
	MillisecondsInMinute = 60000
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

func DurationToString(value time.Duration) string {
	return MillisecondsToString(float64(value) / 1000000.0)
}

func MillisecondsToString(value float64) string {
	switch {
	case value < MillisecondsInSecond:
		return fmt.Sprintf("%.3f ms", value)
	case value < MillisecondsInMinute:
		return fmt.Sprintf("%.3f s", value/1000.0)
	default:
		return fmt.Sprintf("%.3f min", value/60000.0)
	}
}

func FormatTime(t time.Time) string {
	f := t.UTC()
	return fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d UTC", f.Year(), f.Month(),
		f.Day(), f.Hour(), f.Minute(), f.Second())
}
