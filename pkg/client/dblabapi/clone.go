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
	"net/url"
	"time"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/pkg/models"
	"gitlab.com/postgres-ai/database-lab/pkg/observer"
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

// CreateClone creates a new Database Lab clone.
func (c *Client) CreateClone(ctx context.Context, cloneRequest types.CloneCreateRequest) (*models.Clone, error) {
	u := c.URL("/clone")

	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(cloneRequest); err != nil {
		return nil, errors.Wrap(err, "failed to encode CloneCreateRequest")
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

	if clone.Status.Code == "" {
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

	if clone.Status.Code != models.StatusOK {
		return nil, errors.Errorf("failed to create clone, unexpected status given: %v", clone.Status.Code)
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

			if clone.Status.Code != initialStatusCode {
				return clone, nil
			}

			pollingTimer.Reset(c.pollingInterval)

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// CreateCloneAsync asynchronously creates a new Database Lab clone.
func (c *Client) CreateCloneAsync(ctx context.Context, cloneRequest types.CloneCreateRequest) (*models.Clone, error) {
	u := c.URL("/clone")

	var clone models.Clone

	err := c.request(ctx, u, cloneRequest, &clone)

	return &clone, err
}

// UpdateClone updates an existing Database Lab clone.
func (c *Client) UpdateClone(ctx context.Context, cloneID string, updateRequest types.CloneUpdateRequest) (*models.Clone, error) {
	u := c.URL(fmt.Sprintf("/clone/%s", cloneID))

	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(updateRequest); err != nil {
		return nil, errors.Wrap(err, "failed to encode CloneUpdateRequest")
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

	if clone.Status.Code == models.StatusOK {
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
		if err, ok := errors.Cause(err).(models.Error); ok && err.Code == models.ErrCodeNotFound {
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

// StartObservation starts a clone observation.
func (c *Client) StartObservation(ctx context.Context, startRequest types.StartObservationRequest) (*observer.Session, error) {
	u := c.URL("/observation/start")

	var session observer.Session

	err := c.request(ctx, u, startRequest, &session)

	return &session, err
}

// StopObservation stops a clone observation.
func (c *Client) StopObservation(ctx context.Context, stopRequest types.StopObservationRequest) (*models.ObservationResult, error) {
	u := c.URL("/observation/stop")

	var observationResult models.ObservationResult

	err := c.request(ctx, u, stopRequest, &observationResult)

	return &observationResult, err
}

func (c *Client) request(ctx context.Context, u *url.URL, requestObject, responseObject interface{}) error {
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(requestObject); err != nil {
		return errors.Wrap(err, "failed to encode a request object")
	}

	request, err := http.NewRequest(http.MethodPost, u.String(), body)
	if err != nil {
		return errors.Wrap(err, "failed to make a request")
	}

	response, err := c.Do(ctx, request)
	if err != nil {
		return errors.Wrap(err, "failed to get response")
	}

	defer func() { _ = response.Body.Close() }()

	if err := json.NewDecoder(response.Body).Decode(&responseObject); err != nil {
		return errors.Wrap(err, "failed to decode a response body")
	}

	return nil
}
