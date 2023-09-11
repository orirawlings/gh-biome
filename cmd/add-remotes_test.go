package cmd

import (
	"bytes"
	"testing"
)

func TestRemotes(t *testing.T) {
	saved := byName([]remote{
		{
			Name:     "github.com/foo/bar",
			FetchURL: "https://github.com/foo/bar.git",
		},
		{
			Name:     "github.com/foo/archived",
			FetchURL: "https://github.com/foo/archived.git",
			Archived: true,
		},
		{
			Name:     "github.com/foo/disabled",
			FetchURL: "https://github.com/foo/disabled.git",
			Disabled: true,
		},
	})
	var b bytes.Buffer
	saved.save(&b)
	var loaded remotes
	loaded.load(&b)

	if len(saved) != len(loaded) {
		t.Errorf("persisted remote data has unexpected number of entries: expected %d, was %d", len(saved), len(loaded))
	}
	for name, r := range saved {
		if r != loaded[name] {
			t.Errorf("remote data entry for %s not persisted as expected: expected %v, was %v", name, r, loaded[name])
		}
	}
}

func byName(remotes []remote) remotes {
	result := make(map[string]remote)
	for _, r := range remotes {
		result[r.Name] = r
	}
	return result
}
