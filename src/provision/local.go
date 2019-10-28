/*
2019 © Postgres.ai
*/

package provision

import (
	"fmt"

	"../log"
)

const (
	PORT = 5432
)

type LocalConfig struct {
}

type provisionLocal struct {
	provision
	runner Runner
}

func NewProvisionLocal(config Config) (Provision, error) {
	provisionLocal := &provisionLocal{
		runner: NewLocalRunner(),
	}
	provisionLocal.config = config

	return provisionLocal, nil
}

func isValidConfigModeLocal(config Config) bool {
	return true
}

func NewRollbackError(err error) error {
	return fmt.Errorf("Unable to rollback the disk with the database to the initial state. %v", err)
}

func (j *provisionLocal) Init() error {
	return nil
}

func (j *provisionLocal) Reinit() error {
	return fmt.Errorf("Unsupported in `local` mode.")
}

// Provision interface implementaion.
func (j *provisionLocal) StartSession(options ...string) (*Session, error) {
	snapshot := j.config.InitialSnapshot
	if len(options) > 0 {
		snapshot = options[0]
	}

	err := j.rollbackState(snapshot)
	if err != nil {
		return nil, err
	}

	session := &Session{
		Id: "",

		Host:     "localhost",
		Port:     PORT,
		User:     j.config.DbUsername,
		Password: j.config.DbPassword,
	}
	return session, nil
}

func (j *provisionLocal) StopSession(session *Session) error {
	return nil
}

func (j *provisionLocal) ResetSession(session *Session, options ...string) error {
	snapshot := j.config.InitialSnapshot
	if len(options) > 0 {
		snapshot = options[0]
	}

	err := j.rollbackState(snapshot)
	if err != nil {
		return err
	}

	return nil
}

// Make a new snapshot.
func (j *provisionLocal) CreateSnapshot(name string) error {
	// TODO(anatoly): Implement.
	return fmt.Errorf("Unsupported in `local` mode.")
}

func (j *provisionLocal) RunPsql(session *Session, command string) (string, error) {
	pgConf := j.getPgConfig(session.Name, session.Port)
	return runPsqlStrict(j.runner, command, pgConf)
}

// Private methods.
func (j *provisionLocal) rollbackState(snapshot string) error {
	log.Dbg("Rollback the disk with the database to the specified snapshot.")

	err := PostgresStop(j.runner, j.getPgConfig("", 0))
	if err != nil {
		return NewRollbackError(err)
	}

	err = ZfsRollbackSnapshot(j.runner, j.config.ZfsPool, snapshot)
	if err != nil {
		return NewRollbackError(err)
	}

	err = PostgresStart(j.runner, j.getPgConfig("", PORT))
	if err != nil {
		return NewRollbackError(err)
	}

	return nil
}

func (j *provisionLocal) getPgConfig(name string, port uint) *PgConfig {
	return &PgConfig{
		Version:  j.config.PgVersion,
		Bindir:   j.config.PgBindir,
		Datadir:  name + j.config.PgDataSubdir,
		Host:     j.config.DbHost,
		Port:     port,
		Username: j.config.DbUsername,
		Password: j.config.DbPassword,
	}
}
