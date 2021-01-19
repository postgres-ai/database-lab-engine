/*
2020 © Postgres.ai
*/

// Package snapshot provides components for preparing initial snapshots.
package snapshot

import (
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/dbmarker"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/runners"
)

func extractDataStateAt(dbMarker *dbmarker.Marker) string {
	dbMark, err := dbMarker.GetConfig()
	if err != nil {
		log.Err("Failed to retrieve dataStateAt from DBMarker config:", err)
		return ""
	}

	return dbMark.DataStateAt
}

func runPreprocessingScript(preprocessingScript string) error {
	commandOutput, err := runners.NewLocalRunner(false).Run(preprocessingScript)
	if err != nil {
		return errors.Wrap(err, "failed to run custom script")
	}

	log.Msg(commandOutput)

	return nil
}
