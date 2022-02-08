/*
2019 © Postgres.ai
*/

package util

import (
	"testing"
)

func TestUnique(t *testing.T) {
	if s := Unique([]string{"x", "y", "z"}); len(s) != 3 {
		t.FailNow()
	}

	if s := Unique([]string{"x", "y", "y"}); len(s) != 2 {
		t.FailNow()
	}
}
