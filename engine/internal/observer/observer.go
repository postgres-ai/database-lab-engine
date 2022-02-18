/*
2020 © Postgres.ai
*/

// Package observer provides clone monitoring.
package observer

import (
	"bytes"
	"context"
	"encoding/csv"
	"io"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/pglog"
)

const (
	defaultIntervalSeconds        = 10
	defaultMaxLockDurationSeconds = 10
	defaultMaxDurationSeconds     = 60 * 60 // 1 hour.

	statusPassed = "passed"
	statusFailed = "failed"

	observerApplicationName = "observer"
)

// Observer manages observation sessions.
type Observer struct {
	dockerClient     *client.Client
	sessionMu        *sync.Mutex
	storage          map[string]*ObservingClone
	cfg              *Config
	replacementRules []ReplacementRule
	pm               *pool.Manager
}

// Config defines configuration options for observer.
type Config struct {
	ReplacementRules map[string]string `yaml:"replacementRules"`
}

// ReplacementRule describes replacement rules.
type ReplacementRule struct {
	re      *regexp.Regexp
	replace string
}

// NewObserver creates an Observer instance.
func NewObserver(dockerClient *client.Client, cfg *Config, pm *pool.Manager) *Observer {
	observer := &Observer{
		dockerClient:     dockerClient,
		sessionMu:        &sync.Mutex{},
		storage:          make(map[string]*ObservingClone),
		cfg:              cfg,
		pm:               pm,
		replacementRules: []ReplacementRule{},
	}

	for pattern, replace := range cfg.ReplacementRules {
		rule := ReplacementRule{
			re:      regexp.MustCompile(pattern),
			replace: replace,
		}
		observer.replacementRules = append(observer.replacementRules, rule)
	}

	return observer
}

// GetCloneLog gets clone logs.
// TODO (akartasov): Split log to chunks.
func (o *Observer) GetCloneLog(ctx context.Context, port string, obsClone *ObservingClone) ([]byte, error) {
	clonePort, err := strconv.Atoi(port)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse clone port")
	}

	fileSelector := pglog.NewSelector(obsClone.pool.ClonePath(uint(clonePort)))
	fileSelector.SetMinimumTime(obsClone.session.StartedAt)

	if err := fileSelector.DiscoverLogDir(); err != nil {
		return nil, errors.Wrap(err, "failed to init file selector")
	}

	buf := bytes.NewBuffer(make([]byte, 0, len(obsClone.CsvFields())))
	buf.WriteString(obsClone.CsvFields())
	buf.WriteString("\n")

	for {
		filename, err := fileSelector.Next()
		if err != nil {
			if err == pglog.ErrLastFile {
				break
			}

			return nil, errors.Wrap(err, "failed to get a CSV log filename")
		}

		if err := o.processCSVLogFile(ctx, buf, filename, obsClone); err != nil {
			if err == pglog.ErrTimeBoundary {
				break
			}

			return nil, errors.Wrap(err, "failed to process a CSV log file")
		}
	}

	return buf.Bytes(), nil
}

func (o *Observer) processCSVLogFile(ctx context.Context, buf io.Writer, filename string, obsClone *ObservingClone) error {
	logFile, err := os.Open(filename)
	if err != nil {
		return errors.Wrap(err, "failed to open a CSV log file")
	}

	defer func() {
		if err := logFile.Close(); err != nil {
			log.Errf("Failed to close a CSV log file: %s", err.Error())
		}
	}()

	if err := o.scanCSVLogFile(ctx, logFile, buf, obsClone); err != nil {
		return err
	}

	return nil
}

func (o *Observer) scanCSVLogFile(ctx context.Context, reader io.Reader, writer io.Writer, obsClone *ObservingClone) error {
	csvReader := csv.NewReader(reader)
	csvWriter := csv.NewWriter(writer)

	defer csvWriter.Flush()

	for {
		if ctx.Err() != nil {
			break
		}

		entry, err := csvReader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}

			return err
		}

		// Check an application name to skip observer log entries.
		if entry[22] == observerApplicationName {
			continue
		}

		logTime, err := time.Parse("2006-01-02 15:04:05.999 MST", entry[0])
		if err != nil {
			return err
		}

		if logTime.Before(obsClone.session.StartedAt) {
			continue
		}

		if logTime.After(obsClone.session.FinishedAt) {
			return pglog.ErrTimeBoundary
		}

		if len(o.replacementRules) > 0 {
			o.maskLogs(entry, obsClone.maskedIndexes)
		}

		if err := csvWriter.Write(entry); err != nil {
			return err
		}
	}

	return nil
}

func (o *Observer) maskLogs(entry []string, maskedFieldIndexes []int) {
	for _, maskedFieldIndex := range maskedFieldIndexes {
		for _, rule := range o.replacementRules {
			entry[maskedFieldIndex] = rule.re.ReplaceAllString(entry[maskedFieldIndex], rule.replace)
		}
	}
}

// AddObservingClone adds a new observing session to storage.
func (o *Observer) AddObservingClone(cloneID string, port uint, session *ObservingClone) {
	o.sessionMu.Lock()
	defer o.sessionMu.Unlock()
	session.pool = o.pm.First().Pool()
	session.cloneID = cloneID
	session.port = port

	o.storage[cloneID] = session
}

// GetObservingClone returns an observation session from storage.
func (o *Observer) GetObservingClone(cloneID string) (*ObservingClone, error) {
	o.sessionMu.Lock()
	defer o.sessionMu.Unlock()

	session, ok := o.storage[cloneID]
	if !ok {
		return nil, errors.New("observer not found")
	}

	return session, nil
}

// RemoveObservingClone removes an observing clone from storage.
func (o *Observer) RemoveObservingClone(cloneID string) {
	o.sessionMu.Lock()
	defer o.sessionMu.Unlock()

	delete(o.storage, cloneID)

	log.Dbg("Observing clone has been removed: ", cloneID)
}
