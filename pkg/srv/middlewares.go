package srv

import (
	"net/http"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
)

// VerificationTokenHeader defines a verification token name that should be passed in request headers.
const VerificationTokenHeader = "Verification-Token"

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Msg("-> ", r.Method, r.RequestURI)
		next.ServeHTTP(w, r)
	})
}

func (s *Server) authorized(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get(VerificationTokenHeader)
		if len(token) == 0 || s.Config.VerificationToken != token {
			failUnauthorized(w, r)
			return
		}

		h(w, r)
	}
}
