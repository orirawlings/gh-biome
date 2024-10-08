package config

import (
	"errors"
	"os/exec"
	"testing"
)

func TestConfig_Init(t *testing.T) {
	t.Run("bare repo", func(t *testing.T) {
		path := newRepo(t)
		c := New(path)
		if err := c.Init(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		validate(t, c, true)

		// reinitialization should succeed
		if err := c.Init(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		validate(t, c, true)
	})

	t.Run("repo with bad biome version", func(t *testing.T) {
		path := newRepo(t)
		_ = execute(t, "git", "-C", path, "config", "set", "--local", biomeVersionKey, "foobar")
		c := New(path)
		if err := c.Init(); err == nil {
			t.Errorf("expected initialization to fail, but did not")
		}
	})
}

func TestConfig_Validate(t *testing.T) {
	t.Run("newly initialized biome", func(t *testing.T) {
		path := newRepo(t)
		c := New(path)
		validate(t, c, false)
		if err := c.Init(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		validate(t, c, true)
	})

	t.Run("repo with bad biome version", func(t *testing.T) {
		path := newRepo(t)
		_ = execute(t, "git", "-C", path, "config", "set", "--local", biomeVersionKey, "foobar")
		c := New(path)
		validate(t, c, false)
	})

	t.Run("non-repo", func(t *testing.T) {
		path := t.TempDir()
		c := New(path)
		validate(t, c, false)
	})
}

func validate(t *testing.T, c Config, expectedValid bool) {
	err := c.Validate()
	if expectedValid && err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if !expectedValid && err == nil {
		t.Errorf("expected biome config to be invalid, but passed validation")
	}
}

func newRepo(t *testing.T) string {
	path := t.TempDir()
	_ = execute(t, "git", "-C", path, "init", "--bare")
	return path
}

func execute(t *testing.T, command ...string) string {
	cmd := exec.Command(command[0], command[1:]...)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			t.Errorf("unexpected error exec'ing %q: %v\n\nstdout:\n%s\n\nstderr:\n%s\n", cmd.String(), err, out, exitErr.Stderr)
		} else {
			t.Errorf("unexpected error exec'ing %q: %v", cmd.String(), err)
		}
	}
	return string(out)
}
