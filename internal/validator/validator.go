/*
2019 © Postgres.ai
*/

// Package validator provides a validation service.
package validator

import (
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi/types"
)

// Service provides a validation service.
type Service struct {
}

// ValidateCloneRequest validates a clone request.
func (v Service) ValidateCloneRequest(cloneRequest *types.CloneCreateRequest) error {
	if cloneRequest.DB == nil {
		return errors.New("missing both DB username and password")
	}

	if cloneRequest.DB.Username == "" {
		return errors.New("missing DB username")
	}

	if cloneRequest.DB.Password == "" {
		return errors.New("missing DB password")
	}

	return nil
}
