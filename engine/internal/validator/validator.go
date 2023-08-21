/*
2019 © Postgres.ai
*/

// Package validator provides a validation service.
package validator

import (
	"fmt"

	"github.com/pkg/errors"
	passwordvalidator "github.com/wagslane/go-password-validator"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi/types"
)

const minEntropyBits = 60

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

	if err := passwordvalidator.Validate(cloneRequest.DB.Password, minEntropyBits); err != nil {
		return fmt.Errorf("password validation: %w", err)
	}

	return nil
}
