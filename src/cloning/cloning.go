package cloning

import (
	"fmt"
	"time"

	"../log"
	m "../models"
	p "../provision"
	"../util"

	"github.com/rs/xid"
)

// TODO(anatoly): const IDLE_TICK_DURATION = 120 * time.Minute

type Config struct {
	AutoDelete bool   `yaml:"autoDelete"`
	IdleTime   uint   `yaml:"idleTime"`
	AccessHost string `yaml:"accessHost"`
}

type Cloning struct {
	Config *Config

	clones         map[string]*CloneWrapper
	instanceStatus *m.InstanceStatus
	snapshots      []*m.Snapshot

	provision p.Provision
	sessions  map[string]*p.Session
}

type CloneWrapper struct {
	clone   *m.Clone
	session *p.Session

	timeCreatedAt time.Time
	timeStartedAt time.Time
}

func NewCloneWrapper(clone *m.Clone) *CloneWrapper {
	w := &CloneWrapper{
		clone: clone,
	}

	if clone.Db == nil {
		clone.Db = &m.Database{}
	}

	return w
}

func NewCloning(cfg *Config, provision p.Provision) *Cloning {
	var instanceStatusActualStatus = &m.Status{
		Code:    "OK",
		Message: "Instance is ready",
	}

	var disk = &m.Disk{
		Size: 10000,
		Free: 100,
	}

	var instanceStatus = m.InstanceStatus{
		Status:              instanceStatusActualStatus,
		Disk:                disk,
		DataSize:            100000,
		ExpectedCloningTime: 5.0,
		NumClones:           2,
		Clones:              make([]*m.Clone, 0),
	}

	// TODO(anatoly): Fetch snapshots.
	var snapshot1 = m.Snapshot{
		Id:        "1000:10:10 (latest)",
		Timestamp: "2019-12-02 16:25:55 UTC",
	}

	var snapshots = []*m.Snapshot{
		&snapshot1,
	}

	cloning := &Cloning{
		Config:         cfg,
		clones:         make(map[string]*CloneWrapper),
		snapshots:      snapshots,
		instanceStatus: &instanceStatus,
		provision:      provision,
		sessions:       make(map[string]*p.Session),
	}

	return cloning
}

func (c *Cloning) Run() error {
	err := c.provision.Init()
	if err != nil {
		log.Err("CloningRun:", err)
		return err
	}

	// TODO(anatoly): Run interval for stopping idle sessions.
	/*
		_ = util.RunInterval(IDLE_TICK_DURATION, func() {
			log.Dbg("Stop idle sessions tick")
			b.stopIdleSessions()
		})
	*/

	return nil
}

func (c *Cloning) CreateClone(clone *m.Clone) error {
	if len(clone.Name) == 0 {
		return fmt.Errorf("Missing required fields.")
	}

	clone.Id = xid.New().String()
	w := NewCloneWrapper(clone)
	c.clones[clone.Id] = w

	clone.Status = statusCreating

	w.timeCreatedAt = time.Now()
	clone.CreatedAt = util.FormatTime(w.timeCreatedAt)

	go func() {
		session, err := c.provision.StartSession()
		if err != nil {
			// TODO(anatoly): Empty room case.
			log.Err("Failed to create a clone:", err)
			clone.Status = statusFatal
			return
		}

		w.session = session

		w.timeStartedAt = time.Now()
		clone.CloningTime = w.timeStartedAt.Sub(w.timeCreatedAt).Seconds()

		clone.Status = statusOk
		clone.Db.Port = fmt.Sprintf("%d", session.Port)

		// TODO(anatoly): Use username and passsword from clone creation request.
		clone.Db.Username = "postgres"
		clone.Db.Password = "postgres"

		clone.Db.Host = c.Config.AccessHost
		clone.Db.ConnStr = fmt.Sprintf("host=%s port=%s username=postgres password=[ENTER_PASSWORD]", clone.Db.Host, clone.Db.Port)

		// TODO(anatoly): Remove mock data.
		clone.CloneSize = 10
		clone.Snapshot = "latest"
	}()

	return nil
}

