package envvar

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandStrict(t *testing.T) {
	t.Setenv("DBLAB_TOKEN", "secret-from-env")
	t.Setenv("EMPTY_VAR", "")

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr string
	}{
		{name: "empty string", input: "", want: ""},
		{name: "plain value", input: "plain-secret", want: "plain-secret"},
		{name: "braced placeholder", input: "${DBLAB_TOKEN}", want: "secret-from-env"},
		{name: "unbraced placeholder", input: "$DBLAB_TOKEN", want: "secret-from-env"},
		{name: "explicitly empty env var", input: "${EMPTY_VAR}", want: ""},
		{name: "unset variable", input: "${DBLAB_MISSING}", wantErr: `environment variable "DBLAB_MISSING" is not set`},
		{name: "regex backreference looks like unset var", input: "***$1", wantErr: `environment variable "1" is not set`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExpandStrict(tc.input)
			if tc.wantErr != "" {
				require.EqualError(t, err, tc.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestExpandFields(t *testing.T) {
	t.Setenv("FIELD_A", "value-a")
	t.Setenv("FIELD_B", "value-b")

	t.Run("expands all fields in place", func(t *testing.T) {
		a := "${FIELD_A}"
		b := "${FIELD_B}"
		c := "plain"

		require.NoError(t, ExpandFields([]Field{
			{Name: "a", Ptr: &a},
			{Name: "b", Ptr: &b},
			{Name: "c", Ptr: &c},
		}))

		assert.Equal(t, "value-a", a)
		assert.Equal(t, "value-b", b)
		assert.Equal(t, "plain", c)
	})

	t.Run("wraps error with field name on missing var", func(t *testing.T) {
		token := "${MISSING_VAR}"

		err := ExpandFields([]Field{{Name: "server.verificationToken", Ptr: &token}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "config field server.verificationToken")
		assert.Contains(t, err.Error(), `"MISSING_VAR" is not set`)
	})

	t.Run("skips nil pointer", func(t *testing.T) {
		require.NoError(t, ExpandFields([]Field{{Name: "skip", Ptr: nil}}))
	})
}
