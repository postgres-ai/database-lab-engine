/*
2019 © Postgres.ai
*/

package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetVersion(t *testing.T) {
	version = "0.0.1"
	buildTime = "20200427-0551"

	assert.Equal(t, "0.0.1-20200427-0551", GetVersion())
}
