/*
2026 © Postgres.ai
*/

package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSnapshot_IsProtected(t *testing.T) {
	tests := []struct {
		name     string
		snapshot Snapshot
		expected bool
	}{
		{name: "not protected", snapshot: Snapshot{Protected: false}, expected: false},
		{name: "protected with no expiry (infinite)", snapshot: Snapshot{Protected: true, ProtectedTill: nil}, expected: true},
		{name: "protected with future expiry", snapshot: Snapshot{Protected: true, ProtectedTill: NewLocalTime(time.Now().Add(time.Hour))}, expected: true},
		{name: "protected with past expiry (expired)", snapshot: Snapshot{Protected: true, ProtectedTill: NewLocalTime(time.Now().Add(-time.Hour))}, expected: false},
		{name: "not protected but has expiry time", snapshot: Snapshot{Protected: false, ProtectedTill: NewLocalTime(time.Now().Add(time.Hour))}, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.snapshot.IsProtected())
		})
	}
}

func TestSnapshot_ProtectionExpiresIn(t *testing.T) {
	tests := []struct {
		name        string
		snapshot    Snapshot
		expectZero  bool
		minDuration time.Duration
		maxDuration time.Duration
	}{
		{name: "not protected returns zero", snapshot: Snapshot{Protected: false}, expectZero: true},
		{name: "protected with no expiry returns zero", snapshot: Snapshot{Protected: true, ProtectedTill: nil}, expectZero: true},
		{name: "protected with past expiry returns zero", snapshot: Snapshot{Protected: true, ProtectedTill: NewLocalTime(time.Now().Add(-time.Hour))}, expectZero: true},
		{name: "protected with future expiry", snapshot: Snapshot{Protected: true, ProtectedTill: NewLocalTime(time.Now().Add(time.Hour))}, expectZero: false, minDuration: 59 * time.Minute, maxDuration: 61 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.snapshot.ProtectionExpiresIn()
			if tt.expectZero {
				assert.Equal(t, time.Duration(0), result)
				return
			}

			assert.GreaterOrEqual(t, result, tt.minDuration)
			assert.LessOrEqual(t, result, tt.maxDuration)
		})
	}
}
