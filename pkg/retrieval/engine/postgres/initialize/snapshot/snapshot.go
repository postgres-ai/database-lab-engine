/*
2020 © Postgres.ai
*/

// Package snapshot provides components for preparing initial snapshots.
package snapshot

import (
	"os/exec"

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
