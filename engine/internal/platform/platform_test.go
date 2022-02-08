/*
2019 © Postgres.ai
*/

package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIfPersonalTokenEnabled(t *testing.T) {
	s := Service{}
	assert.Equal(t, s.IsPersonalTokenEnabled(), false)

	s.cfg.EnablePersonalToken = true
	assert.Equal(t, s.IsPersonalTokenEnabled(), true)
}

func TestIfOrganizationIsAllowed(t *testing.T) {
	s := Service{}
	assert.Equal(t, s.isAllowedOrganization(0), false)

	s.organizationID = 1
	assert.Equal(t, s.isAllowedOrganization(0), false)
	assert.Equal(t, s.isAllowedOrganization(1), true)
}
