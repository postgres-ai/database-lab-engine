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
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
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

	defaultSessionCloneSize = 10

	dockerLogHeaderLength = 8
)

// ModeZfsPortPool describes an available port range of ZFS pool.
type ModeZfsPortPool struct {
	From uint `yaml:"from"`
	To   uint `yaml:"to"`
}

// ModeZfsConfig describes provisioning configs for ZFS mode.
type ModeZfsConfig struct {
	PortPool             ModeZfsPortPool `yaml:"portPool"`
	ZfsPool              string          `yaml:"pool"`
	MountDir             string          `yaml:"mountDir"`
	UnixSocketDir        string          `yaml:"unixSocketDir"`
	SnapshotFilterSuffix string          `yaml:"snapshotFilterSuffix"`
	DockerImage          string          `yaml:"dockerImage"`
	UseSudo              bool            `yaml:"useSudo"`
}

type provisionModeZfs struct {
	provision
	dockerClient   *client.Client
	runner         Runner
	ports          []bool
	sessionCounter uint
}

// NewProvisionModeZfs creates a new Provision instance of ModeZfs.
func NewProvisionModeZfs(ctx context.Context, config Config, dockerClient *client.Client) (Provision, error) {
	p := &provisionModeZfs{
		runner:         NewLocalRunner(config.ModeZfs.UseSudo),
		sessionCounter: 0,
		dockerClient:   dockerClient,
		provision: provision{
			config: config,
			ctx:    ctx,
		},
	}

	if len(p.config.ModeZfs.MountDir) == 0 {
		p.config.ModeZfs.MountDir = "/var/lib/dblab/clones/"
	}

	if len(p.config.ModeZfs.UnixSocketDir) == 0 {
		p.config.ModeZfs.UnixSocketDir = "/var/lib/dblab/sockets/"
	}

	if !strings.HasSuffix(p.config.ModeZfs.MountDir, Slash) {
		p.config.ModeZfs.MountDir += Slash
	}

	if !strings.HasSuffix(p.config.ModeZfs.UnixSocketDir, Slash) {
		p.config.ModeZfs.UnixSocketDir += Slash
	}

	if len(p.config.PgMgmtUsername) == 0 {
		p.config.PgMgmtUsername = DefaultUsername
	}

	if len(p.config.PgMgmtPassword) == 0 {
		p.config.PgMgmtPassword = DefaultPassword
	}

	return p, nil
}

func isValidConfigModeZfs(config Config) bool {
	result := true

	portPool := config.ModeZfs.PortPool

	if portPool.From == 0 {
		log.Err(`Wrong configuration: "portPool.from" must be defined and be greather than 0.`)

		result = false
	}

	if portPool.To == 0 {
		log.Err(`Wrong configuration: "portPool.to" must be defined and be greather than 0.`)

		result = false
	}

	if portPool.To <= portPool.From {
		log.Err(`Wrong configuration: port pool must consist of at least one port.`)

		result = false
	}

	return result
}

// Provision interface implementation.
func (j *provisionModeZfs) Init() error {
	err := j.stopAllSessions()
	if err != nil {
		return errors.Wrap(err, "failed to stop all session")
	}

	err = j.initPortPool()
	if err != nil {
		return errors.Wrap(err, "failed to init port pool")
	}

	imageExists, err := DockerImageExists(j.runner, j.config.ModeZfs.DockerImage)
	if err != nil {
		return errors.Wrap(err, "cannot check docker image existence")
	}

	if imageExists {
		return nil
	}

	err = DockerPullImage(j.runner, j.config.ModeZfs.DockerImage)
	if err != nil {
		return errors.Wrap(err, "cannot pull docker image")
	}

	return nil
}

func (j *provisionModeZfs) Reinit() error {
	return fmt.Errorf(`"Reinit" method is unsupported in "ZFS" mode`)
}

