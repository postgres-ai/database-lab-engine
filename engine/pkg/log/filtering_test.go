package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilteringMatchesSecretsLiterally(t *testing.T) {
	f := newFiltering([]string{"p@$$w0rd-abcdef", "p@ssw0rd(unclosed"})

	assert.Equal(t, "token is ******** here", string(f.ReplaceAll([]byte("token is p@$$w0rd-abcdef here"))))
	assert.Equal(t, "token is ******** here", string(f.ReplaceAll([]byte("token is p@ssw0rd(unclosed here"))))
}
