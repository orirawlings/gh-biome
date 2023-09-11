package cmd

import (
	"bytes"
	"testing"
)

var (
	barRemote = remote{
		Name:     "github.com/foo/bar",
		FetchURL: "https://github.com/foo/bar.git",
		Head:     "refs/remotes/github.com/foo/bar/heads/main",
	}
	archivedRemote = remote{
		Name:     "github.com/foo/archived",
		FetchURL: "https://github.com/foo/archived.git",
		Archived: true,
		Head:     "refs/remotes/github.com/foo/archived/heads/master",
	}
	disabledRemote = remote{
		Name:     "github.com/foo/disabled",
		FetchURL: "https://github.com/foo/disabled.git",
		Disabled: true,
		Head:     "refs/remotes/github.com/foo/disabled/heads/main",
	}
	lockedRemote = remote{
		Name:     "github.com/foo/locked",
		FetchURL: "https://github.com/foo/locked.git",
		Disabled: true,
		Head:     "refs/remotes/github.com/foo/locked/heads/main",
	}
	headlessRemote = remote{
		Name:     "github.com/foo/headless",
		FetchURL: "https://github.com/foo/headless.git",
	}
)

func TestRemotes(t *testing.T) {
	saved := byName([]remote{
		barRemote,
		archivedRemote,
		disabledRemote,
		lockedRemote,
		headlessRemote,
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

func TestNewRemotes(t *testing.T) {
	built := newRemotes([]repository{
		{
			URL: "https://github.com/foo/bar",
			DefaultBranchRef: &ref{
				Name:   "main",
				Prefix: "refs/heads/",
			},
		},
		{
			IsArchived: true,
			URL:        "https://github.com/foo/archived",
			DefaultBranchRef: &ref{
				Name:   "master",
				Prefix: "refs/heads/",
			},
		},
		{
			IsDisabled: true,
			URL:        "https://github.com/foo/disabled",
			DefaultBranchRef: &ref{
				Name:   "main",
				Prefix: "refs/heads/",
			},
		},
		{
			IsLocked: true,
			URL:      "https://github.com/foo/locked",
			DefaultBranchRef: &ref{
				Name:   "main",
				Prefix: "refs/heads/",
			},
		},
		{
			URL: "https://github.com/foo/headless",
		},
	})
	expected := byName([]remote{
		barRemote,
		archivedRemote,
		disabledRemote,
		lockedRemote,
		headlessRemote,
	})
	if len(expected) != len(built) {
		t.Errorf("remote data has unexpected number of entries: expected %d, was %d", len(expected), len(built))
	}
	for name, r := range expected {
		if r != built[name] {
			t.Errorf("remote data entry for %s not as expected: expected %v, was %v", name, r, built[name])
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
