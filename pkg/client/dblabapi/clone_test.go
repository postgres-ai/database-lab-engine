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

	"gitlab.com/postgres-ai/database-lab/v2/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/models"
)

func TestClientListClones(t *testing.T) {
	expectedClones := []*models.Clone{{
		ID: "testCloneID",
		Metadata: models.CloneMetadata{
			CloneDiffSize: 450,
			CloningTime:   1,
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
	}}

	mockClient := NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal(t, req.URL.String(), "https://example.com/status")

		// Prepare response.
		body, err := json.Marshal(models.InstanceStatus{Cloning: models.Cloning{Clones: expectedClones}})
		require.NoError(t, err)

		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBuffer(body)),
			Header:     make(http.Header),
		}
	})

	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "token",
	})
	require.NoError(t, err)

	c.client = mockClient

	// Send a request.
	cloneList, err := c.ListClones(context.Background())
	require.NoError(t, err)

	assert.EqualValues(t, expectedClones, cloneList)
}

func TestClientListClonesWithFailedRequest(t *testing.T) {
	mockClient := NewTestClient(func(r *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBuffer([]byte{})),
			Header:     make(http.Header),
		}
	})

	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "token",
	})
	require.NoError(t, err)

	c.client = mockClient

	cloneList, err := c.ListClones(context.Background())
	require.EqualError(t, err, "failed to decode a response body: EOF")
	require.Nil(t, cloneList)
}

func TestClientCreateClone(t *testing.T) {
	expectedClone := models.Clone{
		ID: "testCloneID",
		Metadata: models.CloneMetadata{
			CloneDiffSize: 450,
			CloningTime:   1,
		},
		Protected: true,
		DeleteAt:  "2020-01-10 00:00:05.000 UTC",
		CreatedAt: "2020-01-10 00:00:00.000 UTC",
		Status: models.Status{
			Code:    "OK",
			Message: "Clone is ready",
		},
		DB: models.Database{
			Username: "john",
			Password: "doe",
		},
	}

	mockClient := NewTestClient(func(r *http.Request) *http.Response {
		clone := expectedClone

		// skip while refresh status
		if r.Method == http.MethodPost {
			assert.Equal(t, r.URL.String(), "https://example.com/clone")

			requestBody, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			defer func() { _ = r.Body.Close() }()

			cloneRequest := types.CloneCreateRequest{}
			err = json.Unmarshal(requestBody, &cloneRequest)
			require.NoError(t, err)
			clone = expectedClone
			clone.Status = models.Status{
				Code:    models.StatusCreating,
				Message: models.CloneMessageCreating,
			}
		}

		responseBody, err := json.Marshal(clone)
		require.NoError(t, err)

		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBuffer(responseBody)),
			Header:     make(http.Header),
		}
	})

	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "token",
	})
	require.NoError(t, err)

	c.client = mockClient
	c.pollingInterval = time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Send a request.
	newClone, err := c.CreateClone(ctx, types.CloneCreateRequest{
		ID: "testCloneID",

		Protected: true,
		DB: &types.DatabaseRequest{
			Username: "john",
			Password: "doe",
		},
	})
	require.NoError(t, err)

	assert.EqualValues(t, expectedClone, *newClone)
}

func TestClientCreateCloneAsync(t *testing.T) {
	expectedClone := models.Clone{
		ID: "testCloneID",
		Metadata: models.CloneMetadata{
			CloneDiffSize: 450,
			CloningTime:   1,
		},
		Protected: true,
		DeleteAt:  "2020-01-10 00:00:05.000 UTC",
		CreatedAt: "2020-01-10 00:00:00.000 UTC",
		Status: models.Status{
			Code:    "OK",
			Message: "Clone is ready",
		},
		DB: models.Database{
			Username: "john",
			Password: "doe",
		},
	}

	mockClient := NewTestClient(func(r *http.Request) *http.Response {
		assert.Equal(t, r.URL.String(), "https://example.com/clone")

		requestBody, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer func() { _ = r.Body.Close() }()

		cloneRequest := types.CloneCreateRequest{}
		err = json.Unmarshal(requestBody, &cloneRequest)
		require.NoError(t, err)

		responseBody, err := json.Marshal(expectedClone)
		require.NoError(t, err)

		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBuffer(responseBody)),
			Header:     make(http.Header),
		}
	})

	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "token",
	})
	require.NoError(t, err)

	c.client = mockClient
	c.pollingInterval = time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Send a request.
	newClone, err := c.CreateCloneAsync(ctx, types.CloneCreateRequest{
		ID:        "testCloneID",
		Protected: true,
		DB: &types.DatabaseRequest{
			Username: "john",
			Password: "doe",
		},
	})
	require.NoError(t, err)

	assert.EqualValues(t, expectedClone, *newClone)
}

