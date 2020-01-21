/*
2019 © Postgres.ai
*/

// Package util provides utility functions. Config related utils.
package util

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// GetConfigPath return path to configs directory.
func GetConfigPath(name string) (string, error) {
	bindir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return "", errors.Wrap(err, "failed to get abs filepath of a bin directory")
	}

	dir, err := filepath.Abs(filepath.Dir(bindir))
	if err != nil {
		return "", errors.Wrap(err, "failed to get abs filepath of a root directory")
	}

	path := dir + string(os.PathSeparator) + "configs" + string(os.PathSeparator) + name

	return path, nil
}
