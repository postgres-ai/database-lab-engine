/*
2019 © Postgres.ai
*/

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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

	return nil
}
