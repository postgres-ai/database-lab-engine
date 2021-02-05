/*
2019 © Postgres.ai
*/

package srv

import (
	"context"
	"net/http"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/platform"
)

// VerificationTokenHeader defines a verification token name that should be passed in request headers.
const VerificationTokenHeader = "Verification-Token"

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Msg("-> ", r.Method, r.RequestURI)
		next.ServeHTTP(w, r)
	})
}

// authMW defines an authorization middleware of the Database Lab HTTP server.
type authMW struct {
	verificationToken     string
	personalTokenVerifier platform.PersonalTokenVerifier
}

func (a *authMW) authorized(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get(VerificationTokenHeader)
		if !a.isAccessAllowed(r.Context(), token) {
			sendUnauthorizedError(w, r)
			return
		}

		h(w, r)
	}
}

func (a *authMW) isAccessAllowed(ctx context.Context, token string) bool {
	if token == "" {
		return false
	}

	if a.verificationToken == token {
		return true
	}

	if a.personalTokenVerifier != nil && a.personalTokenVerifier.IsPersonalTokenEnabled() &&
		a.personalTokenVerifier.IsAllowedToken(ctx, token) {
		return true
	}

	return false
}
