/*
2026 © Postgres.ai
*/

// Package mountcheck inspects the engine container for a mount layout that stacks
// duplicate bind mounts on the host on every container start.
package mountcheck

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/client"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

const (
	mountInfoPath = "/proc/self/mountinfo"

	// mountInfoMinFields is the number of leading fields in a mountinfo line up to and including the mount point.
	mountInfoMinFields = 5

	// stackedThreshold is the number of mounts of one subtree on one path from which duplicates start.
	stackedThreshold = 2

	// maxMountInfoLine bounds a single mountinfo line; overlay mounts list every layer in one line
	// and overrun the default scanner buffer, which would cut the scan short and hide duplicates.
	maxMountInfoLine = 1 << 20
)

// mountInfoUnescaper decodes the octal escapes mountinfo uses for whitespace in paths.
var mountInfoUnescaper = strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`)

// nestedBind describes a mount whose destination lies inside another mount with shared propagation.
// Docker mounts the child inside the container after the parent, and shared propagation replays
// that mount onto the host, so each container start leaves one more mount stacked on the host path.
type nestedBind struct {
	parent      string
	child       string
	propagation mount.Propagation
}

// stackedMount describes a mount point that carries several mounts of one subtree on top of each other.
type stackedMount struct {
	path  string
	count int
}

// shellQuote wraps s in single quotes so that a path with spaces or shell metacharacters
// survives being pasted into a shell verbatim.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// Report logs a warning for every nested bind of the engine container and for every
// stacked mount visible from inside it. The check is best-effort and never fails startup.
func Report(ctx context.Context, docker *client.Client, containerName string) {
	inspection, err := docker.ContainerInspect(ctx, containerName, client.ContainerInspectOptions{})
	if err != nil {
		log.Dbg("skip the mount check, failed to inspect the engine container:", err)
		return
	}

	for _, nested := range nestedBinds(inspection.Container.Mounts) {
		log.Warn(fmt.Sprintf("bind mount %s is nested inside %s, which is mounted with %s propagation: "+
			"every start of the engine container stacks one more mount on the host, and docker refuses to start "+
			"the container once the stack reaches the kernel limit. remove the nested --volume from the docker run "+
			"command; the parent mount already exposes the directory", nested.child, nested.parent, nested.propagation))
	}

	mountInfo, err := os.Open(mountInfoPath)
	if err != nil {
		log.Dbg("skip the stacked mount check:", err)
		return
	}

	defer func() { _ = mountInfo.Close() }()

	for _, stacked := range stackedMounts(mountInfo) {
		log.Warn(fmt.Sprintf("%d mounts of the same subtree are stacked at %q, the path as seen from the engine "+
			"container. stop the engine and unmount the duplicates on the host, adjusting the path if the host "+
			"path differs: while umount %s; do :; done", stacked.count, stacked.path, shellQuote(stacked.path)))
	}
}

// nestedBinds returns the mounts whose destination lies inside another mount with shared propagation.
// A child covered by several shared parents is reported once, against the innermost one, because that
// is the mount that replays it onto the host.
func nestedBinds(points []container.MountPoint) []nestedBind {
	byChild := make(map[string]nestedBind)

	for _, parent := range points {
		if !isShared(parent.Propagation) {
			continue
		}

		parentDir := strings.TrimRight(parent.Destination, "/")

		for _, child := range points {
			childDir := strings.TrimRight(child.Destination, "/")

			if !strings.HasPrefix(childDir, parentDir+"/") {
				continue
			}

			if reported, ok := byChild[childDir]; ok && len(reported.parent) >= len(parentDir) {
				continue
			}

			byChild[childDir] = nestedBind{parent: parentDir, child: childDir, propagation: parent.Propagation}
		}
	}

	nested := make([]nestedBind, 0, len(byChild))

	for _, bind := range byChild {
		nested = append(nested, bind)
	}

	sort.Slice(nested, func(i, j int) bool { return nested[i].child < nested[j].child })

	return nested
}

func isShared(propagation mount.Propagation) bool {
	return propagation == mount.PropagationShared || propagation == mount.PropagationRShared
}

// stackedMounts parses mountinfo and returns the mount points where the same filesystem
// subtree is mounted more than once on top of itself.
func stackedMounts(mountInfo io.Reader) []stackedMount {
	counts := make(map[string]int)
	paths := make(map[string]string)

	scanner := bufio.NewScanner(mountInfo)
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), maxMountInfoLine)

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < mountInfoMinFields {
			continue
		}

		// major:minor, root and mount point together identify one bind of one subtree.
		key := fields[2] + " " + fields[3] + " " + fields[4]
		counts[key]++
		paths[key] = mountInfoUnescaper.Replace(fields[4])
	}

	if err := scanner.Err(); err != nil {
		log.Dbg("mountinfo was read only in part, stacked mounts may be missing from the report:", err)
	}

	stacked := make([]stackedMount, 0)

	for key, count := range counts {
		if count < stackedThreshold {
			continue
		}

		stacked = append(stacked, stackedMount{path: paths[key], count: count})
	}

	sort.Slice(stacked, func(i, j int) bool { return stacked[i].path < stacked[j].path })

	return stacked
}
