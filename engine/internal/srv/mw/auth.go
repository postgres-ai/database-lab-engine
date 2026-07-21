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

// ForwardedUserEmailHeader carries the acting user's email asserted by a trusted caller
// authenticated with the shared verification token (e.g. the Platform proxying console
// requests). It is ignored on personal-token requests and when authorization is disabled.
const ForwardedUserEmailHeader = "X-Forwarded-User-Email"

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
		ctx, ok := a.authenticate(r.Context(), r.Header.Get(VerificationTokenHeader), r.Header.Get(ForwardedUserEmailHeader))
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
	_, ok := a.authenticate(ctx, token, "")

	return ok
}

// authenticate validates the token and, for a valid personal token, returns a
// context carrying the resolved user identity. The shared verification token
// carries no identity of its own, but a shared-token caller may assert the
// acting user via forwardedEmail; the assertion is trusted because the shared
// token already grants full instance access.
func (a *Auth) authenticate(ctx context.Context, token, forwardedEmail string) (context.Context, bool) {
	if a.verificationToken == "" {
		return ctx, true
	}

	if subtle.ConstantTimeCompare([]byte(a.verificationToken), []byte(token)) == 1 {
		return withForwardedIdentity(ctx, forwardedEmail), true
	}

	if a.personalTokenVerifier != nil && a.personalTokenVerifier.IsPersonalTokenEnabled() {
		if identity, ok := a.personalTokenVerifier.AuthenticatePersonalToken(ctx, token); ok {
			return WithUserIdentity(ctx, identity), true
		}
	}

	return ctx, false
}

// withForwardedIdentity attaches the identity asserted by a shared-token caller.
func withForwardedIdentity(ctx context.Context, email string) context.Context {
	if email == "" {
		return ctx
	}

	return WithUserIdentity(ctx, platform.UserIdentity{Email: email})
}

// WithUserIdentity returns a context carrying the given user identity, as
// attached by the middleware for personal-token and forwarded identities.
func WithUserIdentity(ctx context.Context, identity platform.UserIdentity) context.Context {
	return context.WithValue(ctx, userIdentityKey, identity)
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
