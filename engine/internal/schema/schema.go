// Package schema provides tools to manage PostgreSQL schemas difference.
package schema

import (
	"github.com/docker/docker/client"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

// Diff defines a schema generator.
type Diff struct {
	d *client.Client
}

// NewDiff creates a new Diff service.
func NewDiff(d *client.Client) *Diff {
	return &Diff{d: d}
}

// GenerateDiff generate difference between database schemas.
func (d *Diff) GenerateDiff(actual, origin *models.Clone) (string, error) {
	return "", nil
}
