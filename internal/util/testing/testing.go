package testing

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/cli/go-gh/v2/pkg/config"
)

// Execute the given system command, ensuring it succeeded, returning the stdout.
func Execute(t testing.TB, command ...string) string {
	t.Helper()
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
	t.Helper()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ExpectError ensures that the given error is non-nil.
func ExpectError(t testing.TB, err error) {
	t.Helper()
	if err == nil {
		t.Errorf("expected error, but was nil")
	}
}

// TempRepo returns a temporary directory that has a bare git repository
// initialized. The directory will be cleaned up when the test completes.
func TempRepo(t testing.TB) string {
	t.Helper()
	path := t.TempDir()
	_ = Execute(t, "git", "-C", path, "init", "--bare")
	return path
}

// BiomeBuild compiles this project to an executable file in a temp directory
// returns the path to the executable. This allows git to call the executable
// as a GIT_EDITOR when necessary during tests.
func BiomeBuild() (string, error) {
	_, thisFilePath, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("could not determine current source file path")
	}
	projectSourceDir := filepath.Join(filepath.Dir(thisFilePath), "../../..")
	path := filepath.Join(os.TempDir(), fmt.Sprintf("biome-%d", rand.Int()))
	cmd := exec.Command("go", "build", "-o", path, projectSourceDir)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return path, fmt.Errorf("unexpected error exec'ing %q: %v\n\nstdout:\n%s\n\nstderr:\n%s", cmd.String(), err, out, exitErr.Stderr)
		} else {
			return path, fmt.Errorf("unexpected error exec'ing %q: %v", cmd.String(), err)
		}
	}
	return path, nil
}

// StubGHConfig sets a stubbed gh CLI configuration YAML string temporarily
// for the duration of the test.
func StubGHConfig(t testing.TB, cfgStr string) {
	t.Helper()
	old := config.Read
	config.Read = func(_ *config.Config) (*config.Config, error) {
		return config.ReadFromString(cfgStr), nil
	}
	t.Cleanup(func() {
		config.Read = old
	})
}
