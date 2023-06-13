package cmd

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cli/go-gh/v2/pkg/api"
	graphql "github.com/cli/shurcooL-graphql"
	"github.com/go-git/go-git/v5/plumbing/format/config"
)

const (
	RemoteGroupPrefix = "ubergit-"
)

func init() {
	rootCmd.AddCommand(addRemotesEditorCmd)
}

var addRemotesEditorCmd = &cobra.Command{
	Use:    "add-remotes-editor [<github-user-or-org> ...] <git-config-file-path>",
	Short:  "A git config editor that adds many remotes, one for each repo owned by the given GitHub user or organization.",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		remotes := make(map[string]remote)
		for i, hostAndOwner := range args {
			if i == len(args)-1 {
				break
			}
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
		path := args[len(args)-1]
		configFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer configFile.Close()
		cfg := config.New()
		config.NewDecoder(configFile).Decode(cfg)
		updateConfig(cfg, remotes)

		w, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer w.Close()
		return config.NewEncoder(w).Encode(cfg)
	},
}

type remote struct {
	Name     string
	FetchURL string
	Disabled bool
}

func (r remote) Group() string {
	dgst := sha1.New()
	dgst.Write([]byte(r.Name))
	sum := dgst.Sum(nil)
	return RemoteGroupPrefix + hex.EncodeToString(sum)[:2]
}

func getRemotes(host, owner string) (map[string]remote, error) {
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
					IsLocked   bool
					IsDisabled bool
					URL        string `graphql:"url"`
					SSHURL     string `graphql:"sshUrl"`
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
	remotes := make(map[string]remote)
	for {
		if err := client.Query("RepositoryOwner", &query, variables); err != nil {
			return remotes, err
		}
		for _, node := range query.RepositoryOwner.Repositories.Nodes {
			r := remote{
				Name:     node.URL[8:],
				FetchURL: node.SSHURL,
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

func updateConfig(cfg *config.Config, remotes map[string]remote) {
	var groupOptions config.Options
	for _, r := range remotes {
		if r.Disabled {
			cfg.RemoveSubsection("remote", r.Name)
			continue
		}
		cfg.SetOption("remote", r.Name, "url", r.FetchURL)
		cfg.SetOption("remote", r.Name, "fetch", fmt.Sprintf("+refs/*:refs/remotes/%s/*", r.Name))
		cfg.SetOption("remote", r.Name, "tagOpt", "--no-tags")
		groupOptions = append(groupOptions, &config.Option{
			Key:   r.Group(),
			Value: r.Name,
		})
	}
	remoteGroups := cfg.Section("remotes")
	for _, opt := range remoteGroups.Options {
		remote, ok := remotes[opt.Value]
		if ok && strings.HasPrefix(opt.Key, RemoteGroupPrefix) {
			// We've already mapped known remotes to the proper ubergit group.
			// Drop duplicates or misassignments.
			continue
		}
		if ok && remote.Disabled {
			// Remove disabled remotes from all existing groups.
			continue
		}
		groupOptions = append(groupOptions, opt)
	}
	remoteGroups.Options = groupOptions
}
