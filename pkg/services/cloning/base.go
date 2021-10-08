/*
2019 © Postgres.ai
*/

package cloning

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/jackc/pgtype/pgxtype"
	"github.com/jackc/pgx/v4"
	_ "github.com/lib/pq" // Register Postgres database driver.
	"github.com/pkg/errors"
	"github.com/rs/xid"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/util"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/util/pglog"
)

const (
	idleCheckDuration = 5 * time.Minute

	defaultDatabaseName = "postgres"
)

// Config contains a cloning configuration.
type Config struct {
	MaxIdleMinutes uint   `yaml:"maxIdleMinutes"`
	AccessHost     string `yaml:"accessHost"`
}

// Base provides cloning service.
type Base struct {
	config        *Config
	cloneMutex    sync.RWMutex
	clones        map[string]*CloneWrapper
	snapshotMutex sync.RWMutex
	snapshots     []models.Snapshot
	provision     *provision.Provisioner
	observingCh   chan string
}

// NewBase instances a new Base service.
func NewBase(cfg *Config, provision *provision.Provisioner, observingCh chan string) *Base {
	return &Base{
		config:      cfg,
		clones:      make(map[string]*CloneWrapper),
		provision:   provision,
		observingCh: observingCh,
	}
}

// Reload reloads base cloning configuration.
func (c *Base) Reload(cfg Config) {
	*c.config = cfg
}

// Run initializes and runs cloning component.
func (c *Base) Run(ctx context.Context) error {
	if err := c.provision.Init(); err != nil {
		return errors.Wrap(err, "failed to run cloning service")
	}

	if _, err := c.GetSnapshots(); err != nil {
		log.Err("No available snapshots: ", err)
	}

	if err := c.RestoreClonesState(); err != nil {
		return fmt.Errorf("failed to load sessions: %w", err)
	}

	c.filterRunningClones(ctx)

	if err := c.cleanupInvalidClones(); err != nil {
		return fmt.Errorf("failed to cleanup invalid clones: %w", err)
	}

	go c.runIdleCheck(ctx)

	return nil
}

func (c *Base) cleanupInvalidClones() error {
	keepClones := make(map[string]struct{})

	c.cloneMutex.Lock()

	for _, clone := range c.clones {
		keepClones[util.GetCloneName(clone.Session.Port)] = struct{}{}
	}

	c.cloneMutex.Unlock()

	log.Dbg("Cleaning up invalid clone instances.\nKeep clones:", keepClones)

	if err := c.provision.StopAllSessions(keepClones); err != nil {
		return fmt.Errorf("failed to stop invalid sessions: %w", err)
	}

	return nil
}

// CreateClone creates a new clone.
func (c *Base) CreateClone(cloneRequest *types.CloneCreateRequest) (*models.Clone, error) {
	cloneRequest.ID = strings.TrimSpace(cloneRequest.ID)

	if _, ok := c.findWrapper(cloneRequest.ID); ok {
		return nil, models.New(models.ErrCodeBadRequest, "clone with such ID already exists")
	}

	if cloneRequest.ID == "" {
		cloneRequest.ID = xid.New().String()
	}

	createdAt := time.Now()

	err := c.fetchSnapshots()
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch snapshots")
	}

	var snapshot models.Snapshot

	if cloneRequest.Snapshot != nil {
		snapshot, err = c.getSnapshotByID(cloneRequest.Snapshot.ID)
	} else {
		snapshot, err = c.getLatestSnapshot()
	}

	if err != nil {
		return nil, errors.Wrap(err, "failed to get snapshot")
	}

	clone := &models.Clone{
		ID:        cloneRequest.ID,
		Snapshot:  &snapshot,
		Protected: cloneRequest.Protected,
		CreatedAt: util.FormatTime(createdAt),
		Status: models.Status{
			Code:    models.StatusCreating,
			Message: models.CloneMessageCreating,
		},
		DB: models.Database{
			Username: cloneRequest.DB.Username,
			DBName:   cloneRequest.DB.DBName,
		},
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

	go func() {
		session, err := c.provision.StartSession(clone.Snapshot.ID, ephemeralUser, cloneRequest.ExtraConf)
		if err != nil {
			// TODO(anatoly): Empty room case.
			log.Errf("Failed to start session: %v.", err)

			if updateErr := c.UpdateCloneStatus(cloneID, models.Status{
				Code:    models.StatusFatal,
				Message: errors.Cause(err).Error(),
			}); updateErr != nil {
				log.Errf("Failed to update clone status: %v", updateErr)
			}

			return
		}

		c.fillCloneSession(cloneID, session)
		c.SaveClonesState()
	}()

	return clone, nil
}

