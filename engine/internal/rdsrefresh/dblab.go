/*
2024 © Postgres.ai
*/

package rdsrefresh

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

const (
	verificationHeader = "Verification-Token"
	contentTypeJSON    = "application/json"
)

// DBLabClient provides methods to interact with the DBLab Engine API.
type DBLabClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewDBLabClient creates a new DBLab API client.
func NewDBLabClient(cfg *DBLabConfig) *DBLabClient {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.Insecure},
	}

	return &DBLabClient{
		baseURL: cfg.APIEndpoint,
		token:   cfg.Token,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   60 * time.Second,
		},
	}
}

// GetStatus returns the current DBLab Engine instance status.
func (c *DBLabClient) GetStatus(ctx context.Context) (*models.InstanceStatus, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/status", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var status models.InstanceStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode status response: %w", err)
	}

	return &status, nil
}

// TriggerFullRefresh triggers a full data refresh on the DBLab Engine.
func (c *DBLabClient) TriggerFullRefresh(ctx context.Context) error {
	resp, err := c.doRequest(ctx, http.MethodPost, "/full-refresh", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result models.Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status != "OK" {
		return fmt.Errorf("full refresh failed: %s", result.Message)
	}

	return nil
}

// UpdateConfig updates the DBLab Engine configuration.
func (c *DBLabClient) UpdateConfig(ctx context.Context, configPatch map[string]interface{}) error {
	body, err := json.Marshal(configPatch)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodPatch, "/config", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// WaitForRefreshComplete polls the DBLab status until refresh is complete or timeout.
func (c *DBLabClient) WaitForRefreshComplete(ctx context.Context, pollInterval, timeout time.Duration) error {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeoutTimer.C:
			return fmt.Errorf("timeout waiting for refresh to complete after %v", timeout)
		case <-ticker.C:
			status, err := c.GetStatus(ctx)
			if err != nil {
				return fmt.Errorf("failed to get status: %w", err)
			}

			retrievalStatus := status.Retrieving.Status

			switch retrievalStatus {
			case models.Finished:
				return nil
			case models.Failed:
				if len(status.Retrieving.Alerts) > 0 {
					for _, alert := range status.Retrieving.Alerts {
						return fmt.Errorf("refresh failed: %s", alert.Message)
					}
				}

				return fmt.Errorf("refresh failed (no details available)")
			case models.Refreshing, models.Snapshotting, models.Renewed:
				// still in progress
				continue
			case models.Inactive, models.Pending:
				// not started yet or pending
				continue
			default:
				continue
			}
		}
	}
}

// IsRefreshInProgress checks if a refresh is currently in progress.
func (c *DBLabClient) IsRefreshInProgress(ctx context.Context) (bool, error) {
	status, err := c.GetStatus(ctx)
	if err != nil {
		return false, err
	}

	switch status.Retrieving.Status {
	case models.Refreshing, models.Snapshotting:
		return true, nil
	default:
		return false, nil
	}
}

// Health checks if the DBLab Engine is healthy.
func (c *DBLabClient) Health(ctx context.Context) error {
	resp, err := c.doRequest(ctx, http.MethodGet, "/healthz", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *DBLabClient) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set(verificationHeader, c.token)

	if body != nil {
		req.Header.Set("Content-Type", contentTypeJSON)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		defer resp.Body.Close()

		bodyBytes, _ := io.ReadAll(resp.Body)

		var errModel models.Error
		if err := json.Unmarshal(bodyBytes, &errModel); err == nil && errModel.Message != "" {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, errModel.Message)
		}

		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}
