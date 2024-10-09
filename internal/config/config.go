package config

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"

	slicesutil "github.com/orirawlings/gh-biome/internal/util/slices"
)

const (
	// versionKey is a git config key that indicates what version of biome
	// configuration settings are used in the repo.
	versionKey = "biome.version"

	// v1 is the first version of biome configuration schema used in a git repo.
	v1 = "1"

	// ownersKey is a git config key that lists which GitHub repository
	// owners have been added to the biome.
	ownersKey = "biome.owners"
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

	// AddOwners records that the given GitHub owners have joined the git biome. An
	// owner should be added to the biome before any of the owner's repositories
	// are added as remotes.
	AddOwners(context.Context, ...string) error

	// Owners retrieves a listing of all owners that have been added the git biome.
	Owners(context.Context) ([]string, error)

	// UpdateRemotes ensures that all the given remotes are recorded to the biome
	// as fetchable git remotes. If a remote is disabled, it will be removed from
	// the biome. Metadata about each remote repository will be recorded as well,
	// such as the remote's archived state.
	UpdateRemotes(context.Context, []Remote) error

	// SetHeads updates the symbolic HEAD reference for each remote in the
	// biome to point to the desired target reference. The reflog updates for
	// the symbolic refs include the given reason.
	SetHeads(ctx context.Context, reason string, remotes []Remote) error
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

// AddOwners records that the given GitHub owners have joined the git biome. An
// owner should be added to the biome before any of the owner's repositories
// are added as remotes.
func (c *config) AddOwners(ctx context.Context, owners ...string) error {
	// TODO (orirawlings): Should we normalize the owner references and remove
	// equivalents here, or leave up to callers?
	owners = slicesutil.SortedUnique(owners)
	for _, owner := range owners {
		if err := c.set(ctx, ownersKey, owner, fmt.Sprintf("--value=^%s$", owner)); err != nil {
			return fmt.Errorf("could not add owner to biome config: %q: %w", owner, err)
		}
	}
	return nil
}

// Owners retrieves a listing of all owners that have been added the git biome.
func (c *config) Owners(ctx context.Context) ([]string, error) {
	owners, err := c.getAll(ctx, ownersKey)
	if err != nil {
		return nil, fmt.Errorf("could not read owners from biome config: %w", err)
	}
	slices.Sort(owners)
	return owners, nil
}

// UpdateRemotes ensures that all the given remotes are recorded to the biome
// as fetchable git remotes. If a remote is disabled, it will be removed from
// the biome. Metadata about each remote repository will be recorded as well,
// such as the remote's archived state.
func (c *config) UpdateRemotes(context.Context, []Remote) error {
	// exec git config edit --local
	// os.Setenv("GIT_EDITOR", fmt.Sprintf("gh biome add-remotes-editor %s", f.Name()))
	// gitConfigCmd := exec.Command("git", "config", "--edit")
	return errors.New("TODO (orirawlings): Implement me")
}

func (c *config) SetHeads(ctx context.Context, reason string, remotes []Remote) error {
	symbolicRefs, err := c.loadSymbolicRefs(ctx)
	if err != nil {
		return fmt.Errorf("could not load current symbolic refs on biome: %w", err)
	}
	for _, remote := range remotes {
		ref := fmt.Sprintf("refs/remotes/%s/HEAD", remote.Name)
		if remote.Head == "" && symbolicRefs[ref] != "" {
			// delete HEAD ref, it should no longer exist
			cmd := exec.Command("git", "-C", c.path, "symbolic-ref", "--delete", ref)
			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("could not %q: %w: %s", cmd.String(), err, out)
			}
		}
		if remote.Head != "" && symbolicRefs[ref] != remote.Head {
			// create or update HEAD ref
			cmd := exec.Command("git", "-C", c.path, "symbolic-ref", "-m", reason, ref, remote.Head)
			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("could not %q: %w: %s", cmd.String(), err, out)
			}
		}
	}
	return nil
}

func (c *config) loadSymbolicRefs(ctx context.Context) (map[string]string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "-C", c.path, "for-each-ref", "--format=%(refname)\t%(symref)")
	var b bytes.Buffer
	cmd.Stderr = &b
	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("could not create output pipe for %q: %w", cmd.String(), err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("could not start %q: %w", cmd.String(), err)
	}

	symbolicRefs := make(map[string]string)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 2 {
			symbolicRefs[fields[0]] = fields[1]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("could not scan output of %q: %w", cmd.String(), err)
	}

	if err := cmd.Wait(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			ee.Stderr = b.Bytes()
		}
		return nil, fmt.Errorf("could not %q: %w", cmd.String(), err)
	}
	return symbolicRefs, nil
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

func (c *config) getAll(ctx context.Context, key string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", c.path, "config", "get", "--local", "--all", key)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			// the config key is unset
			return nil, nil
		}
		return nil, fmt.Errorf("could not %q: %w", cmd.String(), err)
	}
	s := bufio.NewScanner(bytes.NewReader(out))
	return slices.Collect(func(yield func(string) bool) {
		for s.Scan() {
			if ok := yield(s.Text()); !ok {
				break
			}
		}
	}), s.Err()
}
