package projection

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/ptypes"
)

const updatedName = "updated"

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

func TestStoreYaml_NilFieldBeforeNonNilFields(t *testing.T) {
	type nilFirstStruct struct {
		Items  []interface{}          `proj:"nested.items"`
		Config map[string]interface{} `proj:"nested.config"`
		Name   *string                `proj:"nested.name"`
	}

	const yamlData = `
nested:
  items:
    - "--no-privileges"
  config:
    key1: value1
  name: original
`

	node := &yaml.Node{}
	err := yaml.Unmarshal([]byte(yamlData), node)
	require.NoError(t, err)

	newName := updatedName
	s := &nilFirstStruct{Name: &newName}

	err = StoreYaml(s, node, StoreOptions{})
	require.NoError(t, err)

	soft, err := NewSoftYaml(node)
	require.NoError(t, err)

	nameVal, err := soft.Get(FieldGet{Path: []string{"nested", "name"}, Type: ptypes.String})
	require.NoError(t, err)
	require.Equal(t, updatedName, nameVal)

	itemsVal, err := soft.Get(FieldGet{Path: []string{"nested", "items"}, Type: ptypes.Slice})
	require.NoError(t, err)
	require.Equal(t, []interface{}{"--no-privileges"}, itemsVal)

	configVal, err := soft.Get(FieldGet{Path: []string{"nested", "config"}, Type: ptypes.Map})
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{"key1": "value1"}, configVal)
}

func TestStoreYaml_EmptyNonNilSliceOverwritesExisting(t *testing.T) {
	type sliceStruct struct {
		Name  *string       `proj:"nested.name"`
		Items []interface{} `proj:"nested.items"`
	}

	const yamlWithItems = `
nested:
  name: original
  items:
    - "--no-privileges"
    - "--no-owner"
`

	node := &yaml.Node{}
	err := yaml.Unmarshal([]byte(yamlWithItems), node)
	require.NoError(t, err)

	newName := updatedName
	s := &sliceStruct{Name: &newName, Items: []interface{}{}}

	err = StoreYaml(s, node, StoreOptions{})
	require.NoError(t, err)

	soft, err := NewSoftYaml(node)
	require.NoError(t, err)

	nameVal, err := soft.Get(FieldGet{Path: []string{"nested", "name"}, Type: ptypes.String})
	require.NoError(t, err)
	require.Equal(t, updatedName, nameVal)

	itemsVal, err := soft.Get(FieldGet{Path: []string{"nested", "items"}, Type: ptypes.Slice})
	require.NoError(t, err)
	require.Equal(t, []interface{}{}, itemsVal)
}

func TestStoreYaml_EmptyNonNilMapOverwritesExisting(t *testing.T) {
	type mapStruct struct {
		Name   *string                `proj:"nested.name"`
		Config map[string]interface{} `proj:"nested.config"`
	}

	const yamlWithMap = `
nested:
  name: original
  config:
    key1: value1
    key2: value2
`

	node := &yaml.Node{}
	err := yaml.Unmarshal([]byte(yamlWithMap), node)
	require.NoError(t, err)

	newName := updatedName
	s := &mapStruct{Name: &newName, Config: map[string]interface{}{}}

	err = StoreYaml(s, node, StoreOptions{})
	require.NoError(t, err)

	soft, err := NewSoftYaml(node)
	require.NoError(t, err)

	nameVal, err := soft.Get(FieldGet{Path: []string{"nested", "name"}, Type: ptypes.String})
	require.NoError(t, err)
	require.Equal(t, updatedName, nameVal)

	configVal, err := soft.Get(FieldGet{Path: []string{"nested", "config"}, Type: ptypes.Map})
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{}, configVal)
}

func TestStoreYaml_NilSlicePreservesExisting(t *testing.T) {
	type sliceStruct struct {
		Name  *string       `proj:"nested.name"`
		Items []interface{} `proj:"nested.items"`
	}

	const yamlWithItems = `
nested:
  name: original
  items:
    - "--no-privileges"
    - "--no-owner"
`

	node := &yaml.Node{}
	err := yaml.Unmarshal([]byte(yamlWithItems), node)
	require.NoError(t, err)

	newName := updatedName
	s := &sliceStruct{Name: &newName}

	err = StoreYaml(s, node, StoreOptions{})
	require.NoError(t, err)

	soft, err := NewSoftYaml(node)
	require.NoError(t, err)

	nameVal, err := soft.Get(FieldGet{Path: []string{"nested", "name"}, Type: ptypes.String})
	require.NoError(t, err)
	require.Equal(t, updatedName, nameVal)

	itemsVal, err := soft.Get(FieldGet{Path: []string{"nested", "items"}, Type: ptypes.Slice})
	require.NoError(t, err)
	require.Equal(t, []interface{}{"--no-privileges", "--no-owner"}, itemsVal)
}

func TestStoreYaml_NilMapPreservesExisting(t *testing.T) {
	type mapStruct struct {
		Name   *string                `proj:"nested.name"`
		Config map[string]interface{} `proj:"nested.config"`
	}

	const yamlWithMap = `
nested:
  name: original
  config:
    key1: value1
    key2: value2
`

	node := &yaml.Node{}
	err := yaml.Unmarshal([]byte(yamlWithMap), node)
	require.NoError(t, err)

	newName := updatedName
	s := &mapStruct{Name: &newName}

	err = StoreYaml(s, node, StoreOptions{})
	require.NoError(t, err)

	soft, err := NewSoftYaml(node)
	require.NoError(t, err)

	nameVal, err := soft.Get(FieldGet{Path: []string{"nested", "name"}, Type: ptypes.String})
	require.NoError(t, err)
	require.Equal(t, updatedName, nameVal)

	configVal, err := soft.Get(FieldGet{Path: []string{"nested", "config"}, Type: ptypes.Map})
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{"key1": "value1", "key2": "value2"}, configVal)
}
