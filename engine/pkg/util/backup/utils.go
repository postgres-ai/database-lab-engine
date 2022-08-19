package backup

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
)

// getFileTimestamp returns the timestamp of a backup file.
// expected filename format: <name>.<extension>.<timestamp>.bak
func getFileTimestamp(filename string) (time.Time, error) {
	base := filepath.Base(filename)
	split := strings.Split(base, ".")

	const expectedSize = 3

	if len(split) < expectedSize {
		return time.Time{}, fmt.Errorf("invalid filename format: %s", filename)
	}

	timeStr := split[len(split)-2]
	timeStamp, err := time.Parse(util.DataStateAtFormat, timeStr)

	if err != nil {
		return time.Time{}, errors.Wrap(err, "failed to parse timestamp")
	}

	return timeStamp, nil
}
