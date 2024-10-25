package biome

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path"
	"slices"
	"strings"

	"github.com/orirawlings/gh-biome/internal/config"
	slicesutil "github.com/orirawlings/gh-biome/internal/util/slices"

	"github.com/cli/go-gh/v2/pkg/api"
	graphql "github.com/cli/shurcooL-graphql"
)

const (
	// section is a git config section that holds top-level biome configuration
	// settings.
	section = "biome"

	// versionOpt is a git config section option key that indicates which
	// version of biome configuration settings are used in the repo.
	versionOpt = "version"

	// versionKey is a git config key that indicates which version of biome
	// configuration settings are used in the repo.
	versionKey = section + "." + versionOpt

	// v1 is the first version of biome configuration schema used in a git repo.
	v1 = "1"

	// ownersOpt is a git config section option key for listing GitHub
	// repository owners that have been added to the biome.
	ownersOpt = "owners"

	// ownersKey is a git config key that lists which GitHub repository
	// owners that have been added to the biome.
	ownersKey = section + "." + ownersOpt

	// remotesSubsection is a git config subsection for storing metadata about
	// remote repositories that are added to the biome.
	remotesSubsection = "remotes"

	// activeOpt is a git config option key that lists GitHub remote
	// repositories that are active, meaning the remote repository:
	//
	// - is not archived
	// - is not locked
	// - is not disabled
	// - is otherwise supported by biome
	activeOpt = "active"

	// activeKey is a git config key that lists GitHub remote
	// repositories that are active, meaning the remote repository:
	//
	// - is not archived
	// - is not locked
	// - is not disabled
	// - is otherwise supported by biome
	activeKey = section + "." + remotesSubsection + "." + activeOpt

	// archivedOpt is a git config option key which lists GitHub remote
	// repositories that are archived.
	archivedOpt = "archived"

	// archivedKey is a git config key which lists GitHub remote repositories
	// that are archived.
	archivedKey = section + "." + remotesSubsection + "." + archivedOpt

	// disabledOpt is a git config option key which lists GitHub remote
	// repositories that are disabled.
	disabledOpt = "disabled"

	// disabledKey is a git config key which lists GitHub remote repositories
	// that are disabled.
	disabledKey = section + "." + remotesSubsection + "." + disabledOpt

	// lockedOpt is a git config option key which lists GitHub remote
	// repositories that are locked.
	lockedOpt = "locked"

	// lockedKey is a git config key which lists GitHub remote repositories
	// that are locked.
	lockedKey = section + "." + remotesSubsection + "." + lockedOpt

	// unsupportedOpt is a git config option key which lists GitHub remote
	// repositories that are not currently supported by biome.
	unsupportedOpt = "unsupported"

	// unsupportedKey is a git config key which lists GitHub remote repositories
	// that are not currently supported by biome.
	unsupportedKey = section + "." + remotesSubsection + "." + unsupportedOpt
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
	// AddOwners records that the given GitHub repository owners have joined the
	// git biome. An owner should be added to the biome before any of the owner's
	// repositories can be added as remotes.
	AddOwners(context.Context, []Owner) error

	// RemoveOwners removes the given GitHub repository owners from the records
	// on the git biome. If any given owners were not previously added, an
	// error will be returned. All remotes for a removed owner's repositories
	// will be removed in the next [UpdateRemotes] invocation.
	RemoveOwners(context.Context, []Owner) error

	// Owners lists the GitHub repository owners that are currently within the
	// biome.
	Owners(context.Context) ([]Owner, error)

	// UpdateRemotes syncs the git remote configurations. All repositories
	// owned by the biome's owners will be configured as remotes. Any other
	// remotes will be dropped. HEAD references for each remote will be updated
	// as well.
	UpdateRemotes(context.Context) error
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

	// TODO (orirawlings): Explore using reftable and fail gracefully if reftable is not available
	// in the user's version of git. reftable would likely be much faster for bulk and concurrent
	// reads of references, but it does not support concurrent writes. `git fetch --multiple` and
	// `git fetch --all` perform potentially concurrent writes and does not appear to busy-spin
	// with backoff when making ref updates. This is a blocker for parallel fetching for biomes.
	//
	// See https://git-scm.com/docs/reftable#_update_transactions
	//
	// cmd := exec.CommandContext(ctx, "git", "init", "--bare", "--ref-format=reftable", b.path)
	cmd := exec.CommandContext(ctx, "git", "init", "--bare", b.path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("could not %q: %w\n%s", cmd, err, out)
	}

	switch err := b.validate(ctx); err {
	case nil:
		// biome already initialized
		return b, nil
	case errVersionNotSet:
		// git repo has never been initialized as a biome
	default:
		// either path is not a git repo or is git repo with invalid biome settings
		return nil, err
	}

	return b, b.editConfig(ctx, func(ctx context.Context, c *config.Config) (bool, error) {
		c.SetOption(section, "", versionOpt, v1)

		// fetch.parallel Specifies the maximal number of fetch operations to
		// be run in parallel at a time (submodules, or remotes when the
		// --multiple option of git-fetch(1) is in effect).
		// A value of 0 will give some reasonable default. If unset, it
		// defaults to 1.
		c.SetOption("fetch", "", "parallel", "0")

		return true, nil
	})
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

// AddOwners records that the given GitHub repository owners have joined the
// git biome. An owner should be added to the biome before any of the owner's
// repositories can be added as remotes.
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

// RemoveOwners removes the given GitHub repository owners from the records
// on the git biome. If any given owners were not previously added, an
// error will be returned. All remotes for a removed owner's repositories
// will be removed in the next [UpdateRemotes] invocation.
func (b *biome) RemoveOwners(ctx context.Context, owners []Owner) error {
	return b.editConfig(ctx, func(ctx context.Context, cfg *config.Config) (bool, error) {
		biomeSection := cfg.Section(section)

		var ownerRefs []string
		for _, ownerRef := range biomeSection.OptionAll(ownersOpt) {
			if !slices.ContainsFunc(owners, func(owner Owner) bool { return owner.String() == ownerRef }) {
				ownerRefs = append(ownerRefs, ownerRef)
			}
		}

		// clear stored owners
		biomeSection.RemoveOption(ownersOpt)

		// store remaining owners
		for _, ownerRef := range ownerRefs {
			biomeSection.AddOption(ownersOpt, ownerRef)
		}

		return true, nil
	})
}

// Owners lists the GitHub repository owners that are currently within the
// biome.
func (b *biome) Owners(ctx context.Context) ([]Owner, error) {
	var owners []Owner
	err := b.editConfig(ctx, func(ctx context.Context, cfg *config.Config) (bool, error) {
		var err error
		owners, err = b.getOwners(cfg)
		return false, err
	})
	return owners, err
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
		Host: owner.Host(),
	})
	if err != nil {
		return fmt.Errorf("could not create API client: %s: %w", owner.Host(), err)
	}
	var query struct {
		RepositoryOwner struct {
			Id string
		} `graphql:"repositoryOwner(login: $owner)"`
	}
	variables := map[string]interface{}{
		"owner": graphql.String(owner.name),
	}
	return client.QueryWithContext(ctx, "Owner", &query, variables)
}

