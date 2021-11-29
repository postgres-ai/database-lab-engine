/*
2019 © Postgres.ai
*/

package api

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

func TestErrorCodeStatuses(t *testing.T) {
	testCases := []struct {
		error models.ErrorCode
		code  int
	}{
		{
			error: "BAD_REQUEST",
			code:  400,
		},
		{
			error: "UNAUTHORIZED",
			code:  401,
		},
		{
			error: "NOT_FOUND",
			code:  404,
		},
		{
			error: "INTERNAL_ERROR",
			code:  500,
		},
		{
			error: "UNKNOWN_ERROR",
			code:  500,
		},
	}

	for _, tc := range testCases {
		errorCode := toStatusCode(models.Error{Code: tc.error})

		assert.Equal(t, tc.code, errorCode)
	}
}
