package projection

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStoreYaml(t *testing.T) {
	r := require.New(t)
	s := fullTestStruct()
	node := getYamlDiverted(t)

	err := StoreYaml(s, node, StoreOptions{})
	r.NoError(err)

	requireYamlNormal(t, node)
}

func TestStoreYamlNull(t *testing.T) {
	r := require.New(t)
	s := &testStruct{}
	node := getYamlNormal(t)

	err := StoreYaml(s, node, StoreOptions{})
	r.NoError(err)

	// no changes should have been made to the node
	requireYamlNullApplied(t, node)
}
