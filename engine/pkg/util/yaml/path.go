// Package yaml Utilities to work with YAML nodes
package yaml

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// FindNodeAtPathString finds the node at the given path.
func FindNodeAtPathString(node *yaml.Node, path string) (*yaml.Node, bool) {
	return FindNodeAtPath(node, strings.Split(path, "."))
}

// FindNodeAtPath finds the node at the given path.
func FindNodeAtPath(node *yaml.Node, path []string) (*yaml.Node, bool) {
	if len(path) == 0 {
		return node, true
	}

	if node.Kind == yaml.DocumentNode {
		if len(node.Content) < 1 {
			return nil, false
		}

		return FindNodeAtPath(node.Content[0], path)
	}

	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content); i += 2 {
			if node.Content[i].Value == path[0] {
				return FindNodeAtPath(node.Content[i+1], path[1:])
			}
		}
	}

	return nil, false
}
