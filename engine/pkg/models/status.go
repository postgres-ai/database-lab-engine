/*
2019 © Postgres.ai
*/

package models

// Status defines the status of clones and instance.
type Status struct {
	Code    StatusCode `json:"code"`
	Message string     `json:"message"`
}

// Response defines the response structure.
type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
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
	StatusUpgrading StatusCode = "UPGRADING"
	StatusFatal     StatusCode = "FATAL"
	StatusWarning   StatusCode = "WARNING"

	CloneMessageOK        = "Clone is ready to accept Postgres connections."
	CloneMessageCreating  = "Clone is being created."
	CloneMessageResetting = "Clone is being reset."
	CloneMessageDeleting  = "Clone is being deleted."
	CloneMessageUpgrading = "Clone is being upgraded."
	CloneMessageFatal     = "Cloning failure."

	// CloneMessageUpgradeWarning prefixes the message of a clone left in StatusWarning by an
	// upgrade that did not take effect. The clone is running and usable; the appended detail
	// says on which major version and what has to be done next.
	CloneMessageUpgradeWarning = "Upgrade was not applied."

	// CloneMessageUpgradeUnsettled prefixes the message of a clone whose upgrade left it without a
	// container. Unlike CloneMessageUpgradeWarning the clone cannot be connected to, but its data
	// directory is intact and the upgrade is still recorded as pending, so the next engine start
	// finishes or undoes it rather than discarding the clone.
	CloneMessageUpgradeUnsettled = "Upgrade did not settle."

	InstanceMessageOK      = "Instance is ready"
	InstanceMessageWarning = "Subsystems that need attention"

	SyncStatusOK           StatusCode = "OK"
	SyncStatusStarting     StatusCode = "Starting"
	SyncStatusDown         StatusCode = "Down"
	SyncStatusNotAvailable StatusCode = "Not available"
	SyncStatusError        StatusCode = "Error"

	ResponseOK = "OK"
)
