/*
2025 © PostgresAI
*/

package rdsrefresh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/projection"
)

const (
	defaultRequestTimeout = 60 * time.Second
)

// DBLabClient wraps the dblabapi client with additional methods for config management.
type DBLabClient struct {
	client *dblabapi.Client
}

// NewDBLabClient creates a new DBLab API client wrapper.
func NewDBLabClient(cfg *DBLabConfig) (*DBLabClient, error) {
	if cfg.Insecure {
		log.Warn("TLS certificate verification is disabled. This is insecure for production use.")
	}

	client, err := dblabapi.NewClient(dblabapi.Options{
		Host:              cfg.APIEndpoint,
		VerificationToken: cfg.Token,
		Insecure:          cfg.Insecure,
		RequestTimeout:    defaultRequestTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create DBLab API client: %w", err)
	}

	return &DBLabClient{
		client: client,
	}, nil
}

// GetStatus returns the current DBLab Engine instance status.
func (c *DBLabClient) GetStatus(ctx context.Context) (*models.InstanceStatus, error) {
	return c.client.Status(ctx)
}

// Health checks if the DBLab Engine is healthy.
func (c *DBLabClient) Health(ctx context.Context) error {
	_, err := c.client.Health(ctx)
	return err
}

// TriggerFullRefresh triggers a full data refresh on the DBLab Engine.
func (c *DBLabClient) TriggerFullRefresh(ctx context.Context) error {
	_, err := c.client.FullRefresh(ctx)
	return err
}

// IsRefreshInProgress checks if a refresh is currently in progress.
func (c *DBLabClient) IsRefreshInProgress(ctx context.Context) (bool, error) {
	status, err := c.GetStatus(ctx)
	if err != nil {
		return false, err
	}

	switch status.Retrieving.Status {
	case models.Refreshing, models.Snapshotting, models.Pending, models.Renewed:
		return true, nil
	default:
		return false, nil
	}
}

// WaitForRefreshComplete polls the DBLab status until refresh is complete or timeout.
// It first waits for the refresh to start (status changes from finished/inactive),
// then waits for it to complete. This prevents race conditions where stale status
// from a previous refresh could cause premature return.
func (c *DBLabClient) WaitForRefreshComplete(ctx context.Context, pollInterval, timeout time.Duration) error {
	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()

	refreshStarted := false

	// checkStatus handles status evaluation and returns (done, error)
	checkStatus := func() (bool, error) {
		status, err := c.GetStatus(ctx)
		if err != nil {
			return false, fmt.Errorf("failed to get status: %w", err)
		}

		switch status.Retrieving.Status {
		case models.Refreshing, models.Snapshotting, models.Renewed, models.Pending:
			refreshStarted = true

			return false, nil
		case models.Finished:
			if !refreshStarted {
				return false, nil
			}

			return true, nil
		case models.Failed:
			if !refreshStarted {
				return false, nil
			}

			if alert, ok := status.Retrieving.Alerts[models.RefreshFailed]; ok {
				return false, fmt.Errorf("refresh failed: %s", alert.Message)
			}

			// fallback to any available alert if RefreshFailed not present
			for _, alert := range status.Retrieving.Alerts {
				return false, fmt.Errorf("refresh failed: %s", alert.Message)
			}

			return false, fmt.Errorf("refresh failed (no details available)")
		case models.Inactive:
			if refreshStarted {
				return false, fmt.Errorf("refresh stopped unexpectedly (status: inactive)")
			}

			return false, nil
		default:
			return false, nil
		}
	}

	// immediate first check
	done, err := checkStatus()
	if err != nil {
		return err
	}

	if done {
		return nil
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeoutTimer.C:
			if !refreshStarted {
				return fmt.Errorf("timeout waiting for refresh to start after %v", timeout)
			}

			return fmt.Errorf("timeout waiting for refresh to complete after %v", timeout)
		case <-ticker.C:
			done, err := checkStatus()
			if err != nil {
				return err
			}

			if done {
				return nil
			}
		}
	}
}

// SourceConfigUpdate contains source database connection parameters for config update.
type SourceConfigUpdate struct {
	Host     string
	Port     int
	DBName   string
	Username string
	Password string
	// RDSIAMDBInstance is the RDS DB instance identifier for IAM auth. When empty, this field is omitted from the config update.
	RDSIAMDBInstance string
}

// UpdateSourceConfig updates the source database connection in DBLab config.
// DBLab automatically reloads the configuration after the update.
func (c *DBLabClient) UpdateSourceConfig(ctx context.Context, update SourceConfigUpdate) error {
	port64 := int64(update.Port)
	proj := models.ConfigProjection{
		Host:     &update.Host,
		Port:     &port64,
		DBName:   &update.DBName,
		Username: &update.Username,
		Password: &update.Password,
	}

	if update.RDSIAMDBInstance != "" {
		proj.RDSIAMDBInstance = &update.RDSIAMDBInstance
	}

	nested := map[string]interface{}{}

	// defensive error check: StoreJSON only fails if target is not an addressable struct,
	// which cannot happen here since proj is always a valid ConfigProjection value.
	if err := projection.StoreJSON(&proj, nested, projection.StoreOptions{
		Groups: []string{"default", "sensitive"},
	}); err != nil {
		return fmt.Errorf("failed to build config projection: %w", err)
	}

	bodyBytes, err := json.Marshal(nested)
	if err != nil {
		return fmt.Errorf("failed to marshal config update: %w", err)
	}

	url := c.client.URL("/admin/config")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url.String(), bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to update DBLab config: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("failed to update DBLab config: HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
