/*
2020 © Postgres.ai
*/

package commands

import (
	"errors"
	"fmt"
)

// ActionError defines a custom type of CLI action error.
type ActionError struct {
	err error
}

// NewActionError constructs a new Action error.
func NewActionError(msg string) ActionError {
	return ActionError{err: errors.New(msg)}
}

// ToActionError wraps the error to a new action error.
func ToActionError(err error) error {
	if err == nil {
		return nil
	}

	return ActionError{err: err}
}

// ActionErrorf formats according to a format specifier.
func ActionErrorf(format string, args ...interface{}) error {
	return ActionError{
		err: fmt.Errorf(format, args...),
	}
}

// Error returns an output of the action error.
func (e ActionError) Error() string {
	return "[ERROR]: " + e.err.Error()
}
