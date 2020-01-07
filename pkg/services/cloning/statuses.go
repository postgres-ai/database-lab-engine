/*
2019 © Postgres.ai
*/

package cloning

import (
	"gitlab.com/postgres-ai/database-lab/pkg/models"
)

var statusOk = &models.Status{
	Code:    "OK",
	Message: "Clone is ready to accept Postgres connections.",
}

var statusCreating = &models.Status{
	Code:    "CREATING",
	Message: "Clone is being created.",
}

var statusResetting = &models.Status{
	Code:    "RESETTING",
	Message: "Clone is being reset.",
}

var statusDeleting = &models.Status{
	Code:    "DELETING",
	Message: "Clone is being deleted.",
}

var statusFatal = &models.Status{
	Code:    "FATAL",
	Message: "Cloning failure.",
}
