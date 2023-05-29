package webhooks

const (
	// CloneCreatedEvent defines the clone create event type.
	CloneCreatedEvent = "clone_create"
	// CloneResetEvent defines the clone reset event type.
	CloneResetEvent = "clone_reset"
	// CloneDeleteEvent defines the clone delete event type.
	CloneDeleteEvent = "clone_delete"

	// SnapshotCreateEvent defines the snapshot create event type.
	SnapshotCreateEvent = "snapshot_create"

	// SnapshotDeleteEvent defines the snapshot delete event type.
	SnapshotDeleteEvent = "snapshot_delete"

	// BranchCreateEvent defines the branch create event type.
	BranchCreateEvent = "branch_create"

	// BranchDeleteEvent defines the branch delete event type.
	BranchDeleteEvent = "branch_delete"
)

// EventTyper unifies webhook events.
type EventTyper interface {
	GetType() string
}

// BasicEvent defines payload of basic webhook event.
type BasicEvent struct {
	EventType string `json:"event_type"`
	EntityID  string `json:"entity_id"`
}

// GetType returns type of the event.
func (e BasicEvent) GetType() string {
	return e.EventType
}

// CloneEvent defines clone webhook events payload.
type CloneEvent struct {
	BasicEvent
	Host     string `json:"host,omitempty"`
	Port     uint   `json:"port,omitempty"`
	Username string `json:"username,omitempty"`
	DBName   string `json:"dbname,omitempty"`
}
