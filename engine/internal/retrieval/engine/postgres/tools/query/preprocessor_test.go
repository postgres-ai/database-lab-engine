package query

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunParallel_ReportsPanickedQueryAsError(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(path.Join(dir, "query.sql"), []byte("select 1"), 0600))

	processor := NewQueryProcessor(nil, PreprocessorCfg{}, "postgres", "postgres")

	err := processor.runParallel(context.Background(), "container", dir)

	assert.ErrorContains(t, err, "panic in")
}
