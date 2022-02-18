//go:build windows
// +build windows

/*
2020 © Postgres.ai
*/

// Package pool provides components to work with storage pools.
package pool

func (pm *Manager) getFSInfo(path string) (string, error) {
	// Not supported for windows.
	return "", nil
}
