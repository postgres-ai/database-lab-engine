/*
2019 © Postgres.ai
*/

package provision

import (
	"bufio"
	"context"
	"fmt"
	"path"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/databases/postgres"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/docker"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/resources"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/runners"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/thinclones"
	"gitlab.com/postgres-ai/database-lab/pkg/util/pglog"
)

const (
	// ClonePrefix defines a Database Lab clone prefix.
	ClonePrefix = "dblab_clone_"

	// Slash represents a slash symbol.
	Slash = "/"

	// DefaultHost defines a default host name.
	DefaultHost = "localhost"

	// DefaultUsername defines a default user name.
	DefaultUsername = "postgres"

	// DefaultPassword defines a default password.
	DefaultPassword = "postgres"

	// UseUnixSocket defines the need to connect to Postgres using Unix sockets.
	UseUnixSocket = true

	dockerLogHeaderLength = 8

	defaultClonesMountDir = "/var/lib/dblab/clones/"

	defaultUnixSocketDir = "/var/lib/dblab/sockets/"
)

// LocalModePortPool describes an available port range for clones.
type LocalModePortPool struct {
	From uint `yaml:"from"`
	To   uint `yaml:"to"`
}

// LocalModeOptions describes provisioning configs for local mode.
type LocalModeOptions struct {
	PortPool          LocalModePortPool `yaml:"portPool"`
	ClonePool         string            `yaml:"pool"`
	ClonesMountDir    string            `yaml:"clonesMountDir"`
	UnixSocketDir     string            `yaml:"unixSocketDir"`
	PreSnapshotSuffix string            `yaml:"preSnapshotSuffix"`
	DockerImage       string            `yaml:"dockerImage"`
	UseSudo           bool              `yaml:"useSudo"`

	// Thin-clone manager.
	ThinCloneManager string `yaml:"thinCloneManager"`
}

type provisionModeLocal struct {
	provision
	dockerClient     *client.Client
	runner           runners.Runner
	mu               *sync.Mutex
	ports            []bool
	sessionCounter   uint32
	thinCloneManager thinclones.Manager
}

// NewProvisionModeLocal creates a new Provision instance of ModeLocal.
func NewProvisionModeLocal(ctx context.Context, config Config, dockerClient *client.Client) (Provision, error) {
	p := &provisionModeLocal{
		runner:       runners.NewLocalRunner(config.Options.UseSudo),
		mu:           &sync.Mutex{},
		dockerClient: dockerClient,
		provision: provision{
			config: config,
			ctx:    ctx,
		},
	}

	setDefault(&p.config)

	thinCloneManager, err := thinclones.NewManager(p.config.Options.ThinCloneManager,
		p.runner, thinclones.ManagerConfig{
			Pool:              p.config.Options.ClonePool,
			PreSnapshotSuffix: p.config.Options.PreSnapshotSuffix,
			ClonesMountDir:    p.config.Options.ClonesMountDir,
			OSUsername:        p.config.OSUsername,
			ClonePrefix:       ClonePrefix,
		})

	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize thin-clone manager")
	}

	p.thinCloneManager = thinCloneManager

	return p, nil
}

func setDefault(cfg *Config) {
	if !strings.HasSuffix(cfg.Options.ClonesMountDir, Slash) {
		cfg.Options.ClonesMountDir += Slash
	}

	if !strings.HasSuffix(cfg.Options.UnixSocketDir, Slash) {
		cfg.Options.UnixSocketDir += Slash
	}

	if cfg.Options.ClonesMountDir == "" {
		cfg.Options.ClonesMountDir = defaultClonesMountDir
	}

	if cfg.Options.UnixSocketDir == "" {
		cfg.Options.UnixSocketDir = defaultUnixSocketDir
	}

	if cfg.PgMgmtUsername == "" {
		cfg.PgMgmtUsername = DefaultUsername
	}

	if cfg.PgMgmtPassword == "" {
		cfg.PgMgmtPassword = DefaultPassword
	}
}

