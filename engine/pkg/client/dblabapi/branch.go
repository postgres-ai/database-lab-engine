/*
2019 © Postgres.ai
*/

package dblabapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

// ListBranches returns branches list.
func (c *Client) ListBranches(ctx context.Context) ([]string, error) {
	u := c.URL("/branch/list")

	request, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to make a request: %w", err)
	}

	response, err := c.Do(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to get response: %w", err)
	}

	defer func() { _ = response.Body.Close() }()

	branches := make([]models.BranchView, 0)

	if err := json.NewDecoder(response.Body).Decode(&branches); err != nil {
		return nil, fmt.Errorf("failed to get response: %w", err)
	}

	listBranches := make([]string, 0, len(branches))

	for _, branchView := range branches {
		listBranches = append(listBranches, branchView.Name)
	}

	sort.Strings(listBranches)

	return listBranches, nil
}

// CreateBranch creates a new DLE data branch.
//
//nolint:dupl
func (c *Client) CreateBranch(ctx context.Context, branchRequest types.BranchCreateRequest) (*models.Branch, error) {
	u := c.URL("/branch/create")

	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(branchRequest); err != nil {
		return nil, fmt.Errorf("failed to encode BranchCreateRequest: %w", err)
	}

	request, err := http.NewRequest(http.MethodPost, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("failed to make a request: %w", err)
	}

	response, err := c.Do(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to get response: %w", err)
	}

	defer func() { _ = response.Body.Close() }()

	var branch *models.Branch

	if err := json.NewDecoder(response.Body).Decode(&branch); err != nil {
		return nil, fmt.Errorf("failed to get response: %w", err)
	}

	return branch, nil
}

// CreateSnapshotForBranch creates a new snapshot for branch.
//
//nolint:dupl
func (c *Client) CreateSnapshotForBranch(
	ctx context.Context,
	snapshotRequest types.SnapshotCloneCreateRequest) (*types.SnapshotResponse, error) {
	u := c.URL("/branch/snapshot")

	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(snapshotRequest); err != nil {
		return nil, fmt.Errorf("failed to encode SnapshotCreateRequest: %w", err)
	}

	request, err := http.NewRequest(http.MethodPost, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("failed to make a request: %w", err)
	}

	response, err := c.Do(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to get response: %w", err)
	}

	defer func() { _ = response.Body.Close() }()

	var snapshot *types.SnapshotResponse

	if err := json.NewDecoder(response.Body).Decode(&snapshot); err != nil {
		return nil, fmt.Errorf("failed to get response: %w", err)
	}

	return snapshot, nil
}

// BranchLog provides snapshot list for branch.
func (c *Client) BranchLog(ctx context.Context, logRequest types.LogRequest) ([]models.SnapshotDetails, error) {
	u := c.URL("/branch/log")

	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(logRequest); err != nil {
		return nil, fmt.Errorf("failed to encode LogRequest: %w", err)
	}

	request, err := http.NewRequest(http.MethodPost, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("failed to make a request: %w", err)
	}

	response, err := c.Do(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to get response: %w", err)
	}

	defer func() { _ = response.Body.Close() }()

	var snapshots []models.SnapshotDetails

	if err := json.NewDecoder(response.Body).Decode(&snapshots); err != nil {
		return nil, fmt.Errorf("failed to get response: %w", err)
	}

	return snapshots, nil
}

// DeleteBranch deletes data branch.
//nolint:dupl
func (c *Client) DeleteBranch(ctx context.Context, r types.BranchDeleteRequest) error {
	u := c.URL("/branch/delete")

	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(r); err != nil {
		return fmt.Errorf("failed to encode BranchDeleteRequest: %w", err)
	}

	request, err := http.NewRequest(http.MethodPost, u.String(), body)
	if err != nil {
		return fmt.Errorf("failed to make a request: %w", err)
	}

	response, err := c.Do(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to get response: %w", err)
	}

	defer func() { _ = response.Body.Close() }()

	return nil
}
