/*
2019 © Postgres.ai
*/

package dblabapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

// Status provides an instance status.
func (c *Client) Status(ctx context.Context) (*models.InstanceStatus, error) {
	body, err := c.StatusRaw(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get response")
	}

	defer func() { _ = body.Close() }()

	var instanceStatus models.InstanceStatus

	if err := json.NewDecoder(body).Decode(&instanceStatus); err != nil {
		return nil, errors.Wrap(err, "failed to get response")
	}

	return &instanceStatus, nil
}

// StatusRaw provides a raw instance status.
func (c *Client) StatusRaw(ctx context.Context) (io.ReadCloser, error) {
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

// Health provides instance health info.
func (c *Client) Health(ctx context.Context) (*models.Engine, error) {
	request, err := http.NewRequest(http.MethodGet, c.URL("/healthz").String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make a request")
	}

	response, err := c.Do(ctx, request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get response")
	}

	defer func() { _ = response.Body.Close() }()

	var engine models.Engine

	if err := json.NewDecoder(response.Body).Decode(&engine); err != nil {
		return nil, errors.Wrap(err, "failed to get response")
	}

	return &engine, nil
}
