package slices

import (
	"cmp"
	"maps"
	"slices"
)

// SortedUnique returns the given slice in sorted order with duplicates removed.
func SortedUnique[S ~[]E, E cmp.Ordered](x S) []E {
	return slices.Sorted(maps.Keys(maps.Collect[E, any](func(yield func(E, any) bool) {
		for _, k := range x {
			if ok := yield(k, nil); !ok {
				break
			}
		}
	})))
}
