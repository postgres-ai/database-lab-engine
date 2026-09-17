// Package yaml contains utilities to work with YAML nodes
package yaml

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// MaskValue replaces a sensitive value in a config shown to a client. A client that posts it back
// for a key asks the engine to keep the stored value.
const MaskValue = "****"

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

// Yaml masks the values at the configured paths. A scalar is replaced as a whole; a mapping keeps
// its keys and has each scalar value masked, so the shape of the document stays readable.
func (c *Mask) Yaml(node *yaml.Node) {
	for i := 0; i < len(c.paths); i++ {
		child, found := FindNodeAtPath(node, c.paths[i])
		if !found {
			continue
		}

		maskNode(child)
	}
}

func maskNode(node *yaml.Node) {
	switch node.Kind {
	case yaml.ScalarNode:
		node.Value = MaskValue
		node.Tag = "!!str"

	case yaml.MappingNode:
		for i := 1; i < len(node.Content); i += 2 {
			if node.Content[i].Kind == yaml.ScalarNode {
				maskNode(node.Content[i])
			}
		}
	}
}
