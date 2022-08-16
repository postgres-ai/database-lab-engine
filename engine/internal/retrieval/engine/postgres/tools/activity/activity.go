// Package activity observes activities of the data retrieval process.
package activity

// Activity represents job activity.
type Activity struct {
	Source []PGEvent
	Target []PGEvent
}

// PGEvent represents pg_stat_activity event.
type PGEvent struct {
	User          string
	Duration      float64
	Query         string
	WaitEventType string
	WaitEvent     string
}
