/*
2019 © Postgres.ai
*/

package models

// Status defines the status of clones and instance.
type Status struct {
	Code    StatusCode `json:"code"`
	Message string     `json:"message"`
}

// StatusCode defines the status code of clones and instance.
type StatusCode string

// Constants declares available status codes and messages.
const (
	StatusOK        StatusCode = "OK"
	StatusCreating  StatusCode = "CREATING"
	StatusResetting StatusCode = "RESETTING"
	StatusDeleting  StatusCode = "DELETING"
	StatusExporting StatusCode = "EXPORTING"
	StatusFatal     StatusCode = "FATAL"
	StatusWarning   StatusCode = "WARNING"

	CloneMessageOK        = "Clone is ready to accept Postgres connections."
	CloneMessageCreating  = "Clone is being created."
	CloneMessageResetting = "Clone is being reset."
	CloneMessageDeleting  = "Clone is being deleted."
	CloneMessageFatal     = "Cloning failure."

	InstanceMessageOK      = "Instance is ready"
	InstanceMessageWarning = "Subsystems that need attention"
)
