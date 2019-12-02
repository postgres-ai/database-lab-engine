/*
2019 © Postgres.ai
*/

package util

import (
	"testing"
)

func TestEqualStringSlicesUnordered(t *testing.T) {
	if !EqualStringSlicesUnordered([]string{"a", "b", "c", "d"}, []string{"a", "b", "c", "d"}) {
		t.FailNow()
	}

	if !EqualStringSlicesUnordered([]string{"d", "c", "b", "a"}, []string{"a", "b", "c", "d"}) {
		t.FailNow()
	}

	if EqualStringSlicesUnordered([]string{"a", "x", "x", "x"}, []string{"a", "b", "c", "d"}) {
		t.FailNow()
	}

	if EqualStringSlicesUnordered([]string{}, []string{"a"}) {
		t.FailNow()
	}

	if EqualStringSlicesUnordered([]string{"a"}, []string{}) {
		t.FailNow()
	}
}
