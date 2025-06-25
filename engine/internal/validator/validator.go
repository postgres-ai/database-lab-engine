/*
2019 © Postgres.ai
*/

// Package validator provides a validation service.
package validator

import (
	"errors"
	"fmt"
	"regexp"

	passwordvalidator "github.com/wagslane/go-password-validator"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi/types"
)

const minEntropyBits = 60

var cloneIDRegexp = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

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

	if cloneRequest.ID != "" && !cloneIDRegexp.MatchString(cloneRequest.ID) {
		return errors.New("clone ID must start with a letter or number and can only contain letters, numbers, underscores, periods, and hyphens")
	}

	if err := passwordvalidator.Validate(cloneRequest.DB.Password, minEntropyBits); err != nil {
		return fmt.Errorf("password validation: %w", err)
	}

	return nil
}
