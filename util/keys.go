package util

import (
	"sort"
)

func Keys[C any](m map[string]C) []string {
	keys := make([]string, len(m))
	i := 0
	for k, _ := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}
