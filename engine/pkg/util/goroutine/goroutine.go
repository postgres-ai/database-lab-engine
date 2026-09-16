/*
2026 © Postgres.ai
*/

// Package goroutine launches background goroutines that survive a panic instead of killing the engine process.
package goroutine

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

const (
	initialRestartDelay = time.Second
	maxRestartDelay     = time.Minute
	backoffFactor       = 2
)

// Go runs fn in a new goroutine. A panic inside fn is recovered and logged together with
// the goroutine name and the stack; it never propagates and fn is not restarted.
func Go(name string, fn func()) {
	go Run(name, fn)
}

// Loop runs fn in a new goroutine and restarts it with exponential backoff every time it panics,
// until ctx is done. The backoff starts over after a run that lasted at least the maximum delay,
// so a rare panic does not keep a long-running goroutine at the slowest restart rate.
// A normal return of fn ends the loop.
func Loop(ctx context.Context, name string, fn func()) {
	go loop(ctx, name, fn, initialRestartDelay, maxRestartDelay)
}

func loop(ctx context.Context, name string, fn func(), initialDelay, maxDelay time.Duration) {
	delay := initialDelay

	for {
		started := time.Now()

		if !Run(name, fn) {
			return
		}

		if time.Since(started) >= maxDelay {
			delay = initialDelay
		}

		log.Msg(fmt.Sprintf("restarting goroutine %q in %s", name, delay))

		select {
		case <-ctx.Done():
			return

		case <-time.After(delay):
		}

		delay = min(delay*backoffFactor, maxDelay)
	}
}

// Run calls fn synchronously and reports whether it panicked. The panic is recovered and logged with
// the name and the stack, so callers such as cron jobs or request handlers keep running.
func Run(name string, fn func()) (panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			panicked = true

			log.Err(fmt.Sprintf("recovered panic in goroutine %q: %v\n%s", name, r, debug.Stack()))
		}
	}()

	fn()

	return false
}
