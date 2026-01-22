/*
2025 © PostgresAI
*/

package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	verificationHeader = "Verification-Token"
	contentTypeJSON    = "application/json"
)

// RetrievalStatus defines status of refreshing data.
type RetrievalStatus string

const (
	StatusInactive     RetrievalStatus = "inactive"
	StatusPending      RetrievalStatus = "pending"
	StatusFailed       RetrievalStatus = "failed"
	StatusRefreshing   RetrievalStatus = "refreshing"
	StatusRenewed      RetrievalStatus = "renewed"
	StatusSnapshotting RetrievalStatus = "snapshotting"
	StatusFinished     RetrievalStatus = "finished"
)

// InstanceStatus represents the DBLab Engine status response.
type InstanceStatus struct {
	Status     *Status    `json:"status"`
	Retrieving Retrieving `json:"retrieving"`
}

// Status represents a generic status.
type Status struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Retrieving represents state of retrieval subsystem.
type Retrieving struct {
	Mode        string           `json:"mode"`
	Status      RetrievalStatus  `json:"status"`
	LastRefresh string           `json:"lastRefresh"`
	NextRefresh string           `json:"nextRefresh"`
	Alerts      map[string]Alert `json:"alerts"`
}

// Alert describes an alert.
type Alert struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

// APIResponse represents a generic API response.
type APIResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// APIError represents an API error response.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ConfigUpdateRequest represents a request to update DBLab config.
// Uses flat structure matching DBLab's ConfigProjection fields.
type ConfigUpdateRequest struct {
	Host     *string `json:"host,omitempty"`
	Port     *int64  `json:"port,omitempty"`
	DBName   *string `json:"dbname,omitempty"`
	Username *string `json:"username,omitempty"`
	Password *string `json:"password,omitempty"`
}

// DBLabClient provides methods to interact with the DBLab Engine API.
type DBLabClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewDBLabClient creates a new DBLab API client.
func NewDBLabClient(cfg *DBLabConfig, logger Logger) *DBLabClient {
	if cfg.Insecure && logger != nil {
		logger.Error("WARNING: TLS certificate verification is disabled. This is insecure for production use.")
	}

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
func (c *DBLabClient) GetStatus(ctx context.Context) (*InstanceStatus, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/status", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var status InstanceStatus
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

	var result APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status != "OK" {
		return fmt.Errorf("full refresh failed: %s", result.Message)
	}

	return nil
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
		case StatusRefreshing, StatusSnapshotting, StatusRenewed, StatusPending:
			refreshStarted = true
			return false, nil
		case StatusFinished:
			if !refreshStarted {
				return false, nil
			}
			return true, nil
		case StatusFailed:
			if !refreshStarted {
				return false, nil
			}
			if len(status.Retrieving.Alerts) > 0 {
				for _, alert := range status.Retrieving.Alerts {
					return false, fmt.Errorf("refresh failed: %s", alert.Message)
				}
			}
			return false, fmt.Errorf("refresh failed (no details available)")
		case StatusInactive:
			if refreshStarted {
				return false, fmt.Errorf("refresh stopped unexpectedly (status: inactive)")
			}
			return false, nil
		default:
			return false, nil
		}
	}

	// immediate first check
	if done, err := checkStatus(); err != nil {
		return err
	} else if done {
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
			if done, err := checkStatus(); err != nil {
				return err
			} else if done {
				return nil
			}
		}
	}
}

// IsRefreshInProgress checks if a refresh is currently in progress.
// Considers all active states: refreshing, snapshotting, pending, renewed.
func (c *DBLabClient) IsRefreshInProgress(ctx context.Context) (bool, error) {
	status, err := c.GetStatus(ctx)
	if err != nil {
		return false, err
	}

	switch status.Retrieving.Status {
	case StatusRefreshing, StatusSnapshotting, StatusPending, StatusRenewed:
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

// UpdateSourceConfig updates the source database connection in DBLab config.
// DBLab automatically reloads the configuration after the update.
func (c *DBLabClient) UpdateSourceConfig(ctx context.Context, host string, port int, dbname, username, password string) error {
	port64 := int64(port)
	updateReq := ConfigUpdateRequest{
		Host:     &host,
		Port:     &port64,
		DBName:   &dbname,
		Username: &username,
		Password: &password,
	}

	bodyBytes, err := json.Marshal(updateReq)
	if err != nil {
		return fmt.Errorf("failed to marshal config update: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodPut, "/admin/config", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to update DBLab config: %w", err)
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

		var errModel APIError
		if err := json.Unmarshal(bodyBytes, &errModel); err == nil && errModel.Message != "" {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, errModel.Message)
		}

		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}
