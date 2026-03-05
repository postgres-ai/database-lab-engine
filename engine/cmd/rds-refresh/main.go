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
	"time"

	"gitlab.com/postgres-ai/database-lab/v3/internal/rdsrefresh"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "cleanup" {
		if err := runCleanup(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		return
	}

	configPath := flag.String("config", "", "path to configuration file")
	dryRun := flag.Bool("dry-run", false, "validate configuration without creating resources")
	showVersion := flag.Bool("version", false, "show version information")
	stateDir := flag.String("state-dir", "", "directory for state file (default: ./meta/)")
	skipCleanup := flag.Bool("skip-cleanup", false, "skip startup cleanup of orphaned clones")
	help := flag.Bool("help", false, "show help")

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

	opts := runOptions{
		configPath:  *configPath,
		dryRun:      *dryRun,
		stateDir:    *stateDir,
		skipCleanup: *skipCleanup,
	}

	if err := run(opts); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

type runOptions struct {
	configPath  string
	dryRun      bool
	stateDir    string
	skipCleanup bool
}

func run(opts runOptions) error {
	cfg, err := rdsrefresh.LoadConfig(opts.configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// handle interrupt signals for graceful shutdown
	// SIGHUP is included to handle SSH disconnections
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		sig := <-sigCh
		log.Msg("\nreceived signal", sig, "initiating graceful shutdown...")
		cancel()
	}()

	// create state file manager
	stateFile := rdsrefresh.NewStateFile(opts.stateDir)

	// perform startup cleanup unless skipped
	if !opts.skipCleanup && !opts.dryRun {
		if err := performStartupCleanup(ctx, cfg, stateFile); err != nil {
			log.Warn("startup cleanup failed:", err)
		}
	}

	refresher, err := rdsrefresh.NewRefresherWithStateFile(ctx, cfg, stateFile)
	if err != nil {
		return fmt.Errorf("failed to initialize refresher: %w", err)
	}

	if opts.dryRun {
		return refresher.DryRun(ctx)
	}

	result := refresher.Run(ctx)

	log.Msg("=== Refresh Summary ===")
	log.Msg("Success:    ", result.Success)
	log.Msg("Snapshot:   ", result.SnapshotID)
	log.Msg("Clone ID:   ", result.CloneID)

	log.Msg("Duration:   ", result.Duration.Round(time.Second))

	if result.Error != nil {
		return result.Error
	}

	return nil
}

func performStartupCleanup(ctx context.Context, cfg *rdsrefresh.Config, stateFile *rdsrefresh.StateFile) error {
	// only perform cleanup if state file exists, indicating a previous run may have crashed
	if !stateFile.Exists() {
		return nil
	}

	log.Msg("found state file from previous run, performing cleanup...")

	rdsClient, err := rdsrefresh.NewRDSClient(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create RDS client: %w", err)
	}

	// try to clean up from state file
	if err := rdsClient.CleanupFromStateFile(ctx, stateFile); err != nil {
		log.Warn("state file cleanup failed:", err)
	}

	// also scan for stale clones by tags in case state file had empty clone ID
	// (clone was created but process crashed before updating state file)
	log.Msg("scanning for stale clones...")

	result, err := rdsClient.CleanupStaleClones(ctx, false)
	if err != nil {
		return fmt.Errorf("stale clone cleanup failed: %w", err)
	}

	if result.ClonesFound > 0 {
		log.Msg("cleanup complete: found", result.ClonesFound, "stale clones, deleted", result.ClonesDeleted, "failed", result.ClonesFailed)
	} else {
		log.Msg("no stale clones found")
	}

	return nil
}

func runCleanup(args []string) error {
	cleanupFlags := flag.NewFlagSet("cleanup", flag.ExitOnError)
	configPath := cleanupFlags.String("config", "", "path to configuration file (required)")
	dryRun := cleanupFlags.Bool("dry-run", false, "show what would be deleted without deleting")
	stateDir := cleanupFlags.String("state-dir", "", "directory for state file (default: ./meta/)")
	maxAge := cleanupFlags.String("max-age", "", "override max age for stale clone detection (e.g., 24h, 48h)")

	cleanupFlags.Usage = func() {
		fmt.Fprintf(os.Stderr, `rds-refresh cleanup - Clean up stale RDS/Aurora clones

USAGE
    rds-refresh cleanup -config <path> [options]

OPTIONS
    -config <path>   config file (required)
    -dry-run         show what would be deleted without deleting
    -state-dir       directory for state file (default: ./meta/)
    -max-age         override max age for stale clone detection (e.g., 24h, 48h)

DESCRIPTION
    Searches for and deletes stale RDS/Aurora clones created by rds-refresh.

    Clones are identified by:
    - Name prefix: dblab-refresh-*
    - Tags: ManagedBy=dblab-rds-refresh, AutoDelete=true

    Clones are considered stale if:
    - DeleteAfter tag timestamp has passed, OR
    - Clone age exceeds max-age (default: 48h)

EXAMPLES
    # dry run - see what would be deleted
    rds-refresh cleanup -config config.yaml -dry-run

    # delete stale clones older than 24 hours
    rds-refresh cleanup -config config.yaml -max-age 24h
`)
	}

	if err := cleanupFlags.Parse(args); err != nil {
		return err
	}

	if *configPath == "" {
		cleanupFlags.Usage()
		return fmt.Errorf("-config flag is required")
	}

	cfg, err := rdsrefresh.LoadConfig(*configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// override max age if specified
	if *maxAge != "" {
		duration, err := time.ParseDuration(*maxAge)
		if err != nil {
			return fmt.Errorf("invalid max-age: %w", err)
		}

		cfg.RDSClone.MaxAge = rdsrefresh.Duration(duration)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// handle interrupt signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Msg("\nreceived interrupt signal, cancelling...")
		cancel()
	}()

	rdsClient, err := rdsrefresh.NewRDSClient(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create RDS client: %w", err)
	}

	stateFile := rdsrefresh.NewStateFile(*stateDir)

	// clean up from state file first
	if stateFile.Exists() && !*dryRun {
		log.Msg("checking state file for orphaned clone...")

		if err := rdsClient.CleanupFromStateFile(ctx, stateFile); err != nil {
			log.Warn("state file cleanup failed:", err)
		}
	}

	// scan and clean up stale clones
	log.Msg("scanning for stale clones (max-age:", cfg.RDSClone.MaxAge.Duration(), ")...")

	result, err := rdsClient.CleanupStaleClones(ctx, *dryRun)
	if err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	log.Msg("=== Cleanup Summary ===")
	log.Msg("Clones found: ", result.ClonesFound)

	if *dryRun {
		log.Msg("Dry run:       no changes made")
	} else {
		log.Msg("Clones deleted:", result.ClonesDeleted)
		log.Msg("Clones failed: ", result.ClonesFailed)
	}

	if len(result.Errors) > 0 {
		return fmt.Errorf("cleanup completed with %d errors", len(result.Errors))
	}

	return nil
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `rds-refresh - Perform full refresh from RDS/Aurora snapshots (logical mode)

Avoids pg_dump on production (which holds xmin → bloat). Instead, creates a
temporary RDS clone from RDS snapshot, refreshes DBLab from it, then deletes it.

USAGE
    rds-refresh -config <path> [options]
    rds-refresh cleanup -config <path> [options]

COMMANDS
    (default)   run the full refresh workflow
    cleanup     clean up stale/orphaned RDS clones

OPTIONS
    -config <path>   config file (required)
    -dry-run         validate only, no changes
    -state-dir       directory for state file (default: ./meta/)
    -skip-cleanup    skip startup cleanup of orphaned clones
    -version         show version
    -help            show help

CLEANUP
    The tool has multiple layers of protection against orphaned clones:
    1. Defer cleanup on normal exit
    2. Signal handlers (SIGINT, SIGTERM, SIGHUP) for graceful shutdown
    3. State file tracking for recovery after crashes (stored in ./meta/)
    4. AWS tag scan for finding stale clones

    Run 'rds-refresh cleanup -help' for cleanup-specific options.

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
      maxAge: 48h                # max age before clone is considered stale
    dblab:
      apiEndpoint: https://dblab:2345
      token: ${DBLAB_TOKEN}
    aws:
      region: us-east-1

DOCKER
    docker run --rm -v $PWD/config.yaml:/config.yaml \
      -e DB_PASSWORD -e DBLAB_TOKEN -e AWS_ACCESS_KEY_ID -e AWS_SECRET_ACCESS_KEY \
      postgresai/rds-refresh -config /config.yaml

TIP: Run inside screen/tmux to prevent SSH disconnections from orphaning clones.

More info: https://postgres.ai/docs/database-lab-engine
`)
}
