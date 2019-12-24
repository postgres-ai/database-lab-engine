/*
2019 © Postgres.ai
*/

package util

import (
	"fmt"
	"math"
	"time"
)

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
	if value < 1000 {
		return fmt.Sprintf("%.3f ms", value)
	} else if value < 60000 {
		return fmt.Sprintf("%.3f s", value/1000.0)
	} else {
		return fmt.Sprintf("%.3f min", value/60000.0)
	}
}

func FormatTime(t time.Time) string {
	f := t.UTC()
	return fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d UTC", f.Year(), f.Month(),
		f.Day(), f.Hour(), f.Minute(), f.Second())
}
