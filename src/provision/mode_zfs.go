/*
2019 © Postgres.ai
*/

package provision

import (
	"fmt"
	"strconv"
	"strings"

	"../log"
)

const (
	CLONE_PREFIX = "dblab_clone_"
	SLASH        = "/"
	DEFAULT_HOST = "localhost"
)

type ModeZfsPortPool struct {
	From uint `yaml:"from"`
	To   uint `yaml:"to"`
}

type ModeZfsConfig struct {
	PortPool        ModeZfsPortPool `yaml:"portPool"`
	ZfsPool         string          `yaml:"pool"`
	InitialSnapshot string          `yaml:"initialSnapshot"`
	LogsDir         string          `yaml:"logsDir"`
	MountDir        string          `yaml:"mountDir"`
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

	if len(p.config.ModeZfs.LogsDir) == 0 {
		p.config.ModeZfs.LogsDir = "/var/lib/postgresql/dblab/logs/"
	}

	if len(p.config.ModeZfs.MountDir) == 0 {
		p.config.ModeZfs.MountDir = "/var/lib/postgresql/dblab/clones/"
	}

	if !strings.HasSuffix(p.config.ModeZfs.LogsDir, SLASH) {
		p.config.ModeZfs.LogsDir += SLASH
	}

	if !strings.HasSuffix(p.config.ModeZfs.MountDir, SLASH) {
		p.config.ModeZfs.MountDir += SLASH
	}

	if len(p.config.DbUsername) == 0 {
		p.config.DbUsername = "postgres"
	}
	if len(p.config.DbPassword) == 0 {
		p.config.DbPassword = "postgres"
	}

	return p, nil
}

func isValidConfigModeZfs(config Config) bool {
	result := true

	portPool := config.ModeZfs.PortPool

	if portPool.From <= 0 {
		log.Err(`Wrong configuration: "portPool.from" must be defined and be greather than 0.`)
		result = false
	}

	if portPool.To <= 0 {
		log.Err(`Wrong configuration: "portPool.to" must be defined and be greather than 0.`)
		result = false
	}

	if portPool.To-portPool.From <= 0 {
		log.Err(`Wrong configuration: port pool must consist of at least one port.`)
		result = false
	}

	return result
}

// Provision interface implementation.
func (j *provisionModeZfs) Init() error {
	err := j.stopAllSessions()
	if err != nil {
		return err
	}

	err = j.initPortPool()
	if err != nil {
		return err
	}

	return nil
}

func (j *provisionModeZfs) Reinit() error {
	return fmt.Errorf(`"Reinit" method is unsupported in "ZFS" mode.`)
}

func (j *provisionModeZfs) StartSession(username string, password string,
	options ...string) (*Session, error) {
	snapshot := j.config.ModeZfs.InitialSnapshot
	if len(options) > 0 {
		snapshot = options[0]
	}

	// TODO(anatoly): Synchronization or port allocation statuses.
	port, err := j.getFreePort()
	if err != nil {
		return nil, err
	}

	name := j.getName(port)

	log.Dbg(fmt.Sprintf(`Starting session for port: %d.`, port))

	err = ZfsCreateClone(j.runner, j.config.ModeZfs.ZfsPool, name, snapshot,
		j.config.ModeZfs.MountDir)
	if err != nil {
		return nil, err
	}

	err = PostgresStart(j.runner, j.getPgConfig(name, port))
	if err != nil {
		log.Err(`StartSession:`, err)
		log.Dbg(`Reverting "StartSession"...`)

		rerr := ZfsDestroyClone(j.runner, j.config.ModeZfs.ZfsPool, name)
		if rerr != nil {
			log.Err(`Revert:`, rerr)
		}

		return nil, err
	}

	err = j.prepareDb(username, password, j.getPgConfig(name, port))
	if err != nil {
		log.Err(`StartSession:`, err)
		log.Dbg(`Reverting "StartSession"...`)

		rerr := PostgresStop(j.runner, j.getPgConfig(name, 0))
		if rerr != nil {
			log.Err("Revert:", rerr)
		}

		rerr = ZfsDestroyClone(j.runner, j.config.ModeZfs.ZfsPool, name)
		if rerr != nil {
			log.Err(`Revert:`, rerr)
		}

		return nil, err
	}

	err = j.setPort(port, true)
	if err != nil {
		log.Err(`StartSession:`, err)
		log.Dbg(`Reverting "StartSession"...`)

		rerr := PostgresStop(j.runner, j.getPgConfig(name, 0))
		if rerr != nil {
			log.Err(`Revert:`, rerr)
		}

		rerr = ZfsDestroyClone(j.runner, j.config.ModeZfs.ZfsPool, name)
		if rerr != nil {
			log.Err(`Revert:`, rerr)
		}

		return nil, err
	}

	j.sessionCounter++

	session := &Session{
		Id: strconv.FormatUint(uint64(j.sessionCounter), 10),

		Host:              DEFAULT_HOST,
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
		return err
	}

	err = ZfsDestroyClone(j.runner, j.config.ModeZfs.ZfsPool, name)
	if err != nil {
		return err
	}

	err = j.setPort(session.Port, false)
	if err != nil {
		return err
	}

	return nil
}

