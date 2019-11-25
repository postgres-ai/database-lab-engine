package srv

import (
	"encoding/json"
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

func failNotFound(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	b, err := json.MarshalIndent(ERROR_NOT_FOUND, "", "  ")
	if err != nil {
		log.Err(err)
	}
	w.Write(b)
}

func failUnauthorized(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusUnauthorized)
	b, err := json.MarshalIndent(ERROR_UNAUTHORIZED, "", "  ")
	if err != nil {
		log.Err(err)
	}
	w.Write(b)
}