func (c *Base) fillCloneSession(cloneID string, session *resources.Session) {
	c.cloneMutex.Lock()
	defer c.cloneMutex.Unlock()

	w, ok := c.clones[cloneID]
	if !ok {
		log.Errf("Clone %q not found", cloneID)
		return
	}

	w.Session = session
	w.TimeStartedAt = time.Now()

	clone := w.Clone
	clone.Status = models.Status{
		Code:    models.StatusOK,
		Message: models.CloneMessageOK,
	}

	dbName := clone.DB.DBName
	if dbName == "" {
		dbName = defaultDatabaseName
	}

	clone.DB.Port = strconv.FormatUint(uint64(session.Port), 10)
	clone.DB.Host = c.config.AccessHost
	clone.DB.ConnStr = fmt.Sprintf("host=%s port=%s user=%s dbname=%s",
		clone.DB.Host, clone.DB.Port, clone.DB.Username, dbName)

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

	if w.Clone.Protected && w.Clone.Status.Code != models.StatusFatal {
		return models.New(models.ErrCodeBadRequest, "clone is protected")
	}

	if err := c.UpdateCloneStatus(cloneID, models.Status{
		Code:    models.StatusDeleting,
		Message: models.CloneMessageDeleting,
	}); err != nil {
		return errors.Wrap(err, "failed to update clone status")
	}

	if w.Session == nil {
		c.deleteClone(cloneID)

		return nil
	}

	go func() {
		if err := c.provision.StopSession(w.Session); err != nil {
			log.Errf("Failed to delete a clone: %+v.", err)

			if updateErr := c.UpdateCloneStatus(cloneID, models.Status{
				Code:    models.StatusFatal,
				Message: errors.Cause(err).Error(),
			}); updateErr != nil {
				log.Errf("Failed to update clone status: %v", updateErr)
			}

			return
		}

		c.deleteClone(cloneID)
		c.observingCh <- cloneID

		c.SaveClonesState()
	}()

	return nil
}

// GetClone returns clone by ID.
func (c *Base) GetClone(id string) (*models.Clone, error) {
	w, ok := c.findWrapper(id)
	if !ok {
		return nil, errors.New("clone not found")
	}

	if w.Session == nil {
		// Not started yet.
		return w.Clone, nil
	}

	sessionState, err := c.provision.GetSessionState(w.Session)
	if err != nil {
		// Session not ready yet.
		log.Err(errors.Wrap(err, "failed to get a session state"))

		return w.Clone, nil
	}

	w.Clone.Metadata.CloneDiffSize = sessionState.CloneDiffSize
	w.Clone.Metadata.CloneDiffSizeHR = humanize.BigIBytes(big.NewInt(int64(sessionState.CloneDiffSize)))

	return w.Clone, nil
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

	go func() {
		snapshot, err := c.provision.ResetSession(w.Session, snapshotID)
		if err != nil {
			log.Errf("Failed to reset clone: %+v.", err)

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

		if err := c.UpdateCloneStatus(cloneID, models.Status{
			Code:    models.StatusOK,
			Message: models.CloneMessageOK,
		}); err != nil {
			log.Errf("failed to update clone status: %v", err)
		}

		c.SaveClonesState()
	}()

	return nil
}

// GetInstanceState returns the current state of instance.
func (c *Base) GetInstanceState() (*models.InstanceStatus, error) {
	disk, err := c.provision.GetDiskState()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get a disk state")
	}

	instanceStatus := &models.InstanceStatus{
		FileSystem: &models.FileSystem{},
	}

	instanceStatus.FileSystem.Size = disk.Size
	instanceStatus.FileSystem.Free = disk.Free
	instanceStatus.FileSystem.Used = disk.Used
	instanceStatus.DataSize = disk.DataSize
	instanceStatus.FileSystem.SizeHR = humanize.BigIBytes(big.NewInt(int64(disk.Size)))
	instanceStatus.FileSystem.FreeHR = humanize.BigIBytes(big.NewInt(int64(disk.Free)))
	instanceStatus.FileSystem.UsedHR = humanize.BigIBytes(big.NewInt(int64(disk.Used)))
	instanceStatus.DataSizeHR = humanize.BigIBytes(big.NewInt(int64(disk.DataSize)))
	instanceStatus.ExpectedCloningTime = c.getExpectedCloningTime()
	instanceStatus.Clones = c.GetClones()
	instanceStatus.NumClones = uint64(len(instanceStatus.Clones))
	instanceStatus.Status = &models.Status{
		Code:    models.StatusOK,
		Message: models.InstanceMessageOK,
	}

	return instanceStatus, nil
}

