package config

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
)

const (
	// biomeVersionKey is a git config key that indicates what version of biome
	// configuration settings are used in the repo.
	biomeVersionKey = "biome.version"

	// biomeV1 is the first version of biome configuration settings tha are used
	// in a repo.
	biomeV1 = "1"
)

var (
	// errBiomeVersionNotSet indicates that a git repository has not been
	// initialized as a git biome
	errBiomeVersionNotSet = errors.New("biome config version not set")
)

// AssertBiomeVersion returns true if the git repository at the given path is using the given version of biome configuration.
func AssertBiomeVersion(path, version string) (bool, error) {
	cmd := exec.Command("git", "-C", path, "config", "get", "--local", biomeVersionKey)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			// the config key is unset
			return false, nil
		}
		return false, fmt.Errorf("could not %q: %w", cmd.String(), err)
	}
	return string(bytes.TrimSpace(out)) == version, nil
}

type Config interface {
	Init() error
	Validate() error
}

type config struct {
	path string
}

func New(path string) Config {
	return &config{
		path: path,
	}
}

func (c *config) Init() error {
	switch err := c.Validate(); err {
	case nil:
		// already initialized
		return nil
	case errBiomeVersionNotSet:
		// initialize
		if err := c.set(biomeVersionKey, biomeV1); err != nil {
			return fmt.Errorf("could not initialize biome config: %w", err)
		}
		return nil
	default:
		// failed to check current biome config version
		return fmt.Errorf("could not initialize biome config: %w", err)
	}
}

func (c *config) Validate() error {
	version, err := c.get(biomeVersionKey)
	if err != nil {
		return fmt.Errorf("could not assert biome config version: %w", err)
	}
	if version == "" {
		return errBiomeVersionNotSet
	}
	if version != biomeV1 {
		return fmt.Errorf("unexpected biome config version, expected: %q was: %q", biomeV1, version)
	}
	return nil
}

func (c *config) set(key, value string) error {
	cmd := exec.Command("git", "-C", c.path, "config", "set", "--local", key, value)
	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("could not %q: %w", cmd.String(), err)
	}
	return nil
}

func (c *config) get(key string) (string, error) {
	cmd := exec.Command("git", "-C", c.path, "config", "get", "--local", key)
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
