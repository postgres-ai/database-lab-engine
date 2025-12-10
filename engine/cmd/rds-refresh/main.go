/*
2025 © PostgresAI
*/

// Package main provides the entry point for the rds-refresh CLI tool.
// This tool automates DBLab full refresh using temporary RDS/Aurora clones.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-lambda-go/lambda"

	"gitlab.com/postgres-ai/database-lab/v3/internal/rdsrefresh"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	// Check if running in Lambda
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		lambda.Start(rdsrefresh.HandleLambda)
		return
	}

	// CLI mode
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
	cfg, err := rdsrefresh.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		fmt.Printf("\nReceived signal %v, initiating graceful shutdown...\n", sig)
		cancel()
	}()

	logger := &rdsrefresh.DefaultLogger{}

	refresher, err := rdsrefresh.NewRefresher(ctx, cfg, logger)
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

This tool creates a temporary RDS/Aurora clone from a snapshot, triggers
a DBLab Engine full refresh, and then cleans up the temporary clone.

USAGE:
    rds-refresh -config <path> [options]

OPTIONS:
    -config <path>    Path to YAML configuration file (required)
    -dry-run          Validate configuration without creating resources
    -version          Show version information
    -help             Show this help message

LAMBDA MODE:
    When running as an AWS Lambda function (detected via AWS_LAMBDA_FUNCTION_NAME
    environment variable), configuration is loaded from environment variables:

    Required:
        RDS_SOURCE_IDENTIFIER       Source RDS instance or Aurora cluster ID
        RDS_CLONE_INSTANCE_CLASS    Instance class for the clone (e.g., db.t3.medium)
        DBLAB_API_ENDPOINT          DBLab Engine API endpoint
        DBLAB_TOKEN                 DBLab verification token
        AWS_REGION                  AWS region

    Optional:
        RDS_SOURCE_TYPE             "rds" or "aurora-cluster" (default: rds)
        RDS_SNAPSHOT_IDENTIFIER     Specific snapshot ID (default: latest)
        RDS_CLONE_SUBNET_GROUP      DB subnet group name
        RDS_CLONE_SECURITY_GROUPS   JSON array of security group IDs
        RDS_CLONE_PUBLIC            "true" to make clone publicly accessible
        RDS_CLONE_PARAMETER_GROUP   DB parameter group name
        RDS_CLONE_ENABLE_IAM_AUTH   "true" to enable IAM authentication
        RDS_CLONE_STORAGE_TYPE      Storage type (gp2, gp3, io1, etc.)
        RDS_CLONE_TAGS              JSON object of additional tags
        DBLAB_INSECURE              "true" to skip TLS verification

EXAMPLE CONFIGURATION:

    source:
      type: rds
      identifier: production-db

    clone:
      instanceClass: db.t3.medium
      subnetGroup: default-vpc-subnet
      securityGroups:
        - sg-12345678
      publiclyAccessible: false
      enableIAMAuth: true

    dblab:
      apiEndpoint: https://dblab.example.com:2345
      token: ${DBLAB_TOKEN}
      pollInterval: 30s
      timeout: 4h

    aws:
      region: us-east-1

For more information, see:
    https://postgres.ai/docs/database-lab-engine

`)
}
