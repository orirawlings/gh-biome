package cmd

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/orirawlings/gh-biome/internal/biome"
)

func init() {
	initCmd.SetContext(context.Background())
	pushInContext(initCmd)
}

func TestInitCmd_Execute(t *testing.T) {
	path := t.TempDir()
	defer execute(t, "git", "-C", path, "maintenance", "unregister")
	rootCmd.SetArgs([]string{
		"init",
		path,
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error executing command: %v", err)
	}
	execute(t, "git", "-C", path, "fsck")
	if refFormat := strings.TrimSpace(execute(t, "git", "-C", path, "rev-parse", "--show-ref-format")); refFormat != "reftable" {
		t.Errorf("expected reftable format for references, but was %q", refFormat)
	}
	c := biome.New(path)
	if err := c.Validate(context.Background()); err != nil {
		t.Errorf("biome initialized with invalid configuration: %v", err)
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
}

func getGitConfig(t *testing.T, dir, key string) string {
	return strings.TrimSpace(execute(t, "git", "-C", dir, "config", "get", "--local", key))
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
