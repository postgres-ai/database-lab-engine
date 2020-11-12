/*
2019 © Postgres.ai
*/

package cloning

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq" // Register Postgres database driver.
	"github.com/pkg/errors"
	"github.com/rs/xid"

	"gitlab.com/postgres-ai/database-lab/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/models"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/resources"
	"gitlab.com/postgres-ai/database-lab/pkg/util"
	"gitlab.com/postgres-ai/database-lab/pkg/util/pglog"
)

const (
	idleCheckDuration = 5 * time.Minute

	defaultDatabaseName = "postgres"
)

type baseCloning struct {
	cloning

	cloneMutex     sync.RWMutex
	clones         map[string]*CloneWrapper
	instanceStatus *models.InstanceStatus
	snapshotMutex  sync.RWMutex
	snapshots      []models.Snapshot

	provision provision.Provision
}

// NewBaseCloning instances a new base Cloning.
func NewBaseCloning(cfg *Config, provision provision.Provision) Cloning {
	return &baseCloning{
		cloning: cloning{Config: cfg},
		clones:  make(map[string]*CloneWrapper),
		instanceStatus: &models.InstanceStatus{
			Status: &models.Status{
				Code:    models.StatusOK,
				Message: models.InstanceMessageOK,
			},
			FileSystem: &models.FileSystem{},
			Clones:     make([]*models.Clone, 0),
		},
		provision: provision,
	}
}

// Initialize and run cloning component.
func (c *baseCloning) Run(ctx context.Context) error {
	if err := c.provision.Init(); err != nil {
		return errors.Wrap(err, "failed to run cloning")
	}

	if _, err := c.GetSnapshots(); err != nil {
		log.Err("No available snapshots: ", err)
	}

	go c.runIdleCheck(ctx)

	return nil
}

// CreateClone creates a new clone.
func (c *baseCloning) CreateClone(cloneRequest *types.CloneCreateRequest) (*models.Clone, error) {
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
			Password: cloneRequest.DB.Password,
		},
		Project: cloneRequest.Project,
	}

	w := NewCloneWrapper(clone)

	w.username = clone.DB.Username
	w.password = clone.DB.Password
	w.timeCreatedAt = createdAt
	w.snapshot = snapshot

	clone.DB.Password = ""
	cloneID := clone.ID

	c.setWrapper(clone.ID, w)

	go func() {
		session, err := c.provision.StartSession(w.username, w.password, w.snapshot.ID)
		if err != nil {
			// TODO(anatoly): Empty room case.
			log.Errf("Failed to start session: %v.", err)

			if updateErr := c.updateCloneStatus(cloneID, models.Status{
				Code:    models.StatusFatal,
				Message: errors.Cause(err).Error(),
			}); updateErr != nil {
				log.Errf("Failed to update clone status: %v", updateErr)
			}

			return
		}

		c.cloneMutex.Lock()
		defer c.cloneMutex.Unlock()

		w, ok := c.clones[cloneID]
		if !ok {
			log.Errf("Clone %q not found", cloneID)
			return
		}

		w.session = session
		w.timeStartedAt = time.Now()

		clone := w.clone
		clone.Status = models.Status{
			Code:    models.StatusOK,
			Message: models.CloneMessageOK,
		}

		clone.DB.Port = strconv.FormatUint(uint64(session.Port), 10)
		clone.DB.Host = c.Config.AccessHost
		clone.DB.ConnStr = fmt.Sprintf("host=%s port=%s user=%s dbname=%s",
			clone.DB.Host, clone.DB.Port, clone.DB.Username, defaultDatabaseName)

		// TODO(anatoly): Remove mock data.
		clone.Metadata = models.CloneMetadata{
			CloneDiffSize:  cloneDiffSize,
			CloningTime:    w.timeStartedAt.Sub(w.timeCreatedAt).Seconds(),
			MaxIdleMinutes: c.Config.MaxIdleMinutes,
		}
	}()

	return clone, nil
}

func (c *baseCloning) DestroyClone(cloneID string) error {
	w, ok := c.findWrapper(cloneID)
	if !ok {
		return models.New(models.ErrCodeNotFound, "clone not found")
	}

	if w.clone.Protected {
		return models.New(models.ErrCodeBadRequest, "clone is protected")
	}

	if err := c.updateCloneStatus(cloneID, models.Status{
		Code:    models.StatusDeleting,
		Message: models.CloneMessageDeleting,
	}); err != nil {
		return errors.Wrap(err, "failed to update clone status")
	}

	if w.session == nil {
		c.deleteClone(cloneID)

		return nil
	}

	go func() {
		if err := c.provision.StopSession(w.session); err != nil {
			log.Errf("Failed to delete a clone: %+v.", err)

			if updateErr := c.updateCloneStatus(cloneID, models.Status{
				Code:    models.StatusFatal,
				Message: errors.Cause(err).Error(),
			}); updateErr != nil {
				log.Errf("Failed to update clone status: %v", updateErr)
			}

			return
		}

		c.deleteClone(cloneID)
	}()

	return nil
}

