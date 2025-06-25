/*
2020 © Postgres.ai
*/

package resources

import (
	"path"
	"sync"
	"time"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/branching"
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
func (p *Pool) ObserverDir(branch, name string, revision int) string {
	return path.Join(p.ClonePath(branch, name, revision), p.ObserverSubDir)
}

// ClonesDir returns a path to the clones directory of the storage pool.
func (p *Pool) ClonesDir(branch string) string {
	return path.Join(p.MountDir, p.PoolDirName, branching.BranchDir, branch)
}

// ClonePath returns a path to the data clone directory.
func (p *Pool) ClonePath(branchName, name string, revision int) string {
	return path.Join(p.MountDir, p.PoolDirName, branching.BranchDir, branchName, name, branching.RevisionSegment(revision), p.DataSubDir)
}

// CloneLocation returns a path to the initialized clone directory.
func (p *Pool) CloneLocation(branchName, name string, revision int) string {
	return path.Join(p.MountDir, p.PoolDirName, branching.BranchDir, branchName, name, branching.RevisionSegment(revision))
}

// CloneRevisionLocation returns a path to the clone revisions.
func (p *Pool) CloneRevisionLocation(branchName, name string) string {
	return path.Join(p.MountDir, p.PoolDirName, branching.BranchDir, branchName, name)
}

// SocketCloneDir returns a path to the socket clone directory.
func (p *Pool) SocketCloneDir(name string) string {
	return path.Join(p.SocketDir(), name)
}

// BranchName returns a full branch name in the data pool.
func (p *Pool) BranchName(poolName, branchName string) string {
	return branching.BranchName(poolName, branchName)
}

// CloneDataset returns a full clone dataset in the data pool.
func (p *Pool) CloneDataset(branchName, cloneName string) string {
	return branching.CloneDataset(p.Name, branchName, cloneName)
}

// CloneName returns a full clone name in the data pool.
func (p *Pool) CloneName(branchName, cloneName string, revision int) string {
	return branching.CloneName(p.Name, branchName, cloneName, revision)
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
