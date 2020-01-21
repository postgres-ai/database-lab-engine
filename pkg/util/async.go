/*
2019 © Postgres.ai
*/

// Package util provides utility functions. Async related utils.
package util

import (
	"time"
)

func RunInterval(d time.Duration, fn func()) chan struct{} {
	ticker := time.NewTicker(d)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				fn()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

	// Use `close(quit)` to stop interval execution.
	return quit
}
