/*
Provision wrapper

2019 © Postgres.ai
*/

package provision

import (
	"fmt"
	"strings"
	"time"

	"gitlab.com/postgres-ai/database-lab/src/log"
)

const (
	MODE_ZFS = "zfs"
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

	// Database.
	Host     string
	Port     uint
	User     string
	Password string

	// For user-defined username and password.
	ephemeralUser     string
	ephemeralPassword string
}

type Config struct {
	Mode string `yaml:"mode"`

	ModeZfs ModeZfsConfig `yaml:"zfs"`

	// Postgres options.
	PgVersion    string `yaml:"pgVersion"`
	PgBindir     string `yaml:"pgBindir"`
	PgDataSubdir string `yaml:"pgDataSubdir"`

	// Database user will be created with the specified credentials.
	DbUsername string
	DbPassword string
}

// TODO(anatoly): Merge with disk from models?
type Disk struct {
	Size     uint64
	Free     uint64
	DataSize uint64
}

type Snapshot struct {
	Id          string
	CreatedAt   time.Time
	DataStateAt time.Time
}

type SessionState struct {
	CloneSize uint64
}

type Provision interface {
	Init() error
	Reinit() error

	StartSession(string, string, ...string) (*Session, error)
	StopSession(*Session) error
	ResetSession(*Session, ...string) error

	CreateSnapshot(string) error
	GetSnapshots() ([]*Snapshot, error)

	RunPsql(*Session, string) (string, error)

	GetDiskState() (*Disk, error)
	GetSessionState(*Session) (*SessionState, error)
}

type provision struct {
	config Config
}

func NewProvision(config Config) (Provision, error) {
	switch config.Mode {
	case MODE_ZFS:
		log.Dbg("Using ZFS mode.")
		return NewProvisionModeZfs(config)
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
	case MODE_ZFS:
		result = result && isValidConfigModeZfs(c)
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
