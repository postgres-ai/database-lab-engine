package projection

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func getJSONNormal() map[string]interface{} {
	return map[string]interface{}{
		"nested": map[string]interface{}{
			"string":    "string",
			"int":       int64(1),
			"float":     1.1,
			"bool":      true,
			"ptrString": "string",
			"ptrInt":    int64(1),
			"ptrFloat":  1.1,
			"ptrBool":   true,
		},
	}
}

func getJSONNull() map[string]interface{} {
	return map[string]interface{}{
		"nested": map[string]interface{}{
			"string":    nil,
			"int":       nil,
			"float":     nil,
			"bool":      nil,
			"ptrString": nil,
			"ptrInt":    nil,
			"ptrFloat":  nil,
			"ptrBool":   nil,
		},
	}
}

const yamlNormal = `
nested:
  string: "string"
  int: 1
  float: 1.1
  bool: true
  ptrString: "string"
  ptrInt: 1
  ptrFloat: 1.1
  ptrBool: true
`

const yamlNull = `
nested:
  string: null
  int: null
  float: null
  bool: null
  ptrString: null
  ptrInt: null
  ptrFloat: null
  ptrBool: null
`

const yamlDiverted = `
nested:
  string: "to be stored"
  int: 200
  float: 200.2
  bool: false
  ptrString: "to be stored"
  ptrInt: 200
  ptrFloat: 200.2
  ptrBool: false
`

const yamlNullApplied = `
nested:
  string: ""
  int: 0
  float: 0.0
  bool: false
  ptrString: "string"
  ptrInt: 1
  ptrFloat: 1.1
  ptrBool: true
`

type testStruct struct {
	StringField string  `proj:"nested.string"`
	IntField    int64   `proj:"nested.int"`
	FloatField  float64 `proj:"nested.float"`
	BoolField   bool    `proj:"nested.bool"`

	PtrStringField *string  `proj:"nested.ptrString"`
	PtrIntField    *int64   `proj:"nested.ptrInt"`
	PtrFloatField  *float64 `proj:"nested.ptrFloat"`
	PtrBoolField   *bool    `proj:"nested.ptrBool"`

	MissField       string `proj:"nested.miss"`
	MissNestedField string `proj:"nested.missMap.nested"`

	PtrMissField       *string `proj:"nested.ptrMiss"`
	PtrMissNestedField *string `proj:"nested.ptrMissMap.nested"`
}

func fullTestStruct() *testStruct {
	strField := "string"
	intField := int64(1)
	floatField := 1.1
	boolField := true
	missField := "ptrMiss"
	missNestedField := "ptrMissNested"

	return &testStruct{
		StringField:        "string",
		IntField:           int64(1),
		FloatField:         1.1,
		BoolField:          true,
		MissField:          "miss",
		MissNestedField:    "missNested",
		PtrStringField:     &strField,
		PtrIntField:        &intField,
		PtrFloatField:      &floatField,
		PtrBoolField:       &boolField,
		PtrMissField:       &missField,
		PtrMissNestedField: &missNestedField,
	}
}

func getYamlNormal(t *testing.T) *yaml.Node {
	t.Helper()
	node := &yaml.Node{}
	err := yaml.Unmarshal([]byte(yamlNormal), node)
	require.NoError(t, err)
	return node
}

func getYamlNull(t *testing.T) *yaml.Node {
	t.Helper()
	node := &yaml.Node{}
	err := yaml.Unmarshal([]byte(yamlNull), node)
	require.NoError(t, err)
	return node
}

func getYamlDiverted(t *testing.T) *yaml.Node {
	t.Helper()
	node := &yaml.Node{}
	err := yaml.Unmarshal([]byte(yamlDiverted), node)
	require.NoError(t, err)
	return node
}

func requireEmpty(t *testing.T, s *testStruct) {
	t.Helper()
	require.Zero(t, s.StringField)
	require.Zero(t, s.IntField)
	require.Zero(t, s.FloatField)
	require.Zero(t, s.BoolField)
	require.Zero(t, s.MissField)
	require.Zero(t, s.MissNestedField)

	require.Nil(t, s.PtrStringField)
	require.Nil(t, s.PtrIntField)
	require.Nil(t, s.PtrFloatField)
	require.Nil(t, s.PtrBoolField)
	require.Nil(t, s.PtrMissField)
	require.Nil(t, s.PtrMissNestedField)
}

func requireMissEmpty(t *testing.T, s *testStruct) {
	t.Helper()
	require.Zero(t, s.MissField)
	require.Zero(t, s.MissNestedField)

	require.Nil(t, s.PtrMissField)
	require.Nil(t, s.PtrMissNestedField)
}

func requireComplete(t *testing.T, s *testStruct) {
	t.Helper()
	require.Equal(t, "string", s.StringField)
	require.Equal(t, int64(1), s.IntField)
	require.Equal(t, 1.1, s.FloatField)
	require.Equal(t, true, s.BoolField)

	require.Equal(t, "string", *s.PtrStringField)
	require.Equal(t, int64(1), *s.PtrIntField)
	require.Equal(t, 1.1, *s.PtrFloatField)
	require.Equal(t, true, *s.PtrBoolField)
}

func requireYamlNormal(t *testing.T, node *yaml.Node) {
	t.Helper()
	normal := &yaml.Node{}
	err := yaml.Unmarshal([]byte(yamlNormal), normal)
	require.NoError(t, err)
	require.EqualValues(t, normal, node)
}

func requireYamlNullApplied(t *testing.T, node *yaml.Node) {
	t.Helper()
	null := &yaml.Node{}
	err := yaml.Unmarshal([]byte(yamlNullApplied), null)
	require.NoError(t, err)
	require.EqualValues(t, null, node)
}
