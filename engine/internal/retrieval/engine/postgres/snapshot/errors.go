/*
2020 © Postgres.ai
*/

package snapshot

type skipSnapshotErr struct {
	message string
}

func newSkipSnapshotErr(message string) *skipSnapshotErr {
	return &skipSnapshotErr{message: message}
}

func (e *skipSnapshotErr) Error() string {
	return e.message
}