// GetSnapshots returns all available snapshots.
func (c *Base) GetSnapshots() ([]models.Snapshot, error) {
	// TODO(anatoly): Update snapshots dynamically.
	if err := c.fetchSnapshots(); err != nil {
		return nil, errors.Wrap(err, "failed to fetch snapshots")
	}

	snapshots := make([]models.Snapshot, len(c.snapshots))

	c.snapshotMutex.RLock()
	copy(snapshots, c.snapshots)
	c.snapshotMutex.RUnlock()

	return snapshots, nil
}

// GetClones returns the list of clones descend ordered by creation time.
func (c *Base) GetClones() []*models.Clone {
	clones := make([]*models.Clone, 0, c.lenClones())

	c.cloneMutex.RLock()
	for _, cloneWrapper := range c.clones {
		clones = append(clones, cloneWrapper.Clone)
	}
	c.cloneMutex.RUnlock()

	sort.Slice(clones, func(i, j int) bool {
		return clones[i].CreatedAt > clones[j].CreatedAt
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

func (c *Base) fetchSnapshots() error {
	entries, err := c.provision.GetSnapshots()
	if err != nil {
		return errors.Wrap(err, "failed to get snapshots")
	}

	snapshots := make([]models.Snapshot, len(entries))

	for i, entry := range entries {
		snapshots[i] = models.Snapshot{
			ID:          entry.ID,
			CreatedAt:   util.FormatTime(entry.CreatedAt),
			DataStateAt: util.FormatTime(entry.DataStateAt),
		}

		log.Dbg("snapshot:", snapshots[i])
	}

	c.snapshotMutex.Lock()
	c.snapshots = snapshots
	c.snapshotMutex.Unlock()

	return nil
}

// getLatestSnapshot returns the latest snapshot.
func (c *Base) getLatestSnapshot() (models.Snapshot, error) {
	c.snapshotMutex.RLock()
	defer c.snapshotMutex.RUnlock()

	if len(c.snapshots) == 0 {
		return models.Snapshot{}, errors.New("no snapshot found")
	}

	snapshot := c.snapshots[0]

	return snapshot, nil
}

// getSnapshotByID returns the snapshot by ID.
func (c *Base) getSnapshotByID(snapshotID string) (models.Snapshot, error) {
	c.snapshotMutex.RLock()
	defer c.snapshotMutex.RUnlock()

	for _, snapshot := range c.snapshots {
		if snapshot.ID == snapshotID {
			return snapshot, nil
		}
	}

	return models.Snapshot{}, errors.New("no snapshot found")
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
				log.Errf("Failed to check the idleness of clone %s: %v.", cloneWrapper.Clone.ID, err)
				continue
			}

			if isIdleClone {
				log.Msg(fmt.Sprintf("Idle clone %q is going to be removed.", cloneWrapper.Clone.ID))

				if err = c.DestroyClone(cloneWrapper.Clone.ID); err != nil {
					log.Errf("Failed to destroy clone: %+v.", err)
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

	if wrapper.Clone.Protected || wrapper.Clone.Status.Code == models.StatusExporting || wrapper.TimeStartedAt.After(minimumTime) {
		return false, nil
	}

	session := wrapper.Session

	// TODO(akartasov): Remove wrappers without session.
	if session == nil {
		return false, errors.New("failed to get clone session")
	}

	if _, err := c.provision.LastSessionActivity(session, minimumTime); err != nil {
		if err == pglog.ErrNotFound {
			log.Dbg(fmt.Sprintf("Not found recent activity for the session: %q. Clone name: %q",
				session.ID, util.GetCloneName(session.Port)))

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
			log.Err("Cannot close database connection.")
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
