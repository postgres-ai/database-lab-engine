/*
2019 © Postgres.ai
*/

package cloning

import (
	"fmt"
	"strconv"
	"time"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/models"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision"
	"gitlab.com/postgres-ai/database-lab/pkg/util"

	"github.com/pkg/errors"
	"github.com/rs/xid"
)

type baseCloning struct {
	cloning

	clones         map[string]*CloneWrapper
	instanceStatus *models.InstanceStatus
	snapshots      []*models.Snapshot

	provision provision.Provision
}

// TODO(anatoly): Delete idle clones.
// NewBaseCloning instances a new base Cloning.
func NewBaseCloning(cfg *Config, provision provision.Provision) Cloning {
	var instanceStatusActualStatus = &models.Status{
		Code:    "OK",
		Message: "Instance is ready",
	}

	var fs = &models.FileSystem{}

	var instanceStatus = models.InstanceStatus{
		Status:     instanceStatusActualStatus,
		FileSystem: fs,
		Clones:     make([]*models.Clone, 0),
	}

	cloning := &baseCloning{}
	cloning.Config = cfg
	cloning.clones = make(map[string]*CloneWrapper)
	cloning.instanceStatus = &instanceStatus
	cloning.provision = provision

	return cloning
}

// Initialize and run cloning component.
func (c *baseCloning) Run() error {
	if err := c.provision.Init(); err != nil {
		return errors.Wrap(err, "failed to run cloning")
	}

	// TODO(anatoly): Run interval for stopping idle sessions.

	return nil
}

func (c *baseCloning) CreateClone(clone *models.Clone) error {
	if len(clone.Name) == 0 {
		return errors.New("missing clone name")
	}

	if clone.Db == nil {
		return errors.New("missing both DB username and password")
	}

	if len(clone.Db.Username) == 0 {
		return errors.New("missing DB username")
	}

	if len(clone.Db.Password) == 0 {
		return errors.New("missing DB password")
	}

	clone.ID = xid.New().String()
	w := NewCloneWrapper(clone)
	c.clones[clone.ID] = w

	clone.Status = statusCreating

	w.timeCreatedAt = time.Now()
	clone.CreatedAt = util.FormatTime(w.timeCreatedAt)

	w.username = clone.Db.Username
	w.password = clone.Db.Password
	clone.Db.Password = ""

	w.snapshot = clone.Snapshot

	go func() {
		snapshotID := ""
		if w.snapshot != nil && len(w.snapshot.ID) > 0 {
			snapshotID = w.snapshot.ID
		}

		session, err := c.provision.StartSession(w.username, w.password, snapshotID)
		if err != nil {
			// TODO(anatoly): Empty room case.
			log.Errf("failed to start session: %+v", err)
			clone.Status = statusFatal
			return
		}

		w.session = session

		w.timeStartedAt = time.Now()
		clone.CloningTime = w.timeStartedAt.Sub(w.timeCreatedAt).Seconds()

		clone.Status = statusOk
		clone.Db.Port = strconv.FormatUint(uint64(session.Port), 10)

		clone.Db.Host = c.Config.AccessHost
		clone.Db.ConnStr = fmt.Sprintf("host=%s port=%s username=%s",
			clone.Db.Host, clone.Db.Port, clone.Db.Username)

		clone.Snapshot = c.snapshots[len(c.snapshots)-1]

		// TODO(anatoly): Remove mock data.
		clone.CloneSize = 10
	}()

	return nil
}

func (c *baseCloning) DestroyClone(id string) error {
	w, ok := c.clones[id]
	if !ok {
		return errors.New("clone not found")
	}

	if w.clone.Protected {
		return errors.New("clone is protected")
	}

	w.clone.Status = statusDeleting

	if w.session == nil {
		return errors.New("clone is not started yet")
	}

	go func() {
		err := c.provision.StopSession(w.session)
		if err != nil {
			log.Errf("failed to delete clone: %+v", err)
			w.clone.Status = statusFatal
			return
		}

		delete(c.clones, w.clone.ID)
	}()

	return nil
}

func (c *baseCloning) GetClone(id string) (*models.Clone, error) {
	w, ok := c.clones[id]
	if !ok {
		return nil, errors.New("clone not found")
	}

	if w.session == nil {
		// Not started yet.
		return w.clone, nil
	}

	sessionState, err := c.provision.GetSessionState(w.session)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get a session state")
	}

	w.clone.CloneSize = sessionState.CloneSize

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

	if patch.CloneSize > 0 {
		return errors.New("CloneSize cannot be changed")
	}

	if patch.CloningTime > 0 {
		return errors.New("CloningTime cannot be changed")
	}

	if len(patch.Project) > 0 {
		return errors.New("Project cannot be changed")
	}

	if patch.Db != nil {
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

	w, ok := c.clones[id]
	if !ok {
		return errors.New("clone not found")
	}

	// Set fields.
	if len(patch.Name) > 0 {
		w.clone.Name = patch.Name
	}

	w.clone.Protected = patch.Protected

	return nil
}

func (c *baseCloning) ResetClone(id string) error {
	w, ok := c.clones[id]
	if !ok {
		return errors.New("clone not found")
	}

	w.clone.Status = statusResetting

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
			log.Errf("failed to reset session: %+v", err)
			w.clone.Status = statusFatal

			return
		}

		w.clone.Status = statusOk
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

func (c *baseCloning) GetClones() []*models.Clone {
	clones := make([]*models.Clone, 0)
	for _, clone := range c.clones {
		clones = append(clones, clone.clone)
	}
	return clones
}

func (c *baseCloning) getExpectedCloningTime() float64 {
	if len(c.clones) == 0 {
		return 0
	}

	sum := 0.0
	for _, clone := range c.clones {
		sum += clone.clone.CloningTime
	}

	return sum / float64(len(c.clones))
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
