/*
2021 © Postgres.ai
*/

package zfs

import (
	"strings"
)

type dsType string

const (
	snapshotType   dsType = "snapshot"
	fileSystemType dsType = "filesystem"
)

type snapshotFields []string

type snapshotSorting []string

type snapshotFilter struct {
	fields  snapshotFields
	sorting snapshotSorting
	dsType  dsType
	pool    string
}

var defaultFields = snapshotFields{
	"name",
	"used",
	"mountpoint",
	"compressratio",
	"available",
	"type",
	"origin",
	"creation",
	"referenced",
	"logicalreferenced",
	"logicalused",
	"usedbysnapshots",
	"usedbychildren",
	dataStateAtLabel,
}

var defaultSorting = snapshotSorting{
	"-S " + dataStateAtLabel,
	"-S creation",
}

func buildListCommand(filter snapshotFilter) string {
	cmdComponents := []string{
		"zfs list",
		"-po", strings.Join(filter.fields, ","),
		strings.Join(filter.sorting, " "),
		"-t", string(filter.dsType),
	}

	if filter.pool != "" {
		cmdComponents = append(cmdComponents, "-r", filter.pool)
	}

	return strings.Join(cmdComponents, " ")
}
