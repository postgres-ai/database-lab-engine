package srv

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"gitlab.com/postgres-ai/database-lab/pkg/log"

	"github.com/pkg/errors"
)

// writeJSON responds with JSON.
func writeJSON(w http.ResponseWriter, httpStatusCode int, v interface{}) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal response")
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpStatusCode)

	if _, err = w.Write(b); err != nil {
		return errors.Wrap(err, "failed to write response")
	}

	log.Dbg("Response:", v)

	return nil
}

// readJSON reads JSON from request.
func readJSON(r *http.Request, v interface{}) error {
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read a request body")
	}

	log.Dbg("Request:", string(reqBody))

	if err = json.Unmarshal(reqBody, v); err != nil {
		return errors.Wrapf(err, "failed to unmarshal json: %s", string(reqBody))
	}

	log.Dbg("Request:", v)

	return nil
}
