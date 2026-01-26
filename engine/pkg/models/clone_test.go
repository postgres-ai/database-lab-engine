/*
2019 © Postgres.ai
*/

package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClone_IsProtected(t *testing.T) {
	tests := []struct {
		name     string
		clone    Clone
		expected bool
	}{
		{name: "not protected flag is false", clone: Clone{Protected: false}, expected: false},
		{name: "protected with no expiry (infinite)", clone: Clone{Protected: true, ProtectedTill: nil}, expected: true},
		{name: "protected with future expiry", clone: Clone{Protected: true, ProtectedTill: NewLocalTime(time.Now().Add(1 * time.Hour))}, expected: true},
		{name: "protected with past expiry (expired)", clone: Clone{Protected: true, ProtectedTill: NewLocalTime(time.Now().Add(-1 * time.Hour))}, expected: false},
		{name: "protected with expiry exactly now", clone: Clone{Protected: true, ProtectedTill: NewLocalTime(time.Now())}, expected: false},
		{name: "protected with far future expiry", clone: Clone{Protected: true, ProtectedTill: NewLocalTime(time.Now().Add(365 * 24 * time.Hour))}, expected: true},
		{name: "protected with 1 minute future", clone: Clone{Protected: true, ProtectedTill: NewLocalTime(time.Now().Add(1 * time.Minute))}, expected: true},
		{name: "protected with 1 second past", clone: Clone{Protected: true, ProtectedTill: NewLocalTime(time.Now().Add(-1 * time.Second))}, expected: false},
		{name: "not protected but has expiry time", clone: Clone{Protected: false, ProtectedTill: NewLocalTime(time.Now().Add(1 * time.Hour))}, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.clone.IsProtected()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClone_ProtectionExpiresIn(t *testing.T) {
	tests := []struct {
		name        string
		clone       Clone
		expectZero  bool
		minDuration time.Duration
		maxDuration time.Duration
	}{
		{name: "not protected returns zero", clone: Clone{Protected: false}, expectZero: true},
		{name: "protected with no expiry returns zero", clone: Clone{Protected: true, ProtectedTill: nil}, expectZero: true},
		{name: "protected with past expiry returns zero", clone: Clone{Protected: true, ProtectedTill: NewLocalTime(time.Now().Add(-1 * time.Hour))}, expectZero: true},
		{name: "protected with future expiry", clone: Clone{Protected: true, ProtectedTill: NewLocalTime(time.Now().Add(1 * time.Hour))}, expectZero: false, minDuration: 59 * time.Minute, maxDuration: 61 * time.Minute},
		{name: "protected with 30 minute expiry", clone: Clone{Protected: true, ProtectedTill: NewLocalTime(time.Now().Add(30 * time.Minute))}, expectZero: false, minDuration: 29 * time.Minute, maxDuration: 31 * time.Minute},
		{name: "protected with 1 day expiry", clone: Clone{Protected: true, ProtectedTill: NewLocalTime(time.Now().Add(24 * time.Hour))}, expectZero: false, minDuration: 23 * time.Hour, maxDuration: 25 * time.Hour},
		{name: "not protected but has expiry time", clone: Clone{Protected: false, ProtectedTill: NewLocalTime(time.Now().Add(1 * time.Hour))}, expectZero: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.clone.ProtectionExpiresIn()
			if tt.expectZero {
				assert.Equal(t, time.Duration(0), result)
			} else {
				assert.GreaterOrEqual(t, result, tt.minDuration)
				assert.LessOrEqual(t, result, tt.maxDuration)
			}
		})
	}
}

func TestClone_IsProtected_EdgeCases(t *testing.T) {
	t.Run("rapid check near expiry time", func(t *testing.T) {
		expiryTime := time.Now().Add(100 * time.Millisecond)
		clone := Clone{Protected: true, ProtectedTill: NewLocalTime(expiryTime)}

		assert.True(t, clone.IsProtected())

		time.Sleep(150 * time.Millisecond)

		assert.False(t, clone.IsProtected())
	})

	t.Run("zero time value", func(t *testing.T) {
		clone := Clone{Protected: true, ProtectedTill: NewLocalTime(time.Time{})}
		assert.False(t, clone.IsProtected())
	})
}

func TestClone_ProtectionExpiresIn_EdgeCases(t *testing.T) {
	t.Run("expiry in milliseconds", func(t *testing.T) {
		clone := Clone{Protected: true, ProtectedTill: NewLocalTime(time.Now().Add(500 * time.Millisecond))}

		result := clone.ProtectionExpiresIn()
		assert.Greater(t, result, time.Duration(0))
		assert.Less(t, result, 1*time.Second)
	})

	t.Run("very large expiry", func(t *testing.T) {
		clone := Clone{Protected: true, ProtectedTill: NewLocalTime(time.Now().Add(10 * 365 * 24 * time.Hour))}

		result := clone.ProtectionExpiresIn()
		assert.Greater(t, result, 9*365*24*time.Hour)
	})

	t.Run("just expired returns zero", func(t *testing.T) {
		clone := Clone{Protected: true, ProtectedTill: NewLocalTime(time.Now().Add(-1 * time.Nanosecond))}

		result := clone.ProtectionExpiresIn()
		assert.Equal(t, time.Duration(0), result)
	})
}
