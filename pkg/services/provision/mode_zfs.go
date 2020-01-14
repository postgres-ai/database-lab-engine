/*
2019 © Postgres.ai
*/

package provision

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
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

	// UnixSocketDir defines directory of Postgres Unix sockets.
	UnixSocketDir = "/var/run/postgresql/"
)

type ModeZfsPortPool struct {
	From uint `yaml:"from"`
	To   uint `yaml:"to"`
}

type ModeZfsConfig struct {
	PortPool             ModeZfsPortPool `yaml:"portPool"`
	ZfsPool              string          `yaml:"pool"`
	MountDir             string          `yaml:"mountDir"`
	SnapshotFilterSuffix string          `yaml:"snapshotFilterSuffix"`
}

type provisionModeZfs struct {
	provision
	runner         Runner
	ports          []bool
	sessionCounter uint
}

func NewProvisionModeZfs(config Config) (Provision, error) {
	p := &provisionModeZfs{
		runner:         NewLocalRunner(),
		sessionCounter: 0,
	}
	p.config = config

	if len(p.config.ModeZfs.MountDir) == 0 {
		p.config.ModeZfs.MountDir = "/var/lib/postgresql/dblab/clones/"
	}

	if !strings.HasSuffix(p.config.ModeZfs.MountDir, Slash) {
		p.config.ModeZfs.MountDir += Slash
	}

	if len(p.config.DbUsername) == 0 {
		p.config.DbUsername = DefaultUsername
	}
	if len(p.config.DbPassword) == 0 {
		p.config.DbPassword = DefaultPassword
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

	return nil
}

func (j *provisionModeZfs) Reinit() error {
	return fmt.Errorf(`"Reinit" method is unsupported in "ZFS" mode.`)
}

func (j *provisionModeZfs) StartSession(username string, password string, options ...string) (*Session, error) {
	snapshotID, err := j.getSnapshotID(options...)
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
		j.config.ModeZfs.MountDir)
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
		User:              j.config.DbUsername,
		Password:          j.config.DbPassword,
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

func (j *provisionModeZfs) ResetSession(session *Session, options ...string) error {
	name := j.getName(session.Port)

	snapshotID, err := j.getSnapshotID(options...)
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
		j.config.ModeZfs.MountDir)
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

func (j *provisionModeZfs) GetSnapshots() ([]*Snapshot, error) {
	entries, err := ZfsListSnapshots(j.runner, j.config.ModeZfs.ZfsPool)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list snapshots")
	}

	snapshots := make([]*Snapshot, 0, len(entries))
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name, j.config.ModeZfs.SnapshotFilterSuffix) {
			continue
		}

		snapshot := &Snapshot{
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

	var parentPoolEntry *ZfsListEntry
	var poolEntry *ZfsListEntry
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

	dataSize, err := j.getDataSize(poolEntry.MountPoint)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get data size")
	}

	disk := &Disk{
		Size:     parentPoolEntry.Available + parentPoolEntry.Used,
		Free:     parentPoolEntry.Available,
		DataSize: dataSize,
	}

	return disk, nil
}

func (j *provisionModeZfs) GetSessionState(s *Session) (*SessionState, error) {
	state := &SessionState{
		CloneSize: 10,
	}

	entries, err := ZfsListFilesystems(j.runner, j.config.ModeZfs.ZfsPool)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list filesystems")
	}

	entryName := j.config.ModeZfs.ZfsPool + "/" + j.getName(s.Port)
	var sEntry *ZfsListEntry
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

func (j *provisionModeZfs) RunPsql(session *Session, command string) (string, error) {
	pgConf := j.getPgConfig(session.Name, session.Port)
	return runPsqlStrict(j.runner, command, pgConf)
}

// Other methods.
func (j *provisionModeZfs) getDataSize(mountDir string) (uint64, error) {
	log.Dbg("getDataSize: " + mountDir)
	out, err := j.runner.Run("sudo du -d0 -b " + mountDir + j.config.PgDataSubdir)
	if err != nil {
		return 0, errors.Wrap(err, "failed to run command")
	}

	split := strings.SplitN(out, "\t", 2)
	if len(split) != 2 {
		return 0, errors.New(`wrong format for "du"`)
	}

	nbytes, err := strconv.ParseUint(split[0], 10, 64)
	if err != nil {
		return 0, errors.Wrap(err, "failed to parse data size")
	}

	return nbytes, nil
}

func (j *provisionModeZfs) getSnapshotID(options ...string) (string, error) {
	snapshotID := ""
	if len(options) > 0 && len(options[0]) > 0 {
		snapshotID = options[0]
	} else {
		snapshots, err := j.GetSnapshots()
		if err != nil {
			return "", errors.Wrap(err, "failed to get snapshots")
		}

		if len(snapshots) == 0 {
			return "", errors.New("no snapshots available")
		}

		snapshotID = snapshots[0].ID
	}

	return snapshotID, nil
}

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
	if UseUnixSocket {
		host = UnixSocketDir
	}

	return &PgConfig{
		Version:  j.config.PgVersion,
		Bindir:   j.config.PgBindir,
		Datadir:  j.config.ModeZfs.MountDir + name + j.config.PgDataSubdir,
		Host:     host,
		Port:     port,
		Name:     "postgres",
		Username: j.config.DbUsername,
		Password: j.config.DbPassword,
	}
}

func (j *provisionModeZfs) prepareDb(username string, password string, pgConf *PgConfig) error {
	whitelist := []string{j.config.DbUsername}

	if err := PostgresResetAllPasswords(j.runner, pgConf, whitelist); err != nil {
		return errors.Wrap(err, "failed to reset all passwords")
	}

	if err := PostgresCreateUser(j.runner, pgConf, username, password); err != nil {
		return errors.Wrap(err, "failed to create user")
	}

	return nil
}
