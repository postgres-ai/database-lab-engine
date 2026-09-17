package yaml

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const yamlStr = `
root:
  sensitive: "fromValue"
  nonSensitive: 123
`

func TestMask(t *testing.T) {
	r := require.New(t)
	node := &yaml.Node{}

	err := yaml.Unmarshal([]byte(yamlStr), node)
	r.NoError(err)

	mask := NewMask([]string{"root.sensitive"})
	mask.Yaml(node)

	sensitive, _ := FindNodeAtPathString(node, "root.sensitive")
	r.NotNil(sensitive)
	r.Equal(MaskValue, sensitive.Value)

	nonSensitive, _ := FindNodeAtPathString(node, "root.nonSensitive")
	r.NotNil(nonSensitive)
	r.Equal("123", nonSensitive.Value)
}

func TestMask_MappingKeepsKeysAndMasksValues(t *testing.T) {
	const doc = `
retrieval:
  spec:
    physicalRestore:
      options:
        tool: walg
        envs:
          AWS_ACCESS_KEY_ID: AKIAIOSFODNN7EXAMPLE
          AWS_SECRET_ACCESS_KEY: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
          WALG_S3_PREFIX: s3://bucket/path
          PGBACKREST_REPO1_RETENTION_FULL: 2
`

	node := &yaml.Node{}
	require.NoError(t, yaml.Unmarshal([]byte(doc), node))

	DefaultConfigMask().Yaml(node)

	out, err := yaml.Marshal(node)
	require.NoError(t, err)

	for _, secret := range []string{"AKIAIOSFODNN7EXAMPLE", "wJalrXUtnFEMI", "s3://bucket/path"} {
		require.NotContains(t, string(out), secret)
	}

	for _, key := range []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "WALG_S3_PREFIX", "PGBACKREST_REPO1_RETENTION_FULL"} {
		value, found := FindNodeAtPathString(node, "retrieval.spec.physicalRestore.options.envs."+key)
		require.True(t, found, key)
		require.Equal(t, MaskValue, value.Value, key)
		require.Equal(t, "!!str", value.Tag, key)
	}

	tool, _ := FindNodeAtPathString(node, "retrieval.spec.physicalRestore.options.tool")
	require.Equal(t, "walg", tool.Value)
}
