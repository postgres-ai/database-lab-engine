/*
2021 © Postgres.ai
*/

package util

import (
	"path"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBinRootPath(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	expected := path.Dir(filename)

	binRootPath, err := GetBinRootPath()
	require.Nil(t, err)
	assert.Equal(t, expected, binRootPath)
}

func TestSwaggerUIPath(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	expected := path.Dir(filename) + "/web"

	swaggerUIPath, err := GetSwaggerUIPath()
	require.Nil(t, err)
	assert.Equal(t, expected, swaggerUIPath)
}

func TestAPIPath(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	expected := path.Dir(filename) + "/api"

	apiPath, err := GetAPIPath()
	require.Nil(t, err)
	assert.Equal(t, expected, apiPath)
}

func TestStandardDirPath(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	expected := path.Dir(filename) + "/standard/test.yml"

	standardConfigPath, err := GetStandardConfigPath("test.yml")
	require.Nil(t, err)
	assert.Equal(t, expected, standardConfigPath)

	expected = path.Dir(filename) + "/standard/postgres/control/pg_hba.conf"
	standardConfigPath, err = GetStandardConfigPath("postgres/control/pg_hba.conf")
	require.Nil(t, err)
	assert.Equal(t, expected, standardConfigPath)
}

func TestConfigPath(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	expected := path.Dir(filename) + "/configs/server.yml"

	configPath, err := GetConfigPath("server.yml")
	require.Nil(t, err)
	assert.Equal(t, expected, configPath)
}

func TestMetaPath(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	expected := path.Dir(filename) + "/meta/state.json"

	metaPath, err := GetMetaPath("state.json")
	require.Nil(t, err)
	assert.Equal(t, expected, metaPath)
}
