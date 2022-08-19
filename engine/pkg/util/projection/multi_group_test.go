package projection

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type structMulti struct {
	Yaml string `proj:"yaml.yamlValue" groups:"yaml"`
	JSON string `proj:"json.jsonValue" groups:"json"`
}

const yamlMulti = `
yaml:
  yamlValue: "yamlValue"
`

func getJSONMulti() map[string]interface{} {
	return map[string]interface{}{
		"json": map[string]interface{}{
			"jsonValue": "jsonValue",
		},
	}
}

func getYamlMulti(t *testing.T) *yaml.Node {
	t.Helper()
	node := &yaml.Node{}
	err := yaml.Unmarshal([]byte(yamlMulti), node)
	require.NoError(t, err)
	return node
}

func TestLoadJsonMulti(t *testing.T) {
	r := require.New(t)

	s := &structMulti{}
	err := LoadJSON(s, getJSONMulti(), LoadOptions{
		Groups: []string{"json"},
	})
	r.NoError(err)

	err = LoadYaml(s, getYamlMulti(t), LoadOptions{
		Groups: []string{"yaml"},
	})
	r.NoError(err)

	r.Equal("jsonValue", s.JSON)
	r.Equal("yamlValue", s.Yaml)
}
