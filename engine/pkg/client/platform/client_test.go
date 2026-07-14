/*
2019 © Postgres.ai
*/

package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
)

// roundTripFunc represents a mock type.
type roundTripFunc func(req *http.Request) *http.Response

// RoundTrip is a mock function.
func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

// NewTestClient returns a mock of *http.Client.
func NewTestClient(fn roundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

func TestNewClient(t *testing.T) {
	// The test case also checks if the client can be work with a no-ideal URL.
	c, err := NewClient(ClientConfig{
		URL:         "https://example.com//",
		AccessToken: "testVerify",
	})
	require.NoError(t, err)

	assert.IsType(t, &Client{}, c)
	assert.Equal(t, "https://example.com", c.url.String())
	assert.Equal(t, "testVerify", c.accessToken)
	assert.IsType(t, &http.Client{}, c.client)
}

func TestClientURL(t *testing.T) {
	c, err := NewClient(ClientConfig{
		URL:         "https://example.com/",
		AccessToken: "testVerify",
	})
	require.NoError(t, err)

	assert.Equal(t, "https://example.com/test-url", c.buildURL("test-url").String())
}

func TestClientWithEmptyConfigURL(t *testing.T) {
	testCases := []struct {
		url   string
		token string
	}{
		{url: "", token: ""},
		{url: "", token: "non-empty"},
	}

	for _, tc := range testCases {
		platformClient, err := NewClient(ClientConfig{
			URL:         tc.url,
			AccessToken: tc.token,
		})

		require.NotNil(t, platformClient)
		require.NotNil(t, err)
		require.Error(t, err, "invalid config of Platform Client given: URL and AccessToken must not be empty")
	}
}

func TestClientWithEmptyConfigKeys(t *testing.T) {
	testCases := []struct {
		url   string
		token string
	}{
		{url: "non-empty", token: ""},
	}

	for _, tc := range testCases {
		platformClient, err := NewClient(ClientConfig{
			URL:         tc.url,
			AccessToken: tc.token,
		})

		require.NotNil(t, platformClient)
		require.NotNil(t, err)
		require.Error(t, err, "invalid config of Platform Client given: URL and AccessToken must not be empty")
	}
}

func TestClientChecksPlatformToken(t *testing.T) {
	expectedResponse := TokenCheckResponse{
		OrganizationID: 1,
		Personal:       true,
		Email:          "jsmith@acme.com",
	}

	testClient := NewTestClient(func(req *http.Request) *http.Response {
		body, err := json.Marshal(expectedResponse)
		require.NoError(t, err)

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBuffer(body)),
		}
	})

	platformClient, err := NewClient(ClientConfig{
		URL:         "https://example.com/",
		AccessToken: "testVerify",
	})
	require.NoError(t, err)
	platformClient.client = testClient

	platformToken, err := platformClient.CheckPlatformToken(context.Background(), TokenCheckRequest{Token: "PersonalToken"})
	require.NoError(t, err)

	assert.Equal(t, expectedResponse.OrganizationID, platformToken.OrganizationID)
	assert.Equal(t, expectedResponse.Personal, platformToken.Personal)
	assert.Equal(t, expectedResponse.Email, platformToken.Email)
}

func TestClientChecksPlatformSEToken(t *testing.T) {
	expectedResponse := TokenCheckResponse{
		OrganizationID: 2,
	}

	testClient := NewTestClient(func(req *http.Request) *http.Response {
		body, err := json.Marshal(expectedResponse)
		require.NoError(t, err)

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBuffer(body)),
		}
	})

	platformClient, err := NewClient(ClientConfig{
		URL:         "https://example.com/",
		AccessToken: "testVerify",
	})
	require.NoError(t, err)
	platformClient.client = testClient

	platformToken, err := platformClient.CheckPlatformToken(context.Background(), TokenCheckRequest{Token: "PersonalToken"})
	require.NoError(t, err)

	assert.Equal(t, expectedResponse.OrganizationID, platformToken.OrganizationID)
	assert.False(t, platformToken.Personal)
	assert.Empty(t, platformToken.Email)
}

func TestClientChecksPlatformTokenFailed(t *testing.T) {
	expectedResponse := TokenCheckResponse{
		APIResponse: APIResponse{
			Hint:    "Ensure that you use a valid and non-expired token",
			Details: "Cannot find the specified token or it is expired.",
			Message: "Invalid token",
		},
	}

	testClient := NewTestClient(func(req *http.Request) *http.Response {
		body, err := json.Marshal(expectedResponse)
		require.NoError(t, err)

		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(bytes.NewBuffer(body)),
		}
	})

	platformClient, err := NewClient(ClientConfig{
		URL:         "https://example.com/",
		AccessToken: "testVerify",
	})
	require.NoError(t, err)
	platformClient.client = testClient

	platformToken, err := platformClient.CheckPlatformToken(context.Background(), TokenCheckRequest{Token: "PersonalToken"})
	require.NotNil(t, err)

	assert.Equal(t, expectedResponse.APIResponse.Message, platformToken.Message)
	assert.Equal(t, expectedResponse.APIResponse.Hint, platformToken.Hint)
	assert.Equal(t, expectedResponse.APIResponse.Details, platformToken.Details)
}

func TestClientSendUsage(t *testing.T) {
	testCases := []struct {
		name            string
		response        BillingResponse
		wantBillingPage string
	}{
		{name: "response without recognized org", response: BillingResponse{Result: "ok", BillingActive: true}},
		{name: "response with recognized org", response: BillingResponse{Result: "ok", BillingActive: true, Org: &Org{ID: 1, Alias: "acme"}},
			wantBillingPage: "https://example.com/console/acme/billing"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testClient := NewTestClient(func(req *http.Request) *http.Response {
				body, err := json.Marshal(EditionResponse{BillingResponse: tc.response})
				require.NoError(t, err)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBuffer(body)),
				}
			})

			platformClient, err := NewClient(ClientConfig{
				URL:         "https://example.com/",
				AccessToken: "testVerify",
				OrgKey:      "testOrgKey",
			})
			require.NoError(t, err)
			platformClient.client = testClient

			props := &global.EngineProps{}
			respData, err := platformClient.SendUsage(context.Background(), props, InstanceUsage{InstanceID: "test"})
			require.NoError(t, err)

			assert.True(t, respData.BillingActive)
			assert.True(t, props.BillingActive)

			if tc.wantBillingPage == "" {
				assert.Nil(t, respData.Org)
				return
			}

			require.NotNil(t, respData.Org)
			assert.Equal(t, tc.wantBillingPage, respData.Org.BillingPage)
		})
	}
}