func (j *provisionModeZfs) StartSession(username, password, snapshotID string) (*Session, error) {
	snapshotID, err := j.getSnapshotID(snapshotID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get snapshots")
	}

	// TODO(anatoly): Synchronization or port allocation statuses.
	port, err := j.getFreePort()
	if err != nil {
		return nil, errors.New("failed to get a free port")
	}

	name := j.getName(port)

	log.Dbg(fmt.Sprintf(`Starting session for port: %d.`, port))

	err = ZfsCreateClone(j.runner, j.config.ModeZfs.ZfsPool, name, snapshotID,
		j.config.ModeZfs.MountDir, j.config.OSUsername)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a clone")
	}

	err = PostgresStart(j.runner, j.getPgConfig(name, port))
	if err != nil {
		log.Dbg(`Reverting "StartSession"...`)

		if runnerErr := ZfsDestroyClone(j.runner, j.config.ModeZfs.ZfsPool, name); runnerErr != nil {
			log.Err(`Revert:`, runnerErr)
		}

		return nil, errors.Wrap(err, "failed to start Postgres")
	}

	err = j.prepareDb(username, password, j.getPgConfig(name, port))
	if err != nil {
		log.Dbg(`Reverting "StartSession"...`)

		if runnerErr := PostgresStop(j.runner, j.getPgConfig(name, 0)); runnerErr != nil {
			log.Err("Revert:", runnerErr)
		}

		if runnerErr := ZfsDestroyClone(j.runner, j.config.ModeZfs.ZfsPool, name); runnerErr != nil {
			log.Err(`Revert:`, runnerErr)
		}

		return nil, errors.Wrap(err, "failed to prepare a database")
	}

	err = j.setPort(port, true)
	if err != nil {
		log.Dbg(`Reverting "StartSession"...`)

		if runnerErr := PostgresStop(j.runner, j.getPgConfig(name, 0)); runnerErr != nil {
			log.Err(`Revert:`, runnerErr)
		}

		if runnerErr := ZfsDestroyClone(j.runner, j.config.ModeZfs.ZfsPool, name); runnerErr != nil {
			log.Err(`Revert:`, runnerErr)
		}

		return nil, errors.Wrap(err, "failed to set a port")
	}

	j.sessionCounter++

	session := &Session{
		ID: strconv.FormatUint(uint64(j.sessionCounter), 10),

		Host:              DefaultHost,
		Port:              port,
		User:              j.config.PgMgmtUsername,
		Password:          j.config.PgMgmtPassword,
		ephemeralUser:     username,
		ephemeralPassword: password,
	}

	return session, nil
}

func (j *provisionModeZfs) StopSession(session *Session) error {
	name := j.getName(session.Port)

	err := PostgresStop(j.runner, j.getPgConfig(name, 0))
	if err != nil {
		return errors.Wrap(err, "failed to stop Postgres")
	}

	err = ZfsDestroyClone(j.runner, j.config.ModeZfs.ZfsPool, name)
	if err != nil {
		return errors.Wrap(err, "failed to destroy a clone")
	}

	err = j.setPort(session.Port, false)
	if err != nil {
		return errors.Wrap(err, "failed to unbind a port")
	}

	return nil
}

