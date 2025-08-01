package biome

import (
	"fmt"
	"os/exec"
)

// Remote represents a git Remote in the biome configuration. Typically the
// Remote would be stored in git config, so references and objects can be
// fetched from the Remote. The biome configuration would also record metadata
// about whether the Remote repository is still active and fetchable in GitHub.
type Remote struct {

	// Name of the git remote in the biome repository.
	Name string

	// Archived indicates that the remote repository is archived in GitHub,
	// disabled from receiving new content.
	// https://docs.github.com/en/repositories/archiving-a-github-repository
	Archived bool

	// Disabled indicates that the remote repository is disabled in GitHub,
	// unable to be updated. This seems to be a rare and undocumented
	// condition for GitHub repositories. Disabled repositories cannot be
	// fetched.
	Disabled bool

	// Locked indicates that the remote repository is locked in GitHub,
	// disabled from any updates, usually because the repository has been
	// migrated to a different git forge. Locked repositories cannot be
	// fetched.
	// https://docs.github.com/en/migrations/overview/about-locked-repositories
	Locked bool
}

func (r Remote) String() string {
	return r.Name
}

// FetchURL to retrieve references and objects from.
func (r Remote) FetchURL() string {
	return fmt.Sprintf("https://%s.git", r.Name)
}

// FetchRefspec returns the refspec that should be used when fetching
// references from the remote. The refspec will sync all references under
// `refs/*` from the remote repo to `refs/remotes/<remote name>/*` in the
// local repo. The destination part of the refspec is checked with
// `git check-ref-format --refspec-pattern` to ensure it is valid.
//
// See https://git-scm.com/docs/git-check-ref-format
func (r Remote) FetchRefspec() (string, error) {
	src := "refs/*"
	dst := fmt.Sprintf("refs/remotes/%s/*", r.Name)
	c := exec.Command("git", "check-ref-format", "--refspec-pattern", dst)
	if err := c.Run(); err != nil {
		// TODO (orirawlings): Instead of failing here, ideally we could
		// fallback to some alternate, normalized refspec value that we know
		// will be valid. Anyone scripting around `git-for-each-ref` would need
		// to understand what that normalized format looks like, so it can be
		// handled properly. This would allow us to add-remotes even for
		// repos whose names do not make valid refspec path components. For
		// example, `.github` repos:
		// https://docs.github.com/en/communities/setting-up-your-project-for-healthy-contributions/creating-a-default-community-health-file#supported-file-types
		return "", fmt.Errorf("refspec pattern invalid: %q %w", dst, err)
	}
	return fmt.Sprintf("+%s:%s", src, dst), nil
}

// Supported returns true if this remote configuration is currently supported
// by this tool. Unsupported remotes are skipped during configuration setup.
func (r Remote) Supported() bool {
	_, err := r.FetchRefspec()
	return err == nil
}

// Head returns the HEAD reference for the remote repository. This is used to
// determine the default branch of the remote repository. The HEAD reference
// is a symbolic reference that points to the default branch of the remote
// repository, such as `refs/remotes/<remote name>/heads/main` or
// `refs/remotes/<remote name>/heads/master`.
func (r Remote) Head() string {
	return fmt.Sprintf("refs/remotes/%s/HEAD", r.Name)
}

// RemoteCategory represents the category of a remote repository in GitHub.
// Remote repositories can be categorized into one or more categories depending on
// their state in GitHub. The category is used to determine how the remote
// repository should be handled in the configuration, such as whether it can be
// fetched from, or if it should be skipped during configuration setup. RemoteCategory
// values can be used to filter queries for remotes configured in the biome.
type RemoteCategory string

const (
	// Active indicates that the remote repository is active in GitHub,
	// able to be updated and fetched from.
	// This is the default category for remotes that are not archived,
	// disabled, locked, or unsupported.
	Active RemoteCategory = "active"

	// Archived indicates that the remote repository is archived in GitHub,
	// disabled from receiving new content.
	// https://docs.github.com/en/repositories/archiving-a-github-repository
	Archived RemoteCategory = "archived"

	// Disabled indicates that the remote repository is disabled in GitHub,
	// unable to be updated. This seems to be a rare and undocumented
	// condition for GitHub repositories. Disabled repositories cannot be
	// fetched.
	Disabled RemoteCategory = "disabled"

	// Locked indicates that the remote repository is locked in GitHub,
	// disabled from any updates, usually because the repository has been
	// migrated to a different git forge. Locked repositories cannot be
	// fetched.
	// https://docs.github.com/en/migrations/overview/about-locked-repositories
	Locked RemoteCategory = "locked"

	// Unsupported indicates that the remote configuration is currently unsupported
	// by this tool. Unsupported remotes are skipped during remote configuration
	// setup, but are still recorded in the configuration for reference.
	Unsupported RemoteCategory = "unsupported"
)

var (
	// AllRemoteCategories is a list of all remote categories.
	AllRemoteCategories = []RemoteCategory{
		Active,
		Archived,
		Disabled,
		Locked,
		Unsupported,
	}

	// FetchableRemoteCategories is a list of remote categories that are
	// fetchable from GitHub.
	FetchableRemoteCategories = []RemoteCategory{
		Active,
		Archived,
	}
)

type remoteConfig struct {
	Remote Remote
	Head   string
}
