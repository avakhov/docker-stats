package util

import (
	"sort"
)

func Uniq(slice []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, val := range slice {
		_, ok := seen[val]
		if !ok {
			seen[val] = true
			result = append(result, val)
		}
	}
	sort.Strings(result)
	return result
}
