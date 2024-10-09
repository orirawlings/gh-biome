package cmd

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"
)

var (
	barLegacyRemote = remote{
		Name:     "github.com/foo/bar",
		FetchURL: "https://github.com/foo/bar.git",
		Head:     "refs/remotes/github.com/foo/bar/heads/main",
	}
	archivedLegacyRemote = remote{
		Name:     "github.com/foo/archived",
		FetchURL: "https://github.com/foo/archived.git",
		Archived: true,
		Head:     "refs/remotes/github.com/foo/archived/heads/master",
	}
	disabledLegacyRemote = remote{
		Name:     "github.com/foo/disabled",
		FetchURL: "https://github.com/foo/disabled.git",
		Disabled: true,
		Head:     "refs/remotes/github.com/foo/disabled/heads/main",
	}
	lockedLegacyRemote = remote{
		Name:     "github.com/foo/locked",
		FetchURL: "https://github.com/foo/locked.git",
		Disabled: true,
		Head:     "refs/remotes/github.com/foo/locked/heads/main",
	}
	headlessLegacyRemote = remote{
		Name:     "github.com/foo/headless",
		FetchURL: "https://github.com/foo/headless.git",
	}
	dotPrefixLegacyRemote = remote{
		Name:     "github.com/foo/.github",
		FetchURL: "https://github.com/foo/.github.git",
		Head:     "refs/remotes/github.com/foo/.github/heads/main",
	}
)

func init() {
	addRemotesCmd.SetContext(context.Background())
	pushInContext(addRemotesCmd)
}

func TestFetchRefspec(t *testing.T) {
	// remotes with a vaild fetch refspec
	for _, r := range []remote{
		barLegacyRemote,
		archivedLegacyRemote,
		disabledLegacyRemote,
		lockedLegacyRemote,
		headlessLegacyRemote,
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
		dotPrefixLegacyRemote,
	} {
		t.Run(r.Name, func(t *testing.T) {
			_, err := r.FetchRefspec()
			if err == nil {
				t.Errorf("expected error, but was nil")
			}
		})
	}
}

func TestRemotes(t *testing.T) {
	saved := byName([]remote{
		barLegacyRemote,
		archivedLegacyRemote,
		disabledLegacyRemote,
		lockedLegacyRemote,
		headlessLegacyRemote,
		dotPrefixLegacyRemote,
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
	ctx := addRemotesCmd.Context()

	built := newRemotes(ctx, []repository{
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
		{
			URL: "https://github.com/foo/.github",
			DefaultBranchRef: &ref{
				Name:   "main",
				Prefix: "refs/heads/",
			},
		},
	})
	expected := byName([]remote{
		barLegacyRemote,
		archivedLegacyRemote,
		disabledLegacyRemote,
		lockedLegacyRemote,
		headlessLegacyRemote,
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

func TestSetHeads(t *testing.T) {
	ctx := addRemotesCmd.Context()

	repo, commitID := tempGitRepo(t)
	if err := os.Chdir(repo); err != nil {
		t.Fatalf("could not change directory to %s: %v", repo, err)
	}

	mainRef := "refs/remotes/github.com/orirawlings/gh-biome/heads/main"
	if err := exec.Command("git", "update-ref", mainRef, commitID).Run(); err != nil {
		t.Fatalf("could not update-ref %q to %q: %v", mainRef, commitID, err)
	}

	head := "refs/remotes/github.com/orirawlings/gh-biome/HEAD"

	remotes := newRemotes(ctx, []repository{
		{
			URL: "https://github.com/orirawlings/gh-biome",
			DefaultBranchRef: &ref{
				Name:   "main",
				Prefix: "refs/heads/",
			},
		},
	})
	if err := setHeads("test set heads", remotes); err != nil {
		t.Fatalf("could not set heads on repo %s: %v", repo, err)
	}
	checkLooseSymbolicRef(t, head, mainRef)

	aNewMainRef := "refs/remotes/github.com/orirawlings/gh-biome/heads/aNewMainBranch"
	if err := exec.Command("git", "update-ref", aNewMainRef, commitID).Run(); err != nil {
		t.Fatalf("could not update-ref %q to %q: %v", aNewMainRef, commitID, err)
	}
	remotes = newRemotes(ctx, []repository{
		{
			URL: "https://github.com/orirawlings/gh-biome",
			DefaultBranchRef: &ref{
				Name:   "aNewMainBranch",
				Prefix: "refs/heads/",
			},
		},
	})
	if err := setHeads("test set heads", remotes); err != nil {
		t.Fatalf("could not set heads on repo %s: %v", repo, err)
	}
	checkLooseSymbolicRef(t, head, aNewMainRef)

	remotes = newRemotes(ctx, []repository{
		{
			URL: "https://github.com/orirawlings/gh-biome",
		},
	})
	if err := setHeads("test set heads", remotes); err != nil {
		t.Fatalf("could not set heads on repo %s: %v", repo, err)
	}
	if _, err := os.Open(head); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected symbolic ref %q to be deleted, but was not: %v", head, err)
	}
}

func byName(remotes []remote) remotes {
	result := make(map[string]remote)
	for _, r := range remotes {
		result[r.Name] = r
	}
	return result
}

// tempGitRepo initializes a bare git repository in a new temporary directory.
// The repository is initialized with an initial commit object but no
// references. Returns repo path and object ID of the initial commit.
func tempGitRepo(t *testing.T) (string, string) {
	path := t.TempDir()
	cmd := exec.Command("git", "-C", path, "init", "--bare")
	if err := cmd.Run(); err != nil {
		t.Fatalf("could not init test git repo in %s: %v", path, err)
	}

	// Create the empty git tree object, 4b825dc642cb6eb9a060e54bf8d69288fbee4904
	if err := exec.Command("git", "-C", path, "write-tree").Run(); err != nil {
		t.Fatalf("could not create the empty git tree in %q: %v", path, err)
	}

	// Create the initial commit
	cmd = exec.Command("git", "-C", path, "hash-object", "-t", "commit", "-w", "--stdin")
	w, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("could not create stdin pipe for %q: %v", cmd, err)
	}
	r, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("could not create stdout pipe for %q: %v", cmd, err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("could not start %q in %q: %v", cmd, path, err)
	}
	if _, err := fmt.Fprint(w, `tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904
author A <a@example.com> 0 +0000
committer C <c@example.com> 0 +0000

initial commit

`); err != nil {
		t.Fatalf("could not write commit data to %q in %q: %v", cmd, path, err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("could not close commit data to %q in %q: %v", cmd, path, err)
	}
	s := bufio.NewScanner(r)
	var commitID string
	for s.Scan() {
		commitID = s.Text()
	}
	if err := s.Err(); err != nil {
		t.Fatalf("could not read data from %q in %q: %v", cmd, path, err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("%q failed in %q: %v", cmd, path, err)
	}
	if err := exec.Command("git", "-C", path, "cat-file", "-e", commitID).Run(); err != nil {
		t.Fatalf("could not confirm that %q was created in %q: %v", commitID, path, err)
	}
	return path, commitID
}

func checkLooseSymbolicRef(t *testing.T, ref, expected string) {
	f, err := os.Open(ref)
	defer func() {
		_ = f.Close()
	}()
	if err != nil {
		t.Fatalf("could not open symbolic ref %q: %v", ref, err)
	}
	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("could not read symbolic ref %q: %v", ref, err)
	}
	if string(data) != fmt.Sprintf("ref: %s\n", expected) {
		t.Errorf("expected %q to be a symbolic ref to %q, but had content %s", ref, expected, string(data))
	}
}
