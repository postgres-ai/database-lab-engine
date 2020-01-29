/*
2019 © Postgres.ai
*/

// Package models provides Database Lab struct.
package models

import (
	"fmt"
)

const (
	// ErrCodeInternal defines an internal error code.
	ErrCodeInternal = "INTERNAL_ERROR"
)

// Error struct represents a response error.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail"`
	Hint    string `json:"hint"`
}

// Error prints an error message.
func (e Error) Error() string {
	return fmt.Sprintf("Code %q. Message: %s Detail: %s Hint: %s", e.Code, e.Message, e.Detail, e.Hint)
}
