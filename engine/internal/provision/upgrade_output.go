/*
2026 © Postgres.ai
*/

package provision

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/sys/unix"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

const (
	// upgradeLogReadLimit bounds how much of a log the engine reads for a status message. The
	// logs live in a directory the upgrade container writes, so their size is not the engine's
	// to trust.
	upgradeLogReadLimit = 1 << 20

	// preservedOutputDirMode matches the mode the upgrade log directory is created with.
	preservedOutputDirMode = 0755

	// serverErrorContinuationLines bounds how many detail and hint lines follow the fatal line
	// into the status message.
	serverErrorContinuationLines = 2

	// serverErrorLengthLimit bounds the postmaster's text that enters the status message.
	serverErrorLengthLimit = 512

	// serverErrorEllipsis marks a truncated postmaster error.
	serverErrorEllipsis = "..."
)

var (
	// serverLogErrorPattern finds the severity field of a postmaster log line. The two spaces
	// after the colon are how the server separates it from the message, which keeps a message
	// that merely mentions "FATAL:" from matching.
	serverLogErrorPattern = regexp.MustCompile(`(?:^|\s)((?:FATAL|PANIC):  .*)$`)

	// serverLogDetailPattern finds the continuation lines the server writes after an error.
	serverLogDetailPattern = regexp.MustCompile(`(?:^|\s)((?:DETAIL|HINT):  .*)$`)
)

// openBeneath opens base/rel[0]/rel[1]/... without following a symbolic link at any component of
// rel. The base is a path the engine owns; everything below it is inside a tree the upgrade
// container writes as the data directory owner, so a link planted there must not be able to
// point a privileged engine operation at another file. flags apply to the last component only.
func openBeneath(base string, rel []string, flags int) (*os.File, error) {
	fd, err := unix.Open(base, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: base, Err: err}
	}

	for i, name := range rel {
		componentFlags := unix.O_RDONLY | unix.O_DIRECTORY | unix.O_NOFOLLOW | unix.O_CLOEXEC
		if i == len(rel)-1 {
			componentFlags = flags | unix.O_NOFOLLOW | unix.O_CLOEXEC
		}

		next, err := unix.Openat(fd, name, componentFlags, 0)

		_ = unix.Close(fd)

		if err != nil {
			return nil, &os.PathError{Op: "open", Path: path.Join(append([]string{base}, rel[:i+1]...)...), Err: err}
		}

		fd = next
	}

	return os.NewFile(uintptr(fd), path.Join(append([]string{base}, rel...)...)), nil
}

// openRegularBeneath opens a file below base for reading and refuses anything but a regular
// file. The container can leave a FIFO or a device in place of a log, and a blocking open on a
// FIFO without a writer never returns; the file is opened non-blocking and its type is checked
// before a byte is read. O_NONBLOCK has no effect on a regular file.
func openRegularBeneath(base string, rel []string) (*os.File, error) {
	file, err := openBeneath(base, rel, unix.O_RDONLY|unix.O_NONBLOCK)
	if err != nil {
		return nil, err
	}

	var st unix.Stat_t

	if err := unix.Fstat(int(file.Fd()), &st); err != nil {
		_ = file.Close()

		return nil, &os.PathError{Op: "stat", Path: file.Name(), Err: err}
	}

	if st.Mode&unix.S_IFMT != unix.S_IFREG {
		_ = file.Close()

		return nil, &os.PathError{Op: "open", Path: file.Name(), Err: unix.EINVAL}
	}

	return file, nil
}

// upgradeLogsRel is the path of the upgrade log directory relative to the clone directory.
func upgradeLogsRel() []string {
	return []string{upgradeDirName, upgradeLogsDirName}
}

