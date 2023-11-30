/*
2020 © Postgres.ai
*/

package util

const (
	// ClonePrefix defines a Database Lab clone prefix.
	ClonePrefix = "dblab_clone_"
)

// GetPoolName returns pool name.
func GetPoolName(basePool, snapshotSuffix string) string {
	return basePool + "/" + snapshotSuffix
}
