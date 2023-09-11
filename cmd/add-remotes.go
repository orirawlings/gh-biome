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
		remotes := make(remotes)
		for _, hostAndOwner := range args {
			hostAndOwner, _ = strings.CutPrefix(hostAndOwner, "http://")
			hostAndOwner, _ = strings.CutPrefix(hostAndOwner, "https://")
			parts := strings.SplitN(hostAndOwner, "/", 2)
			host, owner := parts[0], parts[1]
			rs, err := getRemotes(host, owner)
			if err != nil {
				return err
			}
			for name, r := range rs {
				remotes[name] = r
			}
		}

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

func getRemotes(host, owner string) (remotes, error) {
	opts := api.ClientOptions{
		Host: host,
	}
	client, err := api.NewGraphQLClient(opts)
	if err != nil {
		return nil, err
	}
	var query struct {
		RepositoryOwner struct {
			Repositories struct {
				Nodes []struct {
					IsDisabled bool
					IsArchived bool
					IsLocked   bool
					URL        string `graphql:"url"`
				}
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
	page := 1
	remotes := make(remotes)
	for {
		if err := client.Query("RepositoryOwner", &query, variables); err != nil {
			return remotes, err
		}
		for _, node := range query.RepositoryOwner.Repositories.Nodes {
			r := remote{
				Name:     node.URL[8:],
				FetchURL: node.URL + ".git",
				Archived: node.IsArchived,
				Disabled: node.IsLocked || node.IsDisabled,
			}
			remotes[r.Name] = r
		}
		if !query.RepositoryOwner.Repositories.PageInfo.HasNextPage {
			break
		}
		variables["endCursor"] = graphql.String(query.RepositoryOwner.Repositories.PageInfo.EndCursor)
		page++
	}
	return remotes, nil
}
