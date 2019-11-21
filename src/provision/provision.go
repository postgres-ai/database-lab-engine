/*
Provision wrapper

2019 © Postgres.ai
*/

package provision

import (
	"fmt"
	"strings"

	"../log"
)

const (
	MODE_AWS     = "aws"
	MODE_LOCAL   = "local"
	MODE_MULOCAL = "mulocal"
)

type NoRoomError string

type State struct {
	InstanceId        string
	InstanceIp        string
	DockerContainerId string
	SessionId         string
}

type Session struct {
	Id   string
	Name string

	// Database
	Host     string
	Port     uint
	User     string
	Password string
}

type Config struct {
	Aws     AwsConfig
	Local   LocalConfig
	MuLocal MuLocalConfig

	Mode  string
	Debug bool

	// ZFS options.
	ZfsPool         string
	InitialSnapshot string

	// Postgres options.
	PgVersion    string
	PgBindir     string
	PgDataSubdir string

	DbHost string
	DbName string

	// Database user will be created with the specified credentials.
	DbUsername string
	DbPassword string
}

type Provision interface {
	Init() error
	Reinit() error

	StartSession(...string) (*Session, error)
	StopSession(*Session) error
	ResetSession(*Session, ...string) error

	CreateSnapshot(string) error

	RunPsql(*Session, string) (string, error)
}

type provision struct {
	config Config
}

func NewProvision(config Config) (Provision, error) {
	switch config.Mode {
	case MODE_AWS:
		log.Dbg("Using AWS mode.")
		return NewProvisionAws(config)
	case MODE_LOCAL:
		log.Dbg("Using Local mode.")
		return NewProvisionLocal(config)
	case MODE_MULOCAL:
		log.Dbg("Using MuLocal mode.")
		return NewProvisionMuLocal(config)
	}

	return nil, fmt.Errorf("Unsupported mode specified.")
}

// Check validity of a configuration and show a message for each violation.
func IsValidConfig(c Config) bool {
	result := true

	if len(c.PgVersion) == 0 && len(c.PgBindir) == 0 {
		log.Err("Either pgVersion or pgBindir should be set.")
		result = false
	}

	if len(c.PgBindir) > 0 && strings.HasSuffix(c.PgBindir, "/") {
		log.Err("Remove tailing slash from pgBindir.")
	}

	switch c.Mode {
	case MODE_AWS:
		result = result && isValidConfigModeAws(c)
	case MODE_LOCAL:
		result = result && isValidConfigModeLocal(c)
	case MODE_MULOCAL:
		result = result && isValidConfigModeMuLocal(c)
	default:
		log.Err("Unsupported mode specified.")
		result = false
	}

	return result
}

func (s *Session) GetConnStr(dbname string) string {
	connStr := "sslmode=disable"

	if len(s.Host) > 0 {
		connStr += " host=" + s.Host
	}

	if s.Port > 0 {
		connStr += fmt.Sprintf(" port=%d", s.Port)
	}

	if len(s.User) > 0 {
		connStr += " user=" + s.User
	}

	if len(s.Password) > 0 {
		connStr += " password=" + s.Password
	}

	if len(dbname) > 0 {
		connStr += " dbname=" + dbname
	}

	return connStr
}

func NewNoRoomError(options ...string) error {
	// TODO(anatoly): Change message.
	msg := "Session cannot be started because there is no room"
	if len(options) > 0 {
		msg += ": " + options[0] + "."
	} else {
		msg += "."
	}

	return NoRoomError(msg)
}

func (f NoRoomError) Error() string {
	return string(f)
}