func (b *biome) getOwners(cfg *config.Config) ([]Owner, error) {
	var owners []Owner
	var errs error
	for _, ownerRef := range cfg.Section(section).OptionAll(ownersOpt) {
		owner, err := ParseOwner(ownerRef)
		if err != nil {
			errs = errors.Join(errs, err)
		} else {
			owners = append(owners, owner)
		}
	}
	return owners, errs
}

// UpdateRemotes syncs the git remote configurations. All repositories
// owned by the biome's owners will be configured as remotes. Any other
// remotes will be dropped. HEAD references for each remote will be updated
// as well.
func (b *biome) UpdateRemotes(ctx context.Context) error {
	remotesToCleanUp := make(map[string]struct{})
	var addedRemotes []remote

	if err := b.editConfig(ctx, func(ctx context.Context, cfg *config.Config) (bool, error) {
		owners, err := b.getOwners(cfg)
		if err != nil {
			return false, fmt.Errorf("could not load repository owners: %w", err)
		}

		gitRemoteSection := cfg.Section("remote")

		for _, ss := range gitRemoteSection.Subsections {
			remotesToCleanUp[ss.Name] = struct{}{}
		}

		// clear existing remote declarations
		gitRemoteSection.Subsections = nil

		// clear metadata about remotes
		biomeRemotesSubsection := cfg.Section(section).Subsection(remotesSubsection)
		biomeRemotesSubsection.
			RemoveOption(activeOpt).
			RemoveOption(archivedOpt).
			RemoveOption(disabledOpt).
			RemoveOption(lockedOpt).
			RemoveOption(unsupportedOpt)

		for _, owner := range owners {
			remotes, err := b.buildRemotes(ctx, owner)
			if err != nil {
				return false, err
			}
			for _, r := range remotes {
				if r.Disabled {
					biomeRemotesSubsection.AddOption(disabledOpt, r.Name)
					continue
				}
				if r.Locked {
					biomeRemotesSubsection.AddOption(lockedOpt, r.Name)
					continue
				}
				refspec, err := r.FetchRefspec()
				if err != nil {
					// TODO (orirawlings): Handle this sensibly. Log that remote is not supported?
					biomeRemotesSubsection.AddOption(unsupportedOpt, r.Name)
					continue
				}

				if r.Archived {
					biomeRemotesSubsection.AddOption(archivedOpt, r.Name)
				} else {
					biomeRemotesSubsection.AddOption(activeOpt, r.Name)
				}

				// Add remote
				delete(remotesToCleanUp, r.Name)
				addedRemotes = append(addedRemotes, r)
				gitRemoteSection.Subsection(r.Name).SetOption("url", r.FetchURL)
				gitRemoteSection.Subsection(r.Name).SetOption("fetch", refspec)
				gitRemoteSection.Subsection(r.Name).SetOption("tagOpt", "--no-tags")
			}
		}
		return true, nil
	}); err != nil {
		return fmt.Errorf("could not update remote configurations: %w", err)
	}

	if err := b.setHeads(ctx, addedRemotes); err != nil {
		return fmt.Errorf("could not set HEAD references for remotes: %w", err)
	}

	if err := b.cleanUpRemotes(ctx, remotesToCleanUp); err != nil {
		return fmt.Errorf("could not clean up old remotes: %w", err)
	}

	return nil
}

