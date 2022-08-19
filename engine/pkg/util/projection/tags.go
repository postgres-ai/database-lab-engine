package projection

import (
	"reflect"
	"strings"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/ptypes"
)

const projectionTag = "proj"
const projectionGroupTag = "groups"

type fieldTag struct {
	path   []string
	groups []string
	isPtr  bool
	fType  ptypes.Type
}

// LoadOptions is used to filter fields to load.
type LoadOptions struct {
	Groups []string
}

// StoreOptions is used to filter fields to store.
type StoreOptions struct {
	Groups []string
}

func getFieldTag(value reflect.StructField) (*fieldTag, error) {
	tag := value.Tag.Get(projectionTag)
	if tag == "" {
		return nil, nil
	}

	options := strings.Split(tag, ",")
	path := strings.Split(options[0], ".")

	var isPtr bool

	var fType ptypes.Type

	var groups []string

	groupTag := value.Tag.Get(projectionGroupTag)
	if groupTag == "" {
		groups = []string{"default"}
	} else {
		groups = strings.Split(groupTag, ",")
	}

	if value.Type.Kind() == reflect.Ptr {
		isPtr = true
		fType = ptypes.MapKindToType(value.Type.Elem().Kind())
	} else {
		isPtr = false
		fType = ptypes.MapKindToType(value.Type.Kind())
	}

	if fType == ptypes.Invalid {
		return nil, errors.Errorf("invalid type: %s", value.Type.Kind())
	}

	return &fieldTag{
		path:   path,
		fType:  fType,
		isPtr:  isPtr,
		groups: groups,
	}, nil
}

func (f *fieldTag) matchesStore(options StoreOptions) bool {
	if len(options.Groups) == 0 {
		return true
	}

	for _, group := range f.groups {
		for _, option := range options.Groups {
			if group == option {
				return true
			}
		}
	}

	return false
}

func (f *fieldTag) matchesLoad(options LoadOptions) bool {
	if len(options.Groups) == 0 {
		return true
	}

	for _, group := range f.groups {
		for _, option := range options.Groups {
			if group == option {
				return true
			}
		}
	}

	return false
}
