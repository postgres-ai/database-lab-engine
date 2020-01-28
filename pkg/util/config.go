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

// GetBinRootPath return path to root directory of сurrent binary module.
func GetBinRootPath() (string, error) {
	bindir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return "", errors.Wrap(err, "failed to get abs filepath of a bin directory")
	}

	path, err := filepath.Abs(filepath.Dir(bindir))
	if err != nil {
		return "", errors.Wrap(err, "failed to get abs filepath of a root directory")
	}

	return path, nil
}

// GetSwaggerUIPath return swagger UI path.
func GetSwaggerUIPath() (string, error) {
	dir, err := GetBinRootPath()
	if err != nil {
		return "", errors.Wrap(err, "cannot get binary root directory")
	}

	return dir + string(os.PathSeparator) + "web" + string(os.PathSeparator), nil
}

// GetAPIPath return swagger UI path.
func GetAPIPath() (string, error) {
	dir, err := GetBinRootPath()
	if err != nil {
		return "", errors.Wrap(err, "cannot get binary root directory")
	}

	return dir + string(os.PathSeparator) + "api" + string(os.PathSeparator), nil
}

// GetConfigPath return path to configs directory.
func GetConfigPath(name string) (string, error) {
	dir, err := GetBinRootPath()
	if err != nil {
		return "", errors.Wrap(err, "failed to get abs filepath of a root directory")
	}

	path := dir + string(os.PathSeparator) + "configs" + string(os.PathSeparator) + name

	return path, nil
}
