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
)

type ModeZfsPortPool struct {
	From uint `yaml:"from"`
	To   uint `yaml:"to"`
}

type ModeZfsConfig struct {
	PortPool        ModeZfsPortPool `yaml:"portPool"`
	ZfsPool         string          `yaml:"pool"`
	InitialSnapshot string          `yaml:"initialSnapshot"`
}

type provisionModeZfs struct {
	provision
	runner         Runner
	ports          []bool
	sessionCounter uint
}

func NewProvisionModeZfs(config Config) (Provision, error) {
	provisionModeZfs := &provisionModeZfs{
		runner:         NewLocalRunner(),
		sessionCounter: 0,
	}
	provisionModeZfs.config = config

	// TODO(anatoly): Get from request params.
	provisionModeZfs.config.DbUsername = "postgres"
	provisionModeZfs.config.DbPassword = "postgres"

	return provisionModeZfs, nil
}

func isValidConfigModeZfs(config Config) bool {
	result := true

	portPool := config.ModeZfs.PortPool

	if portPool.From <= 0 {
		log.Err("Wrong configuration: \"portPool.from\" must be defined and be greather than 0.")
		result = false
	}

	if portPool.To <= 0 {
		log.Err("Wrong configuration: \"portPool.to\" must be defined and be greather than 0.")
		result = false
	}

	if portPool.To-portPool.From <= 0 {
		log.Err("Wrong configuration: port pool must consist of at least one port.")
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
	return fmt.Errorf("\"Reinit\" method is unsupported in \"ZFS\" mode.")
}

func (j *provisionModeZfs) StartSession(options ...string) (*Session, error) {
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

	log.Dbg(fmt.Sprintf("Starting session for port: %d.", port))

	err = ZfsCreateClone(j.runner, j.config.ModeZfs.ZfsPool, name, snapshot)
	if err != nil {
		return nil, err
	}

	err = PostgresStart(j.runner, j.getPgConfig(name, port))
	if err != nil {
		log.Dbg("Reverting session start...")

		rerr := ZfsDestroyClone(j.runner, j.config.ModeZfs.ZfsPool, name)
		if rerr != nil {
			log.Err("Revert error:", rerr)
		}

		return nil, err
	}

	err = j.setPort(port, true)
	if err != nil {
		log.Dbg("Reverting session start...")

		rerr := PostgresStop(j.runner, j.getPgConfig(name, 0))
		if rerr != nil {
			log.Err("Revert error:", rerr)
		}

		rerr = ZfsDestroyClone(j.runner, j.config.ModeZfs.ZfsPool, name)
		if rerr != nil {
			log.Err("Revert error:", rerr)
		}

		return nil, err
	}

	j.sessionCounter++

	session := &Session{
		Id: strconv.FormatUint(uint64(j.sessionCounter), 10),

		Host:     "localhost",
		Port:     port,
		User:     j.config.DbUsername,
		Password: j.config.DbPassword,
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
		rerr := ZfsDestroyClone(j.runner, j.config.ModeZfs.ZfsPool, name)
		if rerr != nil {
			log.Err("Revert session reset:", rerr)
		}

		return err
	}

	err = ZfsDestroyClone(j.runner, j.config.ModeZfs.ZfsPool, name)
	if err != nil {
		log.Err("Session reset:", err)
		return err
	}

	err = ZfsCreateClone(j.runner, j.config.ModeZfs.ZfsPool, name, snapshot)
	if err != nil {
		return err
	}

	err = PostgresStart(j.runner, j.getPgConfig(name, session.Port))
	if err != nil {
		rerr := PostgresStop(j.runner, j.getPgConfig(name, 0))
		if rerr != nil {
			log.Err("Revert session reset:", rerr)
		}

		rerr = ZfsDestroyClone(j.runner, j.config.ModeZfs.ZfsPool, name)
		if rerr != nil {
			log.Err("Revert session reset:", rerr)
		}

		return err
	}

	return nil
}

// Make a new snapshot.
func (j *provisionModeZfs) CreateSnapshot(name string) error {
	// TODO(anatoly): Implement.
	return fmt.Errorf("\"CreateSnapshot\" method is unsupported in \"ZFS\" mode.")
}

func (j *provisionModeZfs) GetDiskState() (*Disk, error) {
	entries, err := ZfsListDetails(j.runner, j.config.ModeZfs.ZfsPool)
	if err != nil {
		log.Err("GetDiskState:", err)
		return &Disk{}, err
	}

	var poolEntry *ZfsListEntry
	for _, entry := range entries {
		if entry.Name == j.config.ModeZfs.ZfsPool {
			poolEntry = entry
		}
	}

	if poolEntry == nil {
		err := fmt.Errorf("Cannot get disk state.")
		log.Err("GetDiskState:", err)
		return &Disk{}, err
	}

	// TODO(anatoly): "sudo du -d0 -b /zfspool" [sudo] password.
	/*dataSize, err := j.getDataSize()
	if err != nil {
		return &Disk{}, err
	}*/

	disk := &Disk{
		Size:     poolEntry.Available + poolEntry.Used,
		Free:     poolEntry.Available,
		DataSize: 0,
	}

	return disk, nil
}

func (j *provisionModeZfs) getDataSize() (uint64, error) {
	/*out, err := j.runner.Run("sudo du -d0 -b /zfspool")
	if err != nil {
		log.Err("GetDataSize:", err)
		return 0, err
	}

	split := strings.SplitN(out, "\t", 2)
	if len(split) != 2 {
		err := fmt.Errorf("Wrong format for \"du\".")
		log.Err(err)
		return 0, err
	}

	nbytes, err := strconv.ParseUint(split[0], 10, 64)
	if err != nil {
		log.Err("GetDataSize:", err)
		return 0, err
	}

	return nbytes, nil*/
	return 0, nil
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
		Version:  j.config.PgVersion,
		Bindir:   j.config.PgBindir,
		Datadir:  MOUNT_PREFIX + name + j.config.PgDataSubdir,
		Host:     "localhost",
		Port:     port,
		Name:     "postgres",
		Username: j.config.DbUsername,
		Password: j.config.DbPassword,
	}
}
