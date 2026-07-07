package srv

import (
	"errors"
	"strings"

	"gopkg.in/yaml.v3"

	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/logical"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/snapshot"
	yamlUtils "gitlab.com/postgres-ai/database-lab/v3/pkg/util/yaml"
)

// ensureLogicalPipeline sets retrieval.jobs to a complete logical pipeline
// (dump, restore, snapshot) so a simplified install produces a working
// configuration. logicalRestore is omitted when the dump uses immediateRestore,
// which loads the data in the same step. Jobs whose spec is absent are skipped
// to keep the config valid.
func ensureLogicalPipeline(document *yaml.Node) error {
	retrievalNode, ok := yamlUtils.FindNodeAtPath(document, []string{"retrieval"})
	if !ok {
		return errors.New("retrieval section not found in config")
	}

	desired := []string{logical.DumpJobType}

	if !immediateRestoreEnabled(retrievalNode) {
		desired = append(desired, logical.RestoreJobType)
	}

	desired = append(desired, snapshot.LogicalSnapshotType)

	jobs := make([]*yaml.Node, 0, len(desired))

	for _, job := range desired {
		if _, ok := yamlUtils.FindNodeAtPath(retrievalNode, []string{"spec", job}); !ok {
			continue
		}

		jobs = append(jobs, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: job})
	}

	if len(jobs) == 0 {
		return errors.New("no logical job specs found in config")
	}

	setMappingValue(retrievalNode, "jobs", &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Content: jobs})

	return nil
}

func immediateRestoreEnabled(retrievalNode *yaml.Node) bool {
	node, ok := yamlUtils.FindNodeAtPath(retrievalNode,
		[]string{"spec", "logicalDump", "options", "immediateRestore", "enabled"})
	if !ok {
		return false
	}

	// match the YAML 1.1 truthy set that the config loader (yaml.v2) accepts,
	// so this detection agrees with how the engine reads the same flag.
	switch strings.ToLower(node.Value) {
	case "true", "yes", "on", "y":
		return true
	}

	return false
}

// setMappingValue sets the value node for key in a mapping node, replacing an
// existing value or appending a new key/value pair.
func setMappingValue(mapping *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content[i+1] = value
			return
		}
	}

	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		value,
	)
}
