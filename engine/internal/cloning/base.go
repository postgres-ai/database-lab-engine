/*
2019 © Postgres.ai
*/

package cloning

import (
	"context"
	"database/sql"
	stderrors "errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgtype/pgxtype"
	"github.com/jackc/pgx/v4"
	_ "github.com/lib/pq" // Register Postgres database driver.
	"github.com/pkg/errors"
	"github.com/rs/xid"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/telemetry"
	"gitlab.com/postgres-ai/database-lab/v3/internal/webhooks"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/pglog"
)

const (
	idleCheckDuration = 5 * time.Minute
)

// Config contains a cloning configuration.
type Config struct {
	MaxIdleMinutes uint   `yaml:"maxIdleMinutes"`
	AccessHost     string `yaml:"accessHost"`
}

// Base provides cloning service.
type Base struct {
	config      *Config
	global      *global.Config
	cloneMutex  sync.RWMutex
	clones      map[string]*CloneWrapper
	snapshotBox SnapshotBox
	provision   *provision.Provisioner
	tm          *telemetry.Agent
	observingCh chan string
	webhookCh   chan webhooks.EventTyper
}

// NewBase instances a new Base service.
func NewBase(cfg *Config, global *global.Config, provision *provision.Provisioner, tm *telemetry.Agent,
	observingCh chan string, whCh chan webhooks.EventTyper) *Base {
	return &Base{
		config:      cfg,
		global:      global,
		clones:      make(map[string]*CloneWrapper),
		provision:   provision,
		tm:          tm,
		observingCh: observingCh,
		webhookCh:   whCh,
		snapshotBox: SnapshotBox{
			items: make(map[string]*models.Snapshot),
		},
	}
}

// Reload reloads base cloning configuration.
func (c *Base) Reload(cfg Config, global global.Config) {
	*c.config = cfg
	*c.global = global
}

// Run initializes and runs cloning component.
func (c *Base) Run(ctx context.Context) error {
	if err := c.provision.RevisePortPool(); err != nil {
		return fmt.Errorf("failed to revise port pool: %w", err)
	}

	if _, err := c.GetSnapshots(); err != nil {
		log.Err("no available snapshots:", err)
	}

	if err := c.RestoreClonesState(); err != nil {
		log.Err("failed to load stored sessions:", err)
	}

	c.restartCloneContainers(ctx)

	c.filterRunningClones(ctx)

	if err := c.cleanupInvalidClones(); err != nil {
		return fmt.Errorf("failed to cleanup invalid clones: %w", err)
	}

	if err := c.provision.RevisePortPool(); err != nil {
		return fmt.Errorf("failed to revise port pool: %w", err)
	}

	go c.runIdleCheck(ctx)

	return nil
}

func (c *Base) cleanupInvalidClones() error {
	keepClones := make(map[string]struct{})

	c.cloneMutex.Lock()

	for _, clone := range c.clones {
		keepClones[clone.Clone.ID] = struct{}{}
	}

	c.cloneMutex.Unlock()

	log.Dbg("Cleaning up invalid clone instances.\nKeep clones:", keepClones)

	if err := c.provision.StopAllSessions(keepClones); err != nil {
		return fmt.Errorf("failed to stop invalid sessions: %w", err)
	}

	return nil
}

// GetLatestSnapshot returns the latest snapshot.
func (c *Base) GetLatestSnapshot() (*models.Snapshot, error) {
	snapshot, err := c.getLatestSnapshot()
	if err != nil {
		return nil, fmt.Errorf("failed to find the latest snapshot: %w", err)
	}

	return snapshot, err
}

