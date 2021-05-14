/*
Provision wrapper

2019-2020 © Postgres.ai
*/

// Package provision provides an interface to provision Database Lab clones.
package provision

// NoRoomError defines a specific error type.
type NoRoomError struct {
	msg string
}

// NewNoRoomError instances a new NoRoomError.
func NewNoRoomError(errorMessage string) error {
	return &NoRoomError{msg: errorMessage}
}

func (e *NoRoomError) Error() string {
	// TODO(anatoly): Change message.
	return "session cannot be started because there is no room: " + e.msg
}
