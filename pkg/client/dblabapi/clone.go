/*
2019 © Postgres.ai
*/

package dblabapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/observer"
)

// ListClones provides a list of Database Lab clones.
func (c *Client) ListClones(ctx context.Context) ([]*models.Clone, error) {
	body, err := c.ListClonesRaw(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get response")
	}

	defer func() { _ = body.Close() }()

	var instanceStatus models.CloneList

	if err := json.NewDecoder(body).Decode(&instanceStatus); err != nil {
		return nil, errors.Wrap(err, "failed to decode a response body")
	}

	return instanceStatus.Cloning.Clones, nil
}

// ListClonesRaw provides a raw list of Database Lab clones.
func (c *Client) ListClonesRaw(ctx context.Context) (io.ReadCloser, error) {
	u := c.URL("/status")

	request, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make a request")
	}

	response, err := c.Do(ctx, request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get response")
	}

	return response.Body, nil
}

// GetClone returns info about a Database Lab clone.
func (c *Client) GetClone(ctx context.Context, cloneID string) (*models.Clone, error) {
	body, err := c.GetCloneRaw(ctx, cloneID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get response")
	}

	defer func() { _ = body.Close() }()

	var clone models.Clone

	if err := json.NewDecoder(body).Decode(&clone); err != nil {
		return nil, errors.Wrap(err, "failed to decode a response body")
	}

	return &clone, nil
}

// GetCloneRaw returns raw info about a Database Lab clone.
func (c *Client) GetCloneRaw(ctx context.Context, cloneID string) (io.ReadCloser, error) {
	u := c.URL(fmt.Sprintf("/clone/%s", cloneID))

	request, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make a request")
	}

	response, err := c.Do(ctx, request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get response")
	}

	return response.Body, nil
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
		return nil, errors.Errorf("failed to create clone, unexpected status given. %v: %s", clone.Status.Code, clone.Status.Message)
	}

	return clone, nil
}

// watchCloneStatus checks the clone status for changing.
func (c *Client) watchCloneStatus(ctx context.Context, cloneID string, initialStatusCode models.StatusCode) (*models.Clone, error) {
	pollingTimer := time.NewTimer(c.pollingInterval)
	defer pollingTimer.Stop()

	var cancel context.CancelFunc

	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithTimeout(ctx, c.requestTimeout)
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
func (c *Client) ResetClone(ctx context.Context, cloneID string, params types.ResetCloneRequest) error {
	u := c.URL(fmt.Sprintf("/clone/%s/reset", cloneID))

	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(params); err != nil {
		return errors.Wrap(err, "failed to encode ResetClone parameters to JSON")
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
func (c *Client) ResetCloneAsync(ctx context.Context, cloneID string, params types.ResetCloneRequest) error {
	u := c.URL(fmt.Sprintf("/clone/%s/reset", cloneID))

	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(params); err != nil {
		return errors.Wrap(err, "failed to encode ResetClone parameters to JSON")
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

// StartObservation starts a new clone observation.
func (c *Client) StartObservation(ctx context.Context, startRequest types.StartObservationRequest) (*observer.Session, error) {
	u := c.URL("/observation/start")

	var session observer.Session

	err := c.request(ctx, u, startRequest, &session)

	return &session, err
}

// StopObservation stops the clone observation.
func (c *Client) StopObservation(ctx context.Context, stopRequest types.StopObservationRequest) (*observer.Session, error) {
	u := c.URL("/observation/stop")

	var observerSession observer.Session

	err := c.request(ctx, u, stopRequest, &observerSession)

	return &observerSession, err
}

// SummaryObservation returns the summary of clone observation.
func (c *Client) SummaryObservation(ctx context.Context, cloneID, sessionID string) (*observer.SummaryArtifact, error) {
	u := c.URL(fmt.Sprintf("/observation/summary/%s/%s", cloneID, sessionID))

	request, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make a request")
	}

	response, err := c.Do(ctx, request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get response")
	}

	defer func() { _ = response.Body.Close() }()

	var observationSummary observer.SummaryArtifact

	if err := json.NewDecoder(response.Body).Decode(&observationSummary); err != nil {
		return nil, errors.Wrap(err, "failed to decode a response body")
	}

	return &observationSummary, err
}

// DownloadArtifact downloads clone observation artifacts.
func (c *Client) DownloadArtifact(ctx context.Context, cloneID, sessionID, artifactType string) (io.ReadCloser, error) {
	u := c.URL("/observation/download")

	values := url.Values{}
	values.Add("clone_id", cloneID)
	values.Add("session_id", sessionID)
	values.Add("artifact_type", artifactType)
	u.RawQuery = values.Encode()

	request, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make a request")
	}

	response, err := c.Do(ctx, request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get response")
	}

	if response.StatusCode != http.StatusOK {
		content, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read response, status code: %d", response.StatusCode)
		}

		return nil, errors.Errorf("status code: %d. content: %s", response.StatusCode, content)
	}

	return response.Body, err
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
