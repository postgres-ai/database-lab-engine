/*
2021 © Postgres.ai
*/

package estimator

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"sync/atomic"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/log"
)

const (
	regExp               = "^[.0-9]+\\s+\\S+\\s+(\\d+)\\s+\\w+\\s+(W|R)\\s+\\d+\\s+(\\d+)\\s+[.0-9]+$"
	countMatches         = 4
	expectedMappingParts = 2
)

var (
	r        = regexp.MustCompile(regExp)
	nsPrefix = []byte("NSpid:")
)

// Monitor observes processes and system activity.
type Monitor struct {
	pid        int
	container  string
	pidMapping map[int]int
	profiler   *Profiler
}

// NewMonitor creates a new monitor.
func NewMonitor(pid int, container string, profiler *Profiler) *Monitor {
	return &Monitor{
		pid:        pid,
		container:  container,
		profiler:   profiler,
		pidMapping: make(map[int]int),
	}
}

// InspectIOBlocks counts physically read blocks.
func (m *Monitor) InspectIOBlocks(ctx context.Context) error {
	log.Dbg("Run read physical")

	cmd := exec.Command("biosnoop")

	r, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	cmd.Stderr = cmd.Stdout

	go m.scanOutput(ctx, r)

	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "failed to run")
	}

	<-m.profiler.exitChan

	log.Dbg("End read physical")

	return nil
}

type bytesEntry struct {
	pid        int
	totalBytes uint64
}

func (m *Monitor) scanOutput(ctx context.Context, r io.Reader) {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		scanBytes := scanner.Bytes()

		if !bytes.Contains(scanBytes, []byte("postgres")) && !bytes.Contains(scanBytes, []byte("psql")) {
			continue
		}

		bytesEntry := m.parseReadBytes(scanBytes)
		if bytesEntry == nil || bytesEntry.totalBytes == 0 {
			continue
		}

		pid, ok := m.pidMapping[bytesEntry.pid]
		if !ok {
			hostPID, err := m.filterPID(bytesEntry.pid)
			m.pidMapping[bytesEntry.pid] = hostPID

			if err != nil {
				// log.Dbg("failed to get PID mapping: ", err)
				continue
			}

			pid = hostPID
		}

		if pid != m.pid {
			continue
		}

		log.Dbg("read bytes: ", bytesEntry.totalBytes)

		atomic.AddUint64(&m.profiler.readBytes, bytesEntry.totalBytes)

		select {
		case <-ctx.Done():
			log.Dbg("context")
			return

		case <-m.profiler.exitChan:
			log.Dbg("exit chan")
			return

		default:
		}
	}
}

func getContainerHash(pid int) (string, error) {
	procParallel, err := exec.Command("cat", fmt.Sprintf("/host_proc/%d/cgroup", pid)).Output()
	if err != nil {
		return "", err
	}

	return isInside(procParallel), nil
}

func isInside(procParallel []byte) string {
	sc := bufio.NewScanner(bytes.NewBuffer(procParallel))

	for sc.Scan() {
		line := sc.Bytes()

		if !bytes.HasPrefix(line, []byte("1:name")) {
			continue
		}

		res := bytes.SplitN(line, []byte("/docker/"), 2)

		if len(res) == 1 {
			return ""
		}

		return string(res[1])
	}

	return ""
}

func (m *Monitor) isValidContainer(hash string) bool {
	return m.container == hash
}

func (m *Monitor) filterPID(pid int) (int, error) {
	hash, err := getContainerHash(pid)
	if err != nil {
		return 0, err
	}

	if !m.isValidContainer(hash) {
		return 0, nil
	}

	procParallel, err := exec.Command("cat", fmt.Sprintf("/host_proc/%d/cmdline", pid)).Output()
	if err != nil {
		return 0, err
	}

	if bytes.Contains(procParallel, []byte("postgres")) &&
		bytes.Contains(procParallel, []byte("parallel worker for PID "+strconv.Itoa(m.pid))) {
		return m.pid, nil
	}

	procStatus, err := exec.Command("cat", fmt.Sprintf("/host_proc/%d/status", pid)).Output()
	if err != nil {
		return 0, err
	}

	return m.parsePIDMapping(procStatus)
}

func (m *Monitor) parsePIDMapping(procStatus []byte) (int, error) {
	sc := bufio.NewScanner(bytes.NewBuffer(procStatus))

	for sc.Scan() {
		line := sc.Bytes()
		if !bytes.HasPrefix(line, nsPrefix) {
			continue
		}

		nsPID := bytes.TrimSpace(bytes.TrimPrefix(line, nsPrefix))

		pidValues := bytes.Fields(nsPID)
		if len(pidValues) < expectedMappingParts {
			return 0, nil
		}

		hostPID, err := strconv.Atoi(string(bytes.TrimSpace(pidValues[1])))
		if err != nil {
			return 0, err
		}

		return hostPID, nil
	}

	return 0, nil
}

func (m *Monitor) parseReadBytes(line []byte) *bytesEntry {
	submatch := r.FindSubmatch(line)
	if len(submatch) != countMatches {
		return nil
	}

	totalBytes, err := strconv.ParseUint(string(submatch[3]), 10, 64)
	if err != nil {
		return nil
	}

	pid, err := strconv.Atoi(string(submatch[1]))
	if err != nil {
		return nil
	}

	return &bytesEntry{
		pid:        pid,
		totalBytes: totalBytes,
	}
}
