package projection

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadJson(t *testing.T) {
	r := require.New(t)
	s := &testStruct{}

	err := LoadJSON(s, getJSONNormal(), LoadOptions{})
	r.NoError(err)

	requireComplete(t, s)
	requireMissEmpty(t, s)
}

func TestLoadJsonNull(t *testing.T) {
	r := require.New(t)
	s := fullTestStruct()

	err := LoadJSON(s, getJSONNull(), LoadOptions{})
	r.NoError(err)

	requireEmpty(t, s)
}