func (j *provisionModeZfs) ResetSession(session *Session, options ...string) error {
	name := j.getName(session.Port)

	snapshot := j.config.ModeZfs.InitialSnapshot
	if len(options) > 0 {
		snapshot = options[0]
	}

	err := PostgresStop(j.runner, j.getPgConfig(name, 0))
	if err != nil {
		log.Err(`ResetSession:`, err)
		log.Dbg(`Reverting "ResetSession"...`)

		rerr := ZfsDestroyClone(j.runner, j.config.ModeZfs.ZfsPool, name)
		if rerr != nil {
			log.Err(`Revert:`, rerr)
		}

		return err
	}

	err = ZfsDestroyClone(j.runner, j.config.ModeZfs.ZfsPool, name)
	if err != nil {
		log.Err(`ResetSession:`, err)
		log.Dbg(`Reverting "ResetSession"...`)

		return err
	}

	err = ZfsCreateClone(j.runner, j.config.ModeZfs.ZfsPool, name, snapshot,
		j.config.ModeZfs.MountDir)
	if err != nil {
		log.Err(`ResetSession:`, err)
		return err
	}

	err = PostgresStart(j.runner, j.getPgConfig(name, session.Port))
	if err != nil {
		log.Err(`ResetSession:`, err)
		log.Dbg(`Reverting "ResetSession"...`)

		rerr := PostgresStop(j.runner, j.getPgConfig(name, 0))
		if rerr != nil {
			log.Err(`Revert:`, rerr)
		}

		rerr = ZfsDestroyClone(j.runner, j.config.ModeZfs.ZfsPool, name)
		if rerr != nil {
			log.Err(`Revert:`, rerr)
		}

		return err
	}

	err = j.prepareDb(session.ephemeralUser, session.ephemeralPassword,
		j.getPgConfig(name, session.Port))
	if err != nil {
		log.Err(`ResetSession:`, err)
		log.Dbg(`Reverting "ResetSession"...`)

		rerr := PostgresStop(j.runner, j.getPgConfig(name, 0))
		if rerr != nil {
			log.Err(`Revert:`, rerr)
		}

		rerr = ZfsDestroyClone(j.runner, j.config.ModeZfs.ZfsPool, name)
		if rerr != nil {
			log.Err(`Revert:`, rerr)
		}

		return err
	}

	return nil
}

// Make a new snapshot.
func (j *provisionModeZfs) CreateSnapshot(name string) error {
	// TODO(anatoly): Implement.
	return fmt.Errorf(`"CreateSnapshot" method is unsupported in "ZFS" mode.`)
}

func (j *provisionModeZfs) GetSnapshots() ([]*Snapshot, error) {
	entries, err := ZfsListSnapshots(j.runner, j.config.ModeZfs.ZfsPool)
	if err != nil {
		log.Err("GetSnapshots:", err)
		return []*Snapshot{}, err
	}

	// Currently DB Lab does not provide an option to choose snapshot other
	// the one specified in the configuration. So it does not make sense
	// to list other snapshots.
	// TODO(anatoly): List all snapshots when snapshot setting become available.

	snapshotName := j.config.ModeZfs.ZfsPool + "@" +
		j.config.ModeZfs.InitialSnapshot

	snapshot := &Snapshot{}
	for _, entry := range entries {
		if entry.Name == snapshotName {
			snapshot.Id = snapshotName
			snapshot.CreatedAt = entry.Creation
			snapshot.DataStateAt = entry.DataStateAt
			break
		}
	}

	return []*Snapshot{snapshot}, nil
}

