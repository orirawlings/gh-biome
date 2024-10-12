package biome

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
)

const (
	// versionKey is a git config key that indicates what version of biome
	// configuration settings are used in the repo.
	versionKey = "biome.version"

	// v1 is the first version of biome configuration schema used in a git repo.
	v1 = "1"
)

var (
	// errNotGitRepo indicates that a path is not a valid git repository
	errNotGitRepo = errors.New("directory path is not a git repository")

	// errVersionNotSet indicates that a git repository has not been
	// initialized as a git biome.
	errVersionNotSet = errors.New("biome config version not set")
)

// Biome is a local git repository that aggregates the objects and references
// of many other remote git repositories.
type Biome interface{}

type biome struct {
	path string
}

// Init initializes a new git biome at the given filesystem directory path.
func Init(ctx context.Context, path string) (Biome, error) {
	b := &biome{
		path: path,
	}
	switch err := b.validate(ctx); err {
	case nil:
		// biome already initialized
		return b, nil
	case errNotGitRepo:
		// path is not a git repo, can be initialized
	default:
		// either couldn't determine if path is a git repo or is git repo with invalid biome settings
		return nil, err
	}

	// TODO (orirawlings): Fail gracefully if reftable is not available in the user's version of git.
	gitInitCmd := exec.CommandContext(ctx, "git", "init", "--bare", "--ref-format=reftable", b.path)
	if out, err := gitInitCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("could not init git repo: %q: %w\n\n%s", gitInitCmd.String(), err, out)
	}

	// record biome schema version in git config
	if err := b.setConfig(ctx, versionKey, v1); err != nil {
		return nil, err
	}

	// fetch.parallel Specifies the maximal number of fetch operations to
	// be run in parallel at a time (submodules, or remotes when the
	// --multiple option of git-fetch(1) is in effect).
	// A value of 0 will give some reasonable default. If unset, it
	// defaults to 1.
	if err := b.setConfig(ctx, "fetch.parallel", "0"); err != nil {
		return nil, err
	}

	// start git maintenance for the repo
	maintenanceStartCmd := exec.Command("git", "-C", path, "maintenance", "start")
	if _, err := maintenanceStartCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("could not %q: %w", maintenanceStartCmd.String(), err)
	}
	return b, nil
}

// Load an existing git biome at the given filesystem directory path.
func Load(ctx context.Context, path string) (Biome, error) {
	b := &biome{
		path: path,
	}
	return b, b.validate(ctx)
}

// validate that the biome is a valid git repository and is using the expected
// biome configuration schema version.
func (b *biome) validate(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "git", "-C", b.path, "rev-parse")
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return errNotGitRepo
		}
		return err
	}
	version, err := b.getConfig(ctx, versionKey)
	if err != nil {
		return fmt.Errorf("could not assert biome config version: %w", err)
	}
	if version == "" {
		return errVersionNotSet
	}
	if version != v1 {
		return fmt.Errorf("unexpected biome config version, expected: %q was: %q", v1, version)
	}
	return nil
}

func (b *biome) setConfig(ctx context.Context, key, value string, options ...string) error {
	args := append([]string{"-C", b.path, "config", "set", "--local"}, options...)
	args = append(args, key, value)
	cmd := exec.CommandContext(ctx, "git", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("could not %q: %w: %s", cmd.String(), err, out)
	}
	return nil
}

func (b *biome) getConfig(ctx context.Context, key string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", b.path, "config", "get", "--local", key)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			// the config key is unset
			return "", nil
		}
		return "", fmt.Errorf("could not %q: %w", cmd.String(), err)
	}
	return string(bytes.TrimSpace(out)), nil
}
