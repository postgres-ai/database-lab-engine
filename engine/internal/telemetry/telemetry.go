/*
2021 © Postgres.ai
*/

// Package telemetry contains tools to collect Database Lab Engine data.
package telemetry

import (
	"context"

	platformSvc "gitlab.com/postgres-ai/database-lab/v3/internal/platform"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/platform"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

const (
	// EngineStartedEvent defines the engine start event.
	EngineStartedEvent = "engine_started"

	// EngineStoppedEvent describes the engine stop event.
	EngineStoppedEvent = "engine_stopped"

	// CloneCreatedEvent describes the clone creation event.
	CloneCreatedEvent = "clone_created"

	// CloneResetEvent describes the clone reset event.
	CloneResetEvent = "clone_reset"

	// CloneDestroyedEvent describes a clone destruction event.
	CloneDestroyedEvent = "clone_destroyed"

	// CloneUpdatedEvent describes a clone update event.
	CloneUpdatedEvent = "clone_updated"

	// SnapshotCreatedEvent describes a snapshot creation event.
	SnapshotCreatedEvent = "snapshot_created"

	// BranchCreatedEvent describes a branch creation event.
	BranchCreatedEvent = "branch_created"

	// BranchDestroyedEvent describes a branch destruction event.
	BranchDestroyedEvent = "branch_destroyed"

	ConfigUpdatedEvent = "config_updated"

	// AlertEvent describes alert events.
	AlertEvent = "alert"
)

// Agent represent a telemetry agent to collect engine data.
type Agent struct {
	instanceID string
	platform   *platformSvc.Service
}

// New creates a new agent.
func New(platformSvc *platformSvc.Service, instanceID string) *Agent {
	return &Agent{
		instanceID: instanceID,
		platform:   platformSvc,
	}
}

// SendEvent sends a telemetry event.
func (a *Agent) SendEvent(ctx context.Context, eventType string, payload interface{}) {
	if !a.platform.IsTelemetryEnabled() {
		return
	}

	_, err := a.platform.Client.SendTelemetryEvent(ctx, platform.TelemetryEvent{
		InstanceID: a.instanceID,
		EventType:  eventType,
		Payload:    payload,
	})

	if err != nil {
		log.Err("Failed to send telemetry event", err)
	}
}
