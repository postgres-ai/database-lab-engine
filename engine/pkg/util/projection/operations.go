package projection

import (
	"reflect"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/ptypes"
)

// Load reads the values of the fields of the target struct from the accessor.
func Load(target interface{}, accessor Accessor, options LoadOptions) error {
	return forEachField(target, func(tag *fieldTag, field reflect.Value) error {
		if !tag.matchesLoad(options) {
			return nil
		}

		accessorValue, err := accessor.Get(FieldGet{
			Path: tag.path, Type: tag.fType,
		})
		if err != nil {
			return err
		}

		if accessorValue == nil {
			field.Set(reflect.Zero(field.Type()))
			return nil
		}

		if tag.isPtr {
			setValue := ptypes.NewPtr(accessorValue)
			if setValue.IsValid() {
				field.Set(setValue)
			}
		} else {
			field.Set(reflect.ValueOf(accessorValue))
		}

		return nil
	},
	)
}

// Store writes the values of the fields of the target struct to the accessor.
func Store(target interface{}, accessor Accessor, options StoreOptions) error {
	return forEachField(target, func(tag *fieldTag, field reflect.Value) error {
		if !tag.matchesStore(options) {
			return nil
		}

		var accessorValue interface{}

		if tag.isPtr {
			if field.IsNil() {
				return nil
			}

			accessorValue = field.Elem().Interface()
		} else {
			accessorValue = field.Interface()
		}

		err := accessor.Set(FieldSet{
			Path: tag.path, Value: accessorValue, Type: tag.fType,
			CreateKey: tag.createKey,
		})

		if err != nil {
			return err
		}

		return nil
	},
	)
}

func forEachField(target interface{}, fn func(tag *fieldTag, field reflect.Value) error) error {
	value := reflect.Indirect(
		reflect.ValueOf(target),
	)

	if value.Kind() != reflect.Struct {
		return errors.Errorf("target must be a struct")
	}

	if !value.CanAddr() {
		return errors.Errorf("target must be addressable")
	}

	valueType := value.Type()
	num := value.NumField()

	for i := 0; i < num; i++ {
		field := value.Field(i)
		fieldType := valueType.Field(i)

		tag, err := getFieldTag(fieldType)
		if err != nil {
			return err
		}

		if tag == nil {
			continue
		}

		err = fn(tag, field)
		if err != nil {
			return err
		}
	}

	return nil
}
