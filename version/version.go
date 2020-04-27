/*
2019 © Postgres.ai
*/

// Package version provides the Database Lab version info.
package version

import (
	"fmt"
)

// ldflag variables.
var (
	version   string
	buildTime string
)

// GetVersion return the app version info.
func GetVersion() string {
	return fmt.Sprintf("%s-%s", version, buildTime)
}
