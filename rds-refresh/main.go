/*
2025 © PostgresAI

rds-refresh - Automate DBLab full refresh using RDS/Aurora snapshots

This tool creates a temporary RDS/Aurora clone from a snapshot, updates
DBLab Engine config with the clone endpoint, triggers a full refresh,
and then cleans up the temporary clone.
*/
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	configPath := flag.String("config", "", "Path to configuration file")
	dryRun := flag.Bool("dry-run", false, "Validate configuration without creating resources")
	showVersion := flag.Bool("version", false, "Show version information")
	help := flag.Bool("help", false, "Show help")

	flag.Usage = printUsage
	flag.Parse()

	if *help {
		printUsage()
		os.Exit(0)
	}

	if *showVersion {
		fmt.Printf("rds-refresh version %s (built: %s)\n", version, buildTime)
		os.Exit(0)
	}

	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "error: -config flag is required")
		printUsage()
		os.Exit(1)
	}

	if err := run(*configPath, *dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(configPath string, dryRun bool) error {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// handle interrupt signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		fmt.Printf("\nReceived signal %v, initiating graceful shutdown...\n", sig)
		cancel()
	}()

	logger := &DefaultLogger{}

	refresher, err := NewRefresher(ctx, cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize refresher: %w", err)
	}

	if dryRun {
		return refresher.DryRun(ctx)
	}

	result := refresher.Run(ctx)

	fmt.Println()
	fmt.Println("=== Refresh Summary ===")
	fmt.Printf("Success:     %v\n", result.Success)
	fmt.Printf("Snapshot:    %s\n", result.SnapshotID)
	fmt.Printf("Clone ID:    %s\n", result.CloneID)
	fmt.Printf("Duration:    %v\n", result.Duration.Round(1e9))

	if result.Error != nil {
		return result.Error
	}

	return nil
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `rds-refresh - Perform full refresh from RDS/Aurora snapshots (logical mode)

Avoids pg_dump on production (which holds xmin → bloat). Instead, creates a
temporary RDS clone from RDS snapshot, refreshes DBLab from it, then deletes it.

USAGE
    rds-refresh -config <path> [-dry-run]

OPTIONS
    -config <path>   Config file (required)
    -dry-run         Validate only, no changes
    -version         Show version
    -help            Show help

EXAMPLE CONFIG
    source:
      type: rds                  # or aurora-cluster
      identifier: my-prod-db
      dbName: postgres
      username: postgres
      password: ${DB_PASSWORD}
    clone:
      instanceClass: db.t3.medium
      securityGroups: [sg-xxx]
    dblab:
      apiEndpoint: https://dblab:2345
      token: ${DBLAB_TOKEN}
    aws:
      region: us-east-1

DOCKER
    docker run --rm -v $PWD/config.yaml:/config.yaml \
      -e DB_PASSWORD -e DBLAB_TOKEN -e AWS_ACCESS_KEY_ID -e AWS_SECRET_ACCESS_KEY \
      postgresai/rds-refresh -config /config.yaml

More info: https://postgres.ai/docs/database-lab-engine
`)
}
