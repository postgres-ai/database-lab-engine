/*
2019 © Postgres.ai
*/

package cloning

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/models"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision"
	"gitlab.com/postgres-ai/database-lab/pkg/util"
	"gitlab.com/postgres-ai/database-lab/pkg/util/pglog"

	"github.com/pkg/errors"
	"github.com/rs/xid"
)

const idleCheckDuration = 5 * time.Minute

type baseCloning struct {
	cloning

	cloneMutex     sync.RWMutex
	clones         map[string]*CloneWrapper
	instanceStatus *models.InstanceStatus
	snapshots      []*models.Snapshot

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

	go c.runIdleCheck(ctx)

	return nil
}

// CreateClone creates a new clone.
func (c *baseCloning) CreateClone(clone *models.Clone) error {
	// TODO(akartasov): Separate validation rules.
	clone.ID = strings.TrimSpace(clone.ID)

	if _, ok := c.findWrapper(clone.ID); ok {
		return errors.New("clone with such ID already exists")
	}

	if clone.DB == nil {
		return errors.New("missing both DB username and password")
	}

	if len(clone.DB.Username) == 0 {
		return errors.New("missing DB username")
	}

	if len(clone.DB.Password) == 0 {
		return errors.New("missing DB password")
	}

	if clone.ID == "" {
		clone.ID = xid.New().String()
	}

	w := NewCloneWrapper(clone)

	c.updateStatus(w.clone, models.Status{
		Code:    models.StatusCreating,
		Message: models.CloneMessageCreating,
	})

	w.timeCreatedAt = time.Now()
	clone.CreatedAt = util.FormatTime(w.timeCreatedAt)

	w.username = clone.DB.Username
	w.password = clone.DB.Password
	clone.DB.Password = ""

	w.snapshot = clone.Snapshot

	err := c.fetchSnapshots()
	if err != nil {
		return errors.Wrap(err, "failed to create clone")
	}

	go func() {
		snapshotID := ""
		if w.snapshot != nil && len(w.snapshot.ID) > 0 {
			snapshotID = w.snapshot.ID
		}

		session, err := c.provision.StartSession(w.username, w.password, snapshotID)
		if err != nil {
			// TODO(anatoly): Empty room case.
			c.updateStatus(w.clone, models.Status{
				Code:    models.StatusFatal,
				Message: models.CloneMessageFatal,
			})

			log.Errf("Failed to start session: %+v.", err)

			return
		}

		w.session = session

		w.timeStartedAt = time.Now()

		c.updateStatus(w.clone, models.Status{
			Code:    models.StatusOK,
			Message: models.CloneMessageOK,
		})

		clone.DB.Port = strconv.FormatUint(uint64(session.Port), 10)

		clone.DB.Host = c.Config.AccessHost
		clone.DB.ConnStr = fmt.Sprintf("host=%s port=%s user=%s",
			clone.DB.Host, clone.DB.Port, clone.DB.Username)

		clone.Snapshot = c.snapshots[len(c.snapshots)-1]

		// TODO(anatoly): Remove mock data.
		clone.Metadata = &models.CloneMetadata{
			CloneSize:      cloneSize,
			CloningTime:    w.timeStartedAt.Sub(w.timeCreatedAt).Seconds(),
			MaxIdleMinutes: c.Config.IdleTime,
		}
	}()

	c.setWrapper(clone.ID, w)

	return nil
}

