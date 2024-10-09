package config

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
	// errVersionNotSet indicates that a git repository has not been
	// initialized as a git biome.
	errVersionNotSet = errors.New("biome config version not set")
)

// Config provides a way to store and retrieve configuration settings for the
// git biome.
type Config interface {
	// Init initializes the git biome repository to store configuration settings
	// with the current biome configuration schema version.
	Init(context.Context) error

	// Validate that the git biome repository is using the expected biome
	// configuration schema version.
	Validate(context.Context) error
}

// config provides an implementation of Config that is backed by local git
// config settings in a git biome repository.
type config struct {
	path string
}

// New create a new Config, backed by local git config settings for the git
// biome repository at the given file path.
func New(path string) Config {
	return &config{
		path: path,
	}
}

// Init initializes the git biome repository to store configuration settings
// with the current biome configuration schema version.
func (c *config) Init(ctx context.Context) error {
	switch err := c.Validate(ctx); err {
	case nil:
		// already initialized
		return nil
	case errVersionNotSet:
		// initialize
		if err := c.set(ctx, versionKey, v1); err != nil {
			return fmt.Errorf("could not initialize biome config: %w", err)
		}
		return nil
	default:
		// failed to check current biome config version
		return fmt.Errorf("could not initialize biome config: %w", err)
	}
}

// Validate that the git biome repository is using the expected biome
// configuration schema version.
func (c *config) Validate(ctx context.Context) error {
	version, err := c.get(ctx, versionKey)
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

func (c *config) set(ctx context.Context, key, value string, options ...string) error {
	args := append([]string{"-C", c.path, "config", "set", "--local"}, options...)
	args = append(args, key, value)
	cmd := exec.CommandContext(ctx, "git", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("could not %q: %w: %s", cmd.String(), err, out)
	}
	return nil
}

func (c *config) get(ctx context.Context, key string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", c.path, "config", "get", "--local", key)
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
