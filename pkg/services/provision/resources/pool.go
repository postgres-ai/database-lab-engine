/*
2020 © Postgres.ai
*/

package resources

import (
	"path"
	"time"

	"gitlab.com/postgres-ai/database-lab/pkg/util"
)

// Pool describes a storage pool.
type Pool struct {
	Name         string
	Mode         string
	DSA          time.Time
	MountDir     string
	CloneSubDir  string
	DataSubDir   string
	SocketSubDir string
}

// NewPool creates a new Pool.
func NewPool(name string) *Pool {
	return &Pool{Name: name}
}

// IsEmpty checks if Pool is empty.
func (p Pool) IsEmpty() bool {
	return p.Name == "" && p.Mode == ""
}

// SetDSA sets a dataStateAt value.
func (p Pool) SetDSA(dsa time.Time) {
	p.DSA = dsa
}

// DataDir returns a path to the data directory of the storage pool.
func (p Pool) DataDir() string {
	return path.Join(p.MountDir, p.Name, p.DataSubDir)
}

// SocketDir returns a path to the sockets directory of the storage pool.
func (p Pool) SocketDir() string {
	return path.Join(p.MountDir, p.Name, p.SocketSubDir)
}

// ClonesDir returns a path to the clones directory of the storage pool.
func (p Pool) ClonesDir() string {
	return path.Join(p.MountDir, p.Name, p.CloneSubDir)
}

// ClonePath returns a path to the initialized clone directory.
func (p Pool) ClonePath(port uint) string {
	return path.Join(p.MountDir, p.Name, p.CloneSubDir, util.GetCloneName(port), p.DataSubDir)
}

// SocketCloneDir returns a path to the socket clone directory.
func (p Pool) SocketCloneDir(name string) string {
	return path.Join(p.SocketDir(), name)
}
