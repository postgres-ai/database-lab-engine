/*
2019 © Postgres.ai
*/

// Package pglog provides helpers for a Postgres logs processing.
package pglog

import (
	"errors"
	"regexp"
	"strings"
	"time"

	errs "github.com/pkg/errors"
)

var (
	// ErrNotFound defines an error if last session activity not found.
	ErrNotFound = errors.New("pglog activity: not found")

	// activityRegexp defines a regexp to parse the last activity time from Postgres logs.
	activityRegexp = regexp.MustCompile(`^([0-9 -:.]+) UTC \[[0-9]+\]`)
)

// GetPostgresLastActivity extracts the time of last session activity.
func GetPostgresLastActivity(text string) (*time.Time, error) {
	const expectedMatchResult = 2

	if !strings.Contains(text, "statement:") {
		return nil, nil
	}

	activitySubmatch := activityRegexp.FindStringSubmatch(text)

	if len(activitySubmatch) == 0 {
		return nil, nil
	}

	if len(activitySubmatch) != expectedMatchResult {
		// TODO(akartasov): Maybe retry.
		// If we find a log entry and cannot parse time, skip the file.
		return nil, errs.New("failed to extract time")
	}

	extractedTime := activitySubmatch[1]

	lastActivityTime, err := time.Parse("2006-01-02 15:04:05.000", extractedTime)
	if err != nil {
		return nil, errs.Wrap(err, "failed to parse the last activity time")
	}

	return &lastActivityTime, nil
}
