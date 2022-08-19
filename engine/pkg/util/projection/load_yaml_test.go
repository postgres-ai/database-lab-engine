package projection

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadYaml(t *testing.T) {
	r := require.New(t)
	s := &testStruct{}
	node := getYamlNormal(t)

	err := LoadYaml(s, node, LoadOptions{})
	r.NoError(err)

	requireMissEmpty(t, s)
	requireComplete(t, s)
}

func TestLoadYamlNull(t *testing.T) {
	r := require.New(t)
	s := fullTestStruct()
	node := getYamlNull(t)

	err := LoadYaml(s, node, LoadOptions{})
	r.NoError(err)

	requireEmpty(t, s)
}