func (c *Cloning) DestroyClone(id string) error {
	session, ok := c.sessions[id]
	if !ok {
		err := fmt.Errorf("Clone not found.")
		log.Err(err)
		return err
	}

	clone, ok := c.GetClone(id)
	if !ok {
		err := fmt.Errorf("Clone not found.")
		log.Err(err)
		return err
	}

	clone.Status = statusDeleting

	go func() {
		err := c.provision.StopSession(session)
		if err != nil {
			log.Err("Failed to delete a clone:", err)
			clone.Status = statusFatal
			return
		}

		delete(c.clones, clone.Id)
	}()

	return nil
}

func (c *Cloning) ResetClone(id string) error {
	// TODO(anatoly): Implement.
	return nil
}

func (c *Cloning) GetInstanceState() (*m.InstanceStatus, error) {
	disk, err := c.provision.GetDiskState()
	if err != nil {
		return &m.InstanceStatus{}, err
	}

	log.Dbg(disk)

	c.instanceStatus.Disk.Size = disk.Size
	c.instanceStatus.Disk.Free = disk.Free
	c.instanceStatus.DataSize = disk.DataSize

	c.instanceStatus.ExpectedCloningTime = c.getExpectedCloningTime()

	// TODO(anatoly): Fix. Dirty.
	c.instanceStatus.Clones = c.GetClones()
	return c.instanceStatus, nil
}

func (c *Cloning) GetSnapshots() []*m.Snapshot {
	return c.snapshots
}

func (c *Cloning) GetClone(cloneId string) (*m.Clone, bool) {
	clone, ok := c.clones[cloneId]
	if !ok {
		return &m.Clone{}, false
	}

	return clone.clone, true
}

func (c *Cloning) GetClones() []*m.Clone {
	clones := make([]*m.Clone, 0)
	for _, clone := range c.clones {
		clones = append(clones, clone.clone)
	}
	return clones
}

func (c *Cloning) getExpectedCloningTime() float64 {
	if len(c.clones) == 0 {
		return 0
	}

	sum := 0.0
	for _, clone := range c.clones {
		sum += clone.clone.CloningTime
	}

	return sum / float64(len(c.clones))
}

// TODO(anatoly):
/*
	session, err := b.Prov.StartSession()
	if err != nil {
		switch err.(type) {
		case provision.NoRoomError:
			err = b.stopIdleSessions()
			if err != nil {
				failMsg(sMsg, err.Error())
				return
			}

			session, err = b.Prov.StartSession()
			if err != nil {
				failMsg(sMsg, err.Error())
				return
			}
		default:
			failMsg(sMsg, err.Error())
			return
		}
	}

func (b *Bot) stopIdleSessions() error {
	chsNotify := make(map[string][]string)

	for _, u := range b.Users {
		if u == nil {
			continue
		}

		s := u.Session
		if s.Provision == nil {
			continue
		}

		interval := u.Session.IdleInterval
		sAgo := util.SecondsAgo(u.Session.LastActionTs)

		if sAgo < interval {
			continue
		}

		log.Dbg("Session idle: %v %v", u, s)

		for _, ch := range u.Session.ChannelIds {
			uId := u.ChatUser.ID
			chNotify, ok := chsNotify[ch]
			if !ok {
				chsNotify[ch] = []string{uId}
				continue
			}

			chsNotify[ch] = append(chNotify, uId)
		}

		b.stopSession(u)
	}

	// Publish message in every channel with a list of users.
	for ch, uIds := range chsNotify {
		if len(uIds) == 0 {
			continue
		}

		list := ""
		for _, uId := range uIds {
			if len(list) > 0 {
				list += ", "
			}
			list += fmt.Sprintf("<@%s>", uId)
		}

		msgText := "Stopped idle sessions for: " + list

		msg, _ := b.Chat.NewMessage(ch)
		err := msg.Publish(msgText)
		if err != nil {
			log.Err("Bot: Cannot publish a message", err)
		}
	}

	return nil
}

func (b *Bot) stopAllSessions() error {
	for _, u := range b.Users {
		if u == nil {
			continue
		}

		s := u.Session
		if s.Provision == nil {
			continue
		}

		b.stopSession(u)
	}

	return nil
}

func (b *Bot) stopSession(u *User) error {
	log.Dbg("Stopping session...")

	err := b.Prov.StopSession(u.Session.Provision)

	u.Session.Provision = nil
	u.Session.PlatformSessionId = ""

	if err != nil {
		log.Err(err)
		return err
	}

	return nil
}*/
