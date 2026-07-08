/*
2019 © Postgres.ai
*/

package mw

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/platform"
	"gitlab.com/postgres-ai/database-lab/v3/internal/srv/ws"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

// Test constants.
const (
	testVerificationToken   = "TestToken"
	testPlatformAccessToken = "PlatformAccessToken"
)

// MockPersonalTokenVerifier mocks personal verifier methods.
type MockPersonalTokenVerifier struct {
	isPersonalTokenEnabled bool
	email                  string
}

func (m MockPersonalTokenVerifier) IsAllowedToken(_ context.Context, token string) bool {
	return testPlatformAccessToken == token
}

func (m MockPersonalTokenVerifier) IsPersonalTokenEnabled() bool {
	return m.isPersonalTokenEnabled
}

func (m MockPersonalTokenVerifier) AuthenticatePersonalToken(_ context.Context, token string) (platform.UserIdentity, bool) {
	if !m.isPersonalTokenEnabled || token != testPlatformAccessToken {
		return platform.UserIdentity{}, false
	}

	return platform.UserIdentity{Email: m.email}, true
}

func TestAccess(t *testing.T) {
	testCases := []struct {
		name                   string
		requestToken           string
		isPersonalTokenEnabled bool
		result                 bool
	}{
		{isPersonalTokenEnabled: false, requestToken: "", result: false, name: "empty RequestToken with disabled PersonalToken"},
		{isPersonalTokenEnabled: false, requestToken: "WrongToken", result: false, name: "wrong RequestToken with disabled PersonalToken"},
		{isPersonalTokenEnabled: false, requestToken: "TestToken", result: true, name: "correct RequestToken with disabled PersonalToken"},
		{isPersonalTokenEnabled: true, requestToken: "", result: false, name: "empty RequestToken with enabled PersonalToken"},
		{isPersonalTokenEnabled: true, requestToken: "WrongToken", result: false, name: "wrong RequestToken with enabled PersonalToken"},
		{isPersonalTokenEnabled: true, requestToken: "TestToken", result: true, name: "correct RequestToken with enabled PersonalToken"},
		{isPersonalTokenEnabled: true, requestToken: "PlatformAccessToken", result: true, name: "correct PersonalToken with enabled PersonalToken"},
	}

	mw := Auth{
		verificationToken: testVerificationToken,
	}

	for _, tc := range testCases {
		t.Log(tc.name)
		mw.personalTokenVerifier = MockPersonalTokenVerifier{isPersonalTokenEnabled: tc.result}

		isAllowed := mw.isAccessAllowed(context.Background(), tc.requestToken)
		assert.Equal(t, tc.result, isAllowed)
	}
}

func TestAccess_EmptyVerificationToken(t *testing.T) {
	mw := Auth{verificationToken: ""}

	assert.True(t, mw.isAccessAllowed(context.Background(), ""))
	assert.True(t, mw.isAccessAllowed(context.Background(), "anything"))
}

func TestAccess_NilPersonalTokenVerifier(t *testing.T) {
	mw := Auth{verificationToken: testVerificationToken, personalTokenVerifier: nil}

	assert.False(t, mw.isAccessAllowed(context.Background(), "WrongToken"))
	assert.True(t, mw.isAccessAllowed(context.Background(), testVerificationToken))
}

func TestAuthorized(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	testCases := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{name: "valid token returns ok", token: testVerificationToken, wantStatus: http.StatusOK},
		{name: "empty token returns unauthorized", token: "", wantStatus: http.StatusUnauthorized},
		{name: "wrong token returns unauthorized", token: "wrong", wantStatus: http.StatusUnauthorized},
	}

	auth := NewAuth(testVerificationToken, nil)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set(VerificationTokenHeader, tc.token)
			rec := httptest.NewRecorder()

			auth.Authorized(okHandler)(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}

func TestAuthorized_ResponseBody(t *testing.T) {
	auth := NewAuth(testVerificationToken, nil)
	handler := auth.Authorized(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	var errResp models.Error
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, models.ErrCodeUnauthorized, errResp.Code)
}

func TestAuthorized_UserIdentity(t *testing.T) {
	testCases := []struct {
		name      string
		token     string
		wantOK    bool
		wantEmail string
	}{
		{name: "personal token attaches identity", token: testPlatformAccessToken, wantOK: true, wantEmail: "u@acme.io"},
		{name: "shared token has no identity", token: testVerificationToken, wantOK: false, wantEmail: ""},
	}

	auth := NewAuth(testVerificationToken, MockPersonalTokenVerifier{isPersonalTokenEnabled: true, email: "u@acme.io"})

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var gotIdentity platform.UserIdentity

			var gotOK bool

			handler := auth.Authorized(func(w http.ResponseWriter, r *http.Request) {
				gotIdentity, gotOK = UserIdentityFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/clone", nil)
			req.Header.Set(VerificationTokenHeader, tc.token)
			rec := httptest.NewRecorder()
			handler(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, tc.wantOK, gotOK)
			assert.Equal(t, tc.wantEmail, gotIdentity.Email)
		})
	}
}

func TestAdminMW(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	testCases := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{name: "valid token passes through", token: testVerificationToken, wantStatus: http.StatusOK},
		{name: "empty token returns unauthorized", token: "", wantStatus: http.StatusUnauthorized},
		{name: "wrong token returns unauthorized", token: "invalid", wantStatus: http.StatusUnauthorized},
	}

	auth := NewAuth(testVerificationToken, nil)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/admin/test", nil)
			req.Header.Set(VerificationTokenHeader, tc.token)
			rec := httptest.NewRecorder()

			auth.AdminMW(okHandler).ServeHTTP(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}

func TestAdminMW_WithPersonalToken(t *testing.T) {
	auth := NewAuth(testVerificationToken, MockPersonalTokenVerifier{isPersonalTokenEnabled: true})
	handler := auth.AdminMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/test", nil)
	req.Header.Set(VerificationTokenHeader, testPlatformAccessToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestWebSocketsMW(t *testing.T) {
	keeper, err := ws.NewTokenKeeper()
	require.NoError(t, err)

	validToken, err := keeper.IssueToken()
	require.NoError(t, err)

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	auth := NewAuth(testVerificationToken, nil)

	testCases := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{name: "empty token returns unauthorized", token: "", wantStatus: http.StatusUnauthorized},
		{name: "invalid token returns unauthorized", token: "invalid-token", wantStatus: http.StatusUnauthorized},
		{name: "valid token passes through", token: validToken, wantStatus: http.StatusOK},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := "/ws"
			if tc.token != "" {
				url = "/ws?token=" + tc.token
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()

			auth.WebSocketsMW(keeper, okHandler)(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}

func TestWebSocketsMW_TokenExpendedAfterUse(t *testing.T) {
	keeper, err := ws.NewTokenKeeper()
	require.NoError(t, err)

	token, err := keeper.IssueToken()
	require.NoError(t, err)

	auth := NewAuth(testVerificationToken, nil)
	handler := auth.WebSocketsMW(keeper, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ws?token="+token, nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	req = httptest.NewRequest(http.MethodGet, "/ws?token="+token, nil)
	rec = httptest.NewRecorder()
	handler(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestNewAuth(t *testing.T) {
	verifier := MockPersonalTokenVerifier{isPersonalTokenEnabled: true}
	auth := NewAuth("my-token", verifier)

	require.NotNil(t, auth)
	assert.Equal(t, "my-token", auth.verificationToken)
	assert.Equal(t, verifier, auth.personalTokenVerifier)
}
