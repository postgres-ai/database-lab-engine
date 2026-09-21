/*
2019 © Postgres.ai
*/

package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/platform"
)

func TestIfPersonalTokenEnabled(t *testing.T) {
	s := Service{}
	assert.Equal(t, s.IsPersonalTokenEnabled(), false)

	s.cfg.EnablePersonalToken = true
	assert.Equal(t, s.IsPersonalTokenEnabled(), true)
}

func TestIfOrganizationIsAllowed(t *testing.T) {
	s := Service{}
	assert.Equal(t, s.isAllowedOrganization(0), false)

	s.token.OrganizationID = 1
	assert.Equal(t, s.isAllowedOrganization(0), false)
	assert.Equal(t, s.isAllowedOrganization(1), true)
}

func TestOriginURL(t *testing.T) {
	s := Service{
		cfg: Config{
			URL: "https://example.com:2345/api/path",
		},
	}

	assert.Equal(t, "https://example.com:2345", s.OriginURL())
}

func TestAuthenticatePersonalToken(t *testing.T) {
	testCases := []struct {
		name      string
		enabled   bool
		org       uint
		personal  bool
		email     string
		wantOK    bool
		wantEmail string
	}{
		{name: "valid token", enabled: true, org: 1, personal: true, email: "u@acme.io", wantOK: true, wantEmail: "u@acme.io"},
		{name: "non-personal token", enabled: true, org: 1, personal: false, email: "u@acme.io", wantOK: false},
		{name: "wrong organization", enabled: true, org: 2, personal: true, email: "u@acme.io", wantOK: false},
		{name: "personal tokens disabled", enabled: false, org: 1, personal: true, email: "u@acme.io", wantOK: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp := platform.TokenCheckResponse{OrganizationID: tc.org, Personal: tc.personal, Email: tc.email}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				require.NoError(t, json.NewEncoder(w).Encode(resp))
			}))
			defer server.Close()

			client, err := platform.NewClient(platform.ClientConfig{URL: server.URL, AccessToken: "test"})
			require.NoError(t, err)

			s := &Service{client: client}
			s.cfg.EnablePersonalToken = tc.enabled
			s.token.OrganizationID = 1

			identity, ok := s.AuthenticatePersonalToken(context.Background(), "token")
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.wantEmail, identity.Email)
		})
	}
}

func TestReloadRacesWithServiceReaders(t *testing.T) {
	t.Parallel()

	s := &Service{cfg: Config{URL: "https://example.com", EnableTelemetry: true}, token: Token{OrganizationID: 1}}

	const iterations = 200

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()

		for i := 0; i < iterations; i++ {
			s.Reload(&Service{
				cfg:   Config{URL: fmt.Sprintf("https://example-%d.com", i), AccessToken: "token", EnablePersonalToken: true},
				token: Token{OrganizationID: uint(i)},
			})
		}
	}()

	go func() {
		defer wg.Done()

		for i := 0; i < iterations; i++ {
			_ = s.IsTelemetryEnabled()
			_ = s.IsPersonalTokenEnabled()
			_ = s.BindClonesToUser()
			_ = s.AccessToken()
			_ = s.OrgKey()
			_ = s.OriginURL()
			_ = s.Token()
			_ = s.Client()
			_ = s.isAllowedOrganization(1)
		}
	}()

	wg.Wait()
}

func TestReloadCopiesEveryField(t *testing.T) {
	s := &Service{cfg: Config{URL: "https://before.example.com", OrgKey: "before"}, token: Token{OrganizationID: 1}}
	client, err := platform.NewClient(platform.ClientConfig{URL: "https://after.example.com", AccessToken: "token"})
	require.NoError(t, err)

	s.Reload(&Service{
		cfg:    Config{URL: "https://after.example.com", OrgKey: "after", EnableTelemetry: true},
		token:  Token{OrganizationID: 2},
		client: client,
	})

	assert.Equal(t, "https://after.example.com", s.OriginURL())
	assert.Equal(t, "after", s.OrgKey())
	assert.True(t, s.IsTelemetryEnabled())
	assert.Equal(t, uint(2), s.Token().OrganizationID)
	assert.Same(t, client, s.Client())
}
