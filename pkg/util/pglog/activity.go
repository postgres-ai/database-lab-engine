/*
2019 © Postgres.ai
*/

// Package pglog provides helpers for a Postgres logs processing.
package pglog

import (
	"errors"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	errs "github.com/pkg/errors"
)

const (
	csvLogDir = "log"

	csvLogFilenameFormat = "postgresql-2006-01-02_150405.csv"
)

var (
	// ErrNotFound defines an error if last session activity not found.
	ErrNotFound = errors.New("pglog activity: not found")

	// ErrLastFile defines an error if no more recent log files to discover last activity.
	ErrLastFile = errors.New("no more recent log files")

	// ErrTimeBoundary defines an error if the upper boundary of the interval exceeded.
	ErrTimeBoundary = errors.New("time boundary exceeded")
)

// Selector describes a struct to select CSV log files.
type Selector struct {
	logDir       string
	currentIndex int
	fileNames    []string
	minimumTime  time.Time
}

// NewSelector creates a new Selector.
func NewSelector(dir string) *Selector {
	return &Selector{
		logDir:    buildLogDirName(dir),
		fileNames: make([]string, 0),
	}
}

// SetMinimumTime sets a minimum allowable time.
func (s *Selector) SetMinimumTime(minimumTime time.Time) {
	s.minimumTime = minimumTime
}

// DiscoverLogDir discovers available CSV log files.
func (s *Selector) DiscoverLogDir() error {
	logFilenames := []string{}

	logDirFiles, err := os.ReadDir(s.logDir)
	if err != nil {
		return errs.Wrap(err, "failed to read a log directory")
	}

	for _, fileInfo := range logDirFiles {
		if fileInfo.IsDir() {
			continue
		}

		if !strings.HasSuffix(fileInfo.Name(), "csv") {
			continue
		}

		logFilenames = append(logFilenames, fileInfo.Name())
	}

	if len(logFilenames) == 0 {
		return errors.New("log files not found")
	}

	sort.Strings(logFilenames)
	s.fileNames = logFilenames

	return nil
}

// Next returns the next CSV log filename to discover.
func (s *Selector) Next() (string, error) {
	if len(s.fileNames) == 0 {
		return "", errors.New("log fileNames not found")
	}

	if s.currentIndex >= len(s.fileNames) {
		return "", ErrLastFile
	}

	logPath := path.Join(s.logDir, s.fileNames[s.currentIndex])
	s.currentIndex++

	return logPath, nil
}

// FilterOldFilesInList filters the original filename list.
func (s *Selector) FilterOldFilesInList() {
	if s.minimumTime.IsZero() {
		return
	}

	startIndex := 0
	minimumTime := s.minimumTime.Format(csvLogFilenameFormat)

	for i := range s.fileNames {
		if len(s.fileNames) > i+1 {
			if minimumTime < s.fileNames[i+1] {
				break
			}

			startIndex = i
		}
	}

	s.fileNames = s.fileNames[startIndex:]
}

// ParsePostgresLastActivity extracts the time of last session activity.
func ParsePostgresLastActivity(logTime, text string) (*time.Time, error) {
	if logTime == "" || !(strings.Contains(text, "statement:") || strings.Contains(text, "duration:")) {
		return nil, nil
	}

	lastActivityTime, err := time.Parse("2006-01-02 15:04:05.000 UTC", logTime)
	if err != nil {
		return nil, errs.Wrap(err, "failed to parse the last activity time")
	}

	return &lastActivityTime, nil
}

func buildLogDirName(cloneDir string) string {
	return path.Join(cloneDir, csvLogDir)
}
