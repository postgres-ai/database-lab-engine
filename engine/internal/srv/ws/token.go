/*
2022 © Postgres.ai
*/

// Package ws contains helpers and services to manage web-socket connections, routes and handlers.
package ws

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const (
	// CleanupInterval defines the interval of cleaning up the token registry.
	CleanupInterval = 5 * time.Minute

	wsTokenExpTime = time.Minute
	issuer         = "DLE"
	keyLength      = 32
)

// TokenKeeper manages access tokens to web-socket handlers.
type TokenKeeper struct {
	signingKey  []byte
	mu          *sync.Mutex
	jwtRegistry map[string]struct{}
}

// NewTokenKeeper creates a new TokenKeeper instance.
func NewTokenKeeper() (*TokenKeeper, error) {
	crRand, err := generateRandomToken(keyLength)
	if err != nil {
		return nil, err
	}

	return &TokenKeeper{
		signingKey:  crRand,
		mu:          &sync.Mutex{},
		jwtRegistry: make(map[string]struct{}),
	}, nil
}

// IssueToken generates a new one-time token.
func (tk *TokenKeeper) IssueToken() (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &jwt.RegisteredClaims{
		ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(wsTokenExpTime)},
		Issuer:    issuer,
	})

	tokenString, err := token.SignedString(tk.signingKey)
	if err != nil {
		return "", err
	}

	tk.storeToken(tokenString)

	return tokenString, nil
}

// ValidateToken makes sure that token is valid.
func (tk *TokenKeeper) ValidateToken(tokenString string) error {
	if !tk.isTokenExists(tokenString) {
		return errors.New("token not found")
	}

	return tk.parseToken(tokenString)
}

// parseToken checks the token signature and claims without touching the registry.
func (tk *TokenKeeper) parseToken(tokenString string) error {
	parsedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return tk.signingKey, nil
	})
	if err != nil {
		return err
	}

	if !parsedToken.Valid {
		return errors.New("invalid token")
	}

	return nil
}

// ExpendToken disposes of the one-time token after use.
func (tk *TokenKeeper) ExpendToken(token string) error {
	tk.mu.Lock()
	defer tk.mu.Unlock()

	if _, ok := tk.jwtRegistry[token]; !ok {
		return errors.New("token not found")
	}

	delete(tk.jwtRegistry, token)

	return nil
}

// RunCleaningUp starts the process of periodically cleaning the registry of tokens from old sessions.
func (tk *TokenKeeper) RunCleaningUp(ctx context.Context) {
	idleTimer := time.NewTimer(CleanupInterval)

	for {
		select {
		case <-idleTimer.C:
			tk.cleanUpTokens()
			idleTimer.Reset(CleanupInterval)

		case <-ctx.Done():
			idleTimer.Stop()
			return
		}
	}
}

func (tk *TokenKeeper) isTokenExists(token string) bool {
	tk.mu.Lock()
	defer tk.mu.Unlock()

	_, ok := tk.jwtRegistry[token]

	return ok
}

// cleanUpTokens removes old tokens. The registry keys are snapshotted under the lock, validated
// lock-free, and the expired set is deleted under a single lock hold; the sweep never holds
// tk.mu while validating because isTokenExists takes the same non-reentrant mutex.
func (tk *TokenKeeper) cleanUpTokens() {
	tokens := tk.registeredTokens()

	expired := make([]string, 0, len(tokens))

	for _, token := range tokens {
		if err := tk.parseToken(token); err != nil {
			expired = append(expired, token)
		}
	}

	if len(expired) == 0 {
		return
	}

	tk.mu.Lock()
	defer tk.mu.Unlock()

	for _, token := range expired {
		delete(tk.jwtRegistry, token)
	}
}

func (tk *TokenKeeper) registeredTokens() []string {
	tk.mu.Lock()
	defer tk.mu.Unlock()

	tokens := make([]string, 0, len(tk.jwtRegistry))

	for token := range tk.jwtRegistry {
		tokens = append(tokens, token)
	}

	return tokens
}

func (tk *TokenKeeper) storeToken(token string) {
	tk.mu.Lock()
	defer tk.mu.Unlock()

	tk.jwtRegistry[token] = struct{}{}
}

func generateRandomToken(tokenLen int) ([]byte, error) {
	b := make([]byte, tokenLen)

	if _, err := rand.Read(b); err != nil {
		return nil, err
	}

	return b, nil
}
