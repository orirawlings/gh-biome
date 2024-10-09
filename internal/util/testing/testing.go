package testing

import (
	"errors"
	"os/exec"
	"testing"
)

// Execute the given system command, ensuring it succeeded, returning the stdout.
func Execute(t testing.TB, command ...string) string {
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

// Check ensures that the given error is nil.
func Check(t testing.TB, err error) {
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TempRepo returns a temporary directory that has a bare git repository
// initialized. The directory will be cleaned up when the test completes.
func TempRepo(t testing.TB) string {
	path := t.TempDir()
	_ = Execute(t, "git", "-C", path, "init", "--bare")
	return path
}
