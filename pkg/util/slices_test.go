/*
2019 © Postgres.ai
*/

package util

import (
	"testing"
)

func TestEqualStringSlicesUnordered(t *testing.T) {
	if !EqualStringSlicesUnordered([]string{"a", "b", "c", "d"},
		[]string{"a", "b", "c", "d"}) {
		t.FailNow()
	}

	if !EqualStringSlicesUnordered([]string{"d", "c", "b", "a"},
		[]string{"a", "b", "c", "d"}) {
		t.FailNow()
	}

	if EqualStringSlicesUnordered([]string{"a", "x", "x", "x"},
		[]string{"a", "b", "c", "d"}) {
		t.FailNow()
	}

	if EqualStringSlicesUnordered([]string{}, []string{"a"}) {
		t.FailNow()
	}

	if EqualStringSlicesUnordered([]string{"a"}, []string{}) {
		t.FailNow()
	}
}

func TestUnique(t *testing.T) {
	if s := Unique([]string{"x", "y", "z"}); len(s) != 3 {
		t.FailNow()
	}

	if s := Unique([]string{"x", "y", "y"}); len(s) != 2 {
		t.FailNow()
	}
}

func TestContains(t *testing.T) {
	if !Contains([]string{"x", "y", "z"}, "z") {
		t.FailNow()
	}

	if Contains([]string{"x", "y", "z"}, "k") {
		t.FailNow()
	}
}
