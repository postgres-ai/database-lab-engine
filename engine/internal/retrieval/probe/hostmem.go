/*
2026 © Postgres.ai
*/

package probe

import (
	"bufio"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
)

const (
	meminfoPath = "proc/meminfo"

	bytesPerKB = 1024

	gibibyte = uint64(1024 * 1024 * 1024)
	mebibyte = uint64(1024 * 1024)

	// 25% of host RAM, i.e. divide by 4.
	sharedBuffersFraction   = 4
	sharedBuffersMaxGB      = uint64(8)
	sharedBuffersMinMB      = uint64(128)
	sharedBuffersFallbackGB = uint64(1)

	// expected meminfo line: "MemTotal: <kb> kB" → 3 whitespace-separated fields.
	meminfoFieldCount = 3
)

// DetectHostMemoryBytes reads /proc/meminfo and returns the host's total
// memory in bytes. It returns 0 when the file is unreadable or unparsable —
// callers should treat 0 as "unknown" and surface that state to the user
// rather than substituting a recommendation silently.
//
// The function intentionally does not consult cgroup v1/v2 limits. DBLab's
// documented install path runs the DBLab Engine directly on the host where /proc/meminfo
// reports the host RAM the engine sees. Containerized deployments where the
// container has a memory limit lower than host RAM are a known caveat:
// callers will recommend a shared_buffers value sized to the host, not the
// container, and operators must tune in Expert mode.
//
// The fsys argument exists for testability — production callers pass
// os.DirFS("/").
func DetectHostMemoryBytes(fsys fs.FS) uint64 {
	f, err := fsys.Open(meminfoPath)
	if err != nil {
		return 0
	}

	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) != meminfoFieldCount {
			return 0
		}

		if fields[2] != "kB" {
			return 0
		}

		kb, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0
		}

		return kb * bytesPerKB
	}

	return 0
}

// RecommendSharedBuffers returns a libpq-friendly shared_buffers value sized
// at 25% of host RAM, capped at 8 GiB and floored at 128 MiB. When
// hostMemBytes is 0 (memory unknown), returns "1GB" as a safe default;
// callers must surface "memory unknown" to the user separately so the default
// is not presented as a recommendation.
func RecommendSharedBuffers(hostMemBytes uint64) string {
	if hostMemBytes == 0 {
		return fmt.Sprintf("%dGB", sharedBuffersFallbackGB)
	}

	target := hostMemBytes / sharedBuffersFraction

	maxBytes := sharedBuffersMaxGB * gibibyte
	if target > maxBytes {
		target = maxBytes
	}

	minBytes := sharedBuffersMinMB * mebibyte
	if target < minBytes {
		target = minBytes
	}

	if target%gibibyte == 0 {
		return fmt.Sprintf("%dGB", target/gibibyte)
	}

	return fmt.Sprintf("%dMB", target/mebibyte)
}
