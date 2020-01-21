/*
2019 © Postgres.ai
*/

// Package util provides utility functions. Slices related utils.
package util

func EqualStringSlicesUnordered(x, y []string) bool {
	xMap := make(map[string]int)
	yMap := make(map[string]int)

	for _, xElem := range x {
		xMap[xElem]++
	}
	for _, yElem := range y {
		yMap[yElem]++
	}

	for xMapKey, xMapVal := range xMap {
		if yMap[xMapKey] != xMapVal {
			return false
		}
	}

	for yMapKey, yMapVal := range yMap {
		if xMap[yMapKey] != yMapVal {
			return false
		}
	}

	return true
}

func Unique(list []string) []string {
	keys := make(map[string]bool)
	uqList := []string{}
	for _, entry := range list {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			uqList = append(uqList, entry)
		}
	}
	return uqList
}

func Contains(list []string, s string) bool {
	for _, item := range list {
		if s == item {
			return true
		}
	}
	return false
}
