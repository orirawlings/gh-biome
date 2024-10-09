package config

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"testing"

	slicesutil "github.com/orirawlings/gh-biome/internal/util/slices"
	testutil "github.com/orirawlings/gh-biome/internal/util/testing"
)

func TestConfig_Init(t *testing.T) {
	ctx := context.Background()

	t.Run("bare repo", func(t *testing.T) {
		c := New(testutil.TempRepo(t))
		testutil.Check(t, c.Init(ctx))
		validate(t, ctx, c, true)

		// reinitialization should succeed
		testutil.Check(t, c.Init(ctx))
		validate(t, ctx, c, true)
	})

	t.Run("repo with bad biome version", func(t *testing.T) {
		path := testutil.TempRepo(t)
		testutil.Execute(t, "git", "-C", path, "config", "set", "--local", versionKey, "foobar")
		c := New(path)
		if err := c.Init(ctx); err == nil {
			t.Errorf("expected initialization to fail, but did not")
		}
	})
}

func TestConfig_Validate(t *testing.T) {
	ctx := context.Background()

	t.Run("newly initialized biome", func(t *testing.T) {
		c := New(testutil.TempRepo(t))
		validate(t, ctx, c, false)
		testutil.Check(t, c.Init(ctx))
		validate(t, ctx, c, true)
	})

	t.Run("repo with bad biome version", func(t *testing.T) {
		path := testutil.TempRepo(t)
		testutil.Execute(t, "git", "-C", path, "config", "set", "--local", versionKey, "foobar")
		c := New(path)
		validate(t, ctx, c, false)
	})

	t.Run("non-repo", func(t *testing.T) {
		path := t.TempDir()
		c := New(path)
		validate(t, ctx, c, false)
	})
}

func TestConfig_Owners(t *testing.T) {
	ctx := context.Background()
	c := newBiome(t, ctx)
	owners, err := c.Owners(ctx)
	testutil.Check(t, err)
	if len(owners) != 0 {
		t.Errorf("expected zero owners in biome, but was: %d", len(owners))
	}

	testutil.Check(t, c.AddOwners(ctx, "github.com/orirawlings"))
	expectOwners(t, ctx, c, []string{
		"github.com/orirawlings",
	})

	testutil.Check(t, c.AddOwners(ctx, "github.com/orirawlings", "github.com/kubernetes"))
	expectOwners(t, ctx, c, []string{
		"github.com/kubernetes",
		"github.com/orirawlings",
	})

	testutil.Check(t, c.AddOwners(ctx, "github.com/git", "github.com/cli"))
	expectOwners(t, ctx, c, []string{
		"github.com/cli",
		"github.com/git",
		"github.com/kubernetes",
		"github.com/orirawlings",
	})

	testutil.Check(t, c.AddOwners(ctx, "github.com/git", "github.com/cli"))
	expectOwners(t, ctx, c, []string{
		"github.com/cli",
		"github.com/git",
		"github.com/kubernetes",
		"github.com/orirawlings",
	})
}

func TestConfig_UpdateRemotes(t *testing.T) {
	ctx := context.Background()
	path := testutil.TempRepo(t)
	c := New(path)
	testutil.Check(t, c.Init(ctx))

	c.AddOwners(ctx, "github.com/orirawlings")
	testutil.Check(t, c.UpdateRemotes(ctx, []Remote{
		barRemote,
		archivedRemote,
		disabledRemote,
		lockedRemote,
		dotPrefixRemote,
	}))

	expectedRemotes := slicesutil.SortedUnique([]string{
		barRemote.Name,
		archivedRemote.Name,
	})
	remotes := strings.Split(strings.TrimSpace(testutil.Execute(t, "git", "-C", path, "remote", "show")), "\n")
	if !slices.Equal(remotes, expectedRemotes) {
		t.Errorf("git remotes are incorrect: wanted %v, was %v", expectedRemotes, remotes)
	}

	t.Error("TODO (orirawlings): Do more assertions")
}

func TestConfig_SetHeads(t *testing.T) {
	ctx := context.Background()

	path := testutil.TempRepo(t)
	c := New(path)
	testutil.Check(t, c.Init(ctx))
	commitID := addInitialCommit(t, path)

	remote := Remote{
		Name: "github.com/orirawlings/gh-biome",
	}
	head := fmt.Sprintf("refs/remotes/%s/HEAD", remote.Name)

	for _, expectedTarget := range []string{
		fmt.Sprintf("refs/remotes/%s/heads/main", remote.Name),
		fmt.Sprintf("refs/remotes/%s/heads/anotherMain", remote.Name),
	} {
		// ensure expected target is a valid git reference
		testutil.Execute(t, "git", "-C", path, "update-ref", expectedTarget, commitID)

		remote.Head = expectedTarget
		testutil.Check(t, c.SetHeads(ctx, "test set heads", []Remote{remote}))
		target := strings.TrimSpace(testutil.Execute(t, "git", "-C", path, "for-each-ref", "--format=%(symref)", head))
		if target != expectedTarget {
			t.Errorf("unexpected target for %s: wanted %q, was %q", head, expectedTarget, target)
		}
	}

	// remove remote HEAD
	remote.Head = ""
	testutil.Check(t, c.SetHeads(ctx, "test set heads", []Remote{remote}))
	target := strings.TrimSpace(testutil.Execute(t, "git", "-C", path, "for-each-ref", head))
	if target != "" {
		t.Errorf("expected ref %q to be deleted, but was not", head)
	}
}

func validate(t *testing.T, ctx context.Context, c Config, expectedValid bool) {
	err := c.Validate(ctx)
	if expectedValid && err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if !expectedValid && err == nil {
		t.Errorf("expected biome config to be invalid, but passed validation")
	}
}

func expectOwners(t *testing.T, ctx context.Context, c Config, expected []string) {
	owners, err := c.Owners(ctx)
	testutil.Check(t, err)
	if !slices.Equal(owners, expected) {
		t.Errorf("unexpected owners: wanted %v, was %v", expected, owners)
	}
}

func newBiome(t *testing.T, ctx context.Context) Config {
	c := New(testutil.TempRepo(t))
	testutil.Check(t, c.Init(ctx))
	return c
}

// addInitialCommit object to the git repository at the given path and return the
// object ID of the commit.
func addInitialCommit(t *testing.T, path string) string {
	// create the empty git tree object, 4b825dc642cb6eb9a060e54bf8d69288fbee4904
	_ = testutil.Execute(t, "git", "-C", path, "write-tree")

	// create an initial commit
	cmd := exec.Command("git", "-C", path, "hash-object", "-t", "commit", "-w", "--stdin")
	cmd.Stdin = bytes.NewReader([]byte(`tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904
author A <a@example.com> 0 +0000
committer C <c@example.com> 0 +0000

initial commit

`))
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			t.Errorf("unexpected error exec'ing %q: %v\n\nstdout:\n%s\n\nstderr:\n%s\n", cmd.String(), err, out, exitErr.Stderr)
		} else {
			t.Errorf("unexpected error exec'ing %q: %v", cmd.String(), err)
		}
	}
	commitID := string(bytes.TrimSpace(out))

	// validate commit exists
	_ = testutil.Execute(t, "git", "-C", path, "cat-file", "-e", commitID)
	return commitID
}
