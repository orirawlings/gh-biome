package biome

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"slices"

	"github.com/cli/go-gh/v2/pkg/api"
	graphql "github.com/cli/shurcooL-graphql"
	"github.com/orirawlings/gh-biome/internal/config"
	slicesutil "github.com/orirawlings/gh-biome/internal/util/slices"
)

const (
	// section is the name of the git config section that holds top-level biome
	// configuration settings.
	section = "biome"

	// versionKey is a git config key that indicates what version of biome
	// configuration settings are used in the repo.
	versionKey = section + ".version"

	// v1 is the first version of biome configuration schema used in a git repo.
	v1 = "1"

	// ownersOpt is the git config section option key for listing GitHub
	// repository owners that have been added to the biome.
	ownersOpt = "owners"

	// ownersKey is a git config key that lists which GitHub repository
	// owners that have been added to the biome.
	ownersKey = section + "." + ownersOpt
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
type Biome interface {
	AddOwners(context.Context, []Owner) error
	Owners(context.Context) ([]Owner, error)
}

type biome struct {
	path          string
	editorOptions []config.EditorOption
}

// Init initializes a new git biome at the given filesystem directory path.
func Init(ctx context.Context, path string, opts ...BiomeOption) (Biome, error) {
	b := &biome{
		path: path,
	}
	for _, opt := range opts {
		opt(b)
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
func Load(ctx context.Context, path string, opts ...BiomeOption) (Biome, error) {
	b := &biome{
		path: path,
	}
	for _, opt := range opts {
		opt(b)
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

// AddOwners records that the given GitHub owners have joined the git biome. An
// owner should be added to the biome before any of the owner's repositories
// are added as remotes.
func (b *biome) AddOwners(ctx context.Context, owners []Owner) error {
	if err := b.validateOwners(ctx, owners); err != nil {
		return err
	}
	return b.editConfig(ctx, func(ctx context.Context, cfg *config.Config) (bool, error) {
		biomeSection := cfg.Section(section)

		ownerRefs := slicesutil.SortedUnique(slices.Collect(func(yield func(string) bool) {
			// stored owners
			slices.Values(biomeSection.OptionAll(ownersOpt))(yield)

			// new owners
			for _, owner := range owners {
				if !yield(owner.String()) {
					return
				}
			}
		}))

		// clear stored owners
		biomeSection.RemoveOption(ownersOpt)

		// store all owners
		for _, ownerRef := range ownerRefs {
			biomeSection.AddOption(ownersOpt, ownerRef)
		}

		return true, nil
	})
}

func (b *biome) Owners(ctx context.Context) ([]Owner, error) {
	ownerRefs, err := b.getAllConfig(ctx, ownersKey)
	if err != nil {
		return nil, fmt.Errorf("could not read owners from biome config: %w", err)
	}
	var owners []Owner
	var errs error
	for _, ownerRef := range ownerRefs {
		owner, err := ParseOwner(ownerRef)
		if err != nil {
			errs = errors.Join(errs, err)
		} else {
			owners = append(owners, owner)
		}
	}
	return owners, errs
}

func (b *biome) validateOwners(ctx context.Context, owners []Owner) error {
	var errs []error
	for _, owner := range owners {
		if err := b.validateOwner(ctx, owner); err != nil {
			errs = append(errs, fmt.Errorf("could not validate owner: %s: %w", owner, err))
		}
	}
	return errors.Join(errs...)
}

func (b *biome) validateOwner(ctx context.Context, owner Owner) error {
	client, err := api.NewGraphQLClient(api.ClientOptions{
		Host: owner.host,
	})
	if err != nil {
		return fmt.Errorf("could not create API client: %s: %w", owner.host, err)
	}
	var query struct {
		RepositoryOwner struct {
			Id string
		} `graphql:"repositoryOwner(login: $owner)"`
	}
	variables := map[string]interface{}{
		"login": graphql.String(owner.name),
	}
	return client.QueryWithContext(ctx, "RepositoryOwner", &query, variables)
}

func (b *biome) editConfig(ctx context.Context, do func(context.Context, *config.Config) (bool, error)) error {
	return config.NewEditor(b.path, b.editorOptions...).Edit(ctx, do)
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

func (b *biome) getAllConfig(ctx context.Context, key string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", b.path, "config", "get", "--local", "--all", key)
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

type BiomeOption func(*biome)

func EditorOptions(opts ...config.EditorOption) BiomeOption {
	return func(b *biome) {
		b.editorOptions = opts
	}
}
