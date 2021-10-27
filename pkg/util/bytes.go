/*
2019 © Postgres.ai
*/

// Package util provides utility functions. Data size related functions.
package util

import (
	"crypto/sha1"
	"encoding/hex"
	"strconv"
)

// ParseBytes returns number of bytes from string.
func ParseBytes(str string) (uint64, error) {
	return strconv.ParseUint(str, 10, 64)
}

// HashID returns a hash of provided string.
func HashID(id string) string {
	h := sha1.New()
	_, _ = h.Write([]byte(id))

	return hex.EncodeToString(h.Sum(nil))
}
