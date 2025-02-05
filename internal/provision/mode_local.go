/*
2019 © Postgres.ai
*/

package provision

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/databases/postgres"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/docker"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones/zfs"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/runners"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/networks"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/pglog"
)

const (
	maxNumberOfPortsToCheck = 5
	portCheckingTimeout     = 3 * time.Second
	unknownVersion          = "unknown"
)

// PortPool describes an available port range for clones.
type PortPool struct {
	From uint `yaml:"from"`
	To   uint `yaml:"to"`
}

// Config defines configuration for provisioning.
type Config struct {
	PortPool          PortPool          `yaml:"portPool"`
	DockerImage       string            `yaml:"dockerImage"`
	UseSudo           bool              `yaml:"useSudo"`
	KeepUserPasswords bool              `yaml:"keepUserPasswords"`
	ContainerConfig   map[string]string `yaml:"containerConfig"`
}

// Provisioner describes a struct for ports and clones management.
type Provisioner struct {
	config         *Config
	dbCfg          *resources.DB
	ctx            context.Context
	dockerClient   *client.Client
	runner         runners.Runner
	mu             *sync.Mutex
	ports          []bool
	sessionCounter uint32
	portChecker    portChecker
	pm             *pool.Manager
	networkID      string
	instanceID     string
}

// New creates a new Provisioner instance.
func New(ctx context.Context, cfg *Config, dbCfg *resources.DB, docker *client.Client, pm *pool.Manager,
	instanceID, networkID string) (*Provisioner, error) {
	if err := IsValidConfig(*cfg); err != nil {
		return nil, errors.Wrap(err, "configuration is not valid")
	}

	p := &Provisioner{
		runner:       runners.NewLocalRunner(cfg.UseSudo),
		mu:           &sync.Mutex{},
		dockerClient: docker,
		config:       cfg,
		dbCfg:        dbCfg,
		ctx:          ctx,
		portChecker:  &localPortChecker{},
		pm:           pm,
		networkID:    networkID,
		instanceID:   instanceID,
		ports:        make([]bool, cfg.PortPool.To-cfg.PortPool.From),
	}

	return p, nil
}

// IsValidConfig defines a method for validation of a configuration.
func IsValidConfig(cfg Config) error {
	return isValidConfigModeLocal(cfg)
}

func isValidConfigModeLocal(config Config) error {
	portPool := config.PortPool

	if portPool.From == 0 {
		return errors.New(`"portPool.from" must be defined and be greater than 0`)
	}

	if portPool.To == 0 {
		return errors.New(`"portPool.to" must be defined and be greater than 0`)
	}

	if portPool.To <= portPool.From {
		return errors.New(`"portPool" must include at least one port`)
	}

	return nil
}

// Init inits provision.
func (p *Provisioner) Init() error {
	if err := p.RevisePortPool(); err != nil {
		return fmt.Errorf("failed to revise port pool: %w", err)
	}

	if err := docker.PrepareImage(p.runner, p.config.DockerImage); err != nil {
		return fmt.Errorf("cannot prepare docker image %s: %w", p.config.DockerImage, err)
	}

	return nil
}

// Reload reloads provision configuration.
func (p *Provisioner) Reload(cfg Config, dbCfg resources.DB) {
	*p.config = cfg
	*p.dbCfg = dbCfg
}

// ContainerOptions returns provisioner configuration for running containers.
func (p *Provisioner) ContainerOptions() models.ContainerOptions {
	return models.ContainerOptions{
		DockerImage:     p.config.DockerImage,
		ContainerConfig: p.config.ContainerConfig,
	}
}