// CreateClone creates a new clone.
func (c *Base) CreateClone(cloneRequest *types.CloneCreateRequest) (*models.Clone, error) {
	cloneRequest.ID = strings.TrimSpace(cloneRequest.ID)

	if _, ok := c.findWrapper(cloneRequest.ID); ok {
		return nil, models.New(models.ErrCodeBadRequest, fmt.Sprintf("clone with ID %q already exists", cloneRequest.ID))
	}

	if cloneRequest.ID == "" {
		cloneRequest.ID = xid.New().String()
	}

	createdAt := time.Now()

	err := c.fetchSnapshots()
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch snapshots")
	}

	snapshot, err := c.getLatestSnapshot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to find the latest snapshot")
	}

	if cloneRequest.Snapshot != nil {
		snapshot, err = c.getSnapshotByID(cloneRequest.Snapshot.ID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to find the requested snapshot")
		}
	}

	if cloneRequest.Branch == "" {
		cloneRequest.Branch = snapshot.Branch
	}

	clone := &models.Clone{
		ID:        cloneRequest.ID,
		Snapshot:  snapshot,
		Branch:    cloneRequest.Branch,
		Protected: cloneRequest.Protected,
		CreatedAt: models.NewLocalTime(createdAt),
		Status: models.Status{
			Code:    models.StatusCreating,
			Message: models.CloneMessageCreating,
		},
		DB: models.Database{
			Username: cloneRequest.DB.Username,
			DBName:   cloneRequest.DB.DBName,
		},
		Revision: cloneRequest.Revision,
	}

	w := NewCloneWrapper(clone, createdAt)
	cloneID := clone.ID

	c.setWrapper(clone.ID, w)

	ephemeralUser := resources.EphemeralUser{
		Name:        cloneRequest.DB.Username,
		Password:    cloneRequest.DB.Password,
		Restricted:  cloneRequest.DB.Restricted,
		AvailableDB: cloneRequest.DB.DBName,
	}

	c.IncrementCloneNumber(clone.Snapshot.ID)

	go func() {
		session, err := c.provision.StartSession(clone, ephemeralUser, cloneRequest.ExtraConf)
		if err != nil {
			// TODO(anatoly): Empty room case.
			log.Errf("failed to start session: %v", err)

			if updateErr := c.UpdateCloneStatus(cloneID, models.Status{
				Code:    models.StatusFatal,
				Message: errors.Cause(err).Error(),
			}); updateErr != nil {
				log.Errf("failed to update clone status: %v", updateErr)
			}

			return
		}

		c.fillCloneSession(cloneID, session)
		c.SaveClonesState()

		c.webhookCh <- webhooks.CloneEvent{
			BasicEvent: webhooks.BasicEvent{
				EventType: webhooks.CloneCreatedEvent,
				EntityID:  cloneID,
			},
			Host:          c.config.AccessHost,
			Port:          session.Port,
			Username:      clone.DB.Username,
			DBName:        clone.DB.DBName,
			ContainerName: cloneID,
		}
	}()

	return clone, nil
}

func (c *Base) fillCloneSession(cloneID string, session *resources.Session) {
	c.cloneMutex.Lock()
	defer c.cloneMutex.Unlock()

	w, ok := c.clones[cloneID]
	if !ok {
		log.Errf("clone %q not found", cloneID)
		return
	}

	w.Session = session
	w.TimeStartedAt = time.Now()

	clone := w.Clone
	clone.Status = models.Status{
		Code:    models.StatusOK,
		Message: models.CloneMessageOK,
	}

	if dbName := clone.DB.DBName; dbName == "" {
		clone.DB.DBName = c.global.Database.Name()
	}

	clone.DB.Port = strconv.FormatUint(uint64(session.Port), 10)
	clone.DB.Host = c.config.AccessHost
	clone.DB.ConnStr = fmt.Sprintf("host=%s port=%s user=%s dbname=%s",
		clone.DB.Host, clone.DB.Port, clone.DB.Username, clone.DB.DBName)

	clone.Metadata = models.CloneMetadata{
		CloningTime:    w.TimeStartedAt.Sub(w.TimeCreatedAt).Seconds(),
		MaxIdleMinutes: c.config.MaxIdleMinutes,
	}
}

