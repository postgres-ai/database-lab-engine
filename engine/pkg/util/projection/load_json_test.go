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

func TestLoadJsonEmptyStringUnsetsNumericFields(t *testing.T) {
	r := require.New(t)
	s := fullTestStruct()

	err := LoadJSON(s, getJSONEmptyString(), LoadOptions{})
	r.NoError(err)

	r.Zero(s.IntField)
	r.Zero(s.FloatField)
	r.Zero(s.BoolField)
	r.Nil(s.PtrIntField)
	r.Nil(s.PtrFloatField)
	r.Nil(s.PtrBoolField)
}

func TestLoadJsonNull(t *testing.T) {
	r := require.New(t)
	s := fullTestStruct()

	err := LoadJSON(s, getJSONNull(), LoadOptions{})
	r.NoError(err)

	requireEmpty(t, s)
}
