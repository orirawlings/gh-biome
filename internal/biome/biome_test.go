package biome

import (
	"context"
	"strings"
	"testing"

	testutil "github.com/orirawlings/gh-biome/internal/util/testing"
)

func TestInit(t *testing.T) {
	ctx := context.Background()

	path := t.TempDir()

	defer testutil.Execute(t, "git", "-C", path, "maintenance", "unregister")

	t.Run("new biome", func(t *testing.T) {
		_, err := Init(ctx, path)
		testutil.Check(t, err)

		testutil.Execute(t, "git", "-C", path, "fsck")
		if refFormat := strings.TrimSpace(testutil.Execute(t, "git", "-C", path, "rev-parse", "--show-ref-format")); refFormat != "reftable" {
			t.Errorf("expected reftable format for references, but was %q", refFormat)
		}
		if fetchParallel := getGitConfig(t, path, "fetch.parallel"); fetchParallel != "0" {
			t.Errorf("expected parallel fetch to be enabled, but was: %q", fetchParallel)
		}
		if autoMaintenanceEnabled := getGitConfig(t, path, "maintenance.auto"); autoMaintenanceEnabled != "false" {
			t.Errorf("expected auto maintenance to be disabled, but was not")
		}
		if maintenanceStrategy := getGitConfig(t, path, "maintenance.strategy"); maintenanceStrategy != "incremental" {
			t.Errorf("expected incremental maintenance strategy, but was: %q", maintenanceStrategy)
		}
	})

	t.Run("existing biome", func(t *testing.T) {
		// assert that Init is idempotent
		_, err := Init(ctx, path)
		testutil.Check(t, err)
	})

	t.Run("existing repo with bad biome version", func(t *testing.T) {
		path := testutil.TempRepo(t)
		testutil.Execute(t, "git", "-C", path, "config", "set", "--local", versionKey, "foobar")
		if _, err := Init(ctx, path); err == nil {
			t.Errorf("expected initialization to fail, but did not")
		}
	})
}

func getGitConfig(t *testing.T, dir, key string) string {
	return strings.TrimSpace(testutil.Execute(t, "git", "-C", dir, "config", "get", "--local", key))
}

func TestLoad(t *testing.T) {

	ctx := context.Background()

	t.Run("newly initialized biome", func(t *testing.T) {
		path := t.TempDir()
		_, err := Init(ctx, path)
		testutil.Check(t, err)
		load(t, ctx, path, true)
	})

	t.Run("repo with bad biome version", func(t *testing.T) {
		path := testutil.TempRepo(t)
		testutil.Execute(t, "git", "-C", path, "config", "set", "--local", versionKey, "foobar")
		load(t, ctx, path, false)
	})

	t.Run("non-repo", func(t *testing.T) {
		path := t.TempDir()
		load(t, ctx, path, false)
	})
}

func load(t *testing.T, ctx context.Context, path string, shouldSucceed bool) Biome {
	b, err := Load(ctx, path)
	if shouldSucceed && err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if !shouldSucceed && err == nil {
		t.Errorf("expected biome to be invalid, but loaded successfully")
	}
	return b
}
