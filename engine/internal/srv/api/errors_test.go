/*
2019 © Postgres.ai
*/

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

func TestErrorCodeStatuses(t *testing.T) {
	testCases := []struct {
		error models.ErrorCode
		code  int
	}{
		{
			error: "BAD_REQUEST",
			code:  400,
		},
		{
			error: "UNAUTHORIZED",
			code:  401,
		},
		{
			error: "NOT_FOUND",
			code:  404,
		},
		{
			error: "INTERNAL_ERROR",
			code:  500,
		},
		{
			error: "UNKNOWN_ERROR",
			code:  500,
		},
	}

	for _, tc := range testCases {
		errorCode := toStatusCode(models.Error{Code: tc.error})

		assert.Equal(t, tc.code, errorCode)
	}
}

func TestSendError(t *testing.T) {
	badRequest := models.New(models.ErrCodeBadRequest, "bad input")
	notFound := models.New(models.ErrCodeNotFound, "missing")
	plain := fmt.Errorf("something broke")

	testCases := []struct {
		name     string
		err      error
		status   int
		wantCode models.ErrorCode
	}{
		{name: "models error is preserved", err: badRequest, status: 400, wantCode: models.ErrCodeBadRequest},
		{name: "plain error becomes internal", err: plain, status: 500, wantCode: models.ErrCodeInternal},
		{name: "pkg-wrapped models error", err: errors.Wrap(notFound, "context"), status: 404, wantCode: models.ErrCodeNotFound},
		{name: "std-wrapped models error", err: fmt.Errorf("reset: %w", notFound), status: 404, wantCode: models.ErrCodeNotFound},
		{name: "wrapped plain error becomes internal", err: errors.Wrap(plain, "wrap"), status: 500, wantCode: models.ErrCodeInternal},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			SendError(rec, req, tc.err)

			assert.Equal(t, tc.status, rec.Code)
			assert.Equal(t, JSONContentType, rec.Header().Get("Content-Type"))

			var errResp models.Error
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
			assert.Equal(t, tc.wantCode, errResp.Code)
		})
	}
}

func TestSendBadRequestError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/clone", nil)
	rec := httptest.NewRecorder()

	SendBadRequestError(rec, req, "invalid clone id")

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp models.Error
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, models.ErrCodeBadRequest, errResp.Code)
	assert.Equal(t, "invalid clone id", errResp.Message)
}

func TestSendUnauthorizedError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()

	SendUnauthorizedError(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var errResp models.Error
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, models.ErrCodeUnauthorized, errResp.Code)
	assert.Contains(t, errResp.Message, "verification token")
}

func TestSendNotFoundError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/clone/missing", nil)
	rec := httptest.NewRecorder()

	SendNotFoundError(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var errResp models.Error
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, models.ErrCodeNotFound, errResp.Code)
}

func TestErrDetailsMsg(t *testing.T) {
	testCases := []struct {
		name     string
		method   string
		url      string
		err      error
		contains []string
	}{
		{name: "basic get error", method: http.MethodGet, url: "/test", err: fmt.Errorf("something failed"), contains: []string{"[ERROR]", "GET", "/test", "something failed"}},
		{name: "post with query params", method: http.MethodPost, url: "/api?key=value", err: fmt.Errorf("bad request"), contains: []string{"POST", "/api?key=value", "bad request"}},
		{name: "url with encoded characters", method: http.MethodGet, url: "/api?name=hello%20world", err: fmt.Errorf("err"), contains: []string{"hello world"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.url, nil)
			msg := errDetailsMsg(req, tc.err)

			for _, s := range tc.contains {
				assert.Contains(t, msg, s)
			}
		})
	}
}
