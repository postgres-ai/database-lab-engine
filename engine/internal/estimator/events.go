/*
2021 © Postgres.ai
*/

package estimator

const (
	// ReadyEventType defines ready event type.
	ReadyEventType = "ready"

	// ResultEventType defines result event type.
	ResultEventType = "result"

	// ReadBlocksType defines client event that provides a number of reading blocks.
	ReadBlocksType = "read_blocks"
)

// Event defines the websocket event structure.
type Event struct {
	EventType string
}

// ResultEvent defines a result event.
type ResultEvent struct {
	EventType string
	Payload   Result
}

// ReadBlocksEvent defines a read blocks event.
type ReadBlocksEvent struct {
	EventType  string
	ReadBlocks uint64
}
