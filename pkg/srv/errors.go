/*
2019 © Postgres.ai
*/

package srv

import (
	"fmt"
	"net/http"
	"net/url"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/models"

	"github.com/pkg/errors"
)

func sendError(w http.ResponseWriter, r *http.Request, err error) {
	log.Err(errDetailsMsg(r, err))

	errorInternalServer, ok := errors.Cause(err).(models.Error)
	if !ok {
		errorInternalServer = models.Error{
			Code:    models.ErrCodeInternal,
			Message: errors.Cause(err).Error(),
		}
	}

	_ = writeJSON(w, toStatusCode(errorInternalServer), errorInternalServer)
}

func sendBadRequestError(w http.ResponseWriter, r *http.Request, message string) {
	errorBadRequest := models.Error{
		Code:    models.ErrCodeBadRequest,
		Message: message,
	}

	sendError(w, r, errorBadRequest)
}

func sendUnauthorizedError(w http.ResponseWriter, r *http.Request) {
	errorUnauthorized := models.Error{
		Code:    models.ErrCodeUnauthorized,
		Message: "Check your verification token.",
	}

	sendError(w, r, errorUnauthorized)
}

func sendNotFoundError(w http.ResponseWriter, r *http.Request) {
	errorNotFound := models.Error{
		Code:    models.ErrCodeNotFound,
		Message: "Requested object does not exist. Specify your request.",
	}

	sendError(w, r, errorNotFound)
}

// errDetailsMsg formats details of an error message.
func errDetailsMsg(r *http.Request, err error) string {
	queryString := r.URL.String()
	if queryUnescape, e := url.QueryUnescape(queryString); e == nil {
		queryString = queryUnescape
	}

	return fmt.Sprintf("[ERROR] - %s %s - %v", r.Method, queryString, err)
}

// toStatusCode converts an error to an HTTP status code.
func toStatusCode(err models.Error) int {
	switch err.Code {
	case models.ErrCodeBadRequest:
		return http.StatusBadRequest

	case models.ErrCodeUnauthorized:
		return http.StatusUnauthorized

	case models.ErrCodeNotFound:
		return http.StatusNotFound

	case models.ErrCodeInternal:
		return http.StatusInternalServerError

	default:
		return http.StatusInternalServerError
	}
}
