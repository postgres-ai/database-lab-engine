package projection

import "gitlab.com/postgres-ai/database-lab/v3/pkg/util/ptypes"

// FieldSet represents a field to be set in the accessor.
type FieldSet struct {
	Path      []string
	Value     interface{}
	Type      ptypes.Type
	CreateKey bool
}

// FieldGet is used to retrieve a field value from an accessor.
type FieldGet struct {
	Path []string
	Type ptypes.Type
}

// Accessor is an interface for getting and setting values from a json / yaml / anything else
type Accessor interface {
	Set(set FieldSet) error
	Get(get FieldGet) (interface{}, error)
}
