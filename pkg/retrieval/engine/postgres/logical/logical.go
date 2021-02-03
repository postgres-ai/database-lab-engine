/*
2020 © Postgres.ai
*/

// Package logical provides jobs for logical initial operations.
package logical

import (
	"strconv"
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
