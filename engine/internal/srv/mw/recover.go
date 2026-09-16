/*
2026 © Postgres.ai
*/

package mw

import (
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

	"gitlab.com/postgres-ai/database-lab/v3/internal/srv/api"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

// Recover turns a panic in a downstream handler into a 500 JSON error so a single bad request
// cannot take down the server. http.ErrAbortHandler is re-raised to keep its net/http semantics.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}

			if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(rec)
			}

			log.Err(fmt.Sprintf("panic while handling %s %s: %v\n%s", r.Method, r.RequestURI, rec, debug.Stack()))

			api.SendError(w, r, errors.New("internal server error"))
		}()

		next.ServeHTTP(w, r)
	})
}