func (b *biome) setHeads(ctx context.Context, remotes []remote) error {
	w, err := b.updateRefs(ctx)
	if err != nil {
		return err
	}

	for _, r := range remotes {
		head := fmt.Sprintf("refs/remotes/%s/HEAD", r.Name)
		if r.Head == "" {
			if _, err := fmt.Fprintf(w, "option no-deref\nsymref-delete %s\n", head); err != nil {
				return fmt.Errorf("could not delete HEAD ref for %s: %w", r.Name, err)
			}
		} else {
			if _, err := fmt.Fprintf(w, "option no-deref\nsymref-update %s %s\n", head, r.Head); err != nil {
				return fmt.Errorf("could not update HEAD ref for %s: %w", r.Name, err)
			}
		}
	}

	return w.Close()
}

func (b *biome) cleanUpRemotes(ctx context.Context, remotesToCleanUp map[string]struct{}) error {
	w, err := b.updateRefs(ctx)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	args := []string{
		"-C",
		b.path,
		"for-each-ref",
		"--format=%(if)%(symref)%(then)option no-deref\nsymref-delete %(refname)%(else)delete %(refname)%(end)",
	}
	for remote := range remotesToCleanUp {
		args = append(args, fmt.Sprintf("refs/remotes/%s", remote))
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stdout = w
	cmd.Stderr = &buf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not %q: %w: %s", cmd.String(), err, buf.String())
	}

	return w.Close()
}

func (b *biome) updateRefs(ctx context.Context) (io.WriteCloser, error) {
	return newRefUpdater(ctx, b.path)
}

