/*
2026 © Postgres.ai
*/

package physical

import (
	"context"
	"fmt"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"

	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/cont"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

const (
	syncCheckInterval = 10 * time.Second

	// syncDownPeriod covers several health checks at the default interval: "no response" means nothing listens
	// on the socket, which does not happen while Postgres starts up or recovers.
	syncDownPeriod      = 30 * time.Second
	syncRestartCooldown = 10 * time.Minute

	// syncMaxRestarts limits restarts that do not bring Postgres back to accepting connections.
	syncMaxRestarts = 3

	// pg_isready exit codes recorded by the container health check.
	pgIsReadyAccepting  = 0
	pgIsReadyNoResponse = 2
)

// monitorSyncInstance restarts the sync container when Postgres inside it stops responding, until ctx is canceled.
// The container entrypoint keeps running after Postgres exits, so without this a dead sync instance stays unnoticed.
// Full refresh and engine shutdown cancel ctx before they stop the sync instance on purpose.
func (r *RestoreJob) monitorSyncInstance(ctx context.Context) {
	var (
		downSince, lastRestart time.Time
		restarts               int
		gaveUp                 bool
	)

	ticker := time.NewTicker(syncCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
		}

		syncContainer, err := r.dockerClient.ContainerInspect(ctx, r.syncInstanceName(), client.ContainerInspectOptions{})
		if err != nil {
			continue
		}

		state := syncContainer.Container.State

		if lastHealthCheckIs(state, pgIsReadyAccepting) {
			restarts, gaveUp = 0, false
		}

		if !lastHealthCheckIs(state, pgIsReadyNoResponse) || gaveUp {
			downSince = time.Time{}
			continue
		}

		now := time.Now()

		if downSince.IsZero() {
			downSince = now
		}

		if !restartDue(now, downSince, lastRestart) {
			continue
		}

		if restarts == syncMaxRestarts {
			log.Err(fmt.Sprintf("sync instance %s is still not responding after %d restarts, no more attempts until it recovers",
				r.syncInstanceName(), syncMaxRestarts))

			gaveUp = true

			continue
		}

		log.Err(fmt.Sprintf("sync instance %s is not responding, restarting it, attempt %d of %d",
			r.syncInstanceName(), restarts+1, syncMaxRestarts))

		r.collectSyncDiagnostics(ctx)

		if _, err := r.dockerClient.ContainerRestart(ctx, r.syncInstanceName(), client.ContainerRestartOptions{
			Timeout: pointer.ToInt(cont.StopPhysicalTimeout),
		}); err != nil {
			log.Err("failed to restart sync instance: ", err)
		}

		downSince, lastRestart = time.Time{}, now
		restarts++
	}
}

// lastHealthCheckIs reports whether the container is running and its last health check exited with exitCode.
// Only "no response" means Postgres is down: sync reports "rejecting connections" while it replays WAL to a consistent
// state after a restart.
func lastHealthCheckIs(state *container.State, exitCode int) bool {
	if state == nil || !state.Running || state.Health == nil || len(state.Health.Log) == 0 {
		return false
	}

	lastCheck := state.Health.Log[len(state.Health.Log)-1]

	return lastCheck != nil && lastCheck.ExitCode == exitCode
}

// restartDue reports whether Postgres has been down long enough and the previous restart is old enough.
func restartDue(now, downSince, lastRestart time.Time) bool {
	return now.Sub(downSince) >= syncDownPeriod && now.Sub(lastRestart) >= syncRestartCooldown
}
