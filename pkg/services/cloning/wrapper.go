/*
2019 © Postgres.ai
*/

// Package cloning provides a cloning service.
package cloning

import (
	"time"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/resources"
)

// CloneWrapper represents a cloning service wrapper.
type CloneWrapper struct {
	clone   *models.Clone
	session *resources.Session

	timeCreatedAt time.Time
	timeStartedAt time.Time

	username string
	password string

	snapshot models.Snapshot
}

// NewCloneWrapper constructs a new CloneWrapper.
func NewCloneWrapper(clone *models.Clone) *CloneWrapper {
	w := &CloneWrapper{
		clone: clone,
	}

	return w
}

// IsProtected checks if clone is protected.
func (cw CloneWrapper) IsProtected() bool {
	return cw.clone != nil && cw.clone.Protected
}
