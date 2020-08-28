/*
2020 © Postgres.ai
*/

// Package snapshot provides components for preparing initial snapshots.
package snapshot

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/dbmarker"
)

func extractDataStateAt(dbMarker *dbmarker.Marker) string {
	dataStateAt := ""

	dbMark, err := dbMarker.GetConfig()
	if err != nil {
		log.Err("Failed to retrieve dataStateAt from DBMarker config:", err)
	} else {
		dataStateAt = dbMark.DataStateAt
	}

	return dataStateAt
}

func runPreprocessingScript(preprocessingScript string) error {
	commandOutput, err := exec.Command("bash", preprocessingScript).Output()
	if err != nil {
		return errors.Wrap(err, "failed to run custom script")
	}

	log.Msg(string(commandOutput))

	return nil
}

func applyUsersConfigs(usersConfig map[string]string, filename string) error {
	configFile, err := os.OpenFile(filename, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return errors.Wrapf(err, "failed to open configuration file: %v", filename)
	}

	defer func() { _ = configFile.Close() }()

	sb := strings.Builder{}
	sb.WriteString("\n")

	for configKey, configValue := range usersConfig {
		sb.WriteString(fmt.Sprintf("%s = '%s'\n", configKey, configValue))
	}

	_, err = configFile.WriteString(sb.String())

	return err
}
