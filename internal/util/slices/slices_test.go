package slices

import (
	"slices"
	"testing"
)

func TestSortedUnique(t *testing.T) {
	for name, run := range map[string]struct {
		in       []string
		expected []string
	}{
		"empty": {
			in:       nil,
			expected: nil,
		},
		"one": {
			in:       []string{"a"},
			expected: []string{"a"},
		},
		"no duplicates, unsorted": {
			in:       []string{"b", "a"},
			expected: []string{"a", "b"},
		},
		"duplicates, unsorted": {
			in:       []string{"b", "a", "a", "b", "a"},
			expected: []string{"a", "b"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			result := SortedUnique(run.in)
			if !slices.Equal(result, run.expected) {
				t.Errorf("unexpected result, wanted: %s, was: %s", run.expected, result)
			}
		})
	}
}
