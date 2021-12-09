/*
2021 © Postgres.ai
*/

package main

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"

	"gitlab.com/postgres-ai/database-lab/v3/internal/bootstrap"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

func main() {
	cfg, err := config.LoadConfiguration()
	if err != nil {
		log.Fatal(fmt.Errorf("failed to parse config: %w", err))
	}

	docker, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatal("Failed to create a Docker client:", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := bootstrap.StartDLE(ctx, docker, cfg); err != nil {
		log.Err(err)
		return
	}

	bootstrap.ReportLaunching(cfg)
}
