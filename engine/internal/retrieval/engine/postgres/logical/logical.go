/*
2020 © Postgres.ai
*/

// Package logical provides jobs for logical initial operations.
package logical

import (
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/mount"
)

func buildAnalyzeCommand(conn Connection, parallelJobs int) []string {
	analyzeCmd := []string{
		"vacuumdb",
		"--analyze",
		"--jobs", strconv.Itoa(parallelJobs),
		"--username", conn.Username,
		"--all",
	}

	return analyzeCmd
}

func isAlreadyMounted(mounts []mount.Mount, dir string) bool {
	dir = strings.Trim(dir, "/")

	for _, mountPoint := range mounts {
		if strings.Trim(mountPoint.Source, "/") == dir || strings.Trim(mountPoint.Target, "/") == dir {
			return true
		}
	}

	return false
}
