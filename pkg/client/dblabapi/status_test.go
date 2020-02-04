package dblabapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/pkg/models"
)

func TestClientStatus(t *testing.T) {
	expectedStatus := &models.InstanceStatus{
		Status: &models.Status{
			Code:    "OK",
			Message: "Instance is ready",
		},
		FileSystem: &models.FileSystem{
			Size: 15489651156451,
			Free: 15429651156451,
		},
		DataSize:            2654568125,
		ExpectedCloningTime: 0,
		NumClones:           1,
		Clones: []*models.Clone{{
			ID:   "testCloneID",
			Name: "mockClone",
			Metadata: &models.CloneMetadata{
				CloneSize:   45054685181,
				CloningTime: 1,
			},
			Protected: true,
			DeleteAt:  "2020-01-10 00:00:05.000 UTC",
			CreatedAt: "2020-01-10 00:00:00.000 UTC",
			Status: &models.Status{
				Code:    "OK",
				Message: "Instance is ready",
			},
			DB: &models.Database{
				Username: "john",
				Password: "doe",
			},
			Project: "testProject",
		}},
	}

	mockClient := NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal(t, req.URL.String(), "https://example.com/status")

		// Prepare response.
		body, err := json.Marshal(expectedStatus)
		require.NoError(t, err)

		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBuffer(body)),
			Header:     make(http.Header),
		}
	})

	logger, _ := test.NewNullLogger()
	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "testVerify",
	}, logger)
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
			Body:       ioutil.NopCloser(bytes.NewBuffer([]byte{})),
			Header:     make(http.Header),
		}
	})

	logger, _ := test.NewNullLogger()
	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "testVerify",
	}, logger)
	require.NoError(t, err)

	c.client = mockClient

	status, err := c.Status(context.Background())
	require.EqualError(t, err, "failed to get response: EOF")
	require.Nil(t, status)
}
