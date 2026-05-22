package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvironmentResolvedToken(t *testing.T) {
	t.Run("expands environment placeholders", func(t *testing.T) {
		t.Setenv("DBLAB_TOKEN", "secret-from-env")

		env := Environment{Token: "${DBLAB_TOKEN}"}

		got, err := env.ResolvedToken()
		require.NoError(t, err)
		assert.Equal(t, "secret-from-env", got)
	})

	t.Run("preserves plain values", func(t *testing.T) {
		env := Environment{Token: "plain-secret"}

		got, err := env.ResolvedToken()
		require.NoError(t, err)
		assert.Equal(t, "plain-secret", got)
	})

	t.Run("empty token round-trips without lookup", func(t *testing.T) {
		env := Environment{Token: ""}

		got, err := env.ResolvedToken()
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("errors on unset variable", func(t *testing.T) {
		env := Environment{Token: "${DBLAB_TOKEN_MISSING}"}

		_, err := env.ResolvedToken()
		require.Error(t, err)
		assert.Contains(t, err.Error(), `"DBLAB_TOKEN_MISSING" is not set`)
	})
}
