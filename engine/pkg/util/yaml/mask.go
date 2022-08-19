// Package yaml contains utilities to work with YAML nodes
package yaml

import (
	"strings"

	"gopkg.in/yaml.v3"
)

const maskValue = "****"

// Mask is a YAML masking utility
type Mask struct {
	paths [][]string
}

// NewMask creates a new YAML copy configuration.
func NewMask(paths []string) *Mask {
	c := &Mask{}

	for _, path := range paths {
		pathSplit := strings.Split(path, ".")
		c.paths = append(c.paths, pathSplit)
	}

	return c
}

// Yaml copies node values
func (c *Mask) Yaml(node *yaml.Node) {
	for i := 0; i < len(c.paths); i++ {
		child, found := FindNodeAtPath(node, c.paths[i])
		if !found {
			continue
		}

		if child.Kind != yaml.ScalarNode {
			continue
		}

		child.Value = maskValue
		child.Tag = "!!str"
	}
}
