/*
2020 © Postgres.ai
*/

// Package observer provides clone monitoring.
package observer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/models"
	"gitlab.com/postgres-ai/database-lab/pkg/util"
)

const (
	csvFields = "log_time,user_name,database_name,process_id,connection_from,session_id,session_line_num,command_tag," +
		"session_start_time,virtual_transaction_id,transaction_id,error_severity,sql_state_code,message,detail,hint," +
		"internal_query,internal_query_pos,context,query,query_pos,location,application_name"
)

// Session describes a session of service monitoring.
type Session struct {
	SessionID                uint64            `json:"session_id"`
	StartedAt                time.Time         `json:"started_at"`
	FinishedAt               time.Time         `json:"finished_at"`
	Config                   types.Config      `json:"config"`
	Tags                     map[string]string `json:"tags"`
	csvFields                string
	models.ObservationResult `json:"-"`

	ctx       context.Context
	cancel    context.CancelFunc
	done      chan struct{}
	socketDir string
}

// NewSession creates an Session instance.
func NewSession(config types.Config) *Session {
	if config.ObservationInterval == 0 {
		config.ObservationInterval = defaultIntervalSeconds
	}

	if config.MaxLockDuration == 0 {
		config.MaxLockDuration = defaultMaxLockDurationSeconds
	}

	if config.MaxDuration == 0 {
		config.MaxDuration = defaultMaxDurationSeconds
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Session{
		Config:    config,
		ctx:       ctx,
		cancel:    cancel,
		done:      make(chan struct{}, 1),
		csvFields: csvFields,
	}
}

// CsvFields returns a comma-separated list of available csv fields.
func (s *Session) CsvFields() string {
	return s.csvFields
}

// Start runs observation session.
func (s *Session) Start(clone *models.Clone) error {
	log.Dbg("Start observation for SessionID: ", s.SessionID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := initConnection(clone, s.socketDir)
	if err != nil {
		return errors.Wrap(err, "cannot connect to database")
	}

	defer func() {
		if err := db.Close(ctx); err != nil {
			log.Err("Failed to close a database connection after observation for SessionID: ", s.SessionID)
		}
	}()

	if err := s.discoverLogFields(ctx, db); err != nil {
		return errors.Wrap(err, "failed to discover available log fields")
	}

	timestamp := time.Now()
	sleepDuration := time.Duration(s.Config.ObservationInterval) * time.Second
	timer := time.NewTimer(sleepDuration)

	defer timer.Stop()

	for {
		select {
		case <-s.ctx.Done():
		case <-timer.C:
		}

		dangerousLocks, err := runQuery(ctx, db, buildLocksMetricQuery(observerApplicationName, s.Config.MaxLockDuration))
		if err != nil {
			return errors.Wrap(err, "cannot query metrics")
		}

		s.Summary.TotalIntervals++
		s.Summary.TotalDuration = time.Since(s.StartedAt).Seconds()

		if dangerousLocks != "" {
			s.Summary.WarningIntervals++
		}

		interval := models.Interval{
			StartedAt: timestamp,
			Duration:  time.Since(timestamp).Seconds(),
			Warning:   dangerousLocks,
		}

		s.Intervals = append(s.Intervals, interval)
		timestamp = time.Now()

		if err := s.ctx.Err(); err != nil {
			log.Dbg("Stop observation for SessionID: ", s.SessionID)
			s.done <- struct{}{}

			return nil
		}

		timer.Reset(sleepDuration)
	}
}

func (s *Session) discoverLogFields(ctx context.Context, db *pgx.Conn) error {
	row := db.QueryRow(ctx, `select coalesce(string_agg(column_name, ','), '') 
from information_schema.columns 
where table_name = 'postgres_log'`)

	fields := ""
	if err := row.Scan(&fields); err != nil {
		return err
	}

	if fields != "" {
		s.csvFields = fields
	}

	return nil
}

// Stop stops an observation session.
func (s *Session) Stop() {
	log.Msg(fmt.Sprintf("Observation session %v is stopping...", s.SessionID))

	s.cancel()

	// Waiting for the observation process stops.
	<-s.done

	s.summarize()

	log.Msg(fmt.Sprintf("Observation session %v has been stopped.", s.SessionID))
}

// summarize calculates an observation result.
func (s *Session) summarize() {
	s.Status = statusFailed
	s.FinishedAt = time.Now()
	s.Summary.Checklist.Duration = s.CheckDuration()
	s.Summary.Checklist.Locks = s.CheckLocks()

	if s.Summary.Checklist.Duration && s.Summary.Checklist.Locks {
		s.Status = statusPassed
	}
}

// PrintSummary prints monitoring summary.
func (s *Session) PrintSummary() string {
	maxDuration := time.Duration(s.Config.MaxDuration) * time.Second

	var summary strings.Builder

	summary.WriteString("Summary:\n")
	summary.WriteString(formatSummaryItem(fmt.Sprintf("Duration: %f s.", s.Summary.TotalDuration)))
	summary.WriteString(formatSummaryItem(fmt.Sprintf("Intervals with dangerous locks: %d", s.Summary.WarningIntervals)))
	summary.WriteString(formatSummaryItem(fmt.Sprintf("Total number of observed intervals: %d", s.Summary.TotalIntervals)))
	summary.WriteString("\nPerformance checklist:\n")
	summary.WriteString(formatChecklistItem(fmt.Sprintf("Duration < %s", util.DurationToString(maxDuration)), s.CheckDuration()))
	summary.WriteString(formatChecklistItem("No dangerous locks", s.CheckLocks()))

	return summary.String()
}

// CheckPerformanceRequirements checks monitoring data and returns an error if any of performance requires was not satisfied.
func (s *Session) CheckPerformanceRequirements() error {
	if !s.CheckDuration() || !s.CheckLocks() {
		return errors.New("performance requirements not satisfied")
	}

	return nil
}

// CheckDuration checks duration of the operation.
func (s *Session) CheckDuration() bool {
	return uint64(s.Summary.TotalDuration) < s.Config.MaxDuration
}

// CheckLocks checks long-lasting locks during the operation.
func (s *Session) CheckLocks() bool {
	return s.Summary.WarningIntervals == 0
}

func formatSummaryItem(str string) string {
	return "  " + str + "\n"
}

func formatChecklistItem(str string, state bool) string {
	stateStr := colorizeRed("FAILED")

	if state {
		stateStr = colorizeGreen("PASSED")
	}

	return "  " + str + ": " + stateStr + "\n"
}

func colorizeRed(str string) string {
	return fmt.Sprintf("\033[1;31m%s\033[0m", str)
}

func colorizeGreen(str string) string {
	return fmt.Sprintf("\033[1;32m%s\033[0m", str)
}