func (c *baseCloning) GetClone(id string) (*models.Clone, error) {
	w, ok := c.findWrapper(id)
	if !ok {
		return nil, errors.New("clone not found")
	}

	if w.session == nil {
		// Not started yet.
		return w.clone, nil
	}

	sessionState, err := c.provision.GetSessionState(w.session)
	if err != nil {
		// Session not ready yet.
		log.Err(errors.Wrap(err, "failed to get a session state"))

		return w.clone, nil
	}

	w.clone.Metadata.CloneDiffSize = sessionState.CloneDiffSize

	return w.clone, nil
}

func (c *baseCloning) UpdateClone(id string, patch *types.CloneUpdateRequest) (*models.Clone, error) {
	w, ok := c.findWrapper(id)
	if !ok {
		return nil, models.New(models.ErrCodeNotFound, "clone not found")
	}

	var clone *models.Clone

	// Set fields.
	c.cloneMutex.Lock()
	w.clone.Protected = patch.Protected

	clone = w.clone
	c.cloneMutex.Unlock()

	return clone, nil
}

func (c *baseCloning) ResetClone(cloneID string) error {
	w, ok := c.findWrapper(cloneID)
	if !ok {
		return models.New(models.ErrCodeNotFound, "clone not found")
	}

	if err := c.updateCloneStatus(cloneID, models.Status{
		Code:    models.StatusResetting,
		Message: models.CloneMessageResetting,
	}); err != nil {
		return errors.Wrap(err, "failed to update clone status")
	}

	if w.session == nil {
		return models.New(models.ErrCodeNotFound, "clone is not started yet")
	}

	go func() {
		err := c.provision.ResetSession(w.session, w.snapshot.ID)
		if err != nil {
			log.Errf("Failed to reset a clone: %+v.", err)

			if updateErr := c.updateCloneStatus(cloneID, models.Status{
				Code:    models.StatusFatal,
				Message: errors.Cause(err).Error(),
			}); updateErr != nil {
				log.Errf("failed to update clone status: %v", updateErr)
			}

			return
		}

		if err := c.updateCloneStatus(cloneID, models.Status{
			Code:    models.StatusOK,
			Message: models.CloneMessageOK,
		}); err != nil {
			log.Errf("failed to update clone status: %v", err)
		}
	}()

	return nil
}

func (c *baseCloning) GetInstanceState() (*models.InstanceStatus, error) {
	disk, err := c.provision.GetDiskState()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get a disk state")
	}

	c.instanceStatus.FileSystem.Size = disk.Size
	c.instanceStatus.FileSystem.Free = disk.Free
	c.instanceStatus.FileSystem.Used = disk.Used
	c.instanceStatus.DataSize = disk.DataSize
	c.instanceStatus.ExpectedCloningTime = c.getExpectedCloningTime()
	c.instanceStatus.Clones = c.GetClones()
	c.instanceStatus.NumClones = uint64(len(c.instanceStatus.Clones))

	return c.instanceStatus, nil
}

