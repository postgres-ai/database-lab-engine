/*
2019 © Postgres.ai
*/

package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi/types"
)

func TestValidationCloneRequest(t *testing.T) {
	validator := Service{}
	err := validator.ValidateCloneRequest(
		&types.CloneCreateRequest{
			DB: &types.DatabaseRequest{
				Username: "username",
				Password: "password",
			}})

	assert.Nil(t, err)
}

func TestValidationCloneRequestErrors(t *testing.T) {
	validator := Service{}

	testCases := []struct {
		createRequest types.CloneCreateRequest
		error         string
	}{
		{
			createRequest: types.CloneCreateRequest{},
			error:         "missing both DB username and password",
		},
		{
			createRequest: types.CloneCreateRequest{DB: &types.DatabaseRequest{Username: "user"}},
			error:         "missing DB password",
		},
		{
			createRequest: types.CloneCreateRequest{DB: &types.DatabaseRequest{Password: "password"}},
			error:         "missing DB username",
		},
	}

	for _, tc := range testCases {
		err := validator.ValidateCloneRequest(&tc.createRequest)

		assert.EqualError(t, err, tc.error)
	}
}
