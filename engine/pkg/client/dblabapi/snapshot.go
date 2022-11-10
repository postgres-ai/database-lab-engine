/*
2019 © Postgres.ai
*/

package dblabapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

// ListSnapshots provides a snapshot list.
func (c *Client) ListSnapshots(ctx context.Context) ([]*models.Snapshot, error) {
	body, err := c.ListSnapshotsRaw(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get response")
	}

	defer func() { _ = body.Close() }()

	var snapshots []*models.Snapshot

	if err := json.NewDecoder(body).Decode(&snapshots); err != nil {
		return nil, errors.Wrap(err, "failed to get response")
	}

	return snapshots, nil
}

// ListSnapshotsRaw provides a snapshot list in raw format.
func (c *Client) ListSnapshotsRaw(ctx context.Context) (io.ReadCloser, error) {
	u := c.URL("/snapshots")

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

// CreateSnapshot creates a new snapshot.
func (c *Client) CreateSnapshot(ctx context.Context, snapshotRequest types.SnapshotCreateRequest) (*models.Snapshot, error) {
	u := c.URL("/snapshot/create")

	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(snapshotRequest); err != nil {
		return nil, errors.Wrap(err, "failed to encode SnapshotCreateRequest")
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

	var snapshot *models.Snapshot

	if err := json.NewDecoder(response.Body).Decode(&snapshot); err != nil {
		return nil, errors.Wrap(err, "failed to get response")
	}

	return snapshot, nil
}

// DeleteSnapshot deletes snapshot.
func (c *Client) DeleteSnapshot(ctx context.Context, snapshotRequest types.SnapshotDestroyRequest) error {
	u := c.URL("/snapshot/delete")

	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(snapshotRequest); err != nil {
		return errors.Wrap(err, "failed to encode snapshotDestroyRequest")
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
