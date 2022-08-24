/*
2019 © Postgres.ai
*/

// Package util provides utility functions. Slices related utils.
package util

// Unique returns unique values of slice.
func Unique(list []string) []string {
	keys := make(map[string]struct{})
	uqList := []string{}

	for _, entry := range list {
		if _, value := keys[entry]; !value {
			keys[entry] = struct{}{}

			uqList = append(uqList, entry)
		}
	}

	return uqList
}

// IncludesString checks if a string is included in a slice.
func IncludesString(list []string, value string) bool {
	for _, entry := range list {
		if entry == value {
			return true
		}
	}

	return false
}
