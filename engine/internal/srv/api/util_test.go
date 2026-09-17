/*
2019 © Postgres.ai
*/

package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteJSON(t *testing.T) {
	type response struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	testCases := []struct {
		name       string
		statusCode int
		value      interface{}
	}{
		{name: "ok response", statusCode: http.StatusOK, value: response{Name: "test", Count: 42}},
		{name: "created response", statusCode: http.StatusCreated, value: response{Name: "new", Count: 1}},
		{name: "empty struct", statusCode: http.StatusOK, value: response{}},
		{name: "nil value", statusCode: http.StatusOK, value: nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			err := WriteJSON(rec, tc.statusCode, tc.value)
			require.NoError(t, err)

			assert.Equal(t, tc.statusCode, rec.Code)
			assert.Equal(t, JSONContentType, rec.Header().Get("Content-Type"))

			if tc.value != nil {
				var got response
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
				assert.Equal(t, tc.value, got)
			}
		})
	}
}

func TestWriteJSON_UnmarshalableValue(t *testing.T) {
	rec := httptest.NewRecorder()

	err := WriteJSON(rec, http.StatusOK, make(chan int))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal response")
}

func TestReadJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
		ID   int    `json:"id"`
	}

	testCases := []struct {
		name    string
		body    string
		want    payload
		wantErr bool
	}{
		{name: "valid json", body: `{"name":"test","id":1}`, want: payload{Name: "test", ID: 1}},
		{name: "empty fields", body: `{}`, want: payload{}},
		{name: "invalid json", body: `{invalid`, wantErr: true},
		{name: "empty body", body: ``, wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(tc.body))
			var got payload

			err := ReadJSON(req, &got)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestWriteData(t *testing.T) {
	rec := httptest.NewRecorder()
	data := []byte(`{"status":"ok"}`)

	err := WriteData(rec, http.StatusOK, data)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, JSONContentType, rec.Header().Get("Content-Type"))
	assert.Equal(t, data, rec.Body.Bytes())
}

func TestWriteDataTyped(t *testing.T) {
	testCases := []struct {
		name        string
		statusCode  int
		contentType string
		data        []byte
	}{
		{name: "json content", statusCode: http.StatusOK, contentType: JSONContentType, data: []byte(`{"key":"val"}`)},
		{name: "yaml content", statusCode: http.StatusOK, contentType: YamlContentType, data: []byte("key: val\n")},
		{name: "empty body", statusCode: http.StatusNoContent, contentType: JSONContentType, data: nil},
		{name: "zero-length body", statusCode: http.StatusNoContent, contentType: JSONContentType, data: []byte{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			err := WriteDataTyped(rec, tc.statusCode, tc.contentType, tc.data)
			require.NoError(t, err)

			assert.Equal(t, tc.statusCode, rec.Code)
			assert.Equal(t, tc.contentType, rec.Header().Get("Content-Type"))

			if len(tc.data) > 0 {
				assert.Equal(t, tc.data, rec.Body.Bytes())
			} else {
				assert.Zero(t, rec.Body.Len())
			}
		})
	}
}

func TestReadJSON_LargePayload(t *testing.T) {
	type payload struct {
		Data string `json:"data"`
	}

	largeString := strings.Repeat("a", 10000)
	body := `{"data":"` + largeString + `"}`
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))

	var got payload
	err := ReadJSON(req, &got)
	require.NoError(t, err)
	assert.Equal(t, largeString, got.Data)
}

// jsonBodyOfSize returns a valid JSON document of exactly size bytes.
func jsonBodyOfSize(t *testing.T, size int) string {
	t.Helper()

	const frame = `{"data":""}`

	body := `{"data":"` + strings.Repeat("a", size-len(frame)) + `"}`
	require.Len(t, body, size)

	return body
}

func TestReadJSON_BodyTooLarge(t *testing.T) {
	t.Run("a body at the cap is accepted", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(jsonBodyOfSize(t, MaxRequestBodyBytes)))

		var got map[string]string
		require.NoError(t, ReadJSON(req, &got))
		assert.Len(t, got["data"], MaxRequestBodyBytes-len(`{"data":""}`))
	})

	t.Run("one byte over the cap is refused", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(jsonBodyOfSize(t, MaxRequestBodyBytes+1)))

		var got map[string]string
		err := ReadJSON(req, &got)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrBodyTooLarge)

		var maxBytesErr *http.MaxBytesError
		assert.ErrorAs(t, err, &maxBytesErr)
		assert.NotContains(t, err.Error(), "aaaa", "the error never echoes the body")
	})
}

func TestReadJSON_ErrorOmitsBody(t *testing.T) {
	const secret = "AKIAIOSFODNN7EXAMPLE"

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"password": "`+secret+`"`))

	var got map[string]string
	err := ReadJSON(req, &got)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), secret)
	assert.Contains(t, err.Error(), "failed to unmarshal json")
}

func TestSendDecodeError(t *testing.T) {
	testCases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "oversized body", err: fmt.Errorf("%w: %w", ErrBodyTooLarge, &http.MaxBytesError{Limit: 1}), status: http.StatusRequestEntityTooLarge, code: "PAYLOAD_TOO_LARGE"},
		{name: "malformed body", err: errors.New("failed to unmarshal json: unexpected end of JSON input"), status: http.StatusBadRequest, code: "BAD_REQUEST"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/test", nil)

			SendDecodeError(rec, req, tc.err)

			assert.Equal(t, tc.status, rec.Code)

			var response map[string]string
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
			assert.Equal(t, tc.code, response["code"])
			assert.Equal(t, tc.err.Error(), response["message"])
		})
	}
}
