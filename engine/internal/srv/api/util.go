/*
2019 © Postgres.ai
*/

package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

// YamlContentType is the content type header for YAML.
const YamlContentType = "application/yaml; charset=utf-8"

// JSONContentType is the content type header for JSON.
const JSONContentType = "application/json; charset=utf-8"

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

// ReadJSON reads JSON from request.
func ReadJSON(r *http.Request, v interface{}) error {
	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read a request body")
	}

	if err = json.Unmarshal(reqBody, v); err != nil {
		return errors.Wrapf(err, "failed to unmarshal json: %s", string(reqBody))
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

	if _, err := w.Write(b); err != nil {
		return errors.Wrap(err, "failed to write response")
	}

	return nil
}
