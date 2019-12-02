package srv

import (
	"net/http"

	"../log"
	m "../models"
)

var ERROR_NOT_FOUND = m.Error{
	Code:    "NOT_FOUND",
	Message: "Not found.",
	Detail:  "Requested model does not exist.",
	Hint:    "Specify your request.",
}

var ERROR_UNAUTHORIZED = m.Error{
	Code:    "UNAUTHORIZED",
	Message: "Unauthorized.",
	Detail:  "",
	Hint:    "Check your verification token.",
}

var ERROR_BAD_REQUEST = m.Error{
	Code:    "BAD_REQUEST",
	Message: "Wrong request format.",
	Detail:  "",
	Hint:    "Check request params.",
}

var ERROR_INTERNAL_SERVER = m.Error{
	Code:    "INTERNAL_ERROR",
	Message: "Internal server error.",
	Detail:  "",
	Hint:    "",
}

func failNotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	writeJson(w, ERROR_NOT_FOUND)
	log.Dbg(r.RequestURI)
	log.Dbg("Not found")
}

func failUnauthorized(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusUnauthorized)
	writeJson(w, ERROR_UNAUTHORIZED)
	log.Dbg("Unauthorized")
}

func failBadRequest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	writeJson(w, ERROR_BAD_REQUEST)
	log.Dbg("Bad request")
}

func failInternalServer(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	writeJson(w, ERROR_INTERNAL_SERVER)
	log.Dbg("Internal server error")
}