// StartSession starts a new session.
func (p *Provisioner) StartSession(snapshotID string, user resources.EphemeralUser,
	extraConfig map[string]string) (*resources.Session, error) {
	snapshot, err := p.getSnapshot(snapshotID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get snapshots")
	}

	port, err := p.allocatePort()
	if err != nil {
		return nil, errors.New("failed to get a free port")
	}

	name := util.GetCloneName(port)

	fsm, err := p.pm.GetFSManager(snapshot.Pool)
	if err != nil {
		return nil, fmt.Errorf("cannot work with pool %s: %w", snapshot.Pool, err)
	}

	log.Dbg(fmt.Sprintf(`Starting session for port: %d.`, port))

	defer func() {
		if err != nil {
			p.revertSession(fsm, name)

			if portErr := p.FreePort(port); portErr != nil {
				log.Err(portErr)
			}
		}
	}()

	if err = fsm.CreateClone(name, snapshot.ID); err != nil {
		return nil, errors.Wrap(err, "failed to create clone")
	}

	appConfig := p.getAppConfig(fsm.Pool(), name, port)
	appConfig.SetExtraConf(extraConfig)

	if err = postgres.Start(p.runner, appConfig); err != nil {
		return nil, errors.Wrap(err, "failed to start a container")
	}

	if err = p.prepareDB(appConfig, user); err != nil {
		return nil, errors.Wrap(err, "failed to prepare a database")
	}

	atomic.AddUint32(&p.sessionCounter, 1)

	session := &resources.Session{
		ID:            strconv.FormatUint(uint64(p.sessionCounter), 10),
		Pool:          fsm.Pool().Name,
		Port:          port,
		User:          appConfig.DB.Username,
		SocketHost:    appConfig.Host,
		EphemeralUser: user,
		ExtraConfig:   extraConfig,
	}

	return session, nil
}

// StopSession stops an existing session.
func (p *Provisioner) StopSession(session *resources.Session) error {
	fsm, err := p.pm.GetFSManager(session.Pool)
	if err != nil {
		return errors.Wrap(err, "failed to find a filesystem manager of this session")
	}

	name := util.GetCloneName(session.Port)

	if err := postgres.Stop(p.runner, fsm.Pool(), name); err != nil {
		return errors.Wrap(err, "failed to stop a container")
	}

	if err := fsm.DestroyClone(name); err != nil {
		return errors.Wrap(err, "failed to destroy a clone")
	}

	if err := p.FreePort(session.Port); err != nil {
		return errors.Wrap(err, "failed to unbind a port")
	}

	return nil
}

// ResetSession resets an existing session.
func (p *Provisioner) ResetSession(session *resources.Session, snapshotID string) (*models.Snapshot, error) {
	fsm, err := p.pm.GetFSManager(session.Pool)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find filesystem manager of this session")
	}

	name := util.GetCloneName(session.Port)

	snapshot, err := p.getSnapshot(snapshotID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get snapshots")
	}

	log.Dbg("Snapshot ID to reset session: ", snapshot.ID)

	newFSManager := fsm

	if snapshot.Pool != session.Pool {
		newFSManager, err = p.pm.GetFSManager(snapshot.Pool)
		if err != nil {
			return nil, errors.Wrap(err, "failed to find filesystem manager for a new session")
		}

		session.Pool = snapshot.Pool
		session.SocketHost = newFSManager.Pool().SocketCloneDir(name)
	}

	defer func() {
		if err != nil {
			p.revertSession(newFSManager, name)
		}
	}()

	if err = postgres.Stop(p.runner, fsm.Pool(), name); err != nil {
		return nil, errors.Wrap(err, "failed to stop container")
	}

	if err = fsm.DestroyClone(name); err != nil {
		return nil, errors.Wrap(err, "failed to destroy clone")
	}

	if err = newFSManager.CreateClone(name, snapshot.ID); err != nil {
		return nil, errors.Wrap(err, "failed to create clone")
	}

	appConfig := p.getAppConfig(newFSManager.Pool(), name, session.Port)
	appConfig.SetExtraConf(session.ExtraConfig)

	if err = postgres.Start(p.runner, appConfig); err != nil {
		return nil, errors.Wrap(err, "failed to start container")
	}

	if err = p.prepareDB(appConfig, session.EphemeralUser); err != nil {
		return nil, errors.Wrap(err, "failed to prepare database")
	}

	snapshotModel := &models.Snapshot{
		ID:          snapshot.ID,
		CreatedAt:   util.FormatTime(snapshot.CreatedAt),
		DataStateAt: util.FormatTime(snapshot.DataStateAt),
	}

	return snapshotModel, nil
}

