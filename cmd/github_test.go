package cmd

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/orirawlings/gh-biome/internal/biome"
	testutil "github.com/orirawlings/gh-biome/internal/util/testing"

	"gopkg.in/h2non/gock.v1"
)

var (
	github_com_cli, _ = biome.ParseOwner("github.com/cli")

	github_com_orirawlings, _ = biome.ParseOwner("github.com/orirawlings")

	my_github_biz_foobar, _ = biome.ParseOwner("my.github.biz/foobar")

	owners = []biome.Owner{
		github_com_cli,
		github_com_orirawlings,
		my_github_biz_foobar,
	}

	ownerIds = map[string]string{
		github_com_cli.String():         "MDEyOk9yZ2FuaXphdGlvbjU5NzA0NzEx",
		github_com_orirawlings.String(): "MDQ6VXNlcjU3MjEz",
		my_github_biz_foobar.String():   "foobar",
	}
)

var (
	github_com_orirawlings_bar = repository{
		URL: "https://github.com/orirawlings/bar",
		DefaultBranchRef: &ref{
			Name:   "main",
			Prefix: "refs/heads/",
		},
	}

	github_com_orirawlings_archived = repository{
		IsArchived: true,
		URL:        "https://github.com/orirawlings/archived",
		DefaultBranchRef: &ref{
			Name:   "master",
			Prefix: "refs/heads/",
		},
	}

	github_com_orirawlings_headless = repository{
		URL: "https://github.com/orirawlings/headless",
	}

	github_com_cli_cli = repository{
		URL: "https://github.com/cli/cli",
		DefaultBranchRef: &ref{
			Name:   "trunk",
			Prefix: "refs/heads/",
		},
	}

	my_github_biz_foobar_bazbiz = repository{
		URL: "https://my.github.biz/foobar/bazbiz",
		DefaultBranchRef: &ref{
			Name:   "main",
			Prefix: "refs/heads/",
		},
	}

	repositories = map[string][]repository{
		github_com_cli.String(): {
			github_com_cli_cli,
		},
		github_com_orirawlings.String(): {
			github_com_orirawlings_bar,
			github_com_orirawlings_archived,
			github_com_orirawlings_headless,
		},
		my_github_biz_foobar.String(): {
			my_github_biz_foobar_bazbiz,
		},
	}

	repositoriesStubs map[string]*gock.Response
)

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

func stubGitHub(t testing.TB) {
	const testGHConfig = `
hosts:
  github.com:
    user: user1
    oauth_token: abc123
  my.github.biz:
    user: bizuser1
    oauth_token: def456
`
	t.Helper()
	testutil.StubGHConfig(t, testGHConfig)
	t.Cleanup(gock.Off)

	// uncomment to print intercepted HTTP requests
	// gock.Observe(gock.DumpRequest)

	repositoriesStubs = make(map[string]*gock.Response)
	for _, o := range owners {
		host := o.Host()
		if host == "github.com" {
			host = "api.github.com"
		}

		gock.New(fmt.Sprintf("https://%s", host)).
			Post("/graphql").
			HeaderPresent("Authorization").
			BodyString(fmt.Sprintf(`{"query":"query Owner($owner:String!){repositoryOwner(login: $owner){id}}","variables":{"owner":%q}}`, o.Name())).
			Persist().
			Reply(200).
			JSON(fmt.Sprintf(`
				{
				  "data": {
					"repositoryOwner": {
					  "id": "%s"
					}
				  }
				}
			`, ownerIds[o.String()]))

		repositoriesStubs[o.String()] = gock.New(fmt.Sprintf("https://%s", host)).
			Post("/graphql").
			HeaderPresent("Authorization").
			BodyString(fmt.Sprintf(`{"query":"query OwnerRepositories($endCursor:String$owner:String!){repositoryOwner(login: $owner){repositories(first: 100, after: $endCursor){nodes{isDisabled,isArchived,isLocked,url,defaultBranchRef{name,prefix}},pageInfo{hasNextPage,endCursor}}}}","variables":{"endCursor":null,"owner":%q}}`, o.Name())).
			Persist().
			Reply(200)

		updateStubbedGitHubRepositories(t, o, repositories[o.String()])
	}
}

func updateStubbedGitHubRepositories(t testing.TB, owner biome.Owner, repos []repository) {
	marshalled, err := json.Marshal(repos)
	if err != nil {
		t.Fatalf("could not marshal repositories for %s in stubs: %v", owner, err)
	}

	repositoriesStubs[owner.String()].JSON(fmt.Sprintf(`
		{
		  "data": {
			"repositoryOwner": {
			  "repositories": {
				"nodes": %s,
				"pageInfo": {
				  "hasNextPage": false
				}
			  }
			}
		  }
		}
	`, string(marshalled)))
}
