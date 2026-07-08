/*
2026 © Postgres.ai
*/

package models

import (
	"fmt"
	"time"
)

// ProtectionForever is the stored dle:protected_till value that marks indefinite
// protection (protected with no expiry timestamp).
const ProtectionForever = "forever"

// isProtected reports whether an entity with the given protection flag and
// optional expiry is currently protected.
func isProtected(protected bool, till *LocalTime) bool {
	if !protected {
		return false
	}

	if till == nil {
		return true
	}

	return till.After(time.Now())
}

// protectionExpiresIn returns the duration until protection expires. it returns
// 0 if the entity is not protected, protection has no expiry, or it already expired.
func protectionExpiresIn(protected bool, till *LocalTime) time.Duration {
	if !protected || till == nil {
		return 0
	}

	duration := time.Until(till.Time)
	if duration < 0 {
		return 0
	}

	return duration
}

// ParseProtectedTill interprets a stored dle:protected_till value: an empty value
// means not protected, ProtectionForever means protected without an expiry, and an
// RFC3339 timestamp means protected until that time. A non-empty value that is
// neither ProtectionForever nor valid RFC3339 is treated as not protected and the
// error is returned so the caller can log it — this avoids silently turning a
// corrupted value into permanent, undeletable protection.
func ParseProtectedTill(value string) (bool, *LocalTime, error) {
	if value == "" {
		return false, nil, nil
	}

	if value == ProtectionForever {
		return true, nil, nil
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return false, nil, fmt.Errorf("invalid protected_till value %q: %w", value, err)
	}

	return true, NewLocalTime(parsed), nil
}

// CalculateProtectionTime computes the protection-expiry time for a requested duration.
// A nil durationMinutes falls back to defaultMin; a resulting 0 means indefinite protection
// (nil) unless maxMin caps it, and any duration is capped at maxMin when maxMin > 0.
func CalculateProtectionTime(durationMinutes *uint, defaultMin, maxMin uint) *LocalTime {
	minutes := defaultMin
	if durationMinutes != nil {
		minutes = *durationMinutes
	}

	if minutes == 0 {
		if maxMin == 0 {
			return nil
		}

		minutes = maxMin
	}

	if maxMin > 0 && minutes > maxMin {
		minutes = maxMin
	}

	return NewLocalTime(time.Now().Add(time.Duration(minutes) * time.Minute))
}

// ProtectedTillActive reports whether a stored dle:protected_till value currently
// protects an entity — the ProtectionForever sentinel or a not-yet-expired timestamp.
// A malformed value is treated as not protected (consistent with ParseProtectedTill).
func ProtectedTillActive(protectedTill string) bool {
	protected, till, err := ParseProtectedTill(protectedTill)
	if err != nil {
		return false
	}

	return isProtected(protected, till)
}

// ParseDeleteAt interprets a stored dle:delete_at value: an empty value means no
// scheduled deletion, and an RFC3339 timestamp means deletion is scheduled for that
// time. A non-empty value that is not valid RFC3339 returns an error and no schedule.
func ParseDeleteAt(value string) (*LocalTime, error) {
	if value == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, fmt.Errorf("invalid delete_at value %q: %w", value, err)
	}

	return NewLocalTime(parsed), nil
}
