package biome

import (
	"fmt"
	"os/exec"
)

// remote represents a git remote in the biome configuration. Typically the
// remote would be stored in git config, so references and objects can be
// fetched from the remote. The biome configuration would also record metadata
// about whether the remote repository is still active and fetchable in GitHub.
type remote struct {

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

func (r remote) String() string {
	return r.Name
}

// FetchURL to retrieve references and objects from.
func (r remote) FetchURL() string {
	return fmt.Sprintf("https://%s.git", r.Name)
}

// FetchRefspec returns the refspec that should be used when fetching
// references from the remote. The refspec will sync all references under
// `refs/*` from the remote repo to `refs/remotes/<remote name>/*` in the
// local repo. The destination part of the refspec is checked with
// `git check-ref-format --refspec-pattern` to ensure it is valid.
//
// See https://git-scm.com/docs/git-check-ref-format
func (r remote) FetchRefspec() (string, error) {
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
func (r remote) Supported() bool {
	_, err := r.FetchRefspec()
	return err == nil
}

type remoteConfig struct {
	Remote remote
	Head   string
}