func (c *baseCloning) DestroyClone(id string) error {
	w, ok := c.findWrapper(id)
	if !ok {
		return errors.New("clone not found")
	}

	if w.clone.Protected {
		return errors.New("clone is protected")
	}

	c.updateStatus(w.clone, models.Status{
		Code:    models.StatusDeleting,
		Message: models.CloneMessageDeleting,
	})

	if w.session == nil {
		return errors.New("clone is not started yet")
	}

	go func() {
		if err := c.provision.StopSession(w.session); err != nil {
			c.updateStatus(w.clone, models.Status{
				Code:    models.StatusFatal,
				Message: models.CloneMessageFatal,
			})

			log.Errf("Failed to delete clone: %+v.", err)

			return
		}

		c.deleteClone(w.clone.ID)
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

	w.clone.Metadata.CloneSize = sessionState.CloneSize

	return w.clone, nil
}

func (c *baseCloning) UpdateClone(id string, patch *models.Clone) error {
	// TODO(anatoly): Nullable fields?
	// Check unmodifiable fields.
	if len(patch.ID) > 0 {
		return errors.New("ID cannot be changed")
	}

	if patch.Snapshot != nil {
		return errors.New("Snapshot cannot be changed")
	}

	if patch.Metadata != nil {
		return errors.New("Metadata cannot be changed")
	}

	if len(patch.Project) > 0 {
		return errors.New("Project cannot be changed")
	}

	if patch.DB != nil {
		return errors.New("Database cannot be changed")
	}

	if patch.Status != nil {
		return errors.New("Status cannot be changed")
	}

	if len(patch.DeleteAt) > 0 {
		return errors.New("DeleteAt cannot be changed")
	}

	if len(patch.CreatedAt) > 0 {
		return errors.New("CreatedAt cannot be changed")
	}

	w, ok := c.findWrapper(id)
	if !ok {
		return errors.New("clone not found")
	}

	// Set fields.

	c.cloneMutex.Lock()
	w.clone.Protected = patch.Protected
	c.cloneMutex.Unlock()

	return nil
}

func (c *baseCloning) ResetClone(id string) error {
	w, ok := c.findWrapper(id)
	if !ok {
		return errors.New("clone not found")
	}

	c.updateStatus(w.clone, models.Status{
		Code:    models.StatusResetting,
		Message: models.CloneMessageResetting,
	})

	if w.session == nil {
		return errors.New("clone is not started yet")
	}

	go func() {
		snapshotID := ""
		if w.snapshot != nil && len(w.snapshot.ID) > 0 {
			snapshotID = w.snapshot.ID
		}

		err := c.provision.ResetSession(w.session, snapshotID)
		if err != nil {
			c.updateStatus(w.clone, models.Status{
				Code:    models.StatusFatal,
				Message: models.CloneMessageFatal,
			})

			log.Errf("Failed to reset session: %+v.", err)

			return
		}

		c.updateStatus(w.clone, models.Status{
			Code:    models.StatusOK,
			Message: models.CloneMessageOK,
		})
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
	c.instanceStatus.DataSize = disk.DataSize
	c.instanceStatus.ExpectedCloningTime = c.getExpectedCloningTime()
	c.instanceStatus.Clones = c.GetClones()
	c.instanceStatus.NumClones = uint64(len(c.instanceStatus.Clones))

	return c.instanceStatus, nil
}

func (c *baseCloning) GetSnapshots() ([]*models.Snapshot, error) {
	// TODO(anatoly): Update snapshots dynamically.
	if err := c.fetchSnapshots(); err != nil {
		return nil, errors.Wrap(err, "failed to fetch snapshots")
	}

	return c.snapshots, nil
}

// GetClones returns all clones.
func (c *baseCloning) GetClones() []*models.Clone {
	clones := make([]*models.Clone, 0, c.lenClones())

	c.cloneMutex.RLock()
	for _, clone := range c.clones {
		clones = append(clones, clone.clone)
	}
	c.cloneMutex.RUnlock()

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

// updateStatus updates the clone status.
func (c *baseCloning) updateStatus(clone *models.Clone, status models.Status) {
	c.cloneMutex.Lock()
	clone.Status = &status
	c.cloneMutex.Unlock()
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
		if cloneWrapper.clone.Metadata != nil {
			sum += cloneWrapper.clone.Metadata.CloningTime
		}
	}
	c.cloneMutex.RUnlock()

	return sum / float64(lenClones)
}

func (c *baseCloning) fetchSnapshots() error {
	entries, err := c.provision.GetSnapshots()
	if err != nil {
		return errors.Wrap(err, "failed to get snapshots")
	}

	snapshots := make([]*models.Snapshot, len(entries))

	for i, entry := range entries {
		snapshots[i] = &models.Snapshot{
			ID:          entry.ID,
			CreatedAt:   util.FormatTime(entry.CreatedAt),
			DataStateAt: util.FormatTime(entry.DataStateAt),
		}

		log.Dbg("snapshot:", snapshots[i])
	}

	c.snapshots = snapshots

	return nil
}

func (c *baseCloning) runIdleCheck(ctx context.Context) {
	if c.Config.IdleTime == 0 {
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
				log.Errf("Failed to check the idleness of clone %s: %+v.", cloneWrapper.clone.ID, err)
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

func (c *baseCloning) isIdleClone(wrapper *CloneWrapper) (bool, error) {
	currentTime := time.Now()

	idleDuration := time.Duration(c.Config.IdleTime) * time.Minute

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

			return true, nil
		}

		return false, errors.Wrap(err, "failed to get the last session activity")
	}

	// Check extracted activity time.
	availableSessionActivity := lastSessionActivity.Add(idleDuration)
	if availableSessionActivity.After(currentTime) {
		return false, nil
	}

	return true, nil
}