func isValidConfigModeLocal(config Config) error {
	portPool := config.Options.PortPool

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

// Provision interface implementation.
func (j *provisionModeLocal) Init() error {
	err := j.stopAllSessions()
	if err != nil {
		return errors.Wrap(err, "failed to stop all session")
	}

	err = j.initPortPool()
	if err != nil {
		return errors.Wrap(err, "failed to init port pool")
	}

	imageExists, err := docker.ImageExists(j.runner, j.config.Options.DockerImage)
	if err != nil {
		return errors.Wrap(err, "cannot check docker image existence")
	}

	if imageExists {
		return nil
	}

	err = docker.PullImage(j.runner, j.config.Options.DockerImage)
	if err != nil {
		return errors.Wrap(err, "cannot pull docker image")
	}

	return nil
}

func (j *provisionModeLocal) Reinit() error {
	return fmt.Errorf(`"Reinit" method is unsupported in "local" mode`)
}

// ThinCloneManager provides a thin clone manager.
func (j *provisionModeLocal) ThinCloneManager() thinclones.Manager {
	return j.thinCloneManager
}

func (j *provisionModeLocal) StartSession(username, password, snapshotID string) (*resources.Session, error) {
	snapshotID, err := j.getSnapshotID(snapshotID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get snapshots")
	}

	port, err := j.allocatePort()
	if err != nil {
		return nil, errors.New("failed to get a free port")
	}

	name := j.getName(port)

	log.Dbg(fmt.Sprintf(`Starting session for port: %d.`, port))

	defer func() {
		if err != nil {
			j.revertSession(name)

			if portErr := j.freePort(port); portErr != nil {
				log.Err(portErr)
			}
		}
	}()

	err = j.thinCloneManager.CreateClone(name, snapshotID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a clone")
	}

	err = postgres.Start(j.runner, j.getAppConfig(name, port))
	if err != nil {
		return nil, errors.Wrap(err, "failed to start a container")
	}

	err = j.prepareDB(username, password, j.getAppConfig(name, port))
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare a database")
	}

	atomic.AddUint32(&j.sessionCounter, 1)

	appConfig := j.getAppConfig(name, port)

	session := &resources.Session{
		ID:                strconv.FormatUint(uint64(j.sessionCounter), 10),
		Host:              DefaultHost,
		Port:              port,
		User:              j.config.PgMgmtUsername,
		Password:          j.config.PgMgmtPassword,
		SocketHost:        appConfig.Host,
		EphemeralUser:     username,
		EphemeralPassword: password,
	}

	return session, nil
}

func (j *provisionModeLocal) StopSession(session *resources.Session) error {
	name := j.getName(session.Port)

	err := postgres.Stop(j.runner, j.getAppConfig(name, 0))
	if err != nil {
		return errors.Wrap(err, "failed to stop a container")
	}

	err = j.thinCloneManager.DestroyClone(name)
	if err != nil {
		return errors.Wrap(err, "failed to destroy a clone")
	}

	err = j.freePort(session.Port)
	if err != nil {
		return errors.Wrap(err, "failed to unbind a port")
	}

	return nil
}

func (j *provisionModeLocal) ResetSession(session *resources.Session, snapshotID string) error {
	name := j.getName(session.Port)

	snapshotID, err := j.getSnapshotID(snapshotID)
	if err != nil {
		return errors.Wrap(err, "failed to get snapshots")
	}

	defer func() {
		if err != nil {
			j.revertSession(name)
		}
	}()

	err = postgres.Stop(j.runner, j.getAppConfig(name, 0))
	if err != nil {
		return errors.Wrap(err, "failed to stop a container")
	}

	err = j.thinCloneManager.DestroyClone(name)
	if err != nil {
		return errors.Wrap(err, "failed to destroy clone")
	}

	err = j.thinCloneManager.CreateClone(name, snapshotID)
	if err != nil {
		return errors.Wrap(err, "failed to create a clone")
	}

	err = postgres.Start(j.runner, j.getAppConfig(name, session.Port))
	if err != nil {
		return errors.Wrap(err, "failed to start a container")
	}

	err = j.prepareDB(session.EphemeralUser, session.EphemeralPassword, j.getAppConfig(name, session.Port))
	if err != nil {
		return errors.Wrap(err, "failed to prepare a database")
	}

	return nil
}

func (j *provisionModeLocal) GetSnapshots() ([]resources.Snapshot, error) {
	return j.thinCloneManager.GetSnapshots()
}

func (j *provisionModeLocal) GetDiskState() (*resources.Disk, error) {
	return j.thinCloneManager.GetDiskState()
}

func (j *provisionModeLocal) GetSessionState(s *resources.Session) (*resources.SessionState, error) {
	return j.thinCloneManager.GetSessionState(j.getName(s.Port))
}

// Other methods.
func (j *provisionModeLocal) revertSession(name string) {
	log.Dbg(`Reverting start of a session...`)

	if runnerErr := postgres.Stop(j.runner, j.getAppConfig(name, 0)); runnerErr != nil {
		log.Err(`Revert:`, runnerErr)
	}

	if runnerErr := j.thinCloneManager.DestroyClone(name); runnerErr != nil {
		log.Err(`Revert:`, runnerErr)
	}
}

func (j *provisionModeLocal) getSnapshotID(snapshotID string) (string, error) {
	if snapshotID != "" {
		return snapshotID, nil
	}

	snapshots, err := j.GetSnapshots()
	if err != nil {
		return "", errors.Wrap(err, "failed to get snapshots")
	}

	if len(snapshots) == 0 {
		return "", errors.New("no snapshots available")
	}

	return snapshots[0].ID, nil
}

