/*
2019 © Postgres.ai
*/

package client

import (
	"context"
	"encoding/json"
	"net/http"

	"gitlab.com/postgres-ai/database-lab/pkg/models"

	"github.com/pkg/errors"
)

// Status provides an instance status.
func (c *Client) Status(ctx context.Context) (*models.InstanceStatus, error) {
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
		return nil, errors.Wrap(err, "failed to get response")
	}

	return &instanceStatus, nil
}
