/*
2019 © Postgres.ai
*/

// Package models provides Database Lab struct.
package models

// ErrorCode defines a response error type.
type ErrorCode string

// ErrCode constants define a response error codes.
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
}

var _ error = &Error{}

// New creates ClientError instance with given code and message.
func New(code ErrorCode, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// Error prints an error message.
func (e Error) Error() string {
	return e.Message
}
