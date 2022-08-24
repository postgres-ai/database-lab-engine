package projection

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStoreJson(t *testing.T) {
	r := require.New(t)
	s := fullTestStruct()
	node := map[string]interface{}{}

	err := StoreJSON(s, node, StoreOptions{})
	r.NoError(err)

	var expected = map[string]interface{}{
		"nested": map[string]interface{}{
			"string": "string",
			"int":    int64(1),
			"float":  1.1,
			"bool":   true,
			"miss":   "miss",
			"missMap": map[string]interface{}{
				"nested": "missNested",
			},
			"create": "create",

			"ptrString": "string",
			"ptrInt":    int64(1),
			"ptrFloat":  1.1,
			"ptrBool":   true,
			"ptrMiss":   "ptrMiss",
			"ptrMissMap": map[string]interface{}{
				"nested": "ptrMissNested",
			},
			"ptrCreate": "ptrCreate",
		},
	}
	r.EqualValues(expected, node)
}

func TestStoreJsonNull(t *testing.T) {
	r := require.New(t)
	s := &testStruct{}
	node := getJSONNormal()

	err := StoreJSON(s, node, StoreOptions{})
	r.NoError(err)

	expected := map[string]interface{}{
		"nested": map[string]interface{}{
			"string": "",
			"int":    int64(0),
			"float":  0.0,
			"bool":   false,
			"miss":   "",
			"missMap": map[string]interface{}{
				"nested": "",
			},
			"create": "",

			"ptrString": "string",
			"ptrInt":    int64(1),
			"ptrFloat":  1.1,
			"ptrBool":   true,
			"ptrCreate": "ptrCreate",
		},
	}

	r.EqualValues(expected, node)
}