// preserveUpgradeOutput moves pg_upgrade's output directory out of the new cluster before that
// cluster is discarded and returns the names of the runs it moved, oldest first. Each timestamped
// run directory is moved on its own, so the runs of repeated attempts accumulate next to the
// script log instead of replacing each other, and the oldest are pruned once there are more than
// preservedUpgradeRuns. It never fails: the output is a diagnostic, and recovery must go on
// without it.
//
// Both ends of the move are inside directories the upgrade container owns, so every component is
// opened without following links and the rename is done relative to those directories: a link
// the container left in place of either directory cannot redirect the move into the engine's
// own files.
func preserveUpgradeOutput(appConfig *resources.AppConfig) []string {
	cloneDir := appConfig.CloneDir()

	sourceRel, err := filepath.Rel(cloneDir, newDataDir(appConfig))
	if err != nil {
		log.Err("failed to locate the new data directory inside the clone directory:", err)

		return nil
	}

	source, err := openBeneath(cloneDir, append(strings.Split(sourceRel, "/"), upgradeOutputDirName), unix.O_DIRECTORY)
	if err != nil {
		if !errors.Is(err, unix.ENOENT) {
			log.Err("failed to open the pg_upgrade output directory:", err)
		}

		return nil
	}

	defer func() { _ = source.Close() }()

	runs, err := readDirSorted(source)
	if err != nil {
		log.Err("failed to read the pg_upgrade output directory:", err)

		return nil
	}

	target, err := openPreservedOutputDir(cloneDir)
	if err != nil {
		log.Err("failed to open the preserved pg_upgrade output directory:", err)

		return nil
	}

	defer func() { _ = target.Close() }()

	moved := make([]string, 0, len(runs))

	for _, run := range runs {
		if !run.Type().IsDir() {
			log.Dbg(fmt.Sprintf("skipping a pg_upgrade output entry that is not a run directory: %q", run.Name()))

			continue
		}

		if err := unix.Renameat(int(source.Fd()), run.Name(), int(target.Fd()), run.Name()); err != nil {
			log.Err(fmt.Sprintf("failed to preserve the pg_upgrade output %q:", run.Name()), err)

			continue
		}

		moved = append(moved, run.Name())
	}

	prunePreservedOutput(target)

	return moved
}

// openPreservedOutputDir opens the directory preserved runs go to, creating it inside the upgrade
// log directory when it is missing. An existing entry of that name is accepted only when it is a
// real directory.
func openPreservedOutputDir(cloneDir string) (*os.File, error) {
	logs, err := openBeneath(cloneDir, upgradeLogsRel(), unix.O_DIRECTORY)
	if err != nil {
		return nil, err
	}

	defer func() { _ = logs.Close() }()

	if err := unix.Mkdirat(int(logs.Fd()), upgradeOutputDirName, preservedOutputDirMode); err != nil && !errors.Is(err, unix.EEXIST) {
		return nil, &os.PathError{Op: "mkdir", Path: upgradeOutputPath(cloneDir), Err: err}
	}

	return openBeneath(cloneDir, append(upgradeLogsRel(), upgradeOutputDirName), unix.O_DIRECTORY)
}

// prunePreservedOutput removes the oldest preserved runs beyond preservedUpgradeRuns. Runs are
// named by timestamp, so the directory listing order is their age.
func prunePreservedOutput(target *os.File) {
	entries, err := readDirSorted(target)
	if err != nil {
		log.Err("failed to list the preserved pg_upgrade output:", err)

		return
	}

	runs := make([]string, 0, len(entries))

	for _, entry := range entries {
		if entry.Type().IsDir() {
			runs = append(runs, entry.Name())
		}
	}

	if len(runs) <= preservedUpgradeRuns {
		return
	}

	for _, name := range runs[:len(runs)-preservedUpgradeRuns] {
		if err := removeTreeAt(int(target.Fd()), name); err != nil {
			log.Err(fmt.Sprintf("failed to prune the preserved pg_upgrade output %q:", name), err)
		}
	}
}

// readDirSorted lists an open directory by name. Run directories are named by timestamp, so the
// order is their age, oldest first.
func readDirSorted(dir *os.File) ([]os.DirEntry, error) {
	entries, err := dir.ReadDir(-1)
	if err != nil {
		return nil, err
	}

	slices.SortFunc(entries, func(a, b os.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })

	return entries, nil
}

