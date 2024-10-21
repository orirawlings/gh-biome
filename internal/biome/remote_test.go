package biome

import (
	"testing"
)

var (
	barRemote = remote{
		Name:     "github.com/orirawlings/bar",
		FetchURL: "https://github.com/orirawlings/bar.git",
		Head:     "refs/remotes/github.com/orirawlings/bar/heads/main",
	}
	archivedRemote = remote{
		Name:     "github.com/orirawlings/archived",
		FetchURL: "https://github.com/orirawlings/archived.git",
		Archived: true,
		Head:     "refs/remotes/github.com/orirawlings/archived/heads/master",
	}
	disabledRemote = remote{
		Name:     "github.com/orirawlings/disabled",
		FetchURL: "https://github.com/orirawlings/disabled.git",
		Disabled: true,
		Head:     "refs/remotes/github.com/orirawlings/disabled/heads/main",
	}
	lockedRemote = remote{
		Name:     "github.com/orirawlings/locked",
		FetchURL: "https://github.com/orirawlings/locked.git",
		Disabled: true,
		Head:     "refs/remotes/github.com/orirawlings/locked/heads/main",
	}
	headlessRemote = remote{
		Name:     "github.com/orirawlings/headless",
		FetchURL: "https://github.com/orirawlings/headless.git",
	}
	dotPrefixRemote = remote{
		Name:     "github.com/orirawlings/.github",
		FetchURL: "https://github.com/orirawlings/.github.git",
		Head:     "refs/remotes/github.com/orirawlings/.github/heads/main",
	}
)

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
