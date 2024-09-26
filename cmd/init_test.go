package cmd

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func init() {
	initCmd.SetContext(context.Background())
	pushInContext(initCmd)
}

func TestInitCmd_Execute(t *testing.T) {
	dir := t.TempDir()
	defer execute(t, "git", "-C", dir, "maintenance", "unregister")
	rootCmd.SetArgs([]string{
		"init",
		dir,
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error executing command: %v", err)
	}
	execute(t, "git", "-C", dir, "fsck")
	if refFormat := strings.TrimSpace(execute(t, "git", "-C", dir, "rev-parse", "--show-ref-format")); refFormat != "reftable" {
		t.Errorf("expected reftable format for references, but was %q", refFormat)
	}
	if v := getGitConfig(t, dir, biomeVersionKey); v != biomeV1 {
		t.Errorf("expected %s %q, was %q", biomeVersionKey, biomeV1, v)
	}
	if fetchParallel := getGitConfig(t, dir, "fetch.parallel"); fetchParallel != "0" {
		t.Errorf("expected parallel fetch to be enabled, but was: %q", fetchParallel)
	}
	if autoMaintenanceEnabled := getGitConfig(t, dir, "maintenance.auto"); autoMaintenanceEnabled != "false" {
		t.Errorf("expected auto maintenance to be disabled, but was not")
	}
	if maintenanceStrategy := getGitConfig(t, dir, "maintenance.strategy"); maintenanceStrategy != "incremental" {
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
