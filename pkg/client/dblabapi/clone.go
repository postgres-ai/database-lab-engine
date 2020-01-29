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
	"time"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/models"
)

// ListClones provides a list of Database Lab clones.
func (c *Client) ListClones(ctx context.Context) ([]*models.Clone, error) {
	u := c.URL("/status")

	request, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make a request")
	}

	response, err := c.Do(ctx, request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get response")
	}

	defer func() { _ = response.Body.Close() }()

	var instanceStatus models.InstanceStatus

	if err := json.NewDecoder(response.Body).Decode(&instanceStatus); err != nil {
		return nil, errors.Wrap(err, "failed to decode a response body")
	}

	return instanceStatus.Clones, nil
}

// GetClone returns info about a Database Lab clone.
func (c *Client) GetClone(ctx context.Context, cloneID string) (*models.Clone, error) {
	u := c.URL(fmt.Sprintf("/clone/%s", cloneID))

	request, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make a request")
	}

	response, err := c.Do(ctx, request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get response")
	}

	defer func() { _ = response.Body.Close() }()

	var clone models.Clone

	if err := json.NewDecoder(response.Body).Decode(&clone); err != nil {
		return nil, errors.Wrap(err, "failed to decode a response body")
	}

	return &clone, nil
}

// CreateRequest represents clone params of a create request.
type CreateRequest struct {
	Name      string           `json:"name"`
	Project   string           `json:"project"`
	Protected bool             `json:"protected"`
	DB        *DatabaseRequest `json:"db"`
}

// DatabaseRequest represents database params of a clone request.
type DatabaseRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// CreateClone creates a new Database Lab clone.
func (c *Client) CreateClone(ctx context.Context, cloneRequest CreateRequest) (*models.Clone, error) {
	u := c.URL("/clone")

	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(cloneRequest); err != nil {
		return nil, errors.Wrap(err, "failed to encode CreateRequest")
	}

	request, err := http.NewRequest(http.MethodPost, u.String(), body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make a request")
	}

	response, err := c.Do(ctx, request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get response")
	}

	defer func() { _ = response.Body.Close() }()

	clone := &models.Clone{}

	if err := json.NewDecoder(response.Body).Decode(clone); err != nil {
		return nil, errors.Wrap(err, "failed to decode a response body")
	}

	if clone.Status == nil {
		return nil, errors.New("empty clone status given")
	}

	if clone.Status.Code == models.StatusOK {
		return clone, nil
	}

	if clone.Status.Code != models.StatusCreating {
		return nil, errors.Errorf("unexpected clone status given: %v", clone.Status)
	}

	clone, err = c.watchCloneStatus(ctx, clone.ID, clone.Status.Code)
	if err != nil {
		return nil, errors.Wrap(err, "failed to watch the clone status")
	}

	return clone, nil
}

