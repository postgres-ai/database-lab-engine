/*
2026 © Postgres.ai
*/

package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseProtectedTill(t *testing.T) {
	tests := []struct {
		name          string
		value         string
		wantProtected bool
		wantTill      bool
		wantErr       bool
	}{
		{name: "empty is not protected", value: "", wantProtected: false, wantTill: false},
		{name: "forever is indefinite protection", value: ProtectionForever, wantProtected: true, wantTill: false},
		{name: "rfc3339 is timed protection", value: "2026-06-17T14:30:00Z", wantProtected: true, wantTill: true},
		{name: "malformed is not protected and errors", value: "2026-13-99", wantProtected: false, wantTill: false, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			protected, till, err := ParseProtectedTill(tt.value)
			assert.Equal(t, tt.wantProtected, protected)
			assert.Equal(t, tt.wantTill, till != nil)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestProtectedTillActive(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "empty is not active", value: "", want: false},
		{name: "forever is active", value: ProtectionForever, want: true},
		{name: "future timestamp is active", value: time.Now().Add(time.Hour).UTC().Format(time.RFC3339), want: true},
		{name: "past timestamp is not active", value: time.Now().Add(-time.Hour).UTC().Format(time.RFC3339), want: false},
		{name: "malformed is not active", value: "nope", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ProtectedTillActive(tt.value))
		})
	}
}

func TestParseDeleteAt(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantSet bool
		wantErr bool
	}{
		{name: "empty has no schedule", value: "", wantSet: false},
		{name: "rfc3339 is scheduled", value: "2026-06-18T00:00:00Z", wantSet: true},
		{name: "malformed has no schedule and errors", value: "nope", wantSet: false, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deleteAt, err := ParseDeleteAt(tt.value)
			assert.Equal(t, tt.wantSet, deleteAt != nil)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