func TestClientCreateCloneWithFailedRequest(t *testing.T) {
	mockClient := NewTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBuffer([]byte{})),
			Header:     make(http.Header),
		}
	})

	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "token",
	})
	require.NoError(t, err)

	c.client = mockClient

	clone, err := c.CreateClone(context.Background(), types.CloneCreateRequest{})
	require.EqualError(t, err, "failed to decode a response body: EOF")
	require.Nil(t, clone)
}

func TestClientGetClone(t *testing.T) {
	expectedClone := &models.Clone{
		ID: "testCloneID",
		Metadata: models.CloneMetadata{
			CloneDiffSize: 450,
			CloningTime:   1,
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
	}

	mockClient := NewTestClient(func(r *http.Request) *http.Response {
		assert.Equal(t, r.URL.String(), "https://example.com/clone/testCloneID")

		// Prepare response.
		responseBody, err := json.Marshal(expectedClone)
		require.NoError(t, err)

		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBuffer(responseBody)),
			Header:     make(http.Header),
		}
	})

	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "token",
	})
	require.NoError(t, err)

	c.client = mockClient

	// Send a request.
	clone, err := c.GetClone(context.Background(), expectedClone.ID)
	require.NoError(t, err)

	assert.EqualValues(t, expectedClone, clone)
}

func TestClientGetCloneWithFailedRequest(t *testing.T) {
	mockClient := NewTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBuffer([]byte{})),
			Header:     make(http.Header),
		}
	})

	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "token",
	})
	require.NoError(t, err)

	c.client = mockClient

	clone, err := c.GetClone(context.Background(), "cloneID")
	require.EqualError(t, err, "failed to decode a response body: EOF")
	require.Nil(t, clone)
}

func TestClientUpdateClone(t *testing.T) {
	cloneModel := &models.Clone{
		ID: "testCloneID",
		Metadata: models.CloneMetadata{
			CloneDiffSize: 450,
			CloningTime:   1,
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
	}

	mockClient := NewTestClient(func(r *http.Request) *http.Response {
		assert.Equal(t, r.URL.String(), "https://example.com/clone/testCloneID")

		requestBody, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer func() { _ = r.Body.Close() }()

		updateRequest := types.CloneUpdateRequest{}
		err = json.Unmarshal(requestBody, &updateRequest)
		require.NoError(t, err)

		cloneModel.Protected = updateRequest.Protected

		// Prepare response.
		responseBody, err := json.Marshal(cloneModel)
		require.NoError(t, err)

		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBuffer(responseBody)),
			Header:     make(http.Header),
		}
	})

	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "token",
	})
	require.NoError(t, err)

	c.client = mockClient

	// Send a request.
	newClone, err := c.UpdateClone(context.Background(), cloneModel.ID, types.CloneUpdateRequest{
		Protected: false,
	})
	require.NoError(t, err)

	assert.EqualValues(t, cloneModel, newClone)
}

func TestClientUpdateCloneWithFailedRequest(t *testing.T) {
	mockClient := NewTestClient(func(req *http.Request) *http.Response {
		errorBadRequest := models.Error{
			Code:    "BAD_REQUEST",
			Message: "Check request params.",
		}

		responseBody, err := json.Marshal(errorBadRequest)
		require.NoError(t, err)

		return &http.Response{
			StatusCode: 400,
			Body:       io.NopCloser(bytes.NewBuffer(responseBody)),
			Header:     make(http.Header),
		}
	})

	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "token",
	})
	require.NoError(t, err)

	c.client = mockClient

	clone, err := c.UpdateClone(context.Background(), "testCloneID", types.CloneUpdateRequest{})
	require.EqualError(t, err, `failed to get response: Check request params.`)
	require.Nil(t, clone)
}

