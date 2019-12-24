/*
2019 © Postgres.ai
*/

package cloning

import (
	m "../models"
)

var statusOk = &m.Status{
	Code:    "OK",
	Message: "Clone is ready to accept Postgres connections.",
}

var statusCreating = &m.Status{
	Code:    "CREATING",
	Message: "Clone is being created.",
}

var statusResetting = &m.Status{
	Code:    "RESETTING",
	Message: "Clone is being reset.",
}

var statusDeleting = &m.Status{
	Code:    "DELETING",
	Message: "Clone is being deleted.",
}

var statusFatal = &m.Status{
	Code:    "FATAL",
	Message: "Cloning failure.",
}