func (b *biome) editConfig(ctx context.Context, do func(context.Context, *config.Config) (bool, error)) error {
	return config.NewEditor(b.path, b.editorOptions...).Edit(ctx, do)
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

func (b *biome) buildRemotes(ctx context.Context, owner Owner) ([]remote, error) {
	client, err := api.NewGraphQLClient(api.ClientOptions{
		Host: owner.Host(),
	})
	if err != nil {
		return nil, fmt.Errorf("could not create API client: %s: %w", owner.Host(), err)
	}
	var query struct {
		RepositoryOwner struct {
			Repositories struct {
				Nodes    []repository
				PageInfo struct {
					HasNextPage bool
					EndCursor   string
				}
			} `graphql:"repositories(first: 100, after: $endCursor)"`
		} `graphql:"repositoryOwner(login: $owner)"`
	}
	variables := map[string]interface{}{
		"owner":     graphql.String(owner.name),
		"endCursor": (*graphql.String)(nil),
	}
	var remotes []remote
	for {
		if err := client.QueryWithContext(ctx, "OwnerRepositories", &query, variables); err != nil {
			return remotes, fmt.Errorf("could not query repos for %s: %w", owner, err)
		}
		for _, repo := range query.RepositoryOwner.Repositories.Nodes {
			remotes = append(remotes, repo.Remote())
		}
		if !query.RepositoryOwner.Repositories.PageInfo.HasNextPage {
			break
		}
		variables["endCursor"] = graphql.String(query.RepositoryOwner.Repositories.PageInfo.EndCursor)
	}
	slices.SortFunc(remotes, func(a, b remote) int {
		return strings.Compare(a.Name, b.Name)
	})
	return remotes, nil
}

type BiomeOption func(*biome)

// EditorOptions overrides the options to use when provisioning a
// `git config edit` helper.
func EditorOptions(opts ...config.EditorOption) BiomeOption {
	return func(b *biome) {
		b.editorOptions = opts
	}
}

type ref struct {
	Name   string
	Prefix string
}

type repository struct {
	IsDisabled       bool
	IsArchived       bool
	IsLocked         bool
	URL              string `graphql:"url" json:"url"`
	DefaultBranchRef *ref
}

func (r repository) Remote() remote {
	remote := remote{
		Name:     r.URL[8:],
		FetchURL: r.URL + ".git",
		Archived: r.IsArchived,
		Disabled: r.IsDisabled,
		Locked:   r.IsLocked,
	}
	if r.DefaultBranchRef != nil {
		remote.Head = path.Join("refs/remotes/", remote.Name, strings.TrimPrefix(r.DefaultBranchRef.Prefix, "refs/"), r.DefaultBranchRef.Name)
	}
	return remote
}

type refUpdater struct {
	cmd    *exec.Cmd
	w      io.WriteCloser
	out    bytes.Buffer
	cancel func()
}

var _ io.WriteCloser = &refUpdater{}

func newRefUpdater(ctx context.Context, path string) (r *refUpdater, err error) {
	r = &refUpdater{}
	ctx, r.cancel = context.WithCancel(ctx)

	r.cmd = exec.CommandContext(ctx, "git", "-C", path, "update-ref", "--stdin")
	r.w, err = r.cmd.StdinPipe()
	if err != nil {
		return r, fmt.Errorf("could not create stdin pipe for %q: %w", r.cmd, err)
	}
	r.cmd.Stdout = &r.out
	r.cmd.Stderr = &r.out
	if err := r.cmd.Start(); err != nil {
		return r, fmt.Errorf("could not start %q: %w", r.cmd, err)
	}

	// start transaction
	if _, err := fmt.Fprintln(r.w, "start"); err != nil {
		defer r.cancel()
		return r, fmt.Errorf("could not start transaction in %q session: %w", r.cmd, err)
	}

	return
}

func (r *refUpdater) Write(p []byte) (int, error) {
	return r.w.Write(p)
}

func (r *refUpdater) Close() error {
	defer r.cancel()

	// end transaction
	if _, err := fmt.Fprintln(r.w, "prepare\ncommit"); err != nil {
		defer r.cancel()
		return fmt.Errorf("could not end transaction in %q session: %w", r.cmd, err)
	}
	if err := r.w.Close(); err != nil {
		return fmt.Errorf("could not close stdin for %q: %w", r.cmd, err)
	}

	if err := r.cmd.Wait(); err != nil {
		return fmt.Errorf("could not %q: %w: %s", r.cmd, err, r.out.String())
	}

	return nil
}
