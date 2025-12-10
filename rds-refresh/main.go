/*
2024 © Postgres.ai

rds-refresh - Automate DBLab full refresh using RDS/Aurora snapshots

This tool creates a temporary RDS/Aurora clone from a snapshot, triggers
a DBLab Engine full refresh, and then cleans up the temporary clone.
*/
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-lambda-go/lambda"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	// Check if running in Lambda
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		lambda.Start(HandleLambda)
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
	cfg, err := LoadConfig(configPath)
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

// LambdaEvent is the input event for the Lambda function.
type LambdaEvent struct {
	// DryRun, if true, only validates configuration without creating resources.
	DryRun bool `json:"dryRun"`
	// ConfigOverrides allows overriding configuration values.
	ConfigOverrides *ConfigOverrides `json:"configOverrides"`
}

// ConfigOverrides allows partial configuration overrides via the Lambda event.
type ConfigOverrides struct {
	SnapshotIdentifier string `json:"snapshotIdentifier"`
}

// LambdaResponse is the output response from the Lambda function.
type LambdaResponse struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	SnapshotID    string `json:"snapshotId,omitempty"`
	CloneID       string `json:"cloneId,omitempty"`
	CloneEndpoint string `json:"cloneEndpoint,omitempty"`
	DurationSec   int64  `json:"durationSeconds,omitempty"`
	Error         string `json:"error,omitempty"`
}

// HandleLambda is the Lambda function handler.
func HandleLambda(ctx context.Context, event LambdaEvent) (LambdaResponse, error) {
	logger := &DefaultLogger{}

	cfg, err := loadLambdaConfig()
	if err != nil {
		return LambdaResponse{
			Success: false,
			Error:   err.Error(),
			Message: "failed to load configuration",
		}, nil
	}

	// Apply overrides
	if event.ConfigOverrides != nil && event.ConfigOverrides.SnapshotIdentifier != "" {
		cfg.Source.SnapshotIdentifier = event.ConfigOverrides.SnapshotIdentifier
	}

	refresher, err := NewRefresher(ctx, cfg, logger)
	if err != nil {
		return LambdaResponse{
			Success: false,
			Error:   err.Error(),
			Message: "failed to initialize refresher",
		}, nil
	}

	if event.DryRun {
		if err := refresher.DryRun(ctx); err != nil {
			return LambdaResponse{
				Success: false,
				Error:   err.Error(),
				Message: "dry run failed",
			}, nil
		}

		return LambdaResponse{
			Success: true,
			Message: "dry run completed successfully",
		}, nil
	}

	result := refresher.Run(ctx)

	resp := LambdaResponse{
		Success:       result.Success,
		SnapshotID:    result.SnapshotID,
		CloneID:       result.CloneID,
		CloneEndpoint: result.CloneEndpoint,
		DurationSec:   int64(result.Duration.Seconds()),
	}

	if result.Error != nil {
		resp.Error = result.Error.Error()
		resp.Message = "refresh failed"
	} else {
		resp.Message = "refresh completed successfully"
	}

	return resp, nil
}

// loadLambdaConfig loads configuration from environment variables.
func loadLambdaConfig() (*Config, error) {
	cfg := &Config{}

	// Source configuration
	cfg.Source.Type = getEnvOrDefault("RDS_SOURCE_TYPE", "rds")
	cfg.Source.Identifier = os.Getenv("RDS_SOURCE_IDENTIFIER")
	cfg.Source.SnapshotIdentifier = os.Getenv("RDS_SNAPSHOT_IDENTIFIER")

	// Clone configuration
	cfg.Clone.InstanceClass = os.Getenv("RDS_CLONE_INSTANCE_CLASS")
	cfg.Clone.DBSubnetGroupName = os.Getenv("RDS_CLONE_SUBNET_GROUP")

	if sgJSON := os.Getenv("RDS_CLONE_SECURITY_GROUPS"); sgJSON != "" {
		if err := json.Unmarshal([]byte(sgJSON), &cfg.Clone.VPCSecurityGroupIDs); err != nil {
			return nil, fmt.Errorf("invalid RDS_CLONE_SECURITY_GROUPS JSON: %w", err)
		}
	}

	cfg.Clone.PubliclyAccessible = os.Getenv("RDS_CLONE_PUBLIC") == "true"
	cfg.Clone.ParameterGroupName = os.Getenv("RDS_CLONE_PARAMETER_GROUP")
	cfg.Clone.OptionGroupName = os.Getenv("RDS_CLONE_OPTION_GROUP")
	cfg.Clone.DBClusterParameterGroupName = os.Getenv("RDS_CLONE_CLUSTER_PARAMETER_GROUP")
	cfg.Clone.EnableIAMAuth = os.Getenv("RDS_CLONE_ENABLE_IAM_AUTH") == "true"
	cfg.Clone.StorageType = os.Getenv("RDS_CLONE_STORAGE_TYPE")

	// Parse tags from JSON
	if tagsJSON := os.Getenv("RDS_CLONE_TAGS"); tagsJSON != "" {
		if err := json.Unmarshal([]byte(tagsJSON), &cfg.Clone.Tags); err != nil {
			return nil, fmt.Errorf("invalid RDS_CLONE_TAGS JSON: %w", err)
		}
	}

	// DBLab configuration
	cfg.DBLab.APIEndpoint = os.Getenv("DBLAB_API_ENDPOINT")
	cfg.DBLab.Token = os.Getenv("DBLAB_TOKEN")
	cfg.DBLab.Insecure = os.Getenv("DBLAB_INSECURE") == "true"

	// AWS configuration
	cfg.AWS.Region = os.Getenv("AWS_REGION")

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	cfg.SetDefaults()

	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return defaultValue
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
