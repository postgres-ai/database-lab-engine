/*
2026 © Postgres.ai
*/

package dblabapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

func newConfigTestClient(t *testing.T, handler func(*http.Request) *http.Response) *Client {
	t.Helper()

	c, err := NewClient(Options{Host: "https://example.com/", VerificationToken: "testVerify"})
	require.NoError(t, err)

	c.client = NewTestClient(handler)

	return c
}

func jsonResponse(t *testing.T, status int, v interface{}) *http.Response {
	t.Helper()

	body, err := json.Marshal(v)
	require.NoError(t, err)

	return &http.Response{StatusCode: status, Body: io.NopCloser(bytes.NewBuffer(body)), Header: make(http.Header)}
}

func TestClientProbeSource(t *testing.T) {
	expected := &models.ProposedConfig{
		Source:           models.SourceConnection{Host: "db.example.com", Port: 5432, Username: "alice", DBName: "shop"},
		DetectedProvider: "rds",
		DockerImage:      "rds",
		DockerTag:        "16-0.8.0",
		ResolvedImage:    "registry.gitlab.com/postgres-ai/se-images/rds:16-0.8.0",
		PgMajorVersion:   16,
		CollationVersion: "2.36",
		Databases:        []string{"shop"},
		SharedBuffers:    "4GB",
	}

	c := newConfigTestClient(t, func(req *http.Request) *http.Response {
		assert.Equal(t, "https://example.com/admin/probe-source", req.URL.String())
		assert.Equal(t, http.MethodPost, req.Method)

		var got models.ProbeSourceRequest
		require.NoError(t, json.NewDecoder(req.Body).Decode(&got))
		assert.Equal(t, "postgres://alice@db.example.com/shop", got.URL)
		assert.Equal(t, "secret", got.Password)

		return jsonResponse(t, http.StatusOK, expected)
	})

	got, err := c.ProbeSource(context.Background(), models.ProbeSourceRequest{
		URL: "postgres://alice@db.example.com/shop", Password: "secret",
	})
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestClientProbeSource_BadRequest(t *testing.T) {
	c := newConfigTestClient(t, func(_ *http.Request) *http.Response {
		return jsonResponse(t, http.StatusBadRequest, models.Error{
			Code:    "BAD_REQUEST",
			Message: "connect to source: connection refused",
		})
	})

	got, err := c.ProbeSource(context.Background(), models.ProbeSourceRequest{URL: "postgres://x@y/z"})
	require.Error(t, err)
	require.Nil(t, got)
	assert.Contains(t, err.Error(), "connect to source")
}

func TestClientApplyConfig(t *testing.T) {
	projection := json.RawMessage(`{"retrievalMode":"logical","databaseContainer":{"dockerImage":"postgresai/extended-postgres:16"}}`)
	applied := json.RawMessage(`{"retrievalMode":"logical","databaseContainer":{"dockerImage":"postgresai/extended-postgres:16"}}`)

	c := newConfigTestClient(t, func(req *http.Request) *http.Response {
		assert.Equal(t, "https://example.com/admin/config", req.URL.String())
		assert.Equal(t, http.MethodPost, req.Method)

		sent, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		assert.JSONEq(t, string(projection), string(sent))

		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(applied)), Header: make(http.Header)}
	})

	got, err := c.ApplyConfig(context.Background(), projection)
	require.NoError(t, err)
	assert.JSONEq(t, string(applied), string(got))
}

func TestClientApplyConfig_Error(t *testing.T) {
	c := newConfigTestClient(t, func(_ *http.Request) *http.Response {
		return jsonResponse(t, http.StatusBadRequest, models.Error{
			Code:    "BAD_REQUEST",
			Message: "configuration management via UI/API disabled by admin",
		})
	})

	got, err := c.ApplyConfig(context.Background(), json.RawMessage(`{}`))
	require.Error(t, err)
	require.Nil(t, got)
	assert.Contains(t, err.Error(), "disabled by admin")
}