// watchCloneStatus checks the clone status for changing.
func (c *Client) watchCloneStatus(ctx context.Context, cloneID string, initialStatusCode models.StatusCode) (*models.Clone, error) {
	pollingTimer := time.NewTimer(c.pollingInterval)
	defer pollingTimer.Stop()

	var cancel context.CancelFunc

	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithTimeout(ctx, defaultPollingTimeout)
		defer cancel()
	}

	for {
		select {
		case <-pollingTimer.C:
			clone, err := c.GetClone(ctx, cloneID)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get clone info")
			}

			if clone.Status != nil && clone.Status.Code != initialStatusCode {
				return clone, nil
			}

			pollingTimer.Reset(c.pollingInterval)

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// CreateCloneAsync asynchronously creates a new Database Lab clone.
func (c *Client) CreateCloneAsync(ctx context.Context, cloneRequest CreateRequest) (*models.Clone, error) {
	u := c.URL("/clone")

	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(cloneRequest); err != nil {
		return nil, errors.Wrap(err, "failed to encode CreateRequest")
	}

	request, err := http.NewRequest(http.MethodPost, u.String(), body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make a request")
	}

	response, err := c.Do(ctx, request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get response")
	}

	defer func() { _ = response.Body.Close() }()

	var clone models.Clone

	if err := json.NewDecoder(response.Body).Decode(&clone); err != nil {
		return nil, errors.Wrap(err, "failed to decode a response body")
	}

	return &clone, nil
}

// UpdateRequest represents params of an update request.
type UpdateRequest struct {
	Name      string `json:"name"`
	Protected bool   `json:"protected"`
}

// UpdateClone updates an existing Database Lab clone.
func (c *Client) UpdateClone(ctx context.Context, cloneID string, updateRequest UpdateRequest) (*models.Clone, error) {
	u := c.URL(fmt.Sprintf("/clone/%s", cloneID))

	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(updateRequest); err != nil {
		return nil, errors.Wrap(err, "failed to encode UpdateRequest")
	}

	request, err := http.NewRequest(http.MethodPatch, u.String(), body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make a request")
	}

	response, err := c.Do(ctx, request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get response")
	}

	defer func() { _ = response.Body.Close() }()

	var clone models.Clone

	if err := json.NewDecoder(response.Body).Decode(&clone); err != nil {
		return nil, errors.Wrap(err, "failed to decode a response body")
	}

	return &clone, nil
}

// ResetClone resets a Database Lab clone session.
func (c *Client) ResetClone(ctx context.Context, cloneID string) error {
	u := c.URL(fmt.Sprintf("/clone/%s/reset", cloneID))

	request, err := http.NewRequest(http.MethodPost, u.String(), nil)
	if err != nil {
		return errors.Wrap(err, "failed to make a request")
	}

	response, err := c.Do(ctx, request)
	if err != nil {
		return errors.Wrap(err, "failed to get response")
	}

	defer func() { _ = response.Body.Close() }()

	clone, err := c.watchCloneStatus(ctx, cloneID, models.StatusResetting)
	if err != nil {
		return errors.Wrap(err, "failed to watch the clone status")
	}

	if clone.Status != nil && clone.Status.Code == models.StatusOK {
		return nil
	}

	return errors.Errorf("unexpected clone status given: %v", clone.Status)
}

// ResetCloneAsync asynchronously resets a Database Lab clone session.
func (c *Client) ResetCloneAsync(ctx context.Context, cloneID string) error {
	u := c.URL(fmt.Sprintf("/clone/%s/reset", cloneID))

	request, err := http.NewRequest(http.MethodPost, u.String(), nil)
	if err != nil {
		return errors.Wrap(err, "failed to make a request")
	}

	response, err := c.Do(ctx, request)
	if err != nil {
		return errors.Wrap(err, "failed to get response")
	}

	defer func() { _ = response.Body.Close() }()

	return nil
}

// DestroyClone destroys a Database Lab clone.
func (c *Client) DestroyClone(ctx context.Context, cloneID string) error {
	u := c.URL(fmt.Sprintf("/clone/%s", cloneID))

	request, err := http.NewRequest(http.MethodDelete, u.String(), nil)
	if err != nil {
		return errors.Wrap(err, "failed to make a request")
	}

	response, err := c.Do(ctx, request)
	if err != nil {
		return errors.Wrap(err, "failed to get response")
	}

	defer func() { _ = response.Body.Close() }()

	clone, err := c.watchCloneStatus(ctx, cloneID, models.StatusDeleting)
	if err != nil {
		if err, ok := errors.Cause(err).(models.Error); ok && err.Code == "NOT_FOUND" {
			return nil
		}

		return errors.Wrap(err, "failed to watch the clone status")
	}

	if clone != nil {
		return errors.Errorf("unexpected clone given: %v", clone)
	}

	return nil
}

// DestroyCloneAsync asynchronously destroys a Database Lab clone.
func (c *Client) DestroyCloneAsync(ctx context.Context, cloneID string) error {
	u := c.URL(fmt.Sprintf("/clone/%s", cloneID))

	request, err := http.NewRequest(http.MethodDelete, u.String(), nil)
	if err != nil {
		return errors.Wrap(err, "failed to make a request")
	}

	response, err := c.Do(ctx, request)
	if err != nil {
		return errors.Wrap(err, "failed to get response")
	}

	defer func() { _ = response.Body.Close() }()

	return nil
}