func TestClientDestroyClone(t *testing.T) {
	mockClient := NewTestClient(func(r *http.Request) *http.Response {
		assert.Equal(t, r.URL.String(), "https://example.com/clone/testCloneID")

		var responseBody []byte
		statusCode := 200

		if r.Method != http.MethodDelete {
			errorNotFound := models.Error{
				Code:    "NOT_FOUND",
				Message: "Not found.",
			}

			var err error
			responseBody, err = json.Marshal(errorNotFound)
			require.NoError(t, err)

			statusCode = 404
		}

		return &http.Response{
			StatusCode: statusCode,
			Body:       io.NopCloser(bytes.NewBuffer(responseBody)),
			Header:     make(http.Header),
		}
	})

	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "token",
	})
	require.NoError(t, err)

	c.client = mockClient
	c.pollingInterval = time.Millisecond

	// Send a request.
	err = c.DestroyClone(context.Background(), "testCloneID")
	require.NoError(t, err)
}

func TestClientDestroyCloneAsync(t *testing.T) {
	mockClient := NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal(t, req.URL.String(), "https://example.com/clone/testCloneID")

		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBuffer(nil)),
			Header:     make(http.Header),
		}
	})

	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "token",
	})
	require.NoError(t, err)

	c.client = mockClient

	// Send a request.
	err = c.DestroyCloneAsync(context.Background(), "testCloneID")
	require.NoError(t, err)
}

func TestClientDestroyCloneWithFailedRequest(t *testing.T) {
	errorNotFound := models.Error{
		Code:    "NOT_FOUND",
		Message: "Requested object does not exist. Specify your request.",
	}
	mockClient := NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal(t, req.URL.String(), "https://example.com/clone/testCloneID")

		responseBody, err := json.Marshal(errorNotFound)
		require.NoError(t, err)

		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(bytes.NewBuffer(responseBody)),
			Header:     make(http.Header),
		}
	})

	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "token",
	})
	require.NoError(t, err)

	c.client = mockClient

	// Send a request.
	err = c.DestroyClone(context.Background(), "testCloneID")
	assert.EqualError(t, err, `failed to get response: Requested object does not exist. Specify your request.`)
}

func TestClientResetClone(t *testing.T) {
	mockClient := NewTestClient(func(r *http.Request) *http.Response {
		var responseBody []byte

		if r.Method == http.MethodPost {
			assert.Equal(t, r.URL.String(), "https://example.com/clone/testCloneID/reset")
		} else {
			assert.Equal(t, r.URL.String(), "https://example.com/clone/testCloneID")

			clone := models.Clone{
				ID: "testCloneID",
				Status: models.Status{
					Code:    models.StatusOK,
					Message: models.CloneMessageOK,
				},
			}

			var err error
			responseBody, err = json.Marshal(clone)
			require.NoError(t, err)
		}

		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBuffer(responseBody)),
			Header:     make(http.Header),
		}
	})

	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "token",
	})
	require.NoError(t, err)

	c.client = mockClient
	c.pollingInterval = time.Millisecond

	// Send a request.
	err = c.ResetClone(context.Background(), "testCloneID", types.ResetCloneRequest{Latest: true})
	require.NoError(t, err)
}

func TestClientResetCloneAsync(t *testing.T) {
	mockClient := NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal(t, req.URL.String(), "https://example.com/clone/testCloneID/reset")

		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBuffer(nil)),
			Header:     make(http.Header),
		}
	})

	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "token",
	})
	require.NoError(t, err)

	c.client = mockClient

	// Send a request.
	err = c.ResetCloneAsync(context.Background(), "testCloneID", types.ResetCloneRequest{Latest: true})
	require.NoError(t, err)
}

func TestClientResetCloneWithFailedRequest(t *testing.T) {
	errorUnauthorized := models.Error{
		Code:    "UNAUTHORIZED",
		Message: "Check your verification token.",
	}
	mockClient := NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal(t, req.URL.String(), "https://example.com/clone/testCloneID/reset")

		responseBody, err := json.Marshal(errorUnauthorized)
		require.NoError(t, err)

		return &http.Response{
			StatusCode: 401,
			Body:       io.NopCloser(bytes.NewBuffer(responseBody)),
			Header:     make(http.Header),
		}
	})

	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "token",
	})
	require.NoError(t, err)

	c.client = mockClient

	// Send a request.
	err = c.ResetClone(context.Background(), "testCloneID", types.ResetCloneRequest{Latest: true, SnapshotID: "test"})
	assert.EqualError(t, err, `failed to get response: Check your verification token.`)
}
