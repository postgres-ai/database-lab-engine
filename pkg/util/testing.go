/*
2019 © Postgres.ai
*/

// Package util provides utility functions. Testing related.
package util

import (
	"github.com/sergi/go-diff/diffmatchpatch"
)

func diff(a string, b string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(a, b, false)

	return dmp.DiffPrettyText(diffs)
}
