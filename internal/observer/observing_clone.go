/*
2020 © Postgres.ai
*/

// Package observer provides clone monitoring.
package observer

import (
	"context"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

const (
	csvFields = "log_time,user_name,database_name,process_id,connection_from,session_id,session_line_num,command_tag," +
		"session_start_time,virtual_transaction_id,transaction_id,error_severity,sql_state_code,message,detail,hint," +
		"internal_query,internal_query_pos,context,query,query_pos,location,application_name"
)

// maskedFields contains list of the fields which should be filtered if replacement rules are defined.
var maskedFields = map[string]struct{}{
	"message":        {},
	"detail":         {},
	"hint":           {},
	"internal_query": {},
	"query":          {},
}

// ObservingClone describes an entity containing observability sessions.
type ObservingClone struct {
	pool        *resources.Pool
	cloneID     string
	port        uint
	superUserDB *pgx.Conn

	config types.Config

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
	db     *pgx.Conn

	csvFields     string
	maskedIndexes []int

	session *Session

	// TODO: add lock to prevent running of several session simultaneously.

	registryMu      *sync.Mutex
	sessionRegistry map[uint64]struct{}
}

// Session returns the current observability session.
func (c *ObservingClone) Session() *Session {
	if c.session == nil {
		return nil
	}

	session := *c.session

	return &session
}

// NewObservingClone creates a new observing clone.
func NewObservingClone(config types.Config, sudb *pgx.Conn) *ObservingClone {
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

	observingClone := &ObservingClone{
		config:          config,
		ctx:             ctx,
		cancel:          cancel,
		done:            make(chan struct{}, 1),
		csvFields:       csvFields,
		registryMu:      &sync.Mutex{},
		sessionRegistry: make(map[uint64]struct{}),
		superUserDB:     sudb,
	}

	observingClone.fillMaskedIndexes()

	return observingClone
}

// Config returns config of the observing clone.
func (c *ObservingClone) Config() types.Config {
	return c.config
}

// CsvFields returns a comma-separated list of available csv fields.
func (c *ObservingClone) CsvFields() string {
	return c.csvFields
}

// fillMaskedIndexes discovers indexes of the fields which should be masked.
func (c *ObservingClone) fillMaskedIndexes() {
	c.maskedIndexes = make([]int, 0, len(maskedFields))

	for i, csvField := range strings.Split(c.csvFields, ",") {
		if _, ok := maskedFields[csvField]; ok {
			c.maskedIndexes = append(c.maskedIndexes, i)
		}
	}
}

// AddArtifact adds a new observation session to storage.
func (c *ObservingClone) AddArtifact(sessionID uint64) {
	c.registryMu.Lock()
	defer c.registryMu.Unlock()

	c.sessionRegistry[sessionID] = struct{}{}
}

// GetArtifactList returns available artifact session IDs for the requested clone.
func (c *ObservingClone) GetArtifactList() []uint64 {
	sessionIDs := []uint64{}

	c.registryMu.Lock()
	defer c.registryMu.Unlock()

	for v := range c.sessionRegistry {
		sessionIDs = append(sessionIDs, v)
	}

	return sessionIDs
}

// IsExistArtifacts checks if observation session artifacts exist.
func (c *ObservingClone) IsExistArtifacts(sessionID uint64) bool {
	c.registryMu.Lock()
	defer c.registryMu.Unlock()

	_, ok := c.sessionRegistry[sessionID]

	return ok
}

// ReadSummary reads summary file.
func (c *ObservingClone) ReadSummary(sessionID uint64) ([]byte, error) {
	return c.readFileStats(sessionID, summaryFilename)
}

// Init initializes observation session.
func (c *ObservingClone) Init(clone *models.Clone, sessionID uint64, startedAt time.Time, tags map[string]string) error {
	c.session = NewSession(sessionID, startedAt, c.config, tags)

	log.Dbg("Init observation for SessionID: ", c.session.SessionID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := c.createObservationDir(); err != nil {
		return errors.Wrap(err, "failed to create the observation directory")
	}

	db, err := InitConnection(clone, c.pool.SocketDir())
	if err != nil {
		return errors.Wrap(err, "cannot connect to database")
	}

	c.db = db

	if err := c.discoverLogFields(ctx); err != nil {
		return errors.Wrap(err, "failed to discover available log fields")
	}

	if err := c.getDBSize(ctx, &c.session.state.InitialDBSize); err != nil {
		return errors.Wrap(err, "failed to get the initial database size")
	}

	if err := c.resetStat(ctx); err != nil {
		return errors.Wrap(err, "failed to reset clone statistics")
	}

	return nil
}

// RunSession runs observing session.
func (c *ObservingClone) RunSession() error {
	if c.session == nil || c.db.IsClosed() {
		return errors.New("failed to run session because it has not been initialized")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timestamp := time.Now()
	sleepDuration := time.Duration(c.session.Config.ObservationInterval) * time.Second
	timer := time.NewTimer(sleepDuration)

	defer timer.Stop()

	defer func() {
		if err := c.db.Close(ctx); err != nil {
			log.Err("Failed to close a database connection after observation for SessionID: ", c.session.SessionID)
		}
	}()

	c.session.Result = &models.ObservationResult{}

	for {
		select {
		case <-c.ctx.Done():
		case <-timer.C:
		}

		dangerousLocks, err := runQuery(ctx, c.db, buildLocksMetricQuery(observerApplicationName, c.session.Config.MaxLockDuration))
		if err != nil {
			return errors.Wrap(err, "cannot query metrics")
		}

		c.session.Result.Summary.TotalIntervals++
		c.session.Result.Summary.TotalDuration = time.Since(c.session.StartedAt).Seconds()

		if dangerousLocks != "" {
			c.session.Result.Summary.WarningIntervals++
		}

		interval := models.Interval{
			StartedAt: timestamp,
			Duration:  time.Since(timestamp).Seconds(),
			Warning:   dangerousLocks,
		}

		c.session.Result.Intervals = append(c.session.Result.Intervals, interval)
		timestamp = time.Now()

		if err := c.ctx.Err(); err != nil {
			log.Dbg("Stop observation for SessionID: ", c.session.SessionID)

			if err := c.storeArtifacts(); err != nil {
				log.Err("Failed to store artifacts: ", err)
			}

			c.done <- struct{}{}

			return nil
		}

		timer.Reset(sleepDuration)
	}
}

func (c *ObservingClone) createObservationDir() error {
	sessionPath := c.currentArtifactsSessionPath()

	artifactsPath := path.Join(sessionPath, artifactsSubDir)
	if err := os.RemoveAll(artifactsPath); err != nil {
		return err
	}

	if err := os.MkdirAll(artifactsPath, 0777); err != nil {
		return err
	}

	log.Dbg("Artifacts path", artifactsPath)

	return nil
}

func (c *ObservingClone) resetStat(ctx context.Context) error {
	if _, err := c.superUserDB.Exec(ctx, `create extension if not exists pg_stat_statements`); err != nil {
		return errors.Wrap(err, "failed to reset statistics counters for the current database")
	}

	if _, err := c.superUserDB.Exec(ctx, `create extension if not exists logerrors`); err != nil {
		return errors.Wrap(err, "failed to create the logerrors extension")
	}

	if _, err := c.superUserDB.Exec(ctx, `create extension if not exists pg_stat_kcache`); err != nil {
		return errors.Wrap(err, "failed to create the pg_stat_kcache extension")
	}

	if _, err := c.superUserDB.Exec(ctx,
		`select
		pg_stat_reset(),
		pg_stat_reset_shared('bgwriter')`); err != nil {
		return errors.Wrap(err, "failed to reset statistics counters for the current database")
	}

	if _, err := c.superUserDB.Exec(ctx, `select pg_stat_statements_reset()`); err != nil {
		return errors.Wrap(err, "failed to reset statement statistics for the current database")
	}

	if _, err := c.superUserDB.Exec(ctx, `select pg_stat_kcache_reset()`); err != nil {
		return errors.Wrap(err, "failed to reset kcache statistics for the current database")
	}

	// TODO: uncomment for Postgres 13+
	// if _, err := c.superUserDB.Exec(ctx, `select pg_stat_reset_slru()`); err != nil {
	//   return errors.Wrap(err, "failed to reset slru statistics for the current database")
	// }

	if _, err := c.superUserDB.Exec(ctx, `select pg_log_errors_reset()`); err != nil {
		return errors.Wrap(err, "failed to reset log errors statistics for the current database")
	}

	log.Dbg("Stats have been reset")

	return nil
}

func (c *ObservingClone) storeArtifacts() error {
	log.Dbg("Store observation artifacts for SessionID: ", c.session.SessionID)

	dstPath := path.Join(c.currentArtifactsSessionPath(), artifactsSubDir)
	if err := os.MkdirAll(dstPath, 0666); err != nil {
		return errors.Wrapf(err, "cannot create an artifact directory %s", dstPath)
	}

	ctx := context.Background()

	if err := c.dumpStatementsStats(ctx); err != nil {
		return err
	}

	if err := c.dumpDatabaseStats(ctx); err != nil {
		return err
	}

	if err := c.dumpDatabaseErrors(ctx); err != nil {
		return err
	}

	if err := c.dumpBGWriterStats(ctx); err != nil {
		return err
	}

	if err := c.dumpKCacheStats(ctx); err != nil {
		return err
	}

	if err := c.dumpIndexStats(ctx); err != nil {
		return err
	}

	if err := c.dumpAllTablesStats(ctx); err != nil {
		return err
	}

	if err := c.dumpIOIndexesStats(ctx); err != nil {
		return err
	}

	if err := c.dumpIOTablesStats(ctx); err != nil {
		return err
	}

	if err := c.dumpIOSequencesStats(ctx); err != nil {
		return err
	}

	if err := c.dumpUserFunctionsStats(ctx); err != nil {
		return err
	}

	// TODO: uncomment for Postgres 13+
	// if err := c.dumpSLRUStats(ctx); err != nil {
	// 	 return err
	// }

	if err := c.dumpRelationsSize(ctx); err != nil {
		return err
	}

	if err := c.dumpObjectsSize(ctx); err != nil {
		return err
	}

	if err := c.collectCurrentState(ctx); err != nil {
		return err
	}

	return nil
}

func (c *ObservingClone) collectCurrentState(ctx context.Context) error {
	if err := c.getMaxQueryTime(ctx, &c.session.state.MaxDBQueryTimeMS); err != nil {
		return err
	}

	if err := c.getDBSize(ctx, &c.session.state.CurrentDBSize); err != nil {
		return err
	}

	if err := c.getObjectsSizeStats(ctx, &c.session.state.ObjectStat); err != nil {
		return err
	}

	if err := c.countLogErrors(ctx, &c.session.state.LogErrors); err != nil {
		return err
	}

	return nil
}

func (c *ObservingClone) discoverLogFields(ctx context.Context) error {
	row := c.db.QueryRow(ctx, `select coalesce(string_agg(column_name, ','), '') 
from information_schema.columns 
where table_name = 'postgres_log'`)

	fields := ""
	if err := row.Scan(&fields); err != nil {
		return err
	}

	if fields != "" {
		c.csvFields = fields
		c.fillMaskedIndexes()
	}

	return nil
}

// Stop stops an observation session.
func (c *ObservingClone) Stop() error {
	log.Msg(fmt.Sprintf("Observation session %v is stopping...", c.session.SessionID))

	c.cancel()

	// Waiting for the observation process stops.
	<-c.done

	if c.session == nil {
		return errors.New("failed to summarize session because it has not been initialized")
	}

	c.summarize()

	if err := c.storeSummary(); err != nil {
		log.Err(err)
	}

	c.AddArtifact(c.session.SessionID)

	log.Msg(fmt.Sprintf("Observation session %v has been stopped.", c.session.SessionID))

	return nil
}

// summarize calculates an observation result.
func (c *ObservingClone) summarize() {
	c.session.Result.Status = statusFailed
	c.session.FinishedAt = time.Now()
	c.session.Result.Summary.Checklist.Duration = c.CheckDuration()
	c.session.Result.Summary.Checklist.Locks = c.CheckLocks()
	c.session.Result.Summary.Checklist.Success = c.CheckOverallSuccess()

	if c.session.Result.Summary.Checklist.Duration &&
		c.session.Result.Summary.Checklist.Locks &&
		c.session.Result.Summary.Checklist.Success {
		c.session.Result.Status = statusPassed
	}
}

func (c *ObservingClone) currentArtifactsSessionPath() string {
	return c.artifactsSessionPath(c.session.SessionID)
}

func (c *ObservingClone) artifactsSessionPath(sessionID uint64) string {
	return path.Join(c.pool.ObserverDir(c.port), c.cloneID, strconv.FormatUint(sessionID, 10))
}

// CheckPerformanceRequirements checks monitoring data and returns an error if any of performance requires was not satisfied.
func (c *ObservingClone) CheckPerformanceRequirements() error {
	if !c.CheckDuration() || !c.CheckLocks() {
		return errors.New("performance requirements not satisfied")
	}

	return nil
}

// CheckDuration checks duration of the operation.
func (c *ObservingClone) CheckDuration() bool {
	return uint64(c.session.Result.Summary.TotalDuration) < c.session.Config.MaxDuration
}

// CheckLocks checks long-lasting locks during the operation.
func (c *ObservingClone) CheckLocks() bool {
	return c.session.Result.Summary.WarningIntervals == 0
}

// CheckOverallSuccess checks overall success of queries.
func (c *ObservingClone) CheckOverallSuccess() bool {
	return !c.session.state.OverallError
}

// SetOverallError notes the presence of errors during the session.
func (c *ObservingClone) SetOverallError(overallErrors bool) {
	c.session.state.OverallError = overallErrors
}