// TODO(akartasov): Refactor revert actions.
func (j *provisionModeZfs) ResetSession(session *Session, snapshotID string) error {
	name := j.getName(session.Port)

	snapshotID, err := j.getSnapshotID(snapshotID)
	if err != nil {
		return errors.Wrap(err, "failed to get snapshots")
	}

	err = PostgresStop(j.runner, j.getPgConfig(name, 0))
	if err != nil {
		log.Dbg(`Reverting "ResetSession"...`)

		if runnerErr := ZfsDestroyClone(j.runner, j.config.ModeZfs.ZfsPool, name); runnerErr != nil {
			log.Err(`Revert:`, runnerErr)
		}

		return errors.Wrap(err, "failed to stop Postgres")
	}

	err = ZfsDestroyClone(j.runner, j.config.ModeZfs.ZfsPool, name)
	if err != nil {
		return errors.Wrap(err, "failed to destroy clone")
	}

	err = ZfsCreateClone(j.runner, j.config.ModeZfs.ZfsPool, name, snapshotID,
		j.config.ModeZfs.MountDir, j.config.OSUsername)
	if err != nil {
		return errors.Wrap(err, "failed to create a clone")
	}

	err = PostgresStart(j.runner, j.getPgConfig(name, session.Port))
	if err != nil {
		log.Dbg(`Reverting "ResetSession"...`)

		if runnerErr := PostgresStop(j.runner, j.getPgConfig(name, 0)); runnerErr != nil {
			log.Err(`Revert:`, runnerErr)
		}

		if runnerErr := ZfsDestroyClone(j.runner, j.config.ModeZfs.ZfsPool, name); runnerErr != nil {
			log.Err(`Revert:`, runnerErr)
		}

		return errors.Wrap(err, "failed to start Postgres")
	}

	err = j.prepareDb(session.ephemeralUser, session.ephemeralPassword, j.getPgConfig(name, session.Port))
	if err != nil {
		log.Dbg(`Reverting "ResetSession"...`)

		if runnerErr := PostgresStop(j.runner, j.getPgConfig(name, 0)); runnerErr != nil {
			log.Err(`Revert:`, runnerErr)
		}

		if runnerErr := ZfsDestroyClone(j.runner, j.config.ModeZfs.ZfsPool, name); runnerErr != nil {
			log.Err(`Revert:`, runnerErr)
		}

		return errors.Wrap(err, "failed to prepare a database")
	}

	return nil
}

// Make a new snapshot.
func (j *provisionModeZfs) CreateSnapshot(name string) error {
	// TODO(anatoly): Implement.
	return errors.New(`"CreateSnapshot" method is unsupported in "ZFS" mode`)
}

func (j *provisionModeZfs) GetSnapshots() ([]Snapshot, error) {
	entries, err := ZfsListSnapshots(j.runner, j.config.ModeZfs.ZfsPool)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list snapshots")
	}

	snapshots := make([]Snapshot, 0, len(entries))

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name, j.config.ModeZfs.SnapshotFilterSuffix) {
			continue
		}

		snapshot := Snapshot{
			ID:          entry.Name,
			CreatedAt:   entry.Creation,
			DataStateAt: entry.DataStateAt,
		}

		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

func (j *provisionModeZfs) GetDiskState() (*Disk, error) {
	parts := strings.SplitN(j.config.ModeZfs.ZfsPool, "/", 2)
	parentPool := parts[0]

	entries, err := ZfsListFilesystems(j.runner, parentPool)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list filesystems")
	}

	var parentPoolEntry, poolEntry *ZfsListEntry

	for _, entry := range entries {
		if entry.Name == parentPool {
			parentPoolEntry = entry
		}

		if entry.Name == j.config.ModeZfs.ZfsPool {
			poolEntry = entry
		}

		if parentPoolEntry != nil && poolEntry != nil {
			break
		}
	}

	if parentPoolEntry == nil || poolEntry == nil {
		return nil, errors.New("cannot get disk state: pool entries not found")
	}

	disk := &Disk{
		Size:     parentPoolEntry.Available + parentPoolEntry.Used,
		Free:     parentPoolEntry.Available,
		DataSize: poolEntry.LogicalReferenced,
	}

	return disk, nil
}

