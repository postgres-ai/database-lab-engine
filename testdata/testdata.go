/*
2021 © Postgres.ai
*/

// Package testdata contains data for running tests.
package testdata

import (
	"path"
	"runtime"
)

// GetTestDataDir provides the current directory name.
func GetTestDataDir() string {
	_, filename, _, _ := runtime.Caller(0)

	return path.Dir(filename)
}
