/*
2020 © Postgres.ai
*/

// Package options provides helpers to process retriever options.
package options

import (
	"gopkg.in/yaml.v2"
)

// Unmarshal converts configuration to specific options.
func Unmarshal(in, out interface{}) error {
	// TODO: Parse default yaml values in tags.
	b, err := yaml.Marshal(in)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(b, out)
}
