/*
Provision wrapper

2019-2020 © Postgres.ai
*/

// Package provision provides an interface to provision Database Lab clones.
package provision

import (
	"context"
	"time"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/resources"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/thinclones"
)

// NoRoomError defines a specific error type.
type NoRoomError struct {
	msg string
}

// Config defines configuration for provisioning.
type Config struct {
	Options LocalModeOptions `yaml:"options"`

	// Database user will be created with the specified credentials.
	PgMgmtUsername string `yaml:"pgMgmtUsername"`
	PgMgmtPassword string

	OSUsername string
	MountDir   string
	DataSubDir string

	KeepUserPasswords bool `yaml:"keepUserPasswords"`
}

// Provision defines provision interface.
type Provision interface {
	Init() error
	Reload(Config)
	// TODO (akartasov): Create provision builder to build provision service and clone manager.
	//  Inject clone manager to provision service directly.
	ThinCloneManager() thinclones.Manager

	StartSession(username, password, snapshotID string, extraConfig map[string]string) (*resources.Session, error)
	StopSession(*resources.Session) error
	ResetSession(session *resources.Session, snapshotID string) error

	GetSnapshots() ([]resources.Snapshot, error)

	GetDiskState() (*resources.Disk, error)
	GetSessionState(*resources.Session) (*resources.SessionState, error)
	LastSessionActivity(port uint, minimalTime time.Time) (*time.Time, error)
}

type provision struct {
	config *Config
	ctx    context.Context
}

// New creates a new Provision instance.
func New(ctx context.Context, cfg Config, dockerClient *client.Client) (Provision, error) {
	if err := IsValidConfig(cfg); err != nil {
		return nil, errors.Wrap(err, "configuration is not valid")
	}

	// TODO (akartasov): Support more modes of provisioning.
	return NewProvisionModeLocal(ctx, cfg, dockerClient)
}

// IsValidConfig defines a method for validation of a configuration.
func IsValidConfig(cfg Config) error {
	return isValidConfigModeLocal(cfg)
}

// NewNoRoomError instances a new NoRoomError.
func NewNoRoomError(errorMessage string) error {
	return &NoRoomError{msg: errorMessage}
}

func (e *NoRoomError) Error() string {
	// TODO(anatoly): Change message.
	return "session cannot be started because there is no room: " + e.msg
}
