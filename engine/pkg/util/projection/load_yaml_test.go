package projection

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestLoadYaml(t *testing.T) {
	r := require.New(t)
	s := &testStruct{}
	node := getYamlNormal(t)

	err := LoadYaml(s, node, LoadOptions{})
	r.NoError(err)

	requireMissEmpty(t, s)
	requireComplete(t, s)
}

func TestLoadYamlNull(t *testing.T) {
	r := require.New(t)
	s := fullTestStruct()
	node := getYamlNull(t)

	err := LoadYaml(s, node, LoadOptions{})
	r.NoError(err)

	requireEmpty(t, s)
}

func TestLoadYaml_MergeKey(t *testing.T) {
	type mergeStruct struct {
		Configs      map[string]interface{} `proj:"parent.options.configs"`
		ParallelJobs *int64                 `proj:"parent.options.parallelJobs"`
	}

	const yamlData = `
defaults: &defaults
  configs:
    shared_buffers: 1GB
    work_mem: 100MB

parent:
  options:
    <<: *defaults
    parallelJobs: 4
`

	node := &yaml.Node{}
	err := yaml.Unmarshal([]byte(yamlData), node)
	require.NoError(t, err)

	s := &mergeStruct{}
	err = LoadYaml(s, node, LoadOptions{})
	require.NoError(t, err)

	require.Equal(t, map[string]interface{}{"shared_buffers": "1GB", "work_mem": "100MB"}, s.Configs)
	require.Equal(t, int64(4), *s.ParallelJobs)
}

func TestLoadYaml_EnvPlaceholder(t *testing.T) {
	type placeholderStruct struct {
		Debug *bool   `proj:"global.debug"`
		Port  *int64  `proj:"connection.port"`
		Host  *string `proj:"connection.host"`
	}

	const yamlData = `
global:
  debug: ${DBLAB_DEBUG}
connection:
  host: ${SOURCE_HOST}
  port: ${SOURCE_PORT}
`

	node := &yaml.Node{}
	require.NoError(t, yaml.Unmarshal([]byte(yamlData), node))

	s := &placeholderStruct{}
	require.NoError(t, LoadYaml(s, node, LoadOptions{}))

	require.Nil(t, s.Debug, "an unresolved placeholder has no boolean value yet")
	require.Nil(t, s.Port, "an unresolved placeholder has no numeric value yet")
	require.Equal(t, "${SOURCE_HOST}", *s.Host, "a string field keeps the placeholder verbatim")

	require.NoError(t, StoreYaml(s, node, StoreOptions{}))

	stored, err := yaml.Marshal(node)
	require.NoError(t, err)
	require.Contains(t, string(stored), "${DBLAB_DEBUG}", "storing must not overwrite a placeholder")
	require.Contains(t, string(stored), "${SOURCE_PORT}")
}

func TestStoreYaml_KeepsAnchorOnReencodedMap(t *testing.T) {
	type configs struct {
		Configs map[string]interface{} `proj:"databaseConfigs.configs"`
	}

	doc := "databaseConfigs:\n  configs: &db_configs\n    shared_buffers: 1GB\nrestore:\n  options:\n    <<: *db_configs\n"

	node := &yaml.Node{}
	require.NoError(t, yaml.Unmarshal([]byte(doc), node))
	require.NoError(t, StoreYaml(&configs{Configs: map[string]interface{}{"shared_buffers": "2GB"}}, node, StoreOptions{}))

	stored, err := yaml.Marshal(node)
	require.NoError(t, err)
	require.Contains(t, string(stored), "configs: &db_configs", "the anchor survives the re-encode")

	var decoded map[string]interface{}
	require.NoError(t, yaml.Unmarshal(stored, &decoded), "the merge alias still resolves")
}