// nolint
func (j *provisionModeLocal) initPortPool() error {
	// Init session pool.
	portOpts := j.config.Options.PortPool
	size := portOpts.To - portOpts.From
	j.ports = make([]bool, size)

	//TODO(anatoly): Check ports.
	return nil
}

// allocatePort tries to find a free port and occupy it.
func (j *provisionModeLocal) allocatePort() (uint, error) {
	portOpts := j.config.Options.PortPool

	j.mu.Lock()
	defer j.mu.Unlock()

	for index, binded := range j.ports {
		if !binded {
			port := portOpts.From + uint(index)

			if err := j.setPortStatus(port, true); err != nil {
				return 0, errors.Wrapf(err, "failed to set status for port %v", port)
			}

			return port, nil
		}
	}

	return 0, errors.WithStack(NewNoRoomError("no available ports"))
}

// freePort marks the port as free.
func (j *provisionModeLocal) freePort(port uint) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	return j.setPortStatus(port, false)
}

// setPortStatus updates the port status.
// It's not safe to invoke without ports mutex locking. Use allocatePort and freePort methods.
func (j *provisionModeLocal) setPortStatus(port uint, bind bool) error {
	portOpts := j.config.Options.PortPool

	if port < portOpts.From || port >= portOpts.To {
		return errors.Errorf("port %d is out of bounds of the port pool", port)
	}

	index := port - portOpts.From
	j.ports[index] = bind

	return nil
}

func (j *provisionModeLocal) stopAllSessions() error {
	instances, err := postgres.List(j.runner, j.config.Options.ClonePool)
	if err != nil {
		return errors.Wrap(err, "failed to list containers")
	}

	log.Dbg("Containers running:", instances)

	for _, inst := range instances {
		log.Dbg("Stopping container:", inst)

		if err = postgres.Stop(j.runner, j.getAppConfig(inst, 0)); err != nil {
			return errors.Wrap(err, "failed to container")
		}
	}

	clones, err := j.thinCloneManager.ListClonesNames()
	if err != nil {
		return err
	}

	log.Dbg("VM clones:", clones)

	for _, clone := range clones {
		err = j.thinCloneManager.DestroyClone(clone)
		if err != nil {
			return err
		}
	}

	return nil
}

func (j *provisionModeLocal) getName(port uint) string {
	return ClonePrefix + strconv.FormatUint(uint64(port), 10)
}

func (j *provisionModeLocal) getAppConfig(name string, port uint) *resources.AppConfig {
	host := DefaultHost
	unixSocketCloneDir := j.config.Options.UnixSocketDir + name

	if UseUnixSocket {
		host = unixSocketCloneDir
	}

	appConfig := &resources.AppConfig{
		CloneName:          name,
		ClonePool:          j.config.Options.ClonePool,
		DockerImage:        j.config.Options.DockerImage,
		Datadir:            path.Clean(j.config.Options.ClonesMountDir + name),
		Host:               host,
		Port:               port,
		UnixSocketCloneDir: unixSocketCloneDir,
		OSUsername:         j.config.OSUsername,
	}

	appConfig.SetDBName("postgres")
	appConfig.SetUsername(j.config.PgMgmtUsername)
	appConfig.SetPassword(j.config.PgMgmtPassword)

	return appConfig
}

func (j *provisionModeLocal) LastSessionActivity(session *resources.Session, since time.Duration) (*time.Time, error) {
	cloneName := j.getName(session.Port)

	ctx, cancel := context.WithCancel(j.ctx)
	defer cancel()

	logStream, err := j.dockerClient.ContainerLogs(ctx, cloneName, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Since:      since.String(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed get Docker logs")
	}

	defer func() {
		if err := logStream.Close(); err != nil {
			log.Errf("Failed to close Docker log stream: %s", err.Error())
		}
	}()

	scanner := bufio.NewScanner(logStream)
	for scanner.Scan() {
		if len(scanner.Bytes()) < dockerLogHeaderLength {
			continue
		}

		// Skip stream headers.
		logLine := string(scanner.Bytes()[8:])

		lastActivity, err := pglog.GetPostgresLastActivity(logLine)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get the time of last activity of %q", cloneName)
		}

		if lastActivity == nil {
			continue
		}

		return lastActivity, nil
	}

	return nil, pglog.ErrNotFound
}

func (j *provisionModeLocal) prepareDB(username, password string, pgConf *resources.AppConfig) error {
	whitelist := []string{j.config.PgMgmtUsername}

	if err := postgres.ResetAllPasswords(j.runner, pgConf, whitelist); err != nil {
		return errors.Wrap(err, "failed to reset all passwords")
	}

	if err := postgres.CreateUser(j.runner, pgConf, username, password); err != nil {
		return errors.Wrap(err, "failed to create user")
	}

	return nil
}
