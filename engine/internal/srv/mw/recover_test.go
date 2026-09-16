/*
2026 © Postgres.ai
*/

package mw

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

func TestRecover(t *testing.T) {
	testCases := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus int
		wantCode   models.ErrorCode
	}{
		{name: "healthy handler passes through", handler: func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }, wantStatus: http.StatusOK},
		{name: "nil dereference becomes 500", handler: func(http.ResponseWriter, *http.Request) { var p *models.Error; _ = p.Code }, wantStatus: http.StatusInternalServerError, wantCode: models.ErrCodeInternal},
		{name: "explicit panic becomes 500", handler: func(http.ResponseWriter, *http.Request) { panic("boom") }, wantStatus: http.StatusInternalServerError, wantCode: models.ErrCodeInternal},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := Recover(tc.handler)

			rec := httptest.NewRecorder()
			require.NotPanics(t, func() { handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/clone", nil)) })
			assert.Equal(t, tc.wantStatus, rec.Code)

			if tc.wantCode == "" {
				return
			}

			var apiErr models.Error
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &apiErr))
			assert.Equal(t, tc.wantCode, apiErr.Code)
			assert.NotContains(t, apiErr.Message, "boom", "panic details must not leak to the client")
		})
	}
}

func TestRecover_ServerSurvivesPanic(t *testing.T) {
	calls := 0
	handler := Recover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			panic("first request fails")
		}

		w.WriteHeader(http.StatusOK)
	}))

	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	resp, err = srv.Client().Get(srv.URL)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRecover_AbortHandlerPropagates(t *testing.T) {
	handler := Recover(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) }))

	assert.PanicsWithValue(t, http.ErrAbortHandler, func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	})
}
