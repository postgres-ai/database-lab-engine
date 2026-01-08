/*
2024 © Postgres.ai
*/

package models

// AutoDeleteMode defines the auto-deletion behavior for entities.
type AutoDeleteMode int

// Auto-deletion mode constants.
const (
	// AutoDeleteOff disables auto-deletion.
	AutoDeleteOff AutoDeleteMode = 0
	// AutoDeleteSoft enables auto-deletion only if no dependencies exist.
	AutoDeleteSoft AutoDeleteMode = 1
	// AutoDeleteForce enables auto-deletion with recursive destroy of dependencies.
	AutoDeleteForce AutoDeleteMode = 2
)

// String returns the string representation of the auto-delete mode.
func (m AutoDeleteMode) String() string {
	switch m {
	case AutoDeleteOff:
		return "off"
	case AutoDeleteSoft:
		return "soft"
	case AutoDeleteForce:
		return "force"
	default:
		return "unknown"
	}
}

// IsValid checks if the auto-delete mode value is valid.
func (m AutoDeleteMode) IsValid() bool {
	return m >= AutoDeleteOff && m <= AutoDeleteForce
}
