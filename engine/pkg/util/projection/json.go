package projection

import (
	"fmt"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/ptypes"
)

type softJSON struct {
	root map[string]interface{}
}

// NewSoftJSON creates a new JSON accessor.
func NewSoftJSON(root map[string]interface{}) Accessor {
	return &softJSON{root: root}
}

func (s *softJSON) Set(path []string, value interface{}, _ ptypes.Type) error {
	parent := s.root
	for _, key := range path[:len(path)-1] {
		child, hasChild := parent[key]
		if !hasChild {
			child = make(map[string]interface{})
			parent[key] = child
		}

		switch childTyped := child.(type) {
		case map[string]interface{}:
			parent = childTyped
		default:
			return fmt.Errorf("unsupported type: %T", childTyped)
		}
	}

	key := path[len(path)-1]

	child, ok := parent[key]
	if !ok {
		parent[key] = value
		return nil
	}

	switch child.(type) {
	case map[string]interface{}:
		return fmt.Errorf("node is already a mapping node")
	case []interface{}:
		return fmt.Errorf("node is already a sequence node")
	default:
		parent[key] = value
	}

	return nil
}

func (s *softJSON) Get(path []string, t ptypes.Type) (interface{}, error) {
	parent := s.root
	for _, key := range path[:len(path)-1] {
		child, hasChild := parent[key]
		if !hasChild {
			return nil, nil
		}

		switch childTyped := child.(type) {
		case map[string]interface{}:
			parent = childTyped
		default:
			return nil, fmt.Errorf("unsupported type: %T", childTyped)
		}
	}

	key := path[len(path)-1]

	child, ok := parent[key]
	if !ok {
		return nil, nil
	}

	if child == nil {
		return nil, nil
	}

	typed, err := ptypes.Convert(child, t)
	if err != nil {
		return nil, fmt.Errorf("failed to convert %#v: %w", child, err)
	}

	return typed, nil
}
