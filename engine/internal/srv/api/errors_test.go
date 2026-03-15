/*
2019 © Postgres.ai
*/

package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

func TestErrorCodeStatuses(t *testing.T) {
	testCases := []struct {
		error models.ErrorCode
		code  int
	}{
		{error: "BAD_REQUEST", code: 400},
		{error: "UNAUTHORIZED", code: 401},
		{error: "NOT_FOUND", code: 404},
		{error: "INTERNAL_ERROR", code: 500},
		{error: "UNKNOWN_ERROR", code: 500},
	}

	for _, tc := range testCases {
		errorCode := toStatusCode(models.Error{Code: tc.error})

		assert.Equal(t, tc.code, errorCode)
	}
}

func TestSendError(t *testing.T) {
	t.Run("models.Error preserves error code", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/test", nil)
		modelErr := models.Error{Code: models.ErrCodeNotFound, Message: "clone not found"}
		SendError(w, r, modelErr)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "clone not found")
		assert.Contains(t, w.Body.String(), string(models.ErrCodeNotFound))
	})

	t.Run("generic error becomes internal server error", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/clone", nil)
		SendError(w, r, fmt.Errorf("database connection failed"))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "database connection failed")
	})
}

func TestSendBadRequestError(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/clone", nil)
	SendBadRequestError(w, r, "invalid clone id")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid clone id")
}

func TestSendUnauthorizedError(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/status", nil)
	SendUnauthorizedError(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "verification token")
}

func TestSendNotFoundError(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/clone/nonexistent", nil)
	SendNotFoundError(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "does not exist")
}

func TestWriteJSON_ErrorCases(t *testing.T) {
	t.Run("valid struct serializes correctly", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := map[string]string{"status": "ok"}
		err := WriteJSON(w, http.StatusOK, data)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, JSONContentType, w.Header().Get("Content-Type"))
		assert.Contains(t, w.Body.String(), `"status": "ok"`)
	})

	t.Run("unmarshalable value returns error", func(t *testing.T) {
		w := httptest.NewRecorder()
		err := WriteJSON(w, http.StatusOK, make(chan int))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to marshal")
	})
}

func TestReadJSON(t *testing.T) {
	t.Run("valid json parses correctly", func(t *testing.T) {
		body := `{"id":"clone-1","status":"ok"}`
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		var result map[string]string
		err := ReadJSON(r, &result)
		require.NoError(t, err)
		assert.Equal(t, "clone-1", result["id"])
		assert.Equal(t, "ok", result["status"])
	})

	t.Run("invalid json returns error", func(t *testing.T) {
		body := `{invalid json}`
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		var result map[string]string
		err := ReadJSON(r, &result)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal")
	})

	t.Run("empty body returns error", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
		var result map[string]string
		err := ReadJSON(r, &result)
		require.Error(t, err)
	})
}

func TestErrDetailsMsg(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/clone?id=test%20clone", nil)
	msg := errDetailsMsg(r, fmt.Errorf("connection refused"))
	assert.Contains(t, msg, "POST")
	assert.Contains(t, msg, "/api/clone")
	assert.Contains(t, msg, "connection refused")
}
