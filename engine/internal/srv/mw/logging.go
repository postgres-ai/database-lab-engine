/*
2019 © Postgres.ai
*/

package mw

import (
	"net/http"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

// Logging logs the incoming request.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Msg("-> ", r.Method, r.RequestURI)
		next.ServeHTTP(w, r)
	})
}
