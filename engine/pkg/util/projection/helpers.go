// Package projection helps to bind struct fields to json/yaml paths.
package projection

import (
	"gopkg.in/yaml.v3"
)

// LoadYaml loads struct fields from yaml document.
func LoadYaml(target interface{}, yaml *yaml.Node, options LoadOptions) error {
	soft, err := NewSoftYaml(yaml)
	if err != nil {
		return err
	}

	return Load(target, soft, options)
}

// StoreYaml stores struct fields to yaml document.
func StoreYaml(target interface{}, yaml *yaml.Node, options StoreOptions) error {
	soft, err := NewSoftYaml(yaml)
	if err != nil {
		return err
	}

	return Store(target, soft, options)
}

// LoadJSON loads struct fields from json document.
func LoadJSON(target interface{}, m map[string]interface{}, options LoadOptions) error {
	soft := NewSoftJSON(m)
	return Load(target, soft, options)
}

// StoreJSON stores struct fields to json document.
func StoreJSON(target interface{}, m map[string]interface{}, options StoreOptions) error {
	soft := NewSoftJSON(m)
	return Store(target, soft, options)
}
