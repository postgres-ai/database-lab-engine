package yaml

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const customYamlStr = `
global:
  debug: false
retrieval:
  spec:
    logicalDump:
      options:
        source:
          type: local # local, remote, rds, etc..
          connection:
            dbname: test_22
            host: 172.17.0.78
            port: 5455
            username: tony
            password: mypass
        databases: test1
        envs:
          AWS_SECRET_ACCESS_KEY: john
          PGBACKREST_REPO1_S3_KEY_SECRET: mysecretkey
          TEST_ENV: one
`

func TestTraverseNode(t *testing.T) {
	r := require.New(t)
	node := &yaml.Node{}

	err := yaml.Unmarshal([]byte(customYamlStr), node)
	r.NoError(err)
	TraverseNode(node)

	sensitive, found := FindNodeAtPathString(node, "retrieval.spec.logicalDump.options.envs.AWS_SECRET_ACCESS_KEY")
	r.NotNil(sensitive)
	r.True(found)
	r.Equal(maskValue, sensitive.Value)

	sensitive2, found := FindNodeAtPathString(node, "retrieval.spec.logicalDump.options.envs.PGBACKREST_REPO1_S3_KEY_SECRET")
	r.NotNil(sensitive2)
	r.True(found)
	r.Equal(maskValue, sensitive2.Value)

	nonSensitive, found := FindNodeAtPathString(node, "retrieval.spec.logicalDump.options.envs.TEST_ENV")
	r.NotNil(nonSensitive)
	r.True(found)
	r.Equal("one", nonSensitive.Value)

	password, found := FindNodeAtPathString(node, "retrieval.spec.logicalDump.options.source.connection.password")
	r.NotNil(password)
	r.True(found)
	r.Equal(maskValue, password.Value)

	host, found := FindNodeAtPathString(node, "retrieval.spec.logicalDump.options.source.connection.host")
	r.NotNil(host)
	r.True(found)
	r.Equal("172.17.0.78", host.Value)
}
