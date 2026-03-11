package projection

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/ptypes"
)

type mockAccessor struct {
	sets []FieldSet
}

func (m *mockAccessor) Set(set FieldSet) error {
	m.sets = append(m.sets, set)
	return nil
}

func (m *mockAccessor) Get(get FieldGet) (interface{}, error) {
	return nil, nil
}

func TestStore_NilSliceSkipsSet(t *testing.T) {
	type sliceStruct struct {
		Name  string        `proj:"nested.name"`
		Items []interface{} `proj:"nested.items"`
	}

	s := &sliceStruct{Name: "test"}
	acc := &mockAccessor{}

	err := Store(s, acc, StoreOptions{})
	require.NoError(t, err)
	require.Len(t, acc.sets, 1)
	require.Equal(t, []string{"nested", "name"}, acc.sets[0].Path)
	require.Equal(t, "test", acc.sets[0].Value)
}

func TestStore_NilMapSkipsSet(t *testing.T) {
	type mapStruct struct {
		Name   string                 `proj:"nested.name"`
		Config map[string]interface{} `proj:"nested.config"`
	}

	s := &mapStruct{Name: "test"}
	acc := &mockAccessor{}

	err := Store(s, acc, StoreOptions{})
	require.NoError(t, err)
	require.Len(t, acc.sets, 1)
	require.Equal(t, []string{"nested", "name"}, acc.sets[0].Path)
}

func TestStore_EmptyNonNilSliceCallsSet(t *testing.T) {
	type sliceStruct struct {
		Items []interface{} `proj:"nested.items"`
	}

	s := &sliceStruct{Items: []interface{}{}}
	acc := &mockAccessor{}

	err := Store(s, acc, StoreOptions{})
	require.NoError(t, err)
	require.Len(t, acc.sets, 1)
	require.Equal(t, []string{"nested", "items"}, acc.sets[0].Path)
	require.Equal(t, ptypes.Slice, acc.sets[0].Type)
}

func TestStore_EmptyNonNilMapCallsSet(t *testing.T) {
	type mapStruct struct {
		Config map[string]interface{} `proj:"nested.config"`
	}

	s := &mapStruct{Config: map[string]interface{}{}}
	acc := &mockAccessor{}

	err := Store(s, acc, StoreOptions{})
	require.NoError(t, err)
	require.Len(t, acc.sets, 1)
	require.Equal(t, []string{"nested", "config"}, acc.sets[0].Path)
	require.Equal(t, ptypes.Map, acc.sets[0].Type)
}
