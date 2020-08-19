/*
2020 © Postgres.ai
*/

package tools

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIfDirectoryEmpty(t *testing.T) {
	dirName, err := ioutil.TempDir("", "test")
	defer os.RemoveAll(dirName)

	require.NoError(t, err)

	// Check if the directory is empty.
	isEmpty, err := IsEmptyDirectory(dirName)
	require.NoError(t, err)
	assert.True(t, isEmpty)

	// Create a new file.
	_, err = ioutil.TempFile(dirName, "testFile*")
	require.NoError(t, err)

	// Check if the directory is not empty.
	isEmpty, err = IsEmptyDirectory(dirName)
	require.NoError(t, err)
	assert.False(t, isEmpty)
}
