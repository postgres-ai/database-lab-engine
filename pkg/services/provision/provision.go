/*
Provision wrapper

2019-2020 © Postgres.ai
*/

// Package provision provides an interface to provision Database Lab clones.
package provision

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/resources"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/thinclones"
)

const (
	// ModeLocal defines provisioning for local mode.
	ModeLocal = "local"
)

// NoRoomError defines a specific error type.
type NoRoomError struct {
	msg string
}

// Config defines configuration for provisioning.
type Config struct {
	// Provision mode.
	Mode string `yaml:"mode"`

	ModeLocal ModeLocalConfig `yaml:"local"`

	// Postgres options.
	PgVersion string `yaml:"pgVersion"`

	// Database user will be created with the specified credentials.
	PgMgmtUsername string `yaml:"pgMgmtUsername"`
	PgMgmtPassword string

	OSUsername string
}

// Provision defines provision interface.
type Provision interface {
	Init() error
	Reinit() error
	// TODO (akartasov): Create provision builder to build provision service and clone manager.
	//  Inject clone manager to provision service directly.
	ThinCloneManager() thinclones.Manager

	StartSession(username string, password string, snapshotID string) (*resources.Session, error)
	StopSession(*resources.Session) error
	ResetSession(session *resources.Session, snapshotID string) error

	GetSnapshots() ([]resources.Snapshot, error)

	GetDiskState() (*resources.Disk, error)
	GetSessionState(*resources.Session) (*resources.SessionState, error)
	LastSessionActivity(*resources.Session, time.Duration) (*time.Time, error)
}

type provision struct {
	config Config
	ctx    context.Context
}

// New creates a new Provision instance.
func New(ctx context.Context, config Config) (Provision, error) {
	// nolint
	switch config.Mode {
	case ModeLocal:
		log.Dbg(`Using "local" mode.`)

		// TODO(akartasov): Make it configurable.
		dockerClient, err := client.NewEnvClient()
		if err != nil {
			log.Fatalf(errors.WithMessage(err, `Failed to create Docker client.`))
		}

		return NewProvisionModeLocal(ctx, config, dockerClient)
	}

	return nil, errors.New("unsupported mode specified")
}

// IsValidConfig defines a method for validation of a configuration.
func IsValidConfig(c Config) bool {
	result := true

	if len(c.PgVersion) == 0 {
		log.Err("pgVersion must be set.")

		result = false
	}

	switch c.Mode {
	case ModeLocal:
		result = result && isValidConfigModeLocal(c)
	default:
		log.Err(fmt.Sprintf(`Unsupported mode specified: "%s".`, c.Mode))

		result = false
	}

	return result
}

// NewNoRoomError instances a new NoRoomError.
func NewNoRoomError(errorMessage string) error {
	return &NoRoomError{msg: errorMessage}
}

func (e *NoRoomError) Error() string {
	// TODO(anatoly): Change message.
	return "session cannot be started because there is no room: " + e.msg
}
