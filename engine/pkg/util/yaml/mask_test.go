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
	r.Equal(maskValue, sensitive.Value)

	nonSensitive, _ := FindNodeAtPathString(node, "root.nonSensitive")
	r.NotNil(nonSensitive)
	r.Equal("123", nonSensitive.Value)
}
