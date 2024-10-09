package cmd

import (
	"context"
	"slices"
	"testing"

	"github.com/orirawlings/gh-biome/internal/config"
)

func init() {
	addCmd.SetContext(context.Background())
	pushInContext(addCmd)
}

func TestParseOwnerRef(t *testing.T) {
	for _, run := range []struct {
		owner        string
		expectedHost string
		expectedName string
		invalid      bool
	}{
		{
			owner:        "orirawlings",
			expectedHost: "github.com",
			expectedName: "orirawlings",
		},
		{
			owner:        "github.com/orirawlings",
			expectedHost: "github.com",
			expectedName: "orirawlings",
		},
		{
			owner:        "https://github.com/orirawlings",
			expectedHost: "github.com",
			expectedName: "orirawlings",
		},
		{
			owner:   "https://foobar",
			invalid: true,
		},
		{
			owner:   "https://",
			invalid: true,
		},
		{
			owner:   "",
			invalid: true,
		},
	} {
		t.Run(string(run.owner), func(t *testing.T) {
			host, name, err := parseOwnerRef(run.owner)
			if run.invalid {
				if err == nil {
					t.Error("expected parse error, but was nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected parse error: %v", err)
				}
				if host != run.expectedHost {
					t.Errorf("unexpected host: wanted %q, was %q", run.expectedHost, host)
				}
				if name != run.expectedName {
					t.Errorf("unexpected name: wanted %q, was %q", run.expectedName, name)
				}
			}
		})
	}
}

func TestNormalizeOwners(t *testing.T) {
	owners := normalizeOwners([]string{
		"orirawlings",
		"github.com/orirawlings",
		"https://github.com/orirawlings",
		"https://foobar",
		"https://",
		"",
	})
	expected := []string{
		"github.com/orirawlings",
	}
	if !slices.Equal(owners, expected) {
		t.Errorf("wanted %s, was %s", expected, owners)
	}
}

var (
	barRepo = repository{
		URL: "https://github.com/orirawlings/bar",
		DefaultBranchRef: &ref{
			Name:   "main",
			Prefix: "refs/heads/",
		},
	}
	archivedRepo = repository{
		IsArchived: true,
		URL:        "https://github.com/orirawlings/archived",
		DefaultBranchRef: &ref{
			Name:   "master",
			Prefix: "refs/heads/",
		},
	}
	disabledRepo = repository{
		IsDisabled: true,
		URL:        "https://github.com/orirawlings/disabled",
		DefaultBranchRef: &ref{
			Name:   "main",
			Prefix: "refs/heads/",
		},
	}
	lockedRepo = repository{
		IsLocked: true,
		URL:      "https://github.com/orirawlings/locked",
		DefaultBranchRef: &ref{
			Name:   "main",
			Prefix: "refs/heads/",
		},
	}
	headlessRepo = repository{
		URL: "https://github.com/orirawlings/headless",
	}
	dotPrefixRepo = repository{
		URL: "https://github.com/orirawlings/.github",
		DefaultBranchRef: &ref{
			Name:   "main",
			Prefix: "refs/heads/",
		},
	}

	barRemote = config.Remote{
		Name:     "github.com/orirawlings/bar",
		FetchURL: "https://github.com/orirawlings/bar.git",
		Head:     "refs/remotes/github.com/orirawlings/bar/heads/main",
	}
	archivedRemote = config.Remote{
		Name:     "github.com/orirawlings/archived",
		FetchURL: "https://github.com/orirawlings/archived.git",
		Archived: true,
		Head:     "refs/remotes/github.com/orirawlings/archived/heads/master",
	}
	disabledRemote = config.Remote{
		Name:     "github.com/orirawlings/disabled",
		FetchURL: "https://github.com/orirawlings/disabled.git",
		Disabled: true,
		Head:     "refs/remotes/github.com/orirawlings/disabled/heads/main",
	}
	lockedRemote = config.Remote{
		Name:     "github.com/orirawlings/locked",
		FetchURL: "https://github.com/orirawlings/locked.git",
		Disabled: true,
		Head:     "refs/remotes/github.com/orirawlings/locked/heads/main",
	}
	headlessRemote = config.Remote{
		Name:     "github.com/orirawlings/headless",
		FetchURL: "https://github.com/orirawlings/headless.git",
	}
	dotPrefixRemote = config.Remote{
		Name:     "github.com/orirawlings/.github",
		FetchURL: "https://github.com/orirawlings/.github.git",
		Head:     "refs/remotes/github.com/orirawlings/.github/heads/main",
	}
)

func TestRepository_Remote(t *testing.T) {
	for name, run := range map[string]struct {
		repo     repository
		expected config.Remote
	}{
		"bar": {
			repo:     barRepo,
			expected: barRemote,
		},
		"archived": {
			repo:     archivedRepo,
			expected: archivedRemote,
		},
		"disabled": {
			repo:     disabledRepo,
			expected: disabledRemote,
		},
		"locked": {
			repo:     lockedRepo,
			expected: lockedRemote,
		},
		"headless": {
			repo:     headlessRepo,
			expected: headlessRemote,
		},
		"dotPrefix": {
			repo:     dotPrefixRepo,
			expected: dotPrefixRemote,
		},
	} {
		t.Run(name, func(t *testing.T) {
			remote := run.repo.Remote()
			if remote != run.expected {
				t.Errorf("expected %v, was %v", run.expected, remote)
			}
		})
	}
}

func TestBuildRemotes(t *testing.T) {
	ctx := addRemotesCmd.Context()

	remotes := buildRemotes(ctx, []repository{
		barRepo,
		archivedRepo,
		disabledRepo,
		lockedRepo,
		headlessRepo,
		dotPrefixRepo,
	})
	expected := []config.Remote{
		barRemote,
		archivedRemote,
		disabledRemote,
		lockedRemote,
		headlessRemote,
	}
	if !slices.Equal(remotes, expected) {
		t.Errorf("expected %v, was %v", expected, remotes)
	}
}
