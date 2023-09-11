package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	graphql "github.com/cli/shurcooL-graphql"
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
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// load repos for given owners
		repos, err := getRepos(args)
		if err != nil {
			return fmt.Errorf("could not query repositories: %w", err)
		}

		// persist remote data for git config editor
		f, err := os.CreateTemp("", "")
		if err != nil {
			return fmt.Errorf("cannot create temporary file for remotes data: %w", err)
		}
		defer os.Remove(f.Name()) // clean up
		if err := newRemotes(repos).save(f); err != nil {
			return fmt.Errorf("cannot save remotes data for git config editor: %w", err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("cannot close remotes data file: %w", err)
		}

		// invoke git config editor to add remote configs
		os.Setenv("GIT_EDITOR", fmt.Sprintf("gh ubergit add-remotes-editor %s", f.Name()))
		gitConfigCmd := exec.Command("git", "config", "--edit")
		gitConfigCmd.Stderr = os.Stderr
		gitConfigCmd.Stdout = os.Stdout
		return gitConfigCmd.Run()
	},
}

type remote struct {
	Name     string
	FetchURL string
	Archived bool
	Disabled bool
}

type remotes map[string]remote

func newRemotes(repos []repository) remotes {
	remotes := make(remotes)
	for _, repo := range repos {
		r := remote{
			Name:     repo.URL[8:],
			FetchURL: repo.URL + ".git",
			Archived: repo.IsArchived,
			Disabled: repo.IsLocked || repo.IsDisabled,
		}
		remotes[r.Name] = r
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

type repository struct {
	IsDisabled bool
	IsArchived bool
	IsLocked   bool
	URL        string `graphql:"url"`
}

func getRepos(args []string) ([]repository, error) {
	var repos []repository
	for _, hostAndOwner := range args {
		// parse host and owner
		hostAndOwner, _ = strings.CutPrefix(hostAndOwner, "http://")
		hostAndOwner, _ = strings.CutPrefix(hostAndOwner, "https://")
		parts := strings.SplitN(hostAndOwner, "/", 2)
		host, owner := parts[0], parts[1]

		// query API
		opts := api.ClientOptions{
			Host: host,
		}
		client, err := api.NewGraphQLClient(opts)
		if err != nil {
			return nil, fmt.Errorf("could not create API client: %s: %w", host, err)
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
			"owner":     graphql.String(owner),
			"endCursor": (*graphql.String)(nil),
		}
		for {
			if err := client.Query("RepositoryOwner", &query, variables); err != nil {
				return repos, fmt.Errorf("could not query repos for %s/%s: %w", host, owner, err)
			}
			repos = append(repos, query.RepositoryOwner.Repositories.Nodes...)
			if !query.RepositoryOwner.Repositories.PageInfo.HasNextPage {
				break
			}
			variables["endCursor"] = graphql.String(query.RepositoryOwner.Repositories.PageInfo.EndCursor)
		}
	}
	return repos, nil
}
