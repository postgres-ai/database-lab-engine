/*
2019 © Postgres.ai
*/

package cloning

import (
	"fmt"
	"time"

	"gitlab.com/postgres-ai/database-lab/src/log"
	"gitlab.com/postgres-ai/database-lab/src/models"
	"gitlab.com/postgres-ai/database-lab/src/provision"
	"gitlab.com/postgres-ai/database-lab/src/util"

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
	err := c.provision.Init()
	if err != nil {
		log.Err("CloningRun:", err)
		return err
	}

	// TODO(anatoly): Run interval for stopping idle sessions.

	return nil
}

func (c *baseCloning) CreateClone(clone *models.Clone) error {
	if len(clone.Name) == 0 {
		return fmt.Errorf("Missing clone name.")
	}

	if clone.Db == nil {
		return fmt.Errorf("Missing both DB username and password.")
	}

	if len(clone.Db.Username) == 0 {
		return fmt.Errorf("Missing DB username.")
	}

	if len(clone.Db.Password) == 0 {
		return fmt.Errorf("Missing DB password.")
	}

	clone.Id = xid.New().String()
	w := NewCloneWrapper(clone)
	c.clones[clone.Id] = w

	clone.Status = statusCreating

	w.timeCreatedAt = time.Now()
	clone.CreatedAt = util.FormatTime(w.timeCreatedAt)

	w.username = clone.Db.Username
	w.password = clone.Db.Password
	clone.Db.Password = ""

	w.snapshot = clone.Snapshot

	go func() {
		snapshotId := ""
		if w.snapshot != nil && len(w.snapshot.Id) > 0 {
			snapshotId = w.snapshot.Id
		}

		session, err := c.provision.StartSession(w.username, w.password,
			snapshotId)
		if err != nil {
			// TODO(anatoly): Empty room case.
			log.Err("Failed to create clone:", err)
			clone.Status = statusFatal
			return
		}

		w.session = session

		w.timeStartedAt = time.Now()
		clone.CloningTime = w.timeStartedAt.Sub(w.timeCreatedAt).Seconds()

		clone.Status = statusOk
		clone.Db.Port = fmt.Sprintf("%d", session.Port)

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
		err := fmt.Errorf("Clone not found.")
		log.Err(err)
		return err
	}

	if w.clone.Protected {
		err := fmt.Errorf("Clone is protected.")
		log.Err(err)
		return err
	}

	w.clone.Status = statusDeleting

	if w.session == nil {
		return fmt.Errorf("Clone is not started yet.")
	}

	go func() {
		err := c.provision.StopSession(w.session)
		if err != nil {
			log.Err("Failed to delete clone:", err)
			w.clone.Status = statusFatal
			return
		}

		delete(c.clones, w.clone.Id)
	}()

	return nil
}

func (c *baseCloning) GetClone(id string) (*models.Clone, bool) {
	w, ok := c.clones[id]
	if !ok {
		return &models.Clone{}, false
	}

	if w.session == nil {
		// Not started yet.
		return w.clone, true
	}

	sessionState, err := c.provision.GetSessionState(w.session)
	if err != nil {
		log.Err(err)
		// TODO(anatoly): Error processing.
		return &models.Clone{}, false
	}

	w.clone.CloneSize = sessionState.CloneSize

	return w.clone, true
}

func (c *baseCloning) UpdateClone(id string, patch *models.Clone) error {
	// TODO(anatoly): Nullable fields?

	// Check unmodifiable fields.
	if len(patch.Id) > 0 {
		err := fmt.Errorf("ID cannot be changed.")
		log.Err(err)
		return err
	}

	if patch.Snapshot != nil {
		err := fmt.Errorf("Snapshot cannot be changed.")
		log.Err(err)
		return err
	}

	if patch.CloneSize > 0 {
		err := fmt.Errorf("CloneSize cannot be changed.")
		log.Err(err)
		return err
	}

	if patch.CloningTime > 0 {
		err := fmt.Errorf("CloningTime cannot be changed.")
		log.Err(err)
		return err
	}

	if len(patch.Project) > 0 {
		err := fmt.Errorf("Project cannot be changed.")
		log.Err(err)
		return err
	}

	if patch.Db != nil {
		err := fmt.Errorf("Database cannot be changed.")
		log.Err(err)
		return err
	}

	if patch.Status != nil {
		err := fmt.Errorf("Status cannot be changed.")
		log.Err(err)
		return err
	}

	if len(patch.DeleteAt) > 0 {
		err := fmt.Errorf("DeleteAt cannot be changed.")
		log.Err(err)
		return err
	}

	if len(patch.CreatedAt) > 0 {
		err := fmt.Errorf("CreatedAt cannot be changed.")
		log.Err(err)
		return err
	}

	w, ok := c.clones[id]
	if !ok {
		err := fmt.Errorf("Clone not found.")
		log.Err(err)
		return err
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
		err := fmt.Errorf("Clone not found.")
		log.Err(err)
		return err
	}

	w.clone.Status = statusResetting

	if w.session == nil {
		return fmt.Errorf("Clone is not started yet.")
	}

	go func() {
		snapshotId := ""
		if w.snapshot != nil && len(w.snapshot.Id) > 0 {
			snapshotId = w.snapshot.Id
		}

		err := c.provision.ResetSession(w.session, snapshotId)
		if err != nil {
			log.Err("Failed to reset clone:", err)
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
		return &models.InstanceStatus{}, err
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
	err := c.fetchSnapshots()
	if err != nil {
		log.Err("CloningRun:", err)
		return []*models.Snapshot{}, err
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
		log.Err(err)
		return err
	}

	snapshots := make([]*models.Snapshot, len(entries))

	for i, entry := range entries {
		snapshots[i] = &models.Snapshot{
			Id:          entry.Id,
			CreatedAt:   util.FormatTime(entry.CreatedAt),
			DataStateAt: util.FormatTime(entry.DataStateAt),
		}

		log.Dbg("snapshot:", snapshots[i])
	}

	c.snapshots = snapshots

	return nil
}
