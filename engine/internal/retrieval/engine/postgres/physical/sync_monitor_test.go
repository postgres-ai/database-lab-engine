/*
2026 © Postgres.ai
*/

package physical

import (
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/stretchr/testify/assert"
)

func healthState(running bool, exitCodes ...int) *container.State {
	health := &container.Health{}

	for _, code := range exitCodes {
		health.Log = append(health.Log, &container.HealthcheckResult{ExitCode: code})
	}

	return &container.State{Running: running, Health: health}
}

func TestLastHealthCheckIs(t *testing.T) {
	testCases := []struct {
		name     string
		state    *container.State
		exitCode int
		expected bool
	}{
		{name: "no state", state: nil, exitCode: pgIsReadyNoResponse, expected: false},
		{name: "stopped container", state: healthState(false, 2), exitCode: pgIsReadyNoResponse, expected: false},
		{name: "no health check", state: &container.State{Running: true}, exitCode: pgIsReadyNoResponse, expected: false},
		{name: "no health check results yet", state: healthState(true), exitCode: pgIsReadyAccepting, expected: false},
		{name: "accepting connections", state: healthState(true, 2, 0), exitCode: pgIsReadyAccepting, expected: true},
		{name: "recovered after no response", state: healthState(true, 2, 0), exitCode: pgIsReadyNoResponse, expected: false},
		{name: "rejecting connections during recovery", state: healthState(true, 2, 1), exitCode: pgIsReadyNoResponse, expected: false},
		{name: "health check timed out", state: healthState(true, -1), exitCode: pgIsReadyNoResponse, expected: false},
		{name: "no response", state: healthState(true, 0, 2), exitCode: pgIsReadyNoResponse, expected: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, lastHealthCheckIs(tc.state, tc.exitCode))
		})
	}
}

func TestRestartDue(t *testing.T) {
	now := time.Date(2026, 10, 9, 5, 30, 0, 0, time.UTC)

	testCases := []struct {
		name        string
		downSince   time.Time
		lastRestart time.Time
		expected    bool
	}{
		{name: "down shorter than the down period", downSince: now.Add(-20 * time.Second), expected: false},
		{name: "down for the down period, never restarted", downSince: now.Add(-30 * time.Second), expected: true},
		{name: "recent restart", downSince: now.Add(-5 * time.Minute), lastRestart: now.Add(-9 * time.Minute), expected: false},
		{name: "restart cooldown passed", downSince: now.Add(-5 * time.Minute), lastRestart: now.Add(-10 * time.Minute), expected: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, restartDue(now, tc.downSince, tc.lastRestart))
		})
	}
}
