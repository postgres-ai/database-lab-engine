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

// ctxKey is the type for context keys set by this package.
type ctxKey string

// userIdentityKey is the context key carrying the authenticated user identity.
const userIdentityKey ctxKey = "dblab_user_identity"

// Auth defines an authorization middleware of the Database Lab HTTP server.
type Auth struct {
	verificationToken     string
	personalTokenVerifier platform.PersonalTokenVerifier
}

// NewAuth creates a new Auth middleware.
func NewAuth(verificationToken string, personalTokenVerifier platform.PersonalTokenVerifier) *Auth {
	return &Auth{verificationToken: verificationToken, personalTokenVerifier: personalTokenVerifier}
}

// Authorized checks if the user has permission to access and attaches the
// resolved user identity (for personal tokens) to the request context.
func (a *Auth) Authorized(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, ok := a.authenticate(r.Context(), r.Header.Get(VerificationTokenHeader))
		if !ok {
			api.SendUnauthorizedError(w, r)
			return
		}

		h(w, r.WithContext(ctx))
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
	_, ok := a.authenticate(ctx, token)

	return ok
}

// authenticate validates the token and, for a valid personal token, returns a
// context carrying the resolved user identity. The shared verification token is
// authorized but carries no identity.
func (a *Auth) authenticate(ctx context.Context, token string) (context.Context, bool) {
	if a.verificationToken == "" {
		return ctx, true
	}

	if subtle.ConstantTimeCompare([]byte(a.verificationToken), []byte(token)) == 1 {
		return ctx, true
	}

	if a.personalTokenVerifier != nil && a.personalTokenVerifier.IsPersonalTokenEnabled() {
		if identity, ok := a.personalTokenVerifier.AuthenticatePersonalToken(ctx, token); ok {
			return context.WithValue(ctx, userIdentityKey, identity), true
		}
	}

	return ctx, false
}

// UserIdentityFromContext returns the authenticated user identity attached to
// the context by Authorized, if present.
func UserIdentityFromContext(ctx context.Context) (platform.UserIdentity, bool) {
	identity, ok := ctx.Value(userIdentityKey).(platform.UserIdentity)

	return identity, ok
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
