/*
2019 © Postgres.ai
*/

// Package util provides utility functions. Data size related functions.
package util

import (
	"strconv"
)

// ParseBytes returns number of bytes from string.
func ParseBytes(str string) (uint64, error) {
	return strconv.ParseUint(str, 10, 64)
}
