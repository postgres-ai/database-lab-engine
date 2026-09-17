/*
2019 © Postgres.ai
*/

package api

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

// YamlContentType is the content type header for YAML.
const YamlContentType = "application/yaml; charset=utf-8"

// JSONContentType is the content type header for JSON.
const JSONContentType = "application/json; charset=utf-8"

// MaxRequestBodyBytes is the largest request body ReadJSON accepts. Every JSON body the API takes
// is a small document; the cap keeps a client from holding memory with an unbounded one.
const MaxRequestBodyBytes = 1 << 20

// ErrBodyTooLarge reports a request body over MaxRequestBodyBytes. Callers test for it with
// errors.Is; the *http.MaxBytesError it wraps is an implementation detail.
var ErrBodyTooLarge = stderrors.New("request body is too large")

// WriteJSON responds with JSON.
func WriteJSON(w http.ResponseWriter, httpStatusCode int, v interface{}) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal response")
	}

	w.Header().Set("Content-Type", JSONContentType)
	w.WriteHeader(httpStatusCode)

	if _, err := w.Write(b); err != nil {
		return errors.Wrap(err, "failed to write response")
	}

	log.Dbg("Response:", v)

	return nil
}

// ReadJSON decodes the request body into v. The body is capped at MaxRequestBodyBytes: a larger
// one yields ErrBodyTooLarge. A decode failure names the JSON error only, never the body, which
// may hold a credential.
func ReadJSON(r *http.Request, v interface{}) error {
	reqBody, err := io.ReadAll(http.MaxBytesReader(nil, r.Body, MaxRequestBodyBytes))
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if stderrors.As(err, &maxBytesErr) {
			return fmt.Errorf("%w: %w", ErrBodyTooLarge, maxBytesErr)
		}

		return fmt.Errorf("failed to read a request body: %w", err)
	}

	if err = json.Unmarshal(reqBody, v); err != nil {
		return fmt.Errorf("failed to unmarshal json: %w", err)
	}

	return nil
}

// WriteData responds with JSON.
func WriteData(w http.ResponseWriter, httpStatusCode int, b []byte) error {
	return WriteDataTyped(w, httpStatusCode, JSONContentType, b)
}

// WriteDataTyped responds with data including content type.
func WriteDataTyped(
	w http.ResponseWriter,
	httpStatusCode int,
	contentType string,
	b []byte,
) error {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(httpStatusCode)

	// skip write on empty body to avoid implicit chunked transfer-encoding.
	if len(b) > 0 {
		if _, err := w.Write(b); err != nil {
			return errors.Wrap(err, "failed to write response")
		}
	}

	return nil
}