func (c *baseCloning) GetSnapshots() ([]models.Snapshot, error) {
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
func (c *baseCloning) GetClones() []*models.Clone {
	clones := make([]*models.Clone, 0, c.lenClones())

	c.cloneMutex.RLock()
	for _, cloneWrapper := range c.clones {
		clones = append(clones, cloneWrapper.clone)
	}
	c.cloneMutex.RUnlock()

	sort.Slice(clones, func(i, j int) bool {
		return clones[i].CreatedAt > clones[j].CreatedAt
	})

	return clones
}

// findWrapper retrieves a clone findWrapper by id.
func (c *baseCloning) findWrapper(id string) (*CloneWrapper, bool) {
	c.cloneMutex.RLock()
	w, ok := c.clones[id]
	c.cloneMutex.RUnlock()

	return w, ok
}

// setWrapper adds a clone wrapper to the map of clones.
func (c *baseCloning) setWrapper(id string, wrapper *CloneWrapper) {
	c.cloneMutex.Lock()
	c.clones[id] = wrapper
	c.cloneMutex.Unlock()
}

// updateCloneStatus updates the clone status.
func (c *baseCloning) updateCloneStatus(cloneID string, status models.Status) error {
	c.cloneMutex.Lock()
	defer c.cloneMutex.Unlock()

	w, ok := c.clones[cloneID]
	if !ok {
		return errors.Errorf("clone %q not found", cloneID)
	}

	w.clone.Status = status

	return nil
}

// deleteClone removes the clone by ID.
func (c *baseCloning) deleteClone(cloneID string) {
	c.cloneMutex.Lock()
	delete(c.clones, cloneID)
	c.cloneMutex.Unlock()
}

// lenClones returns the number of clones.
func (c *baseCloning) lenClones() int {
	c.cloneMutex.RLock()
	lenClones := len(c.clones)
	c.cloneMutex.RUnlock()

	return lenClones
}

func (c *baseCloning) getExpectedCloningTime() float64 {
	lenClones := c.lenClones()

	if lenClones == 0 {
		return 0
	}

	sum := 0.0

	c.cloneMutex.RLock()
	for _, cloneWrapper := range c.clones {
		sum += cloneWrapper.clone.Metadata.CloningTime
	}
	c.cloneMutex.RUnlock()

	return sum / float64(lenClones)
}

func (c *baseCloning) fetchSnapshots() error {
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
func (c *baseCloning) getLatestSnapshot() (models.Snapshot, error) {
	c.snapshotMutex.RLock()
	defer c.snapshotMutex.RUnlock()

	if len(c.snapshots) == 0 {
		return models.Snapshot{}, errors.New("no snapshot found")
	}

	snapshot := c.snapshots[0]

	return snapshot, nil
}

// getSnapshotByID returns the snapshot by ID.
func (c *baseCloning) getSnapshotByID(snapshotID string) (models.Snapshot, error) {
	c.snapshotMutex.RLock()
	defer c.snapshotMutex.RUnlock()

	for _, snapshot := range c.snapshots {
		if snapshot.ID == snapshotID {
			return snapshot, nil
		}
	}

	return models.Snapshot{}, errors.New("no snapshot found")
}

func (c *baseCloning) runIdleCheck(ctx context.Context) {
	if c.Config.MaxIdleMinutes == 0 {
		return
	}

	idleTimer := time.NewTimer(idleCheckDuration)

	for {
		select {
		case <-idleTimer.C:
			c.destroyIdleClones(ctx)
			idleTimer.Reset(idleCheckDuration)

		case <-ctx.Done():
			idleTimer.Stop()
			return
		}
	}
}

func (c *baseCloning) destroyIdleClones(ctx context.Context) {
	for _, cloneWrapper := range c.clones {
		select {
		case <-ctx.Done():
			return
		default:
			isIdleClone, err := c.isIdleClone(cloneWrapper)
			if err != nil {
				log.Errf("Failed to check the idleness of clone %s: %v.", cloneWrapper.clone.ID, err)
				continue
			}

			if isIdleClone {
				log.Msg(fmt.Sprintf("Idle clone %q is going to be removed.", cloneWrapper.clone.ID))

				if err = c.DestroyClone(cloneWrapper.clone.ID); err != nil {
					log.Errf("Failed to destroy clone: %+v.", err)
					continue
				}
			}
		}
	}
}

// isIdleClone checks if clone is idle.
func (c *baseCloning) isIdleClone(wrapper *CloneWrapper) (bool, error) {
	currentTime := time.Now()

	idleDuration := time.Duration(c.Config.MaxIdleMinutes) * time.Minute

	availableIdleTime := wrapper.timeStartedAt.Add(idleDuration)
	if wrapper.clone.Protected || availableIdleTime.After(currentTime) {
		return false, nil
	}

	session := wrapper.session

	// TODO(akartasov): Remove wrappers without session.
	if session == nil {
		return false, errors.New("failed to get clone session")
	}

	lastSessionActivity, err := c.provision.LastSessionActivity(session, idleDuration)
	if err != nil {
		if err == pglog.ErrNotFound {
			log.Dbg(fmt.Sprintf("Not found recent activity for the session: %q. Session name: %q",
				session.ID, session.Name))

			return hasNotQueryActivity(session)
		}

		return false, errors.Wrap(err, "failed to get the last session activity")
	}

	// Check extracted activity time.
	availableSessionActivity := lastSessionActivity.Add(idleDuration)
	if availableSessionActivity.After(currentTime) {
		return false, nil
	}

	return hasNotQueryActivity(session)
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
