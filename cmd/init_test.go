package cmd

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/orirawlings/gh-biome/internal/biome"
	"github.com/orirawlings/gh-biome/internal/config"
	testutil "github.com/orirawlings/gh-biome/internal/util/testing"
)

func init() {
	initCmd.SetContext(context.Background())
	pushInContext(initCmd)
}

func TestInitCmd_Execute(t *testing.T) {
	initBiome(t)
}

func initBiome(t *testing.T) {
	t.Helper()

	path := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot determine current working directory: %v", err)
	}

	// override biome options
	oldOptions := biomeOptions
	biomeOptions = []biome.BiomeOption{
		biome.EditorOptions(config.HelperCommand(fmt.Sprintf("%s config-edit-helper", biomeBuildPath))),
	}
	t.Cleanup(func() {
		biomeOptions = oldOptions
	})

	// init biome
	rootCmd.SetArgs([]string{
		"init",
		path,
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error executing command: %v", err)
	}
	t.Cleanup(func() {
		testutil.Execute(t, "git", "-C", path, "maintenance", "unregister")
	})

	// switch to biome directory
	if err := os.Chdir(path); err != nil {
		t.Fatalf("could not change to the biome directory: %v", err)
	}
	t.Cleanup(func() {
		os.Chdir(oldWD)
	})
}