func (j *provisionModeZfs) GetDiskState() (*Disk, error) {
	parts := strings.SplitN(j.config.ModeZfs.ZfsPool, "/", 2)
	parentPool := parts[0]

	entries, err := ZfsListFilesystems(j.runner, parentPool)
	if err != nil {
		log.Err("GetDiskState:", err)
		return &Disk{}, err
	}

	var parentPoolEntry *ZfsListEntry
	var poolEntry *ZfsListEntry
	for _, entry := range entries {
		if entry.Name == parentPool {
			parentPoolEntry = entry
		} else if entry.Name == j.config.ModeZfs.ZfsPool {
			poolEntry = entry
		}

		if parentPoolEntry != nil && poolEntry != nil {
			break
		}
	}

	if parentPoolEntry == nil || poolEntry == nil {
		err := fmt.Errorf("Cannot get disk state. Pool entries not found.")
		log.Err("GetDiskState:", err)
		return &Disk{}, err
	}

	dataSize, err := j.getDataSize(poolEntry.MountPoint)
	if err != nil {
		return &Disk{}, err
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
		log.Err("GetSessionState:", err)
		return &SessionState{}, err
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
		err := fmt.Errorf("Cannot get session state. " +
			"Specified ZFS pool does not exist.")
		log.Err("GetSessionState:", err)
		return &SessionState{}, err
	}

	state.CloneSize = sEntry.Used
	return state, nil
}

func (j *provisionModeZfs) getDataSize(mountDir string) (uint64, error) {
	log.Dbg("getDataSize: " + mountDir)
	out, err := j.runner.Run("sudo du -d0 -b " + mountDir)
	if err != nil {
		log.Err("GetDataSize:", err)
		return 0, err
	}

	split := strings.SplitN(out, "\t", 2)
	if len(split) != 2 {
		err := fmt.Errorf(`Wrong format for "du".`)
		log.Err(err)
		return 0, err
	}

	nbytes, err := strconv.ParseUint(split[0], 10, 64)
	if err != nil {
		log.Err("GetDataSize:", err)
		return 0, err
	}

	return nbytes, nil
}

func (j *provisionModeZfs) RunPsql(session *Session, command string) (string, error) {
	pgConf := j.getPgConfig(session.Name, session.Port)
	return runPsqlStrict(j.runner, command, pgConf)
}

// Other methods.
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

	return 0, NewNoRoomError("No available ports.")
}

func (j *provisionModeZfs) setPort(port uint, bind bool) error {
	portOpts := j.config.ModeZfs.PortPool

	if port < portOpts.From || port >= portOpts.To {
		return fmt.Errorf("Port %d is out of bounds of the port pool.", port)
	}

	index := port - portOpts.From
	j.ports[index] = bind

	return nil
}

func (j *provisionModeZfs) stopAllSessions() error {
	insts, err := PostgresList(j.runner, CLONE_PREFIX)
	if err != nil {
		return err
	}

	log.Dbg("Postgres instances running:", insts)

	for _, inst := range insts {
		err = PostgresStop(j.runner, j.getPgConfig(inst, 0))
		if err != nil {
			return err
		}
	}

	clones, err := ZfsListClones(j.runner, CLONE_PREFIX)
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
	return CLONE_PREFIX + strconv.FormatUint(uint64(port), 10)
}

func (j *provisionModeZfs) getPgConfig(name string, port uint) *PgConfig {
	return &PgConfig{
		Version:    j.config.PgVersion,
		Bindir:     j.config.PgBindir,
		Datadir:    j.config.ModeZfs.MountDir + name + j.config.PgDataSubdir,
		Host:       DEFAULT_HOST,
		Port:       port,
		Name:       "postgres",
		Username:   j.config.DbUsername,
		Password:   j.config.DbPassword,
		LogsPrefix: j.config.ModeZfs.LogsDir,
	}
}

func (j *provisionModeZfs) prepareDb(username string, password string,
	pgConf *PgConfig) error {
	whitelist := []string{j.config.DbUsername}
	err := PostgresResetAllPasswords(j.runner, pgConf, whitelist)
	if err != nil {
		return err
	}

	err = PostgresCreateUser(j.runner, pgConf, username, password)
	if err != nil {
		return err
	}

	return nil
}
