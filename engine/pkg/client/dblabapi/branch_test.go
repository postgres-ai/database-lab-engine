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

	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

func TestClientUpdateBranch(t *testing.T) {
	branchView := &models.BranchView{Name: "feature", Protected: false}

	mockClient := NewTestClient(func(r *http.Request) *http.Response {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Contains(t, r.URL.String(), "/branch/feature")

		requestBody, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer func() { _ = r.Body.Close() }()

		updateRequest := types.BranchUpdateRequest{}
		require.NoError(t, json.Unmarshal(requestBody, &updateRequest))
		require.NotNil(t, updateRequest.Protected)
		branchView.Protected = *updateRequest.Protected

		responseBody, err := json.Marshal(branchView)
		require.NoError(t, err)

		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBuffer(responseBody)),
			Header:     make(http.Header),
		}
	})

	c, err := NewClient(Options{Host: "https://example.com/", VerificationToken: "token"})
	require.NoError(t, err)

	c.client = mockClient

	protected := true

	updated, err := c.UpdateBranch(context.Background(), branchView.Name, types.BranchUpdateRequest{Protected: &protected})
	require.NoError(t, err)
	assert.True(t, updated.Protected)
}
