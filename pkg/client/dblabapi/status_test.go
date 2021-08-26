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

	"gitlab.com/postgres-ai/database-lab/v2/pkg/models"
)

func TestClientStatus(t *testing.T) {
	expectedStatus := &models.InstanceStatus{
		Status: &models.Status{
			Code:    "OK",
			Message: "Instance is ready",
		},
		FileSystem: &models.FileSystem{
			Size:   15489651156451,
			SizeHR: "14 TiB",
			Free:   15429651156451,
			FreeHR: "14 TiB",
		},
		DataSize:            2654568125,
		DataSizeHR:          "2.5 GiB",
		ExpectedCloningTime: 0,
		NumClones:           1,
		Clones: []*models.Clone{{
			ID: "testCloneID",
			Metadata: models.CloneMetadata{
				CloneDiffSize:   450546851,
				CloneDiffSizeHR: "430 MiB",
				CloningTime:     1,
			},
			Protected: true,
			DeleteAt:  "2020-01-10 00:00:05.000 UTC",
			CreatedAt: "2020-01-10 00:00:00.000 UTC",
			Status: models.Status{
				Code:    "OK",
				Message: "Instance is ready",
			},
			DB: models.Database{
				Username: "john",
				Password: "doe",
			},
		}},
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
