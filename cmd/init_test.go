package cmd

import (
	"context"
	"testing"

	testutil "github.com/orirawlings/gh-biome/internal/util/testing"
)

func init() {
	initCmd.SetContext(context.Background())
	pushInContext(initCmd)
}

func TestInitCmd_Execute(t *testing.T) {
	path := t.TempDir()
	defer testutil.Execute(t, "git", "-C", path, "maintenance", "unregister")
	rootCmd.SetArgs([]string{
		"init",
		path,
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error executing command: %v", err)
	}
}
