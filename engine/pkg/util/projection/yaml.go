package projection

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/envvar"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/ptypes"
)

type yamlSoft struct {
	root     *yaml.Node
	document *yaml.Node
}

// NewSoftYaml creates a new yaml accessor
func NewSoftYaml(
	document *yaml.Node,
) (Accessor, error) {
	if document.Kind != yaml.DocumentNode {
		return nil, fmt.Errorf("document is not a document node")
	}

	if len(document.Content) != 1 {
		return nil, fmt.Errorf("document has more than one child")
	}

	if document.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("document has no mapping node")
	}

	return &yamlSoft{
		root:     document.Content[0],
		document: document,
	}, nil
}

func (y *yamlSoft) Set(set FieldSet) error {
	node := y.root
	for i, key := range set.Path {
		if node.Kind != yaml.MappingNode {
			return fmt.Errorf("node is not a mapping node")
		}

		isLastSegment := i == len(set.Path)-1

		var (
			child    *yaml.Node
			hasChild bool
		)

		if isLastSegment && set.CreateKey {
			child, hasChild = findDirectNodeForKey(node, key)
		} else {
			child, hasChild = findNodeForKey(node, key)
		}

		if !hasChild {
			if set.CreateKey && isLastSegment {
				child = &yaml.Node{
					Kind: yaml.ScalarNode,
					Tag:  "!!map",
				}
				keyNode := &yaml.Node{
					Kind:  yaml.ScalarNode,
					Tag:   "!!str",
					Value: key,
				}
				node.Content = append(node.Content, keyNode, child)
			} else {
				return nil
			}
		}

		node = child
	}

	if set.Value == nil {
		return nil
	}

	// Encode rebuilds the node and drops its anchor, which would leave every
	// alias to it, such as a "<<: *db_configs" merge, dangling
	anchor := node.Anchor

	if mv, ok := set.Value.(map[string]interface{}); ok {
		if err := node.Encode(mv); err != nil {
			return fmt.Errorf("cannot encode map: %w", err)
		}

		node.Anchor = anchor

		return nil
	}

	if seq, ok := set.Value.([]interface{}); ok {
		if err := node.Encode(seq); err != nil {
			return fmt.Errorf("cannot encode slice: %w", err)
		}

		node.Anchor = anchor

		return nil
	}

	conv, err := ptypes.Convert(set.Value, ptypes.String)
	if err != nil {
		return err
	}

	node.Value = conv.(string)
	node.Tag = ptypeToNodeTag(set.Type)

	return nil
}

func (y *yamlSoft) Get(get FieldGet) (interface{}, error) {
	node := y.root
	for _, key := range get.Path {
		if node.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("node is not a mapping node")
		}

		child, hasChild := findNodeForKey(node, key)
		if !hasChild {
			return nil, nil
		}

		node = child
	}

	if node.Tag == "!!null" {
		return nil, nil
	}

	if node.Tag == "!!map" {
		return convertMap(node)
	}

	if node.Tag == "!!seq" {
		return convertSlice(node)
	}

	// a field still holding an environment placeholder has no value of the target
	// type yet: the admin API reads the config file raw, by design, so that saving
	// it cannot persist resolved secrets. Report it as absent rather than failing
	// the whole projection — Store then leaves the placeholder in place.
	if get.Type != ptypes.String && envvar.IsPlaceholder(node.Value) {
		return nil, nil
	}

	typed, err := ptypes.Convert(node.Value, get.Type)
	if err != nil {
		return nil, err
	}

	return typed, nil
}

// findDirectNodeForKey looks up a key in a mapping node's direct children only.
func findDirectNodeForKey(node *yaml.Node, key string) (*yaml.Node, bool) {
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1], true
		}
	}

	return nil, false
}

// findNodeForKey looks up a key in a mapping node, resolving YAML merge keys (<<: *alias).
func findNodeForKey(node *yaml.Node, key string) (*yaml.Node, bool) {
	if child, ok := findDirectNodeForKey(node, key); ok {
		return child, true
	}

	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Tag != "!!merge" {
			continue
		}

		merged := node.Content[i+1]
		if merged.Kind == yaml.AliasNode && merged.Alias != nil {
			merged = merged.Alias
		}

		if merged.Kind != yaml.MappingNode {
			continue
		}

		if child, ok := findNodeForKey(merged, key); ok {
			return child, true
		}
	}

	return nil, false
}

func convertMap(node *yaml.Node) (map[string]interface{}, error) {
	convertedMap := make(map[string]interface{}, 0)

	for i := 0; i < len(node.Content); i += 2 {
		switch node.Content[i+1].Tag {
		case "!!null":
			convertedMap[node.Content[i].Value] = struct{}{}

		case "!!map":
			nestedMap, err := convertMap(node.Content[i+1])
			if err != nil {
				return nil, err
			}

			convertedMap[node.Content[i].Value] = nestedMap

		case "!!seq":
			slice, err := convertSlice(node.Content[i+1])
			if err != nil {
				return nil, err
			}

			convertedMap[node.Content[i].Value] = slice

		default:
			typed, err := ptypes.Convert(node.Content[i+1].Value, nodeTagToPType(node.Content[i+1].Tag))
			if err != nil {
				return nil, err
			}

			convertedMap[node.Content[i].Value] = typed
		}
	}

	return convertedMap, nil
}

func convertSlice(node *yaml.Node) ([]interface{}, error) {
	stringSlice := []interface{}{}

	for _, nodeContent := range node.Content {
		typed, err := ptypes.Convert(nodeContent.Value, ptypes.String)
		if err != nil {
			return nil, fmt.Errorf("failed to convert a slice element: %w", err)
		}

		stringSlice = append(stringSlice, typed)
	}

	return stringSlice, nil
}

func ptypeToNodeTag(t ptypes.Type) string {
	switch t {
	case ptypes.String:
		return "!!str"
	case ptypes.Int64:
		return "!!int"
	case ptypes.Float64:
		return "!!float"
	case ptypes.Bool:
		return "!!bool"
	case ptypes.Map:
		return "!!map"
	case ptypes.Slice:
		return "!!seq"
	default:
		return ""
	}
}

func nodeTagToPType(nodeTag string) ptypes.Type {
	switch nodeTag {
	case "!!str":
		return ptypes.String
	case "!!int":
		return ptypes.Int64
	case "!!float":
		return ptypes.Float64
	case "!!bool":
		return ptypes.Bool
	case "!!map":
		return ptypes.Map
	case "!!seq":
		return ptypes.Slice
	default:
		return ptypes.Invalid
	}
}
