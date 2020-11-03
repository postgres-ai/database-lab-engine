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
	"sync"
	"time"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/client/platform"
	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/util/pglog"
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
	dockerClient *client.Client
	Platform     *platform.Client
	storage      map[string]*Session
	sessionMu    *sync.Mutex
	cfg          *Config
}

// Config defines configuration options for observer.
type Config struct {
	CloneDir   string
	DataSubDir string
	SocketDir  string
}

// NewObserver creates an Observer instance.
func NewObserver(dockerClient *client.Client, cfg *Config, platform *platform.Client) *Observer {
	return &Observer{
		dockerClient: dockerClient,
		Platform:     platform,
		storage:      make(map[string]*Session),
		sessionMu:    &sync.Mutex{},
		cfg:          cfg,
	}
}

// GetCloneLog gets clone logs.
// TODO (akartasov): Split log to chunks.
func (o *Observer) GetCloneLog(ctx context.Context, port uint, session *Session) ([]byte, error) {
	fileSelector := pglog.NewSelector(o.cfg.CloneDir, o.cfg.DataSubDir, port)
	fileSelector.SetMinimumTime(session.StartedAt)

	if err := fileSelector.DiscoverLogDir(); err != nil {
		return nil, errors.Wrap(err, "failed to init file selector")
	}

	buf := bytes.NewBuffer(make([]byte, 0, len(session.CsvFields())))
	buf.WriteString(session.CsvFields())
	buf.WriteString("\n")

	for {
		filename, err := fileSelector.Next()
		if err != nil {
			if err == pglog.ErrLastFile {
				break
			}

			return nil, errors.Wrap(err, "failed to get a CSV log filename")
		}

		if err := o.processCSVLogFile(ctx, buf, filename, session.StartedAt, session.FinishedAt); err != nil {
			if err == pglog.ErrTimeBoundary {
				break
			}

			return nil, errors.Wrap(err, "failed to process a CSV log file")
		}
	}

	return buf.Bytes(), nil
}

func (o *Observer) processCSVLogFile(ctx context.Context, buf io.Writer, filename string, since, until time.Time) error {
	logFile, err := os.Open(filename)
	if err != nil {
		return errors.Wrap(err, "failed to open a CSV log file")
	}

	defer func() {
		if err := logFile.Close(); err != nil {
			log.Errf("Failed to close a CSV log file: %s", err.Error())
		}
	}()

	if err := o.scanCSVLogFile(ctx, logFile, buf, since, until); err != nil {
		return err
	}

	return nil
}

func (o *Observer) scanCSVLogFile(ctx context.Context, reader io.Reader, writer io.Writer, since, until time.Time) error {
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

		if logTime.Before(since) {
			continue
		}

		if logTime.After(until) {
			return pglog.ErrTimeBoundary
		}

		if err := csvWriter.Write(entry); err != nil {
			return err
		}
	}

	return nil
}

// AddSession adds a new observation session to storage.
func (o *Observer) AddSession(cloneID string, session *Session) {
	o.sessionMu.Lock()
	defer o.sessionMu.Unlock()
	session.socketDir = o.cfg.SocketDir

	o.storage[cloneID] = session
}

// GetSession returns an observation session from storage.
func (o *Observer) GetSession(cloneID string) (*Session, error) {
	o.sessionMu.Lock()
	defer o.sessionMu.Unlock()

	session, ok := o.storage[cloneID]
	if !ok {
		return nil, errors.New("observer not found")
	}

	return session, nil
}

// RemoveSession removes an observation session from storage.
func (o *Observer) RemoveSession(cloneID string) {
	o.sessionMu.Lock()
	defer o.sessionMu.Unlock()

	delete(o.storage, cloneID)
}
