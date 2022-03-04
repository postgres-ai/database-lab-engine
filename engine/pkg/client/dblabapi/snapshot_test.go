package dblabapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

func TestClientListSnapshots(t *testing.T) {
	expectedSnapshots := []*models.Snapshot{{
		ID:          "testSnapshot1",
		CreatedAt:   &models.LocalTime{Time: time.Date(2020, 01, 10, 0, 0, 5, 0, time.UTC)},
		DataStateAt: &models.LocalTime{Time: time.Date(2020, 01, 10, 0, 0, 0, 0, time.UTC)},
	}, {
		ID:          "testSnapshot2",
		CreatedAt:   &models.LocalTime{Time: time.Date(2020, 1, 11, 8, 02, 11, 0, time.UTC)},
		DataStateAt: &models.LocalTime{Time: time.Date(2020, 1, 11, 8, 02, 0, 0, time.UTC)},
	}}

	mockClient := NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal(t, req.URL.String(), "https://example.com/snapshots")

		// Prepare response.
		body, err := json.Marshal(expectedSnapshots)
		require.NoError(t, err)

		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBuffer(body)),
			Header:     make(http.Header),
		}
	})

	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "testVerify",
	})
	require.NoError(t, err)

	c.client = mockClient

	// Send a request.
	snapshots, err := c.ListSnapshots(context.Background())
	require.NoError(t, err)

	assert.EqualValues(t, expectedSnapshots, snapshots)
}

func TestClientListSnapshotsWithFailedRequest(t *testing.T) {
	mockClient := NewTestClient(func(r *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBuffer([]byte{})),
			Header:     make(http.Header),
		}
	})

	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "testVerify",
	})
	require.NoError(t, err)

	c.client = mockClient

	snapshots, err := c.ListSnapshots(context.Background())
	require.EqualError(t, err, "failed to get response: EOF")
	require.Nil(t, snapshots)
}
