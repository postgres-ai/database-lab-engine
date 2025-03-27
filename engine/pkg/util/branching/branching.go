/*
2023 © Postgres.ai
*/

// Package branching contains branching tools and types.
package branching

import (
	"fmt"
	"path"
	"strings"
)

const (
	// DefaultBranch defines the name of the default branch.
	DefaultBranch = "main"

	// DefaultRevison defines default clone revision.
	DefaultRevision = 0

	// BranchDir defines branch directory in the pool.
	BranchDir = "branch"
)

// BranchName returns a full branch name in the data pool.
func BranchName(poolName, branchName string) string {
	return path.Join(poolName, BranchDir, branchName)
}

// CloneDataset returns a full clone dataset in the data pool.
func CloneDataset(poolName, branchName, cloneName string) string {
	return path.Join(BranchName(poolName, branchName), cloneName)
}

// CloneName returns a full clone name in the data pool.
func CloneName(poolName, branchName, cloneName string, revision int) string {
	return path.Join(BranchName(poolName, branchName), cloneName, RevisionSegment(revision))
}

// RevisionSegment returns a clone path suffix depends on its revision.
func RevisionSegment(revision int) string {
	return fmt.Sprintf("r%d", revision)
}

// ParseCloneName parses clone name from the clone dataset.
func ParseCloneName(cloneDataset, poolName string) (string, bool) {
	const cloneSegmentNumber = 2

	splits := parseCloneDataset(cloneDataset, poolName)

	if len(splits) < cloneSegmentNumber {
		return "", false
	}

	cloneID := splits[1]

	return cloneID, true
}

// ParseBranchName parses branch name from the clone dataset.
func ParseBranchName(cloneDataset, poolName string) (string, bool) {
	splits := parseCloneDataset(cloneDataset, poolName)

	if len(splits) < 1 {
		return "", false
	}

	branch := splits[0]

	return branch, true
}

func parseCloneDataset(cloneDataset, poolName string) []string {
	const splitParts = 3

	// bcrStr contains branch, clone and revision.
	bcrStr := strings.TrimPrefix(cloneDataset, poolName+"/"+BranchDir+"/")

	// Parse branchName/cloneID/revision.
	splits := strings.SplitN(bcrStr, "/", splitParts)
	if len(splits) != splitParts {
		return nil
	}

	return splits
}
