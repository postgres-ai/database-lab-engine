/*
2019 © Postgres.ai
*/

// Package api contains helpers to work with HTTP requests and responses.
package api

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

// SendError sends a server error.
func SendError(w http.ResponseWriter, r *http.Request, err error) {
	log.Err(errDetailsMsg(r, err))

	errorInternalServer, ok := errors.Cause(err).(models.Error)
	if !ok {
		errorInternalServer = models.Error{
			Code:    models.ErrCodeInternal,
			Message: errors.Cause(err).Error(),
		}
	}

	_ = WriteJSON(w, toStatusCode(errorInternalServer), errorInternalServer)
}

// SendBadRequestError sends a bad request error.
func SendBadRequestError(w http.ResponseWriter, r *http.Request, message string) {
	errorBadRequest := models.Error{
		Code:    models.ErrCodeBadRequest,
		Message: message,
	}

	SendError(w, r, errorBadRequest)
}

// SendUnauthorizedError sends an unauthorized request error.
func SendUnauthorizedError(w http.ResponseWriter, r *http.Request) {
	errorUnauthorized := models.Error{
		Code:    models.ErrCodeUnauthorized,
		Message: "Check your verification token.",
	}

	SendError(w, r, errorUnauthorized)
}

// SendNotFoundError sends a not found error.
func SendNotFoundError(w http.ResponseWriter, r *http.Request) {
	errorNotFound := models.Error{
		Code:    models.ErrCodeNotFound,
		Message: "Requested object does not exist. Specify your request.",
	}

	SendError(w, r, errorNotFound)
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
