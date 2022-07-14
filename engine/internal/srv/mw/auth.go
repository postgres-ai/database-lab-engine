/*
2019 © Postgres.ai
*/

// Package mw contains middlewares.
package mw

import (
	"context"
	"crypto/subtle"
	"net/http"

	"gitlab.com/postgres-ai/database-lab/v3/internal/platform"
	"gitlab.com/postgres-ai/database-lab/v3/internal/srv/api"
	"gitlab.com/postgres-ai/database-lab/v3/internal/srv/ws"
)

// VerificationTokenHeader defines the verification token name that should be passed in request headers.
const VerificationTokenHeader = "Verification-Token"

// wsTokenKey defines the name of web-sockets token parameter that should be passed in query string.
const wsTokenKey = "token"

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

// AdminMW checks if the user has permission to access to admin sub-route.
// TODO: check admin permissions.
func (a *Auth) AdminMW(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get(VerificationTokenHeader)
		if !a.isAccessAllowed(r.Context(), token) {
			api.SendUnauthorizedError(w, r)
			return
		}

		h.ServeHTTP(w, r)
	})
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

// WebSocketsMW checks if the user has a token to access to web-socket handlers.
func (a *Auth) WebSocketsMW(holder *ws.TokenKeeper, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authToken := r.URL.Query().Get(wsTokenKey)
		if authToken == "" {
			api.SendUnauthorizedError(w, r)

			return
		}

		if err := holder.ValidateToken(authToken); err != nil {
			api.SendUnauthorizedError(w, r)

			return
		}

		if err := holder.ExpendToken(authToken); err != nil {
			api.SendUnauthorizedError(w, r)

			return
		}

		h(w, r)
	}
}
