/*
2019 © Postgres.ai
*/

// Package models provides Database Lab struct.
package models

import (
	"fmt"
)

// ErrorCode defines a response error type.
type ErrorCode string

// ErrCodeInternal defines a response error codes.
const (
	ErrCodeInternal     ErrorCode = "INTERNAL_ERROR"
	ErrCodeBadRequest   ErrorCode = "BAD_REQUEST"
	ErrCodeUnauthorized ErrorCode = "UNAUTHORIZED"
	ErrCodeNotFound     ErrorCode = "NOT_FOUND"
)

// Error struct represents a response error.
type Error struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Detail  string    `json:"detail"`
	Hint    string    `json:"hint"`
}

// Error prints an error message.
func (e Error) Error() string {
	return fmt.Sprintf("Code %q. Message: %s Detail: %s Hint: %s", e.Code, e.Message, e.Detail, e.Hint)
}
