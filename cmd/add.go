package cmd

import (
	"context"
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/orirawlings/gh-biome/internal/config"

	"github.com/cli/go-gh/v2/pkg/api"
	graphql "github.com/cli/shurcooL-graphql"
	"github.com/spf13/cobra"
)

var (
	skipFetch bool
)

func init() {
	addCmd.Flags().BoolVar(&skipFetch, "skip-fetch", false, "Do not automatically fetch git references and objects from the owners' repositories.")
	rootCmd.AddCommand(addCmd)
}

var addCmd = &cobra.Command{
	Use:   "add <github-owner> [...]",
	Short: "Add all GitHub repositories of a given user or organization to the git biome.",
	Long: `
Add all GitHub repositories of a given owner to this git biome.

An owner is a GitHub user or organization. Owners are specified with the
following format, where <host> is the GitHub server domain and <name> is the
name of the GitHub user or organziation within the server.
If <host> is omitted, "github.com" is assumed.

	[https://][<host>/]<name>

Each repository will be configured as a git remote. All git references are
fetched from the remotes and stored under refs/remotes/<remote-name>/,
including refs/remotes/<remote-name>/tags/ and refs/remotes/<remote-name>/pull/.

Examples:

	add orirawlings

	add github.com/orirawlings

	add https://github.com/orirawlings
`,
	Args: cobra.MatchAll(
		cobra.MinimumNArgs(1),
		func(cmd *cobra.Command, args []string) error {
			for _, owner := range args {
				if _, _, err := parseOwnerRef(owner); err != nil {
					return err
				}
			}
			return nil
		},
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		c := config.New(".")

		// validate current biome configuration
		if err := c.Validate(ctx); err != nil {
			return err
		}

		owners := normalizeOwners(args)

		// load repository data for all owners
		repos, err := getRepos(owners)
		if err != nil {
			return fmt.Errorf("could not load repositories: %w", err)
		}

		// record owners/users in git config if not already present
		if err := c.AddOwners(ctx, owners...); err != nil {
			return err
		}

		// update git remote configurations
		remotes := buildRemotes(ctx, repos)
		if err := c.UpdateRemotes(ctx, remotes); err != nil {
			return err
		}

		// set heads
		reason := strings.Join(append([]string{cmd.CommandPath()}, args...), " ")
		if err := c.SetHeads(ctx, reason, remotes); err != nil {
			return err
		}

		// fetch remotes
		if !skipFetch {
			panic("TODO (orirawlings): implement me")
		}

		return nil
	},
}

// parseOwnerRef identifies the GitHub host and owner name given a reference,
// typically typed in as a command line argument.
//
// GitHub owners are specified with the following format, where <host> is the
// GitHub server domain and <name> is the name of the GitHub user or organziation.
// If <host> is omitted, "github.com" is assumed.
//
//	[https://][<host>/]<name>
//
// Examples:
//
//	orirawlings
//	github.com/orirawlings
//	https://github.com/orirawlings
func parseOwnerRef(owner string) (host, name string, err error) {
	const defaultHost = "github.com"

	var protocolIncluded bool
	s, protocolIncluded := strings.CutPrefix(owner, "http://")
	s, ok := strings.CutPrefix(s, "https://")
	protocolIncluded = protocolIncluded || ok

	err = fmt.Errorf("owner reference %q invalid, valid format is [https://][<host>/]<name>", owner)

	parts := strings.SplitN(s, "/", 2)
	switch len(parts) {
	case 2:
		host, name = parts[0], parts[1]
	case 1:
		if protocolIncluded || parts[0] == "" {
			return "", "", err
		}
		name = parts[0]
	}
	if host == "" {
		host = defaultHost
	}
	return host, name, nil
}

// normalizeOwners provided as command line arguments to remove equivalent and
// invalid references. Result is sorted.
func normalizeOwners(args []string) []string {
	return slices.Sorted(maps.Keys(maps.Collect[string, any](func(yield func(string, any) bool) {
		for _, owner := range args {
			host, name, err := parseOwnerRef(owner)
			if err != nil {
				continue
			}
			if ok := yield(path.Join(host, name), nil); !ok {
				break
			}
		}
	})))
}

type repository struct {
	IsDisabled       bool
	IsArchived       bool
	IsLocked         bool
	URL              string `graphql:"url"`
	DefaultBranchRef *ref
}

type ref struct {
	Name   string
	Prefix string
}

func (r repository) Remote() config.Remote {
	remote := config.Remote{
		Name:     r.URL[8:],
		FetchURL: r.URL + ".git",
		Archived: r.IsArchived,
		Disabled: r.IsLocked || r.IsDisabled,
	}
	if r.DefaultBranchRef != nil {
		remote.Head = path.Join("refs/remotes/", remote.Name, strings.TrimPrefix(r.DefaultBranchRef.Prefix, "refs/"), r.DefaultBranchRef.Name)
	}
	return remote
}

func getRepos(owners []string) ([]repository, error) {
	var repos []repository
	for _, owner := range owners {
		parts := strings.SplitN(owner, "/", 2)
		host, name := parts[0], parts[1]

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
			"owner":     graphql.String(name),
			"endCursor": (*graphql.String)(nil),
		}
		for {
			if err := client.Query("RepositoryOwner", &query, variables); err != nil {
				return repos, fmt.Errorf("could not query repos for %s/%s: %w", host, name, err)
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

func buildRemotes(ctx context.Context, repos []repository) []config.Remote {
	var remotes []config.Remote
	for _, repo := range repos {
		r := repo.Remote()
		if r.Supported() {
			remotes = append(remotes, r)
		} else {
			cmd := commandFrom(ctx)
			fmt.Fprintf(cmd.ErrOrStderr(), "Skipping unsupported repo %s\n", r.Name)
		}
	}
	return remotes
}
