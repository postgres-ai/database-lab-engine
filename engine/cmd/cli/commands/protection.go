/*
2026 © Postgres.ai
*/

package commands

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

const (
	minutesPerHour     = 60
	minutesPerDay      = minutesPerHour * 24
	maxDurationMinutes = 365 * minutesPerDay
)

// ParseProtectedFlag parses the --protected flag value into a three-state protection setting
// and an optional protection duration. A nil bool means the flag was not set (leave protection
// unchanged); a non-nil bool is the requested state, and a non-nil duration requests timed
// protection. The grammar matches clones: 'true'/empty, 'false', plain minutes, or 30m/2h/7d.
func ParseProtectedFlag(cliCtx *cli.Context) (*bool, *uint, error) {
	if !cliCtx.IsSet("protected") {
		return nil, nil, nil
	}

	value := cliCtx.String("protected")

	switch strings.ToLower(value) {
	case "", "true":
		protected := true
		return &protected, nil, nil

	case "false":
		protected := false
		return &protected, nil, nil

	default:
		minutes, err := ParseDurationMinutes(value)
		if err != nil {
			return nil, nil, errors.Errorf(
				"invalid --protected value: %q (use 'true', 'false', minutes, or duration like 30m/2h/7d)", value)
		}

		protected := true
		d := uint(minutes)

		return &protected, &d, nil
	}
}

// ParseDurationMinutes parses a duration string into minutes. Accepted formats: a plain number
// (minutes), or a number with a suffix m (minutes), h (hours), or d (days). Suffix matching is
// case-insensitive.
func ParseDurationMinutes(value string) (uint64, error) {
	lower := strings.ToLower(value)

	var multiplier uint64 = 1

	switch {
	case strings.HasSuffix(lower, "d"):
		multiplier = minutesPerDay
		lower = strings.TrimSuffix(lower, "d")

	case strings.HasSuffix(lower, "h"):
		multiplier = minutesPerHour
		lower = strings.TrimSuffix(lower, "h")

	case strings.HasSuffix(lower, "m"):
		lower = strings.TrimSuffix(lower, "m")
	}

	n, err := strconv.ParseUint(lower, 10, 32)
	if err != nil {
		return 0, err
	}

	result := n * multiplier
	if result > maxDurationMinutes {
		return 0, errors.Errorf("duration too large: %d minutes exceeds maximum of %d minutes (365 days)",
			result, maxDurationMinutes)
	}

	return result, nil
}
