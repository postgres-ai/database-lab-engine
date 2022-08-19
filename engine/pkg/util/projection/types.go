package projection

import "gitlab.com/postgres-ai/database-lab/v3/pkg/util/ptypes"

// Accessor is an interface for getting and setting values from a json / yaml / anything else
type Accessor interface {
	Set(path []string, value interface{}, t ptypes.Type) error
	Get(path []string, t ptypes.Type) (interface{}, error)
}
