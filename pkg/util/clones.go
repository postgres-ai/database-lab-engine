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

// GetCloneName returns a clone name.
func GetCloneName(port uint) string {
	return ClonePrefix + strconv.FormatUint(uint64(port), 10)
}
