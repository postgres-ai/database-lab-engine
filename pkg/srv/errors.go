package srv

import (
	"net/http"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/models"
)

func failNotFound(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotFound)

	errorNotFound := models.Error{
		Code:    "NOT_FOUND",
		Message: "Not found.",
		Detail:  "Requested object does not exist.",
		Hint:    "Specify your request.",
	}

	_ = writeJSON(w, errorNotFound)
	log.Dbg("Not found")
}

func failUnauthorized(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusUnauthorized)

	errorUnauthorized := models.Error{
		Code:    "UNAUTHORIZED",
		Message: "Unauthorized.",
		Detail:  "",
		Hint:    "Check your verification token.",
	}

	_ = writeJSON(w, errorUnauthorized)
	log.Dbg("Unauthorized")
}

func failBadRequest(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusBadRequest)

	errorBadRequest := models.Error{
		Code:    "BAD_REQUEST",
		Message: "Wrong request format.",
		Detail:  "",
		Hint:    "Check request params.",
	}

	_ = writeJSON(w, errorBadRequest)
	log.Dbg("Bad request")
}

func failInternalServer(w http.ResponseWriter, _ *http.Request, detail string) {
	w.WriteHeader(http.StatusInternalServerError)

	errorInternalServer := models.Error{
		Code:    "INTERNAL_ERROR",
		Message: "Internal server error.",
		Detail:  detail,
		Hint:    "",
	}

	_ = writeJSON(w, errorInternalServer)
	log.Dbg("Internal server error")
}
