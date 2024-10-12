package biome

import (
	"context"
	"testing"

	testutil "github.com/orirawlings/gh-biome/internal/util/testing"
)

func TestBiome_Init(t *testing.T) {
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

func TestBiome_Validate(t *testing.T) {
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

func validate(t *testing.T, ctx context.Context, c Biome, expectedValid bool) {
	err := c.Validate(ctx)
	if expectedValid && err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if !expectedValid && err == nil {
		t.Errorf("expected biome to be invalid, but passed validation")
	}
}
