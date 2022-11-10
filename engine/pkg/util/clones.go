/*
2020 © Postgres.ai
*/

package util

import (
	"strconv"
)

const (
	// ClonePrefix defines a Database Lab clone prefix.
	ClonePrefix = "dblab_clone_"
)

// GetCloneName returns clone name.
func GetCloneName(port uint) string {
	return ClonePrefix + strconv.FormatUint(uint64(port), 10)
}

// GetCloneNameStr returns clone name.
func GetCloneNameStr(port string) string {
	return ClonePrefix + port
}

// GetPoolName returns pool name.
func GetPoolName(basePool, snapshotSuffix string) string {
	return basePool + "/" + snapshotSuffix
}