func (j *provisionModeZfs) GetSessionState(s *Session) (*SessionState, error) {
	state := &SessionState{
		CloneSize: defaultSessionCloneSize,
	}

	entries, err := ZfsListFilesystems(j.runner, j.config.ModeZfs.ZfsPool)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list filesystems")
	}

	var sEntry *ZfsListEntry

	entryName := j.config.ModeZfs.ZfsPool + "/" + j.getName(s.Port)

	for _, entry := range entries {
		if entry.Name == entryName {
			sEntry = entry
			break
		}
	}

	if sEntry == nil {
		return nil, errors.New("cannot get session state: specified ZFS pool does not exist")
	}

	state.CloneSize = sEntry.Used

	return state, nil
}

// Other methods.
func (j *provisionModeZfs) getSnapshotID(snapshotID string) (string, error) {
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
func (j *provisionModeZfs) initPortPool() error {
	// Init session pool.
	portOpts := j.config.ModeZfs.PortPool
	size := portOpts.To - portOpts.From
	j.ports = make([]bool, size)

	//TODO(anatoly): Check ports.
	return nil
}

func (j *provisionModeZfs) getFreePort() (uint, error) {
	portOpts := j.config.ModeZfs.PortPool

	for index, binded := range j.ports {
		if !binded {
			port := portOpts.From + uint(index)
			return port, nil
		}
	}

	return 0, errors.WithStack(NewNoRoomError("no available ports"))
}

func (j *provisionModeZfs) setPort(port uint, bind bool) error {
	portOpts := j.config.ModeZfs.PortPool

	if port < portOpts.From || port >= portOpts.To {
		return errors.Errorf("port %d is out of bounds of the port pool", port)
	}

	index := port - portOpts.From
	j.ports[index] = bind

	return nil
}

func (j *provisionModeZfs) stopAllSessions() error {
	insts, err := PostgresList(j.runner, ClonePrefix)
	if err != nil {
		return errors.Wrap(err, "failed to list Postgres")
	}

	log.Dbg("Postgres instances running:", insts)

	for _, inst := range insts {
		log.Dbg("Stopping Postgress instance:", inst)

		if err = PostgresStop(j.runner, j.getPgConfig(inst, 0)); err != nil {
			return errors.Wrap(err, "failed to stop Postgres")
		}
	}

	clones, err := ZfsListClones(j.runner, ClonePrefix)
	if err != nil {
		return err
	}

	log.Dbg("ZFS clones:", clones)

	for _, clone := range clones {
		err = ZfsDestroyClone(j.runner, j.config.ModeZfs.ZfsPool, clone)
		if err != nil {
			return err
		}
	}

	return nil
}

func (j *provisionModeZfs) getName(port uint) string {
	return ClonePrefix + strconv.FormatUint(uint64(port), 10)
}

func (j *provisionModeZfs) getPgConfig(name string, port uint) *PgConfig {
	host := DefaultHost
	unixSocketCloneDir := j.config.ModeZfs.UnixSocketDir + name

	if UseUnixSocket {
		host = unixSocketCloneDir
	}

	return &PgConfig{
		CloneName:          name,
		Version:            j.config.PgVersion,
		DockerImage:        j.config.ModeZfs.DockerImage,
		Datadir:            path.Clean(j.config.ModeZfs.MountDir + name + j.config.PgDataSubdir),
		Host:               host,
		Port:               port,
		UnixSocketCloneDir: unixSocketCloneDir,
		Name:               "postgres",
		Username:           j.config.PgMgmtUsername,
		Password:           j.config.PgMgmtPassword,
		OSUsername:         j.config.OSUsername,
	}
}

func (j *provisionModeZfs) LastSessionActivity(session *Session, since time.Duration) (*time.Time, error) {
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

func (j *provisionModeZfs) prepareDb(username string, password string, pgConf *PgConfig) error {
	whitelist := []string{j.config.PgMgmtUsername}

	if err := PostgresResetAllPasswords(j.runner, pgConf, whitelist); err != nil {
		return errors.Wrap(err, "failed to reset all passwords")
	}

	if err := PostgresCreateUser(j.runner, pgConf, username, password); err != nil {
		return errors.Wrap(err, "failed to create user")
	}

	return nil
}
