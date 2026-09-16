package ws

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
)

func TestNewTokenKeeper(t *testing.T) {
	tk, err := NewTokenKeeper()
	require.NoError(t, err)
	require.IsType(t, &TokenKeeper{}, tk)
	require.Equal(t, keyLength, len(tk.signingKey))
	require.NotNil(t, tk.jwtRegistry)
}

func TestTokenKeeperIssuesToken(t *testing.T) {
	tk, err := NewTokenKeeper()
	require.NoError(t, err)

	token, err := tk.IssueToken()
	require.NoError(t, err)
	require.Greater(t, len(token), 0)

	_, ok := tk.jwtRegistry[token]
	require.True(t, ok)

	err = tk.ValidateToken(token)
	require.Nil(t, err)

	err = tk.ExpendToken(token)
	require.Nil(t, err)

	_, ok = tk.jwtRegistry[token]
	require.False(t, ok)

	err = tk.ValidateToken(token)
	require.EqualError(t, err, "token not found")
}

func TestTokenKeeperValidation(t *testing.T) {
	tk, err := NewTokenKeeper()
	require.NoError(t, err)

	token, err := tk.IssueToken()
	require.NoError(t, err)
	require.Greater(t, len(token), 0)

	_, ok := tk.jwtRegistry[token]
	require.True(t, ok)

	err = tk.ValidateToken(token)
	require.Nil(t, err)

	invalidToken := "invalidToken"

	_, ok = tk.jwtRegistry[invalidToken]
	require.False(t, ok)

	err = tk.ValidateToken(invalidToken)
	require.EqualError(t, err, "token not found")

	invalidJWT, err := jwt.NewWithClaims(jwt.SigningMethodHS256, &jwt.RegisteredClaims{}).SignedString([]byte("invalidKey"))
	require.Nil(t, err)

	tk.jwtRegistry[invalidJWT] = struct{}{}
	err = tk.ValidateToken(invalidJWT)
	require.EqualError(t, err, "signature is invalid")

	expiredJWT, err := jwt.NewWithClaims(jwt.SigningMethodHS256, &jwt.RegisteredClaims{
		ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(-time.Second)},
		Issuer:    issuer,
	}).SignedString(tk.signingKey)
	require.Nil(t, err)

	tk.jwtRegistry[expiredJWT] = struct{}{}
	err = tk.ValidateToken(expiredJWT)
	require.EqualError(t, err, "Token is expired")
}

func TestCleanupTokens(t *testing.T) {
	tk, err := NewTokenKeeper()
	require.NoError(t, err)

	tk.storeToken("token1")
	tk.storeToken("token2")

	require.Equal(t, 2, len(tk.jwtRegistry))
	tk.cleanUpTokens()
	require.Equal(t, 0, len(tk.jwtRegistry))
}

func TestCleanupTokensKeepsValidTokens(t *testing.T) {
	tk, err := NewTokenKeeper()
	require.NoError(t, err)

	validToken, err := tk.IssueToken()
	require.NoError(t, err)

	expiredJWT, err := jwt.NewWithClaims(jwt.SigningMethodHS256, &jwt.RegisteredClaims{
		ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(-time.Second)},
		Issuer:    issuer,
	}).SignedString(tk.signingKey)
	require.NoError(t, err)

	tk.storeToken(expiredJWT)
	tk.storeToken("malformed")

	require.Equal(t, 3, len(tk.jwtRegistry))

	tk.cleanUpTokens()

	require.Equal(t, 1, len(tk.jwtRegistry))
	require.NoError(t, tk.ValidateToken(validToken))
	require.EqualError(t, tk.ValidateToken(expiredJWT), "token not found")
	require.EqualError(t, tk.ValidateToken("malformed"), "token not found")
}

func TestCleanupTokensConcurrentWithStore(t *testing.T) {
	tk, err := NewTokenKeeper()
	require.NoError(t, err)

	const iterations = 500

	done := make(chan struct{})

	go func() {
		defer close(done)

		for i := 0; i < iterations; i++ {
			tk.storeToken(fmt.Sprintf("token-%d", i))
		}
	}()

	for i := 0; i < iterations; i++ {
		tk.cleanUpTokens()
	}

	<-done

	tk.cleanUpTokens()
	require.Equal(t, 0, len(tk.jwtRegistry))
}

func TestRandomToken(t *testing.T) {
	token, err := generateRandomToken(5)
	require.Nil(t, err)
	require.Equal(t, 5, len(token))
}
