package config

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5/plumbing/format/config"
	testutil "github.com/orirawlings/gh-biome/internal/util/testing"
)

func TestEditor(t *testing.T) {
	const (
		section       = "biome-test"
		sectionKey    = "editor"
		configKey     = section + "." + sectionKey
		expectedValue = "foobar"
	)
	t.Run("save edits", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
		defer cancel()
		path := testutil.TempRepo(t)
		assertConfigNotSet(ctx, t, path, configKey)
		err := newEditor(t, path).Edit(ctx, func(ctx context.Context, c *config.Config) (bool, error) {
			c.Section(section).SetOption(sectionKey, expectedValue)
			return true, nil
		})
		testutil.Check(t, err)
		if v := strings.TrimSpace(testutil.Execute(t, "git", "-C", path, "config", "get", "--local", configKey)); v != expectedValue {
			t.Errorf("expected config %q to be updated with value %q, was %q", configKey, expectedValue, v)
		}
	})

	t.Run("do not save edits", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
		defer cancel()
		path := testutil.TempRepo(t)
		err := newEditor(t, path).Edit(ctx, func(ctx context.Context, c *config.Config) (bool, error) {
			c.Section(section).SetOption(sectionKey, expectedValue)
			return false, nil
		})
		testutil.Check(t, err)
		assertConfigNotSet(ctx, t, path, configKey)
	})

	t.Run("editing error", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
		defer cancel()
		path := testutil.TempRepo(t)
		editorErr := errors.New("pretend something went wrong")
		err := newEditor(t, path).Edit(ctx, func(ctx context.Context, c *config.Config) (bool, error) {
			c.Section(section).SetOption(sectionKey, expectedValue)
			return true, editorErr
		})
		if err == nil {
			t.Error("expected error, but was nil")
		}
		if !errors.Is(err, editorErr) {
			t.Error("expected error to wrap the editor's error, but did not")
		}
		assertConfigNotSet(ctx, t, path, configKey)
	})
}

func newEditor(t *testing.T, repoPath string) Editor {
	_, thisFilePath, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine current source file path")
	}
	projectSourceDir := filepath.Join(filepath.Dir(thisFilePath), "../..")
	biomePath := filepath.Join(t.TempDir(), "biome")
	testutil.Execute(t, "go", "build", "-o", biomePath, projectSourceDir)
	return NewEditor(repoPath, helperCommand(fmt.Sprintf("%s config-edit-helper", biomePath)))
}

func assertConfigNotSet(ctx context.Context, t *testing.T, path, configKey string) {
	cmd := exec.CommandContext(ctx, "git", "-C", path, "config", "get", "--local", configKey)
	out, err := cmd.Output()
	if err == nil {
		t.Error("expected error, but was nil")
		return
	}
	ee, ok := err.(*exec.ExitError)
	if !ok {
		t.Errorf("expected exit error, was: %v", err)
		return
	}
	if ee.ExitCode() != 1 {
		t.Errorf("expected config key %q to not exist, instead %q exited %d, output:\n%s", configKey, cmd, ee.ExitCode(), string(out))
	}
}
