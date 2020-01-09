/*
2019 © Postgres.ai
*/

package models

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
