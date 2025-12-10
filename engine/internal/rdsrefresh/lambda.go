/*
2025 © PostgresAI
*/

package rdsrefresh

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

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

// LambdaLogger implements Logger for Lambda/CloudWatch.
type LambdaLogger struct{}

// Info logs an info message.
func (l *LambdaLogger) Info(msg string, args ...interface{}) {
	fmt.Printf("[INFO] "+msg+"\n", args...)
}

// Error logs an error message.
func (l *LambdaLogger) Error(msg string, args ...interface{}) {
	fmt.Printf("[ERROR] "+msg+"\n", args...)
}

// Debug logs a debug message.
func (l *LambdaLogger) Debug(msg string, args ...interface{}) {
	fmt.Printf("[DEBUG] "+msg+"\n", args...)
}

// HandleLambda is the Lambda function handler.
func HandleLambda(ctx context.Context, event LambdaEvent) (LambdaResponse, error) {
	logger := &LambdaLogger{}

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
