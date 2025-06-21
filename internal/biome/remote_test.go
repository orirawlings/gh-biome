package biome

import (
	"testing"
)

var (
	barRemote = remote{
		Name: "github.com/orirawlings/bar",
	}
	archivedRemote = remote{
		Name:     "github.com/orirawlings/archived",
		Archived: true,
	}
	disabledRemote = remote{
		Name:     "github.com/orirawlings/disabled",
		Disabled: true,
	}
	lockedRemote = remote{
		Name:     "github.com/orirawlings/locked",
		Disabled: true,
	}
	headlessRemote = remote{
		Name: "github.com/orirawlings/headless",
	}
	dotPrefixRemote = remote{
		Name: "github.com/orirawlings/.github",
	}
)

var (
	barRemoteCfg = remoteConfig{
		Remote: barRemote,
		Head:   "refs/remotes/github.com/orirawlings/bar/heads/main",
	}
	archivedRemoteCfg = remoteConfig{
		Remote: archivedRemote,
		Head:   "refs/remotes/github.com/orirawlings/archived/heads/master",
	}
)

func TestRemote_String(t *testing.T) {
	for _, r := range []remote{
		barRemote,
		archivedRemote,
		disabledRemote,
		lockedRemote,
		headlessRemote,
		dotPrefixRemote,
	} {
		t.Run(r.Name, func(t *testing.T) {
			if r.String() != r.Name {
				t.Errorf("expected %q, got %q", r.Name, r.String())
			}
		})
	}
}

func TestRemote_FetchURL(t *testing.T) {
	for _, r := range []struct {
		remote   remote
		expected string
	}{
		{
			remote:   barRemote,
			expected: "https://github.com/orirawlings/bar.git",
		},
		{
			remote:   archivedRemote,
			expected: "https://github.com/orirawlings/archived.git",
		},
		{
			remote:   disabledRemote,
			expected: "https://github.com/orirawlings/disabled.git",
		},
		{
			remote:   lockedRemote,
			expected: "https://github.com/orirawlings/locked.git",
		},
		{
			remote:   headlessRemote,
			expected: "https://github.com/orirawlings/headless.git",
		},
		{
			remote:   dotPrefixRemote,
			expected: "https://github.com/orirawlings/.github.git",
		},
	} {
		t.Run(r.remote.Name, func(t *testing.T) {
			if r.remote.FetchURL() != r.expected {
				t.Errorf("expected %q, got %q", r.expected, r.remote.FetchURL())
			}
		})
	}
}

func TestRemote_FetchRefspec(t *testing.T) {
	// remotes with a vaild fetch refspec
	for _, r := range []remote{
		barRemote,
		archivedRemote,
		disabledRemote,
		lockedRemote,
		headlessRemote,
	} {
		t.Run(r.Name, func(t *testing.T) {
			_, err := r.FetchRefspec()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
	// remotes with an invaild fetch refspec
	for _, r := range []remote{
		dotPrefixRemote,
	} {
		t.Run(r.Name, func(t *testing.T) {
			_, err := r.FetchRefspec()
			if err == nil {
				t.Errorf("expected error, but was nil")
			}
		})
	}
}

func TestRemote_Supported(t *testing.T) {
	// remotes with a vaild fetch refspec
	for _, r := range []remote{
		barRemote,
		archivedRemote,
		disabledRemote,
		lockedRemote,
		headlessRemote,
	} {
		t.Run(r.Name, func(t *testing.T) {
			if !r.Supported() {
				t.Errorf("expected remote to be supported, but was not")
			}
		})
	}
	// remotes with an invaild fetch refspec
	for _, r := range []remote{
		dotPrefixRemote,
	} {
		t.Run(r.Name, func(t *testing.T) {
			if r.Supported() {
				t.Errorf("expected remote to not be supported, but was")
			}
		})
	}
}
