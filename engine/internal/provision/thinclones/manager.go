/*
2020 © Postgres.ai
*/

// Package thinclones provides an interface to work different thin-clone managers.
package thinclones

import (
	"fmt"
)

// ResetOptions defines reset options.
type ResetOptions struct {
	// -f
	// -r
}

// SnapshotExistsError defines an error when snapshot already exists.
type SnapshotExistsError struct {
	name string
}

// NewSnapshotExistsError creates a new SnapshotExistsError.
func NewSnapshotExistsError(name string) *SnapshotExistsError {
	return &SnapshotExistsError{name: name}
}

// Error prints a message describing SnapshotExistsError.
func (e *SnapshotExistsError) Error() string {
	return fmt.Sprintf(`snapshot %s already exists`, e.name)
}

// DestroyOptions provides options for destroy commands.
type DestroyOptions struct {
	Force bool
}

// SnapshotProperties describe custom properties of the dataset.
type SnapshotProperties struct {
	Name        string
	Parent      string
	Child       string
	Branch      string
	Root        string
	DataStateAt string
	Message     string
}