// ConnectToClone connects to clone by cloneID.
func (c *Base) ConnectToClone(ctx context.Context, cloneID string) (pgxtype.Querier, error) {
	w, ok := c.findWrapper(cloneID)
	if !ok {
		return nil, errors.New("not found")
	}

	connStr := connectionString(
		w.Session.SocketHost, strconv.FormatUint(uint64(w.Session.Port), 10), w.Session.User, w.Clone.DB.DBName)

	db, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func connectionString(host, port, username, dbname string) string {
	return fmt.Sprintf("host=%s port=%s user=%s database='%s'",
		host, port, username, dbname)
}

// DestroyClone destroys clone.
func (c *Base) DestroyClone(cloneID string) error {
	w, ok := c.findWrapper(cloneID)
	if !ok {
		return models.New(models.ErrCodeNotFound, "clone not found")
	}

	if err := c.destroyPreChecks(cloneID, w); err != nil {
		if stderrors.Is(err, errNoSession) {
			return nil
		}

		return err
	}

	go c.destroyClone(cloneID, w)

	return nil
}

var errNoSession = errors.New("no clone session")

func (c *Base) destroyPreChecks(cloneID string, w *CloneWrapper) error {
	if w.Clone.Protected && w.Clone.Status.Code != models.StatusFatal {
		return models.New(models.ErrCodeBadRequest, "clone is protected")
	}

	if c.hasDependentSnapshots(w) {
		log.Warn("clone has dependent snapshots", cloneID)
	}

	if err := c.UpdateCloneStatus(cloneID, models.Status{
		Code:    models.StatusDeleting,
		Message: models.CloneMessageDeleting,
	}); err != nil {
		return errors.Wrap(err, "failed to update clone status")
	}

	if w.Session == nil {
		c.deleteClone(cloneID)

		if w.Clone.Snapshot != nil {
			c.decrementCloneNumber(w.Clone.Snapshot.ID)
		}

		return errNoSession
	}

	return nil
}

func (c *Base) DestroyCloneSync(cloneID string) error {
	w, ok := c.findWrapper(cloneID)
	if !ok {
		return models.New(models.ErrCodeNotFound, "clone not found")
	}

	if err := c.destroyPreChecks(cloneID, w); err != nil {
		if stderrors.Is(err, errNoSession) {
			return nil
		}

		return err
	}

	c.destroyClone(cloneID, w)

	return nil
}

func (c *Base) destroyClone(cloneID string, w *CloneWrapper) {
	if err := c.provision.StopSession(w.Session, w.Clone); err != nil {
		log.Errf("failed to delete clone: %v", err)

		if updateErr := c.UpdateCloneStatus(cloneID, models.Status{
			Code:    models.StatusFatal,
			Message: errors.Cause(err).Error(),
		}); updateErr != nil {
			log.Errf("failed to update clone status: %v", updateErr)
		}

		return
	}

	c.deleteClone(cloneID)

	if w.Clone.Snapshot != nil {
		c.decrementCloneNumber(w.Clone.Snapshot.ID)
	}
	c.observingCh <- cloneID

	c.SaveClonesState()

	c.webhookCh <- webhooks.CloneEvent{
		BasicEvent: webhooks.BasicEvent{
			EventType: webhooks.CloneDeleteEvent,
			EntityID:  cloneID,
		},
		Host:          c.config.AccessHost,
		Port:          w.Session.Port,
		Username:      w.Clone.DB.Username,
		DBName:        w.Clone.DB.DBName,
		ContainerName: cloneID,
	}
}

// GetClone returns clone by ID.
func (c *Base) GetClone(id string) (*models.Clone, error) {
	w, ok := c.findWrapper(id)
	if !ok {
		return nil, errors.New("clone not found")
	}

	c.refreshCloneMetadata(w)

	return w.Clone, nil
}

func (c *Base) refreshCloneMetadata(w *CloneWrapper) {
	if w == nil || w.Session == nil || w.Clone == nil {
		// Not started yet.
		return
	}

	sessionState, err := c.provision.GetSessionState(w.Session, w.Clone.Branch, w.Clone.ID)
	if err != nil {
		// Session not ready yet.
		log.Err(fmt.Errorf("failed to get session state: %w", err))

		return
	}

	w.Clone.Metadata.CloneDiffSize = sessionState.CloneDiffSize
	w.Clone.Metadata.LogicalSize = sessionState.LogicalReferenced
}

// UpdateClone updates clone.
func (c *Base) UpdateClone(id string, patch types.CloneUpdateRequest) (*models.Clone, error) {
	w, ok := c.findWrapper(id)
	if !ok {
		return nil, models.New(models.ErrCodeNotFound, "clone not found")
	}

	var clone *models.Clone

	// Set fields.
	c.cloneMutex.Lock()
	w.Clone.Protected = patch.Protected
	clone = w.Clone
	c.cloneMutex.Unlock()

	c.SaveClonesState()

	return clone, nil
}

// UpdateCloneStatus updates the clone status.
func (c *Base) UpdateCloneStatus(cloneID string, status models.Status) error {
	c.cloneMutex.Lock()
	defer c.cloneMutex.Unlock()

	w, ok := c.clones[cloneID]
	if !ok {
		return errors.Errorf("clone %q not found", cloneID)
	}

	w.Clone.Status = status

	return nil
}

// UpdateCloneSnapshot updates clone snapshot.
func (c *Base) UpdateCloneSnapshot(cloneID string, snapshot *models.Snapshot) error {
	c.cloneMutex.Lock()
	defer c.cloneMutex.Unlock()

	w, ok := c.clones[cloneID]
	if !ok {
		return errors.Errorf("clone %q not found", cloneID)
	}

	w.Clone.Snapshot = snapshot

	return nil
}

// ResetClone resets clone to chosen snapshot.
func (c *Base) ResetClone(cloneID string, resetOptions types.ResetCloneRequest) error {
	w, ok := c.findWrapper(cloneID)
	if !ok {
		return models.New(models.ErrCodeNotFound, "the clone not found")
	}

	if w.Session == nil || w.Clone == nil {
		return models.New(models.ErrCodeNotFound, "clone is not started yet")
	}

	var snapshotID string

	if resetOptions.SnapshotID != "" {
		snapshot, err := c.getSnapshotByID(resetOptions.SnapshotID)
		if err != nil {
			return errors.Wrap(err, "failed to get snapshot ID")
		}

		snapshotID = snapshot.ID
	}

	// If the snapshotID variable is empty, the latest snapshot will be chosen.
	if snapshotID == "" && !resetOptions.Latest {
		snapshotID = w.Clone.Snapshot.ID
	}

	if err := c.UpdateCloneStatus(cloneID, models.Status{
		Code:    models.StatusResetting,
		Message: models.CloneMessageResetting,
	}); err != nil {
		return errors.Wrap(err, "failed to update clone status")
	}

	if c.hasDependentSnapshots(w) {
		log.Warn("clone has dependent snapshots", cloneID)
		c.cloneMutex.Lock()
		w.Clone.Revision++
		w.Clone.HasDependent = true
		c.cloneMutex.Unlock()
	} else {
		c.cloneMutex.Lock()
		w.Clone.HasDependent = false
		c.cloneMutex.Unlock()
	}

	go func() {
		var originalSnapshotID string

		if w.Clone.Snapshot != nil {
			originalSnapshotID = w.Clone.Snapshot.ID
		}

		snapshot, err := c.provision.ResetSession(w.Session, w.Clone, snapshotID)
		if err != nil {
			log.Errf("failed to reset clone: %v", err)

			if updateErr := c.UpdateCloneStatus(cloneID, models.Status{
				Code:    models.StatusFatal,
				Message: errors.Cause(err).Error(),
			}); updateErr != nil {
				log.Errf("failed to update clone status: %v", updateErr)
			}

			return
		}

		c.cloneMutex.Lock()
		w.Clone.Snapshot = snapshot
		c.cloneMutex.Unlock()
		c.decrementCloneNumber(originalSnapshotID)
		c.IncrementCloneNumber(snapshot.ID)

		if err := c.UpdateCloneStatus(cloneID, models.Status{
			Code:    models.StatusOK,
			Message: models.CloneMessageOK,
		}); err != nil {
			log.Errf("failed to update clone status: %v", err)
		}

		c.SaveClonesState()

		c.webhookCh <- webhooks.CloneEvent{
			BasicEvent: webhooks.BasicEvent{
				EventType: webhooks.CloneResetEvent,
				EntityID:  cloneID,
			},
			Host:          c.config.AccessHost,
			Port:          w.Session.Port,
			Username:      w.Clone.DB.Username,
			DBName:        w.Clone.DB.DBName,
			ContainerName: cloneID,
		}

		c.tm.SendEvent(context.Background(), telemetry.CloneResetEvent, telemetry.CloneCreated{
			ID:          util.HashID(w.Clone.ID),
			CloningTime: w.Clone.Metadata.CloningTime,
			DSADiff:     util.GetDataFreshness(snapshot.DataStateAt.Time),
		})
	}()

	return nil
}

// GetCloningState returns the current state of instance.
func (c *Base) GetCloningState() models.Cloning {
	clones := c.GetClones()
	cloning := models.Cloning{
		ExpectedCloningTime: c.getExpectedCloningTime(),
		Clones:              clones,
		NumClones:           uint64(len(clones)),
	}

	return cloning
}

// GetSnapshots returns all available snapshots.
func (c *Base) GetSnapshots() ([]models.Snapshot, error) {
	// TODO(anatoly): Update snapshots dynamically.
	if err := c.fetchSnapshots(); err != nil {
		return nil, errors.Wrap(err, "failed to fetch snapshots")
	}

	return c.getSnapshotList(), nil
}

// GetSnapshotByID returns snapshot by ID.
func (c *Base) GetSnapshotByID(snapshotID string) (*models.Snapshot, error) {
	return c.getSnapshotByID(snapshotID)
}

// ReloadSnapshots reloads snapshot list.
func (c *Base) ReloadSnapshots() error {
	return c.fetchSnapshots()
}

// GetClones returns the list of clones descend ordered by creation time.
func (c *Base) GetClones() []*models.Clone {
	clones := make([]*models.Clone, 0, c.lenClones())

	c.cloneMutex.RLock()
	for _, cloneWrapper := range c.clones {
		if cloneWrapper.Clone.Snapshot != nil {
			snapshot, err := c.getSnapshotByID(cloneWrapper.Clone.Snapshot.ID)
			if err != nil {
				log.Err("snapshot not found: ", cloneWrapper.Clone.Snapshot.ID)
			}

			if snapshot != nil {
				cloneWrapper.Clone.Snapshot = snapshot
			}
		}

		c.refreshCloneMetadata(cloneWrapper)

		clones = append(clones, cloneWrapper.Clone)
	}
	c.cloneMutex.RUnlock()

	sort.Slice(clones, func(i, j int) bool {
		return clones[i].CreatedAt.After(clones[j].CreatedAt.Time)
	})

	return clones
}

// findWrapper retrieves a clone findWrapper by id.
func (c *Base) findWrapper(id string) (*CloneWrapper, bool) {
	c.cloneMutex.RLock()
	w, ok := c.clones[id]
	c.cloneMutex.RUnlock()

	return w, ok
}

// setWrapper adds a clone wrapper to the map of clones.
func (c *Base) setWrapper(id string, wrapper *CloneWrapper) {
	c.cloneMutex.Lock()
	c.clones[id] = wrapper
	c.cloneMutex.Unlock()
}

// deleteClone removes the clone by ID.
func (c *Base) deleteClone(cloneID string) {
	c.cloneMutex.Lock()
	delete(c.clones, cloneID)
	c.cloneMutex.Unlock()
}

// lenClones returns the number of clones.
func (c *Base) lenClones() int {
	c.cloneMutex.RLock()
	lenClones := len(c.clones)
	c.cloneMutex.RUnlock()

	return lenClones
}

func (c *Base) getExpectedCloningTime() float64 {
	lenClones := c.lenClones()

	if lenClones == 0 {
		return 0
	}

	sum := 0.0

	c.cloneMutex.RLock()
	for _, cloneWrapper := range c.clones {
		sum += cloneWrapper.Clone.Metadata.CloningTime
	}
	c.cloneMutex.RUnlock()

	return sum / float64(lenClones)
}

func (c *Base) runIdleCheck(ctx context.Context) {
	if c.config.MaxIdleMinutes == 0 {
		return
	}

	idleTimer := time.NewTimer(idleCheckDuration)

	for {
		select {
		case <-idleTimer.C:
			c.destroyIdleClones(ctx)
			idleTimer.Reset(idleCheckDuration)
			c.SaveClonesState()

		case <-ctx.Done():
			idleTimer.Stop()
			return
		}
	}
}

func (c *Base) destroyIdleClones(ctx context.Context) {
	for _, cloneWrapper := range c.clones {
		select {
		case <-ctx.Done():
			return
		default:
			isIdleClone, err := c.isIdleClone(cloneWrapper)
			if err != nil {
				log.Errf("failed to check idleness of clone %s: %v", cloneWrapper.Clone.ID, err)
				continue
			}

			if isIdleClone {
				log.Msg(fmt.Sprintf("Idle clone %q is going to be removed.", cloneWrapper.Clone.ID))

				if err = c.DestroyClone(cloneWrapper.Clone.ID); err != nil {
					log.Errf("failed to destroy clone: %v", err)
					continue
				}
			}
		}
	}
}

// isIdleClone checks if clone is idle.
func (c *Base) isIdleClone(wrapper *CloneWrapper) (bool, error) {
	currentTime := time.Now()

	idleDuration := time.Duration(c.config.MaxIdleMinutes) * time.Minute
	minimumTime := currentTime.Add(-idleDuration)

	if wrapper.Clone.Protected || wrapper.Clone.Status.Code == models.StatusExporting || wrapper.TimeStartedAt.After(minimumTime) ||
		c.hasDependentSnapshots(wrapper) {
		return false, nil
	}

	session := wrapper.Session

	if session == nil {
		if wrapper.Clone.Status.Code == models.StatusFatal {
			return true, nil
		}

		return false, errors.New("failed to get clone session")
	}

	if _, err := c.provision.LastSessionActivity(session, wrapper.Clone.Branch, wrapper.Clone.ID, wrapper.Clone.Revision,
		minimumTime); err != nil {
		if err == pglog.ErrNotFound {
			log.Dbg(fmt.Sprintf("Not found recent activity for session: %q. Clone name: %q",
				session.ID, wrapper.Clone.ID))

			return hasNotQueryActivity(session)
		}

		return false, errors.Wrap(err, "failed to get the last session activity")
	}

	return false, nil
}

const pgDriverName = "postgres"

// hasNotQueryActivity opens connection and checks if there is no any query running by a user.
func hasNotQueryActivity(session *resources.Session) (bool, error) {
	log.Dbg(fmt.Sprintf("Check an active query for: %q.", session.ID))

	db, err := sql.Open(pgDriverName, getSocketConnStr(session))

	if err != nil {
		return false, errors.Wrap(err, "cannot connect to database")
	}

	defer func() {
		if err := db.Close(); err != nil {
			log.Err("cannot close database connection")
		}
	}()

	return checkActiveQueryNotExists(db)
}

func getSocketConnStr(session *resources.Session) string {
	return fmt.Sprintf("host=%s user=%s port=%d dbname=postgres", session.SocketHost, session.User, session.Port)
}

// checkActiveQueryNotExists runs query to check a user activity.
func checkActiveQueryNotExists(db *sql.DB) (bool, error) {
	var isRunningQueryNotExists bool

	query := `select not exists (
		select * from pg_stat_activity
		where state <> 'idle' and query not like 'autovacuum: %' and pid <> pg_backend_pid()
	)`
	err := db.QueryRow(query).Scan(&isRunningQueryNotExists)

	return isRunningQueryNotExists, err
}