// GetSnapshots provides a snapshot list from active pools.
func (p *Provisioner) GetSnapshots() ([]resources.Snapshot, error) {
	snapshots := []resources.Snapshot{}

	for _, activeFSManager := range p.pm.GetActiveFSManagers() {
		poolSnapshots, err := activeFSManager.GetSnapshots()
		if err != nil {
			var emptyErr *zfs.EmptyPoolError
			if errors.As(err, &emptyErr) {
				log.Msg(emptyErr.Error())
				continue
			}

			log.Err(fmt.Errorf("failed to get snapshots for pool %s: %w", activeFSManager.Pool().Name, err))
		}

		snapshots = append(snapshots, poolSnapshots...)
	}

	return snapshots, nil
}

// GetSessionState describes the state of the session.
func (p *Provisioner) GetSessionState(s *resources.Session) (*resources.SessionState, error) {
	fsm, err := p.pm.GetFSManager(s.Pool)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find a filesystem manager of this session")
	}

	return fsm.GetSessionState(util.GetCloneName(s.Port))
}

// GetPoolEntryList provides an ordered list of available pools.
func (p *Provisioner) GetPoolEntryList() []models.PoolEntry {
	fsmList := p.pm.GetFSManagerOrderedList()
	pools := make([]models.PoolEntry, 0, len(fsmList))

	for _, fsManager := range fsmList {
		poolEntry, err := buildPoolEntry(fsManager)
		if err != nil {
			log.Err("skip pool entry: ", err.Error())
			continue
		}

		pools = append(pools, poolEntry)
	}

	return pools
}

func buildPoolEntry(fsm pool.FSManager) (models.PoolEntry, error) {
	fsmPool := fsm.Pool()
	if fsmPool == nil {
		return models.PoolEntry{}, errors.New("empty pool")
	}

	listClones, err := fsm.ListClonesNames()
	if err != nil {
		log.Err(fmt.Sprintf("failed to get clone list related to the pool %s", fsmPool.Name))
	}

	fileSystem, err := fsm.GetFilesystemState()
	if err != nil {
		log.Err(fmt.Sprintf("failed to get disk stats for the pool %s", fsmPool.Name))
	}

	var dataStateAt string
	if !fsmPool.DSA.IsZero() {
		dataStateAt = fsmPool.DSA.String()
	}

	poolEntry := models.PoolEntry{
		Name:        fsmPool.Name,
		Mode:        fsmPool.Mode,
		DataStateAt: dataStateAt,
		CloneList:   listClones,
		FileSystem:  fileSystem,
		Status:      fsm.Pool().Status(),
	}

	return poolEntry, nil
}

// Other methods.
func (p *Provisioner) revertSession(fsm pool.FSManager, name string) {
	log.Dbg(`Reverting start of a session...`)

	if runnerErr := postgres.Stop(p.runner, fsm.Pool(), name); runnerErr != nil {
		log.Err("Stop Postgres:", runnerErr)
	}

	if runnerErr := fsm.DestroyClone(name); runnerErr != nil {
		log.Err("Destroy clone:", runnerErr)
	}
}

func (p *Provisioner) getSnapshot(snapshotID string) (*resources.Snapshot, error) {
	snapshots, err := p.GetSnapshots()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get snapshots")
	}

	if len(snapshots) == 0 {
		return nil, errors.New("no snapshots available")
	}

	if snapshotID != "" {
		for _, snapshot := range snapshots {
			if snapshot.ID == snapshotID {
				return &snapshot, nil
			}
		}

		return nil, errors.Errorf("snapshot %q not found", snapshotID)
	}

	return &snapshots[0], nil
}

// RevisePortPool checks and aligns availability of the port range.
func (p *Provisioner) RevisePortPool() error {
	log.Msg(fmt.Sprintf("Revising availability of the port range [%d - %d]", p.config.PortPool.From, p.config.PortPool.To))

	host, err := externalIP()
	if err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	availablePorts := 0

	for port := p.config.PortPool.From; port < p.config.PortPool.To; port++ {
		if err := p.portChecker.checkPortAvailability(host, port); err != nil {
			log.Msg(fmt.Sprintf("port %d is not available, marking as busy", port))

			if err := p.setPortStatus(port, true); err != nil {
				return errors.Wrapf(err, "port %d is not available", port)
			}

			continue
		}

		if err := p.setPortStatus(port, false); err != nil {
			log.Err(fmt.Sprintf("cannot free port %d: %s", port, err))
		}

		availablePorts++
	}

	log.Msg(availablePorts, " ports are available")

	return nil
}

