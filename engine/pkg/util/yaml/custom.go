// Package yaml contains utilities to work with YAML nodes
package yaml

import (
	"strings"

	"gopkg.in/yaml.v3"
)

var secretKeyList = []string{"secret", "key", "token", "password"}

// TraverseNode traverses node and mask sensitive keys.
func TraverseNode(node *yaml.Node) {
	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) < 1 {
			return
		}

		TraverseNode(node.Content[0])

	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			if node.Content[i+1].Kind == yaml.ScalarNode {
				if containsSecret(strings.ToLower(node.Content[i].Value)) {
					node.Content[i+1].Value = maskValue
					node.Content[i+1].Tag = "!!str"
				}

				continue
			}

			TraverseNode(node.Content[i+1])
		}
	}
}

func containsSecret(key string) bool {
	for _, secret := range secretKeyList {
		if strings.Contains(key, secret) {
			return true
		}
	}

	return false
}
