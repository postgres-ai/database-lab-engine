package srv

import (
	"fmt"
	"net/http"
	"net/url"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/models"

	"github.com/pkg/errors"
)

func failNotFound(w http.ResponseWriter, _ *http.Request) {
	errorNotFound := models.Error{
		Code:    models.ErrCodeNotFound,
		Message: "Not found.",
		Detail:  "Requested object does not exist.",
		Hint:    "Specify your request.",
	}

	_ = writeJSON(w, http.StatusNotFound, errorNotFound)

	log.Dbg("Not found")
}

func failUnauthorized(w http.ResponseWriter, _ *http.Request) {
	errorUnauthorized := models.Error{
		Code:    models.ErrCodeUnauthorized,
		Message: "Unauthorized.",
		Detail:  "",
		Hint:    "Check your verification token.",
	}

	_ = writeJSON(w, http.StatusUnauthorized, errorUnauthorized)

	log.Dbg("Unauthorized")
}

func failBadRequest(w http.ResponseWriter, _ *http.Request) {
	errorBadRequest := models.Error{
		Code:    models.ErrCodeBadRequest,
		Message: "Wrong request format.",
		Detail:  "",
		Hint:    "Check request params.",
	}

	_ = writeJSON(w, http.StatusBadRequest, errorBadRequest)

	log.Dbg("Bad request")
}

func failInternalServer(w http.ResponseWriter, r *http.Request, err error) {
	log.Err(errDetailsMsg(r, err, models.ErrCodeInternal))

	errorInternalServer := models.Error{
		Code:    models.ErrCodeInternal,
		Message: "Internal server error.",
		Detail:  errors.Cause(err).Error(),
		Hint:    "",
	}

	w.WriteHeader(http.StatusInternalServerError)
	_ = writeJSON(w, http.StatusInternalServerError, errorInternalServer)

	log.Dbg("Internal server error")
}

func errDetailsMsg(r *http.Request, err error, errCode models.ErrorCode) string {
	queryString := r.URL.String()
	if queryUnescape, e := url.QueryUnescape(queryString); e == nil {
		queryString = queryUnescape
	}

	return fmt.Sprintf("[%s] - %s %s - %+v",
		errCode, r.Method, queryString, err)
}