// allocatePort tries to find a free port and occupy it.
func (p *Provisioner) allocatePort() (uint, error) {
	portOpts := p.config.PortPool

	attempts := 0

	host, err := externalIP()
	if err != nil {
		return 0, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	for index, bind := range p.ports {
		if attempts >= maxNumberOfPortsToCheck {
			break
		}

		if bind {
			continue
		}

		port := portOpts.From + uint(index)

		log.Msg(fmt.Sprintf("checking port %d ...", port))

		if err := p.portChecker.checkPortAvailability(host, port); err != nil {
			log.Msg(fmt.Sprintf("port %d is not available: %v", port, err))
			attempts++

			continue
		}

		if err := p.setPortStatus(port, true); err != nil {
			return 0, errors.Wrapf(err, "failed to set status for port %v", port)
		}

		return port, nil
	}

	return 0, errors.WithStack(NewNoRoomError("no available ports"))
}

func externalIP() (string, error) {
	res, err := exec.Command("bash", "-c", "/sbin/ip route | awk '/default/ { print $3 }'").Output()
	if err != nil {
		return "", err
	}

	return string(bytes.TrimSpace(res)), nil
}

// FreePort marks the port as free.
func (p *Provisioner) FreePort(port uint) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.setPortStatus(port, false)
}

// setPortStatus updates the port status.
// It's not safe to invoke without ports mutex locking. Use allocatePort and FreePort methods.
func (p *Provisioner) setPortStatus(port uint, bind bool) error {
	portOpts := p.config.PortPool

	if port < portOpts.From || port >= portOpts.To {
		return errors.Errorf("port %d is out of bounds of the port pool", port)
	}

	index := port - portOpts.From
	p.ports[index] = bind

	return nil
}

// StopAllSessions stops all running clone containers.
func (p *Provisioner) StopAllSessions(exceptClones map[string]struct{}) error {
	for _, fsm := range p.pm.GetFSManagerList() {
		if err := p.stopPoolSessions(fsm, exceptClones); err != nil {
			return err
		}
	}

	return nil
}

func (p *Provisioner) stopPoolSessions(fsm pool.FSManager, exceptClones map[string]struct{}) error {
	fsPool := fsm.Pool()

	instances, err := postgres.List(p.runner, fsPool.Name)
	if err != nil {
		return errors.Wrap(err, "failed to list containers")
	}

	log.Dbg("Containers running:", instances)

	for _, instance := range instances {
		if _, ok := exceptClones[instance]; ok {
			continue
		}

		log.Dbg("Stopping container:", instance)

		if err = postgres.Stop(p.runner, fsPool, instance); err != nil {
			return errors.Wrap(err, "failed to container")
		}
	}

	clones, err := fsm.ListClonesNames()
	if err != nil {
		return err
	}

	log.Dbg("Clone list:", clones)

	for _, clone := range clones {
		if _, ok := exceptClones[clone]; ok {
			continue
		}

		if err := fsm.DestroyClone(clone); err != nil {
			return err
		}
	}

	return nil
}

func (p *Provisioner) getAppConfig(pool *resources.Pool, name string, port uint) *resources.AppConfig {
	appConfig := &resources.AppConfig{
		CloneName:     name,
		DockerImage:   p.config.DockerImage,
		Host:          pool.SocketCloneDir(name),
		Port:          port,
		DB:            p.dbCfg,
		Pool:          pool,
		ContainerConf: p.config.ContainerConfig,
		NetworkID:     p.networkID,
	}

	return appConfig
}

