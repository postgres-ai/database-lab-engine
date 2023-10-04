/*
2020 © Postgres.ai
*/

// Package fs provides tools for working with the filesystem.
package fs

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
)

const (
	logDirectory = "log"
)

// CopyDirectoryContent copies all files from one directory to another.
func CopyDirectoryContent(sourceDir, dataDir string) error {
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		sourcePath := filepath.Join(sourceDir, entry.Name())
		destPath := filepath.Join(dataDir, entry.Name())

		if err := CopyFile(sourcePath, destPath); err != nil {
			return err
		}
	}

	return nil
}

// CopyFile copies a file from one location to another.
func CopyFile(sourceFilename, destinationFilename string) error {
	dst, err := os.Create(destinationFilename)
	if err != nil {
		return err
	}

	defer func() { _ = dst.Close() }()

	src, err := os.Open(sourceFilename)
	if err != nil {
		return err
	}

	defer func() { _ = src.Close() }()

	_, err = io.Copy(dst, src)
	if err != nil {
		return err
	}

	return nil
}

// AppendFile appends data to a file.
func AppendFile(file string, data []byte) error {
	configFile, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	defer func() { _ = configFile.Close() }()

	if _, err := configFile.Write(data); err != nil {
		return err
	}

	return nil
}

// CleanupLogsDir removes old log files from the clone directory.
func CleanupLogsDir(dataDir string) error {
	logPath := path.Join(dataDir, logDirectory)

	logDir, err := os.ReadDir(logPath)
	if err != nil {
		return fmt.Errorf("cannot read directory %s: %v", logPath, err.Error())
	}

	for _, logFile := range logDir {
		logName := path.Join(logPath, logFile.Name())
		if err := os.RemoveAll(logName); err != nil {
			return fmt.Errorf("cannot remove %s: %v", logName, err.Error())
		}
	}

	return nil
}
