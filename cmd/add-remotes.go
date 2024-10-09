package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(addRemotesCmd)
}

var addRemotesCmd = &cobra.Command{
	Use:   "add-remotes <github-user-or-org> [...]",
	Short: "Add all GitHub repositories of a given user or organization as separate git remotes on the current local git repository.",
	Long: `
Add all GitHub repositories of a given user or organization as separate git
remotes on the current local git repository. Remotes are added with a special
fetch refspec. All references on the remote are retrieved and stored under
refs/remotes/<remote-name>/, including refs/remotes/<remote-name>/tags/ and
refs/remotes/<remote-name>/pull/. This enables analyses of all objects reachable
from any reference on the remote.`,
	Deprecated: "Use the `add` sub-command instead",
	Args:       cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// load repos for given owners
		repos, err := getRepos(args)
		if err != nil {
			return fmt.Errorf("could not query repositories: %w", err)
		}
		remotes := newRemotes(cmd.Context(), repos)

		// persist remote data for git config editor
		f, err := os.CreateTemp("", "")
		if err != nil {
			return fmt.Errorf("cannot create temporary file for remotes data: %w", err)
		}
		defer os.Remove(f.Name()) // clean up
		if err := remotes.save(f); err != nil {
			return fmt.Errorf("cannot save remotes data for git config editor: %w", err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("cannot close remotes data file: %w", err)
		}

		// invoke git config editor to add remote configs
		os.Setenv("GIT_EDITOR", fmt.Sprintf("gh biome add-remotes-editor %s", f.Name()))
		gitConfigCmd := exec.Command("git", "config", "--edit")
		gitConfigCmd.Stderr = cmd.ErrOrStderr()
		gitConfigCmd.Stdout = cmd.OutOrStdout()
		if err := gitConfigCmd.Run(); err != nil {
			return fmt.Errorf("could not edit git config: %w", err)
		}

		// set HEADs for remote repositories
		reason := strings.Join(append([]string{cmd.CommandPath()}, args...), " ")
		return setHeads(reason, remotes)
	},
}

type remote struct {
	Name     string
	FetchURL string
	Archived bool
	Disabled bool
	Head     string
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

type remotes map[string]remote

func newRemotes(ctx context.Context, repos []repository) remotes {
	remotes := make(remotes)
	for _, repo := range repos {
		r := remote{
			Name:     repo.URL[8:],
			FetchURL: repo.URL + ".git",
			Archived: repo.IsArchived,
			Disabled: repo.IsLocked || repo.IsDisabled,
		}
		if repo.DefaultBranchRef != nil {
			r.Head = path.Join("refs/remotes/", r.Name, strings.TrimPrefix(repo.DefaultBranchRef.Prefix, "refs/"), repo.DefaultBranchRef.Name)
		}
		if r.Supported() {
			remotes[r.Name] = r
		} else {
			cmd := commandFrom(ctx)
			fmt.Fprintf(cmd.ErrOrStderr(), "Skipping unsupported repo %s\n", r.Name)
		}
	}
	return remotes
}

func (rs remotes) save(w io.Writer) error {
	var data struct {
		Remotes []remote
	}
	for _, r := range rs {
		data.Remotes = append(data.Remotes, r)
	}
	return json.NewEncoder(w).Encode(&data)
}

func (rs *remotes) load(r io.Reader) error {
	if (*rs) == nil {
		(*rs) = make(remotes)
	}
	var data struct {
		Remotes []remote
	}
	if err := json.NewDecoder(r).Decode(&data); err != nil {
		return err
	}
	for _, r := range data.Remotes {
		(*rs)[r.Name] = r
	}
	return nil
}

// setHeads will set HEADs for each remote, approximating
// `git remote set-head --auto $remote`, but adjusted for the fact that we use
// non-standard fetch refspecs.
func setHeads(reason string, remotes remotes) error {
	gitForEachRefCmd := exec.Command("git", "for-each-ref", "--format=%(refname)	%(symref)")
	gitForEachRefCmd.Stderr = os.Stderr
	r, err := gitForEachRefCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("could not create output pipe for `%s`: %w", gitForEachRefCmd, err)
	}
	if err := gitForEachRefCmd.Start(); err != nil {
		return fmt.Errorf("could not start `%s`: %w", gitForEachRefCmd, err)
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
		return fmt.Errorf("could not scan output of `%s`: %w", gitForEachRefCmd, err)
	}
	if err := gitForEachRefCmd.Wait(); err != nil {
		return fmt.Errorf("`%s` failed: %w", gitForEachRefCmd, err)
	}
	for _, remote := range remotes {
		ref := fmt.Sprintf("refs/remotes/%s/HEAD", remote.Name)
		if remote.Head == "" && symbolicRefs[ref] != "" {
			// delete HEAD ref, it should no longer exist
			cmd := exec.Command("git", "symbolic-ref", "--delete", ref)
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("could not remove HEAD ref for %s: `%s`: %w", remote.Name, cmd, err)
			}
		}
		if remote.Head != "" && symbolicRefs[ref] != remote.Head {
			// create or update HEAD ref
			cmd := exec.Command("git", "symbolic-ref", "-m", reason, ref, remote.Head)
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("could not set HEAD ref for %s: `%s`: %w", remote.Name, cmd, err)
			}
		}
	}
	return nil
}
