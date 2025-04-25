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

func TestClientStatus(t *testing.T) {
	expectedStatus := &models.InstanceStatus{
		Status: &models.Status{
			Code:    "OK",
			Message: "Instance is ready",
		},
		Cloning: models.Cloning{
			ExpectedCloningTime: 0,
			NumClones:           1,
			Clones: []*models.Clone{{
				ID: "testCloneID",
				Metadata: models.CloneMetadata{
					CloneDiffSize: 450546851,
					CloningTime:   1,
				},
				Protected: true,
				DeleteAt:  &models.LocalTime{Time: time.Date(2020, 01, 10, 0, 0, 5, 0, time.UTC)},
				CreatedAt: &models.LocalTime{Time: time.Date(2020, 01, 10, 0, 0, 0, 0, time.UTC)},
				Status: models.Status{
					Code:    "OK",
					Message: "Instance is ready",
				},
				DB: models.Database{
					Username: "john",
					Password: "doe",
				},
			}},
		},
		Pools: []models.PoolEntry{
			{
				Name:        "test_pool",
				Mode:        "zfs",
				DataStateAt: &models.LocalTime{Time: time.Date(2020, 01, 07, 0, 0, 0, 0, time.UTC)},
				Status:      "active",
				CloneList:   []string{"testCloneID"},
				FileSystem: models.FileSystem{
					Size:            167724544000,
					Free:            133059072000,
					Used:            34665472000,
					UsedBySnapshots: 199680000,
					UsedByClones:    1013760000,
					DataSize:        147773952000,
				}},
		},
	}

	mockClient := NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal(t, req.URL.String(), "https://example.com/status")

		// Prepare response.
		body, err := json.Marshal(expectedStatus)
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
	status, err := c.Status(context.Background())
	require.NoError(t, err)

	assert.EqualValues(t, expectedStatus, status)
}

func TestClientStatusWithFailedRequest(t *testing.T) {
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

	status, err := c.Status(context.Background())
	require.EqualError(t, err, "failed to get response: EOF")
	require.Nil(t, status)
}

func TestClientFullRefresh(t *testing.T) {
	expectedResponse := &models.Response{
		Status:  "OK",
		Message: "Full refresh started",
	}

	mockClient := NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal(t, req.URL.String(), "https://example.com/full-refresh")
		assert.Equal(t, req.Method, http.MethodPost)

		body, err := json.Marshal(expectedResponse)
		require.NoError(t, err)

		return &http.Response{
			StatusCode: http.StatusOK,
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

	resp, err := c.FullRefresh(context.Background())
	require.NoError(t, err)
	assert.EqualValues(t, expectedResponse, resp)
}

func TestClientFullRefreshWithFailedDecode(t *testing.T) {
	mockClient := NewTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: http.StatusOK,
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

	resp, err := c.FullRefresh(context.Background())
	require.EqualError(t, err, "failed to get response: EOF")
	require.Nil(t, resp)
}
