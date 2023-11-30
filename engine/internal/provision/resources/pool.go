/*
2020 © Postgres.ai
*/

package resources

import (
	"path"
	"sync"
	"time"
)

// PoolStatus represents a pool status.
type PoolStatus string

const (
	// ActivePool defines an active pool status.
	ActivePool PoolStatus = "active"
	// RefreshingPool defines the status of a pool when data retrieval in progress.
	RefreshingPool PoolStatus = "refreshing"
	// EmptyPool defines the status of an inactive pool.
	EmptyPool PoolStatus = "empty"

	// branchDir defines branch directory in the pool.
	branchDir = "branch"
)

// Pool describes a storage pool.
type Pool struct {
	Name           string
	Mode           string
	DSA            time.Time
	PoolDirName    string
	MountDir       string
	CloneSubDir    string
	DataSubDir     string
	SocketSubDir   string
	ObserverSubDir string
	mu             sync.RWMutex
	status         PoolStatus
}

// NewPool creates a new Pool.
func NewPool(name string) *Pool {
	return &Pool{Name: name, status: EmptyPool}
}

// IsEmpty checks if Pool is empty.
func (p *Pool) IsEmpty() bool {
	return p.Name == "" && p.Mode == ""
}

// SetDSA sets a dataStateAt value.
func (p *Pool) SetDSA(dsa time.Time) {
	p.DSA = dsa
}

// DataDir returns a path to the data directory of the storage pool.
func (p *Pool) DataDir() string {
	return path.Join(p.MountDir, p.PoolDirName, p.DataSubDir)
}

// SocketDir returns a path to the sockets directory of the storage pool.
func (p *Pool) SocketDir() string {
	return path.Join(p.MountDir, p.PoolDirName, p.SocketSubDir)
}

// ObserverDir returns a path to the observer directory of the storage pool.
func (p *Pool) ObserverDir(name string) string {
	return path.Join(p.ClonePath(name), p.ObserverSubDir)
}

// ClonesDir returns a path to the clones directory of the storage pool.
func (p *Pool) ClonesDir() string {
	return path.Join(p.MountDir, p.PoolDirName, p.CloneSubDir)
}

// ClonePath returns a path to the initialized clone directory.
func (p *Pool) ClonePath(name string) string {
	return path.Join(p.MountDir, p.PoolDirName, p.CloneSubDir, name, p.DataSubDir)
}

// SocketCloneDir returns a path to the socket clone directory.
func (p *Pool) SocketCloneDir(name string) string {
	return path.Join(p.SocketDir(), name)
}

// BranchDir returns a path to the branch directory of the storage pool.
func (p *Pool) BranchDir() string {
	return path.Join(p.MountDir, p.PoolDirName, branchDir)
}

// BranchPath returns a path to the specific branch in the storage pool.
func (p *Pool) BranchPath(branchName string) string {
	return path.Join(p.BranchDir(), branchName)
}

// Status gets the pool status.
func (p *Pool) Status() PoolStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.status
}

// SetStatus sets a status to the pool.
func (p *Pool) SetStatus(status PoolStatus) {
	p.mu.Lock()
	p.status = status
	p.mu.Unlock()
}
