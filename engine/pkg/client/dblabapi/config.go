/*
2026 © Postgres.ai
*/

package dblabapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

// ProbeSource inspects a source database (POST /admin/probe-source) and returns
// the engine's proposed retrieval configuration. The password is sent as a
// separate field and never embedded in the URL.
func (c *Client) ProbeSource(ctx context.Context, req models.ProbeSourceRequest) (*models.ProposedConfig, error) {
	u := c.URL("/admin/probe-source")

	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(req); err != nil {
		return nil, fmt.Errorf("failed to encode ProbeSourceRequest: %w", err)
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

	var proposed models.ProposedConfig

	if err := json.NewDecoder(response.Body).Decode(&proposed); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &proposed, nil
}

// ApplyConfig applies a projected configuration (POST /admin/config) and returns
// the projection the engine persisted. The projection is sent verbatim so the
// caller controls the exact field set, including the synthetic retrievalMode.
func (c *Client) ApplyConfig(ctx context.Context, projection json.RawMessage) (json.RawMessage, error) {
	u := c.URL("/admin/config")

	request, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(projection))
	if err != nil {
		return nil, fmt.Errorf("failed to make a request: %w", err)
	}

	response, err := c.Do(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to get response: %w", err)
	}

	defer func() { _ = response.Body.Close() }()

	applied, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return applied, nil
}
