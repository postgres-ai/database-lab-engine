/*
2019 © Postgres.ai
*/

package provision

import (
	"fmt"
	"strconv"

	"../log"
)

const (
	CLONE_PREFIX = "joe_auto_clone_"
)

type MuLocalPortPool struct {
	From uint `yaml:"from"`
	To   uint `yaml:"to"`
}

type MuLocalConfig struct {
	PortPool MuLocalPortPool `yaml:"portPool"`
}

type provisionMuLocal struct {
	provision
	runner         Runner
	ports          []bool
	sessionCounter uint
}

func NewProvisionMuLocal(config Config) (Provision, error) {
	provisionMuLocal := &provisionMuLocal{
		runner:         NewLocalRunner(),
		sessionCounter: 0,
	}
	provisionMuLocal.config = config

	return provisionMuLocal, nil
}

func isValidConfigModeMuLocal(config Config) bool {
	result := true

	portPool := config.MuLocal.PortPool

	if portPool.From <= 0 {
		log.Err(`Wrong configuration: "portPool.from" should be defined and be greather than 0`)
		result = false
	}

	if portPool.To <= 0 {
		log.Err(`Wrong configuration: "portPool.to" should be defined and be greather than 0`)
		result = false
	}

	if portPool.To-portPool.From <= 0 {
		log.Err(`Wrong configuration: Port pool should consist of at least one port`)
		result = false
	}

	return result
}

// Provision interface implementation.
func (j *provisionMuLocal) Init() error {
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

func (j *provisionMuLocal) Reinit() error {
	return fmt.Errorf("Unsupported in `mulocal` mode.")
}

func (j *provisionMuLocal) StartSession(options ...string) (*Session, error) {
	snapshot := j.config.InitialSnapshot
	if len(options) > 0 {
		snapshot = options[0]
	}

	// TODO(anatoly): Synchronization or port allocation statuses.
	port, err := j.getFreePort()
	if err != nil {
		return nil, err
	}

	name := j.getName(port)

	log.Dbg(fmt.Sprintf("Starting session for port: %d", port))

	err = ZfsCreateClone(j.runner, j.config.ZfsPool, name, snapshot)
	if err != nil {
		return nil, err
	}

	err = PostgresStart(j.runner, j.getPgConfig(name, port))
	if err != nil {
		log.Dbg("Reverting session start...")

		rerr := ZfsDestroyClone(j.runner, j.config.ZfsPool, name)
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

		rerr = ZfsDestroyClone(j.runner, j.config.ZfsPool, name)
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

func (j *provisionMuLocal) StopSession(session *Session) error {
	name := j.getName(session.Port)

	err := PostgresStop(j.runner, j.getPgConfig(name, 0))
	if err != nil {
		return err
	}

	err = ZfsDestroyClone(j.runner, j.config.ZfsPool, name)
	if err != nil {
		return err
	}

	err = j.setPort(session.Port, false)
	if err != nil {
		return err
	}

	return nil
}

func (j *provisionMuLocal) ResetSession(session *Session, options ...string) error {
	name := j.getName(session.Port)

	snapshot := j.config.InitialSnapshot
	if len(options) > 0 {
		snapshot = options[0]
	}

	err := PostgresStop(j.runner, j.getPgConfig(name, 0))
	if err != nil {
		rerr := ZfsDestroyClone(j.runner, j.config.ZfsPool, name)
		if rerr != nil {
			log.Err("Revert session reset:", rerr)
		}

		return err
	}

	err = ZfsDestroyClone(j.runner, j.config.ZfsPool, name)
	if err != nil {
		log.Err("Session reset:", err)
		return err
	}

	err = ZfsCreateClone(j.runner, j.config.ZfsPool, name, snapshot)
	if err != nil {
		return err
	}

	err = PostgresStart(j.runner, j.getPgConfig(name, session.Port))
	if err != nil {
		rerr := PostgresStop(j.runner, j.getPgConfig(name, 0))
		if rerr != nil {
			log.Err("Revert session reset:", rerr)
		}

		rerr = ZfsDestroyClone(j.runner, j.config.ZfsPool, name)
		if rerr != nil {
			log.Err("Revert session reset:", rerr)
		}

		return err
	}

	return nil
}

// Make a new snapshot.
func (j *provisionMuLocal) CreateSnapshot(name string) error {
	// TODO(anatoly): Implement.
	return fmt.Errorf("Unsupported in `mulocal` mode.")
}

func (j *provisionMuLocal) RunPsql(session *Session, command string) (string, error) {
	pgConf := j.getPgConfig(session.Name, session.Port)
	return runPsqlStrict(j.runner, command, pgConf)
}

// Other methods.
func (j *provisionMuLocal) initPortPool() error {
	// Init session pool.
	portOpts := j.config.MuLocal.PortPool
	size := portOpts.To - portOpts.From
	j.ports = make([]bool, size)

	//TODO(anatoly): Check ports.
	return nil
}

func (j *provisionMuLocal) getFreePort() (uint, error) {
	portOpts := j.config.MuLocal.PortPool
	for index, binded := range j.ports {
		if !binded {
			port := portOpts.From + uint(index)
			return port, nil
		}
	}

	return 0, NewNoRoomError("No available ports")
}

func (j *provisionMuLocal) setPort(port uint, bind bool) error {
	portOpts := j.config.MuLocal.PortPool

	if port < portOpts.From || port >= portOpts.To {
		return fmt.Errorf("Port is out of bounds of the port pool.")
	}

	index := port - portOpts.From
	j.ports[index] = bind

	return nil
}

func (j *provisionMuLocal) stopAllSessions() error {
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
		err = ZfsDestroyClone(j.runner, j.config.ZfsPool, clone)
		if err != nil {
			return err
		}
	}

	return nil
}

func (j *provisionMuLocal) getName(port uint) string {
	return CLONE_PREFIX + strconv.FormatUint(uint64(port), 10)
}

func (j *provisionMuLocal) getPgConfig(name string, port uint) *PgConfig {
	return &PgConfig{
		Version:  j.config.PgVersion,
		Bindir:   j.config.PgBindir,
		Datadir:  name + j.config.PgDataSubdir,
		Host:     j.config.DbHost,
		Port:     port,
		Name:     j.config.DbName,
		Username: j.config.DbUsername,
		Password: j.config.DbPassword,
	}
}
