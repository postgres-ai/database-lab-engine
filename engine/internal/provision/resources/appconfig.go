/*
2020 © Postgres.ai
*/

package resources

import (
	"path"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/branching"
)

// AppConfig currently stores Postgres configuration (other application in the future too).
type AppConfig struct {
	CloneName      string
	Branch         string
	Revision       int
	DockerImage    string
	Pool           *Pool
	Host           string
	Port           uint
	DB             *DB
	NetworkID      string
	ProvisionHosts string

	ContainerConf map[string]string
	pgExtraConf   map[string]string
}

// DB describes a default database configuration.
type DB struct {
	Username string
	DBName   string
}

// CloneDir returns the path of the clone directory.
func (c *AppConfig) CloneDir() string {
	// TODO(akartasov): Move to pool.
	return path.Join(c.Pool.ClonesDir(c.Branch), c.CloneName, branching.RevisionSegment(c.Revision))
}

// DataDir returns the path of clone data.
func (c *AppConfig) DataDir() string {
	// TODO(akartasov): Move to pool.
	return path.Join(c.Pool.ClonesDir(c.Branch), c.CloneName, branching.RevisionSegment(c.Revision), c.Pool.DataSubDir)
}

// ExtraConf returns a map with an extra configuration.
func (c *AppConfig) ExtraConf() map[string]string {
	return c.pgExtraConf
}

// SetExtraConf sets a map with an extra configuration.
func (c *AppConfig) SetExtraConf(extraConf map[string]string) {
	c.pgExtraConf = extraConf
}
