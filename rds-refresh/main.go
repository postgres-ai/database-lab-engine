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
	fmt.Fprintf(os.Stderr, `rds-refresh - Automate DBLab full refresh using RDS/Aurora snapshots

This tool creates a temporary RDS/Aurora clone from a snapshot, updates
DBLab Engine config with the clone endpoint, triggers a full refresh,
and then cleans up the temporary clone.

USAGE:
    rds-refresh -config <path> [options]

OPTIONS:
    -config <path>    Path to YAML configuration file (required)
    -dry-run          Validate configuration without creating resources
    -version          Show version information
    -help             Show this help message

DEPLOYMENT:
    This tool is designed to run as a container (Docker, ECS Task, Kubernetes Job)
    or directly from the command line. The refresh process can take 1-4 hours
    depending on database size, so long-running execution environments are required.

    Docker:
        docker run -v /path/to/config.yaml:/config.yaml \
            postgres-ai/rds-refresh -config /config.yaml

    ECS Task / Kubernetes Job:
        Schedule as a periodic task (e.g., daily) using your orchestration platform.

    Cron:
        0 2 * * * /usr/local/bin/rds-refresh -config /etc/rds-refresh/config.yaml

EXAMPLE CONFIGURATION:

    source:
      type: rds                    # or "aurora-cluster"
      identifier: production-db
      dbName: myapp
      username: postgres
      password: ${DB_PASSWORD}     # supports environment variable expansion

    clone:
      instanceClass: db.t3.medium
      subnetGroup: default-vpc-subnet
      securityGroups:
        - sg-12345678
      publiclyAccessible: false

    dblab:
      apiEndpoint: https://dblab.example.com:2345
      token: ${DBLAB_TOKEN}
      pollInterval: 30s
      timeout: 4h

    aws:
      region: us-east-1

WORKFLOW:
    1. Verifies DBLab is healthy and not already refreshing
    2. Gets source database info from RDS/Aurora
    3. Finds the latest automated snapshot
    4. Creates a temporary RDS clone from the snapshot
    5. Waits for the clone to be available (10-30 minutes)
    6. Updates DBLab config with the clone endpoint
    7. Triggers DBLab full refresh
    8. Waits for refresh to complete (1-4 hours)
    9. Deletes the temporary clone

For more information, see:
    https://postgres.ai/docs/database-lab-engine

`)
}