// LastSessionActivity returns the time of the last session activity.
func (p *Provisioner) LastSessionActivity(session *resources.Session, minimumTime time.Time) (*time.Time, error) {
	fsm, err := p.pm.GetFSManager(session.Pool)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find a filesystem manager")
	}

	ctx, cancel := context.WithCancel(p.ctx)
	defer cancel()

	fileSelector := pglog.NewSelector(fsm.Pool().ClonePath(session.Port))

	if err := fileSelector.DiscoverLogDir(); err != nil {
		return nil, errors.Wrap(err, "failed to init file selector")
	}

	fileSelector.SetMinimumTime(minimumTime)
	fileSelector.FilterOldFilesInList()

	for {
		filename, err := fileSelector.Next()
		if err != nil {
			if err == pglog.ErrLastFile {
				break
			}

			return nil, errors.Wrap(err, "failed get CSV log filenames")
		}

		activity, err := p.scanCSVLogFile(ctx, filename, minimumTime)
		if err == io.EOF {
			continue
		}

		return activity, err
	}

	return nil, pglog.ErrNotFound
}

const csvMessageLogFieldsLength = 14

func (p *Provisioner) scanCSVLogFile(ctx context.Context, filename string, availableTime time.Time) (*time.Time, error) {
	csvFile, err := os.Open(filename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open a CSV log file")
	}

	defer func() {
		if err := csvFile.Close(); err != nil {
			log.Errf("Failed to close a CSV log file: %s", err.Error())
		}
	}()

	csvReader := csv.NewReader(csvFile)

	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		entry, err := csvReader.Read()
		if err != nil {
			return nil, err
		}

		if len(entry) < csvMessageLogFieldsLength {
			return nil, errors.New("wrong CSV file content")
		}

		logTime := entry[0]
		logMessage := entry[13]

		lastActivity, err := pglog.ParsePostgresLastActivity(logTime, logMessage)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get the time of last activity")
		}

		// Filter invalid and non-recent activity.
		if lastActivity == nil || lastActivity.Before(availableTime) {
			continue
		}

		return lastActivity, nil
	}
}

func (p *Provisioner) prepareDB(pgConf *resources.AppConfig, user resources.EphemeralUser) error {
	if !p.config.KeepUserPasswords {
		whitelist := []string{p.dbCfg.Username}

		if err := postgres.ResetAllPasswords(pgConf, whitelist); err != nil {
			return errors.Wrap(err, "failed to reset all passwords")
		}
	}

	if err := postgres.CreateUser(pgConf, user); err != nil {
		return errors.Wrap(err, "failed to create user")
	}

	return nil
}

// IsCloneRunning checks if clone is running.
func (p *Provisioner) IsCloneRunning(ctx context.Context, cloneName string) bool {
	isRunning, err := docker.IsContainerRunning(ctx, p.dockerClient, cloneName)
	if err != nil {
		log.Err(err)
	}

	return isRunning
}

// ReconnectClone disconnects clone from the old instance network and connect to the actual one.
func (p *Provisioner) ReconnectClone(ctx context.Context, cloneName string) error {
	return networks.Reconnect(ctx, p.dockerClient, p.instanceID, cloneName)
}

// StartCloneContainer starts clone container.
func (p *Provisioner) StartCloneContainer(ctx context.Context, containerName string) error {
	return p.dockerClient.ContainerStart(ctx, containerName, types.ContainerStartOptions{})
}

// DetectDBVersion detects version of the database.
func (p *Provisioner) DetectDBVersion() string {
	fsManager := p.pm.First()
	if fsManager == nil {
		return unknownVersion
	}

	pgVersion, err := tools.DetectPGVersion(fsManager.Pool().DataDir())
	if err != nil {
		return parseImageVersion(p.config.DockerImage)
	}

	return strconv.FormatFloat(pgVersion, 'g', -1, 64)
}

var regDockerImage = regexp.MustCompile(":([.0-9]+)")

func parseImageVersion(image string) string {
	allStringSubmatch := regDockerImage.FindAllStringSubmatch(image, -1)
	if len(allStringSubmatch) == 0 {
		return ""
	}

	if lastMatch := allStringSubmatch[len(allStringSubmatch)-1]; len(lastMatch) > 1 {
		return lastMatch[1]
	}

	return ""
}
