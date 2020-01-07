package srv

import (
	"net/http"

	"gitlab.com/postgres-ai/database-lab/src/log"
	"gitlab.com/postgres-ai/database-lab/src/models"
)

func failNotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	var errorNotFound = models.Error{
		Code:    "NOT_FOUND",
		Message: "Not found.",
		Detail:  "Requested object does not exist.",
		Hint:    "Specify your request.",
	}
	writeJson(w, errorNotFound)
	log.Dbg("Not found")
}

func failUnauthorized(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusUnauthorized)
	var errorUnauthorized = models.Error{
		Code:    "UNAUTHORIZED",
		Message: "Unauthorized.",
		Detail:  "",
		Hint:    "Check your verification token.",
	}
	writeJson(w, errorUnauthorized)
	log.Dbg("Unauthorized")
}

func failBadRequest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	var errorBadRequest = models.Error{
		Code:    "BAD_REQUEST",
		Message: "Wrong request format.",
		Detail:  "",
		Hint:    "Check request params.",
	}
	writeJson(w, errorBadRequest)
	log.Dbg("Bad request")
}

func failInternalServer(w http.ResponseWriter, r *http.Request, detail string) {
	w.WriteHeader(http.StatusInternalServerError)

	var errorInternalServer = models.Error{
		Code:    "INTERNAL_ERROR",
		Message: "Internal server error.",
		Detail:  detail,
		Hint:    "",
	}
	writeJson(w, errorInternalServer)
	log.Dbg("Internal server error")
}
