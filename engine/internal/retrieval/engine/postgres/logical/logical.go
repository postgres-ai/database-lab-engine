/*
2020 © Postgres.ai
*/

// Package logical provides jobs for logical initial operations.
package logical

import (
	"strconv"
	"strings"

	"github.com/moby/moby/api/types/mount"
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

// isAlreadyMounted reports whether dir is reachable through one of the mounts: either mounted
// itself or located inside a writable mount target. A mount nested inside an inherited shared mount
// would be replayed onto the host on every container start, so a covered directory must not be
// mounted again. A mount of dir itself counts whatever its mode, since docker rejects a second
// mount on the same target anyway; the read-only carve-out applies only to the enclosing-target
// case, where dump and restore would otherwise lose the writable bind they need.
func isAlreadyMounted(mounts []mount.Mount, dir string) bool {
	dir = strings.Trim(dir, "/")

	for _, mountPoint := range mounts {
		source, target := strings.Trim(mountPoint.Source, "/"), strings.Trim(mountPoint.Target, "/")

		if source == dir || target == dir {
			return true
		}

		if target != "" && !mountPoint.ReadOnly && strings.HasPrefix(dir, target+"/") {
			return true
		}
	}

	return false
}
