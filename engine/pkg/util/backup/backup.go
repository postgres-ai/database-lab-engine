// Package backup utilities to back up and restore data
package backup

import (
	"time"
)

const (
	backupFileExtension = ".bak"
)

type backup struct {
	Filename string
	Time     time.Time
}

var now = func() time.Time {
	return time.Now().In(time.UTC)
}
