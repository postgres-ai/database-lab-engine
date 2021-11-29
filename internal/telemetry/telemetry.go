/*
2021 © Postgres.ai
*/

// Package telemetry contains tools to collect Database Lab Engine data.
package telemetry

import (
	"context"
	"fmt"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/platform"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
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

	// SnapshotCreatedEvent describes a snapshot creation event.
	SnapshotCreatedEvent = "snapshot_created"

	// AlertEvent describes alert events.
	AlertEvent = "alert"
)

// Agent represent a telemetry agent to collect engine data.
type Agent struct {
	instanceID string
	cfg        global.Telemetry
	platform   *platform.Client
}

// New creates a new agent.
func New(cfg global.Config, engineProps global.EngineProps) (*Agent, error) {
	platformClient, err := platform.NewClient(platform.ClientConfig{
		URL:         cfg.Telemetry.URL,
		AccessToken: engineProps.InstanceID, // Use the instance ID as a token to keep events anonymous and protect API from random bots.
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create a new telemetry client: %w", err)
	}

	return &Agent{
		instanceID: engineProps.InstanceID,
		cfg:        cfg.Telemetry,
		platform:   platformClient,
	}, nil
}

// Reload reloads configuration of the telemetry agent.
func (a *Agent) Reload(cfg global.Config) {
	a.cfg = cfg.Telemetry
}

// IsEnabled checks if telemetry is enabled.
func (a *Agent) IsEnabled() bool {
	return a.cfg.Enabled
}

// SendEvent sends a telemetry event.
func (a *Agent) SendEvent(ctx context.Context, eventType string, payload interface{}) {
	if !a.IsEnabled() {
		return
	}

	_, err := a.platform.SendTelemetryEvent(ctx, platform.TelemetryEvent{
		InstanceID: a.instanceID,
		EventType:  eventType,
		Payload:    payload,
	})

	if err != nil {
		log.Err("Failed to send telemetry event", err)
	}
}
