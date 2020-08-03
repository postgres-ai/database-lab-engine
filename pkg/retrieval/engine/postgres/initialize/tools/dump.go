/*
2020 © Postgres.ai
*/

package tools

import (
	"bufio"
	"io"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	// DataStateAtFormat describes format of dataStateAt.
	DataStateAtFormat = "20060102150405"

	archiveCreatedLabel = "Archive created at "
	archiveTimeFormat   = "2006-01-02 15:04:05 MST"
)

// DiscoverDataStateAt scans input data and discovers time when the archive is created.
func DiscoverDataStateAt(input io.Reader) (string, error) {
	scanner := bufio.NewScanner(input)

	for scanner.Scan() {
		text := scanner.Text()

		// Define the first index in order to cut out insufficient bytes from the start of the line.
		i := strings.Index(text, ";")
		if i < 0 {
			continue
		}

		dataStateAt, err := parseDataStateAt(strings.TrimLeft(text[i:], "; "))
		if err != nil {
			return "", errors.Wrap(err, "failed to parse dataStateAt")
		}

		if dataStateAt == "" {
			continue
		}

		return dataStateAt, nil
	}

	return "", errors.New("dataStateAt not found")
}

// parseDataStateAt retrieves date and time from incoming string and returns a formatted string.
func parseDataStateAt(inputLine string) (string, error) {
	if !strings.HasPrefix(inputLine, archiveCreatedLabel) {
		return "", nil
	}

	datetimeString := strings.TrimPrefix(inputLine, archiveCreatedLabel)

	datetime, err := time.Parse(archiveTimeFormat, datetimeString)
	if err != nil {
		return "", err
	}

	return datetime.Format(DataStateAtFormat), nil
}
