/*
2024 © Postgres.ai
*/

package main

import (
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
	Mode        string          `json:"mode"`
	Status      RetrievalStatus `json:"status"`
	LastRefresh string          `json:"lastRefresh"`
	NextRefresh string          `json:"nextRefresh"`
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
			case StatusFinished:
				return nil
			case StatusFailed:
				if len(status.Retrieving.Alerts) > 0 {
					for _, alert := range status.Retrieving.Alerts {
						return fmt.Errorf("refresh failed: %s", alert.Message)
					}
				}

				return fmt.Errorf("refresh failed (no details available)")
			case StatusRefreshing, StatusSnapshotting, StatusRenewed:
				// still in progress
				continue
			case StatusInactive, StatusPending:
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
	case StatusRefreshing, StatusSnapshotting:
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

		var errModel APIError
		if err := json.Unmarshal(bodyBytes, &errModel); err == nil && errModel.Message != "" {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, errModel.Message)
		}

		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}