// removeTreeAt removes dirfd/name and everything below it without following links.
func removeTreeAt(dirfd int, name string) error {
	var st unix.Stat_t

	if err := unix.Fstatat(dirfd, name, &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return err
	}

	if st.Mode&unix.S_IFMT != unix.S_IFDIR {
		return unix.Unlinkat(dirfd, name, 0)
	}

	fd, err := unix.Openat(dirfd, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}

	dir := os.NewFile(uintptr(fd), name)

	entries, err := dir.ReadDir(-1)
	if err != nil {
		_ = dir.Close()

		return err
	}

	for _, entry := range entries {
		if err := removeTreeAt(int(dir.Fd()), entry.Name()); err != nil {
			_ = dir.Close()

			return err
		}
	}

	_ = dir.Close()

	return unix.Unlinkat(dirfd, name, unix.AT_REMOVEDIR)
}

// readServerLogError returns the first fatal error the postmaster pg_upgrade started wrote in the
// newest of the runs preserved by the current attempt, with the detail and hint lines that
// follow it, or "" when the attempt left no such run. Only runs moved in this recovery count:
// an earlier attempt's run is still on disk, and its error would be reported as this attempt's
// cause otherwise. pg_upgrade's own log only reports that it could not connect to that
// postmaster; this line is the reason. The text is the container's, so it is stripped of
// control characters and bounded before it enters the status.
func readServerLogError(cloneDir string, preserved []string) string {
	if len(preserved) == 0 {
		return ""
	}

	newest := preserved[len(preserved)-1]

	rel := append(upgradeLogsRel(), upgradeOutputDirName, newest, upgradeServerLogDirName, upgradeServerLogName)

	data, err := readBeneath(cloneDir, rel)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")

	for i, line := range lines {
		match := serverLogErrorPattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		parts := []string{strings.TrimSpace(match[1])}

		for _, next := range lines[i+1:] {
			detail := serverLogDetailPattern.FindStringSubmatch(next)
			if detail == nil || len(parts) > serverErrorContinuationLines {
				break
			}

			parts = append(parts, strings.TrimSpace(detail[1]))
		}

		return sanitizeServerError(strings.Join(parts, "; "))
	}

	return ""
}

// sanitizeServerError drops every control and non-printable character from a postmaster message
// and truncates it to serverErrorLengthLimit bytes, so a line the container wrote cannot inject
// terminal sequences or line breaks into the status and the engine log.
func sanitizeServerError(text string) string {
	printable := strings.Map(func(r rune) rune {
		if r == ' ' || unicode.IsPrint(r) {
			return r
		}

		return -1
	}, text)

	if len(printable) <= serverErrorLengthLimit {
		return printable
	}

	cut := serverErrorLengthLimit - len(serverErrorEllipsis)

	for cut > 0 && !utf8.RuneStart(printable[cut]) {
		cut--
	}

	return printable[:cut] + serverErrorEllipsis
}

// readBeneath reads at most upgradeLogReadLimit bytes from the start of a regular file below the
// clone directory, without following links.
func readBeneath(cloneDir string, rel []string) ([]byte, error) {
	file, err := openRegularBeneath(cloneDir, rel)
	if err != nil {
		return nil, err
	}

	defer func() { _ = file.Close() }()

	return io.ReadAll(io.LimitReader(file, upgradeLogReadLimit))
}

// readLogTail returns the last lines of the script log, reading no more than upgradeLogReadLimit
// bytes from its end.
func readLogTail(cloneDir string) string {
	file, err := openRegularBeneath(cloneDir, append(upgradeLogsRel(), upgradeLogFileName))
	if err != nil {
		return ""
	}

	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return ""
	}

	if info.Size() > upgradeLogReadLimit {
		if _, err := file.Seek(info.Size()-upgradeLogReadLimit, io.SeekStart); err != nil {
			return ""
		}
	}

	data, err := io.ReadAll(io.LimitReader(file, upgradeLogReadLimit))
	if err != nil {
		return ""
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) > logTailLines {
		lines = lines[len(lines)-logTailLines:]
	}

	return strings.Join(lines, "\n")
}

// describeFailure appends what the postmaster started by pg_upgrade said to the reported cause,
// when its log recorded a fatal error.
func describeFailure(cause, serverError string) string {
	if serverError == "" {
		return cause
	}

	return cause + "; the postmaster started by pg_upgrade reported: " + serverError
}
