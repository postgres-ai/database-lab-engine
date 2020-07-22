/*
2020 © Postgres.ai
*/

// Package logical provides jobs for logical initial operations.
package logical

import (
	"time"

	"github.com/docker/docker/api/types/container"
)

const (
	// Default values.
	defaultPort     = 5432
	defaultUsername = "postgres"
	defaultDBName   = "postgres"

	// Defines container health check options.
	hcInterval    = 5 * time.Second
	hcTimeout     = 2 * time.Second
	hcStartPeriod = 3 * time.Second
	hcRetries     = 5
)

func getContainerHealthConfig() *container.HealthConfig {
	return &container.HealthConfig{
		Test:        []string{"CMD-SHELL", "pg_isready -U " + defaultUsername},
		Interval:    hcInterval,
		Timeout:     hcTimeout,
		StartPeriod: hcStartPeriod,
		Retries:     hcRetries,
	}
}
