/*
2019 © Postgres.ai
*/

// Package mw contains middlewares.
package mw

import (
	"context"
	"crypto/subtle"
	"net/http"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/platform"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/srv/api"
)

// VerificationTokenHeader defines a verification token name that should be passed in request headers.
const VerificationTokenHeader = "Verification-Token"

// Auth defines an authorization middleware of the Database Lab HTTP server.
type Auth struct {
	verificationToken     string
	personalTokenVerifier platform.PersonalTokenVerifier
}

// NewAuth creates a new Auth middleware.
func NewAuth(verificationToken string, personalTokenVerifier platform.PersonalTokenVerifier) *Auth {
	return &Auth{verificationToken: verificationToken, personalTokenVerifier: personalTokenVerifier}
}

// Authorized checks if the user has permission to access.
func (a *Auth) Authorized(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get(VerificationTokenHeader)
		if !a.isAccessAllowed(r.Context(), token) {
			api.SendUnauthorizedError(w, r)
			return
		}

		h(w, r)
	}
}

func (a *Auth) isAccessAllowed(ctx context.Context, token string) bool {
	if a.verificationToken == "" {
		return true
	}

	if subtle.ConstantTimeCompare([]byte(a.verificationToken), []byte(token)) == 1 {
		return true
	}

	if a.personalTokenVerifier != nil && a.personalTokenVerifier.IsPersonalTokenEnabled() &&
		a.personalTokenVerifier.IsAllowedToken(ctx, token) {
		return true
	}

	return false
}
