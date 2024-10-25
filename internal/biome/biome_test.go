package biome

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/orirawlings/gh-biome/internal/config"
	testutil "github.com/orirawlings/gh-biome/internal/util/testing"
	"gopkg.in/h2non/gock.v1"
)

var (
	biomeBuildPath string
)

func TestMain(m *testing.M) {
	var err error
	biomeBuildPath, err = testutil.BiomeBuild()
	if err != nil {
		panic(err.Error())
	}
	defer os.Remove(biomeBuildPath)
	m.Run()
}

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

	github_com_orirawlings_disabled = repository{
		IsDisabled: true,
		URL:        "https://github.com/orirawlings/disabled",
		DefaultBranchRef: &ref{
			Name:   "main",
			Prefix: "refs/heads/",
		},
	}

	github_com_orirawlings_locked = repository{
		IsLocked: true,
		URL:      "https://github.com/orirawlings/locked",
		DefaultBranchRef: &ref{
			Name:   "main",
			Prefix: "refs/heads/",
		},
	}

	github_com_orirawlings_headless = repository{
		URL: "https://github.com/orirawlings/headless",
	}

	github_com_orirawlings_dotPrefix = repository{
		URL: "https://github.com/orirawlings/.github",
		DefaultBranchRef: &ref{
			Name:   "main",
			Prefix: "refs/heads/",
		},
	}

	github_com_cli_cli = repository{
		URL: "https://github.com/cli/cli",
		DefaultBranchRef: &ref{
			Name:   "trunk",
			Prefix: "refs/heads/",
		},
	}

	github_com_git_git = repository{
		URL: "https://github.com/git/git",
		DefaultBranchRef: &ref{
			Name:   "master",
			Prefix: "refs/heads/",
		},
	}

	github_com_kubernetes_kubernetes = repository{
		URL: "https://github.com/kubernetes/kubernetes",
		DefaultBranchRef: &ref{
			Name:   "master",
			Prefix: "refs/heads/",
		},
	}

	github_com_kubernetes_community = repository{
		URL: "https://github.com/kubernetes/community",
		DefaultBranchRef: &ref{
			Name:   "master",
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
		github_com_git.String(): {
			github_com_git_git,
		},
		github_com_kubernetes.String(): {
			github_com_kubernetes_kubernetes,
			github_com_kubernetes_community,
		},
		github_com_orirawlings.String(): {
			github_com_orirawlings_bar,
			github_com_orirawlings_archived,
			github_com_orirawlings_disabled,
			github_com_orirawlings_locked,
			github_com_orirawlings_headless,
			github_com_orirawlings_dotPrefix,
		},
		my_github_biz_foobar.String(): {
			my_github_biz_foobar_bazbiz,
		},
	}

	repositoriesStubs map[string]*gock.Response
)

func TestInit(t *testing.T) {
	ctx := context.Background()

	path := t.TempDir()

	t.Run("new biome", func(t *testing.T) {
		initBiome(t, ctx, path, true)

		testutil.Execute(t, "git", "-C", path, "fsck")

		expectedRefFormat := "files"
		if refFormat := strings.TrimSpace(testutil.Execute(t, "git", "-C", path, "rev-parse", "--show-ref-format")); refFormat != expectedRefFormat {
			t.Errorf("expected %q format for references, but was %q", expectedRefFormat, refFormat)
		}
		assertGitConfig(t, path, "fetch.parallel", "0")

		// assert that Init is idempotent
		initBiome(t, ctx, path, true)
	})

	t.Run("existing repo with bad biome version", func(t *testing.T) {
		path := testutil.TempRepo(t)
		testutil.Execute(t, "git", "-C", path, "config", "set", "--local", versionKey, "foobar")
		initBiome(t, ctx, path, false)
	})
}

func assertGitConfig(t *testing.T, path, key, expected string) {
	t.Helper()
	actual := getGitConfig(t, path, key)
	if actual != expected {
		t.Errorf("unexpected value for git config setting %q: wanted %q, was %q", key, expected, actual)
	}
}

func getGitConfig(t *testing.T, path, key string) string {
	return strings.TrimSpace(testutil.Execute(t, "git", "-C", path, "config", "get", "--local", key))
}

func TestLoad(t *testing.T) {

	ctx := context.Background()

	t.Run("newly initialized biome", func(t *testing.T) {
		path := t.TempDir()
		initBiome(t, ctx, path, true)
		load(t, ctx, path, true)
	})

	t.Run("repo with bad biome version", func(t *testing.T) {
		path := testutil.TempRepo(t)
		testutil.Execute(t, "git", "-C", path, "config", "set", "--local", versionKey, "foobar")
		load(t, ctx, path, false)
	})

	t.Run("non-repo", func(t *testing.T) {
		path := t.TempDir()
		load(t, ctx, path, false)
	})
}

func initBiome(t testing.TB, ctx context.Context, path string, shouldSucceed bool) Biome {
	t.Helper()
	stubGitHub(t)
	b, err := Init(ctx, path, biomeOptions()...)
	if shouldSucceed && err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if !shouldSucceed && err == nil {
		t.Fatalf("expected error, but initialized successfully")
	}
	return b
}

func load(t testing.TB, ctx context.Context, path string, shouldSucceed bool) Biome {
	stubGitHub(t)
	b, err := Load(ctx, path, biomeOptions()...)
	if shouldSucceed && err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if !shouldSucceed && err == nil {
		t.Fatalf("expected biome to be invalid, but loaded successfully")
	}
	return b
}

func biomeOptions() []BiomeOption {
	return []BiomeOption{
		EditorOptions(config.HelperCommand(fmt.Sprintf("%s config-edit-helper", biomeBuildPath))),
	}
}

func TestBiome_Owners(t *testing.T) {
	ctx := context.Background()
	path := t.TempDir()
	b := initBiome(t, ctx, path, true)

	// no owners
	o, err := b.Owners(ctx)
	testutil.Check(t, err)
	if len(o) != 0 {
		t.Errorf("expected zero owners in biome, but was: %d", len(o))
	}

	// add one owner
	addOwners(t, ctx, b, github_com_orirawlings)
	expectOwners(t, ctx, b, []Owner{
		github_com_orirawlings,
	})

	// readd owner and one other
	addOwners(t, ctx, b, github_com_orirawlings, github_com_kubernetes)
	expectOwners(t, ctx, b, []Owner{
		github_com_kubernetes,
		github_com_orirawlings,
	})

	// add two more owners
	addOwners(t, ctx, b, github_com_git, github_com_cli)
	expectOwners(t, ctx, b, []Owner{
		github_com_cli,
		github_com_git,
		github_com_kubernetes,
		github_com_orirawlings,
	})

	// readd two owners
	addOwners(t, ctx, b, github_com_git, github_com_cli)
	expectOwners(t, ctx, b, []Owner{
		github_com_cli,
		github_com_git,
		github_com_kubernetes,
		github_com_orirawlings,
	})

	// add owner from another github host
	addOwners(t, ctx, b, my_github_biz_foobar)
	expectOwners(t, ctx, b, []Owner{
		github_com_cli,
		github_com_git,
		github_com_kubernetes,
		github_com_orirawlings,
		my_github_biz_foobar,
	})

	ownerRefs := testutil.Execute(t, "git", "-C", path, "config", "get", "--all", ownersKey)
	expected := `github.com/cli
github.com/git
github.com/kubernetes
github.com/orirawlings
my.github.biz/foobar
`
	if ownerRefs != expected {
		t.Errorf("expected %q config values %q, but was %q", ownersKey, expected, ownerRefs)
	}

	// remove an owner
	removeOwners(t, ctx, b, github_com_orirawlings)
	expectOwners(t, ctx, b, []Owner{
		github_com_cli,
		github_com_git,
		github_com_kubernetes,
		my_github_biz_foobar,
	})

	// bad owner in config
	testutil.Execute(t, "git", "-C", path, "config", "set", "--value=bad/bad/bad", ownersKey, "bad/bad/bad")
	_, err = b.Owners(ctx)
	if err == nil {
		t.Error("expected error, but was nil")
	}
}

func TestBiome_UpdateRemotes(t *testing.T) {
	ctx := context.Background()
	path := t.TempDir()
	b := initBiome(t, ctx, path, true)

	// Add github.com/orirawlings
	addOwners(t, ctx, b, github_com_orirawlings)
	testutil.Check(t, b.UpdateRemotes(ctx))
	expectRemotes(t, path, []string{
		barRemote.Name,
		archivedRemote.Name,
		headlessRemote.Name,
	})
	expectActive(t, path, []string{
		barRemote.Name,
		headlessRemote.Name,
	})
	expectArchived(t, path, []string{
		archivedRemote.Name,
	})
	expectDisabled(t, path, []string{
		disabledRemote.Name,
	})
	expectLocked(t, path, []string{
		lockedRemote.Name,
	})
	expectUnsupported(t, path, []string{
		dotPrefixRemote.Name,
	})

	// Add github.com/cli, github.com/git, github.com/kubernetes, my.github.biz/foobar
	addOwners(t, ctx, b, github_com_cli, github_com_git, github_com_kubernetes, my_github_biz_foobar)
	testutil.Check(t, b.UpdateRemotes(ctx))
	expectRemotes(t, path, []string{
		"github.com/cli/cli",
		"github.com/git/git",
		"github.com/kubernetes/community",
		"github.com/kubernetes/kubernetes",
		barRemote.Name,
		archivedRemote.Name,
		headlessRemote.Name,
		"my.github.biz/foobar/bazbiz",
	})
	expectActive(t, path, []string{
		"github.com/cli/cli",
		"github.com/git/git",
		"github.com/kubernetes/community",
		"github.com/kubernetes/kubernetes",
		barRemote.Name,
		headlessRemote.Name,
		"my.github.biz/foobar/bazbiz",
	})
	expectArchived(t, path, []string{
		archivedRemote.Name,
	})
	expectDisabled(t, path, []string{
		disabledRemote.Name,
	})
	expectLocked(t, path, []string{
		lockedRemote.Name,
	})
	expectUnsupported(t, path, []string{
		dotPrefixRemote.Name,
	})

	// Remove all github.com/orirawlings repos except github.com/orirawlings/bar
	updateStubbedGitHubRepositories(t, github_com_orirawlings, []repository{
		github_com_orirawlings_bar,
	})
	testutil.Check(t, b.UpdateRemotes(ctx))
	expectRemotes(t, path, []string{
		"github.com/cli/cli",
		"github.com/git/git",
		"github.com/kubernetes/community",
		"github.com/kubernetes/kubernetes",
		barRemote.Name,
		"my.github.biz/foobar/bazbiz",
	})
	expectActive(t, path, []string{
		"github.com/cli/cli",
		"github.com/git/git",
		"github.com/kubernetes/community",
		"github.com/kubernetes/kubernetes",
		barRemote.Name,
		"my.github.biz/foobar/bazbiz",
	})
	expectArchived(t, path, nil)
	expectDisabled(t, path, nil)
	expectLocked(t, path, nil)
	expectUnsupported(t, path, nil)

	removeOwners(t, ctx, b, github_com_orirawlings, github_com_kubernetes)
	testutil.Check(t, b.UpdateRemotes(ctx))
	expectRemotes(t, path, []string{
		"github.com/cli/cli",
		"github.com/git/git",
		"my.github.biz/foobar/bazbiz",
	})
	expectActive(t, path, []string{
		"github.com/cli/cli",
		"github.com/git/git",
		"my.github.biz/foobar/bazbiz",
	})
	expectArchived(t, path, nil)
	expectDisabled(t, path, nil)
	expectLocked(t, path, nil)
	expectUnsupported(t, path, nil)
}

func addOwners(t *testing.T, ctx context.Context, b Biome, owners ...Owner) {
	t.Helper()
	testutil.Check(t, b.AddOwners(ctx, owners))
}

func removeOwners(t *testing.T, ctx context.Context, b Biome, owners ...Owner) {
	t.Helper()
	testutil.Check(t, b.RemoveOwners(ctx, owners))
}

func expectOwners(t *testing.T, ctx context.Context, b Biome, expected []Owner) {
	t.Helper()
	owners, err := b.Owners(ctx)
	testutil.Check(t, err)
	if !slices.Equal(owners, expected) {
		t.Errorf("unexpected owners: wanted %v, was %v", expected, owners)
	}
}

func expectRemotes(t *testing.T, path string, expected []string) {
	t.Helper()
	remotes := strings.Split(strings.TrimSpace(testutil.Execute(t, "git", "-C", path, "remote")), "\n")
	expected = slices.Sorted(slices.Values(expected))
	if !slices.Equal(remotes, expected) {
		t.Errorf("unexpected remotes, wanted %v, was %v:", expected, remotes)
	}
}

func expectRemotesForConfigKey(t *testing.T, path, key string, expected []string) {
	t.Helper()
	cmd := exec.Command("git", "-C", path, "config", "get", "--all", key)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.ExitCode() != 1 {
				t.Errorf("unexpected error exec'ing %q: %v\n\nstdout:\n%s\n\nstderr:\n%s\n", cmd.String(), err, out, exitErr.Stderr)
			}
		} else {
			t.Errorf("unexpected error exec'ing %q: %v", cmd.String(), err)
		}
	}
	var actual []string
	if len(out) != 0 {
		actual = strings.Split(strings.TrimSpace(string(out)), "\n")
	}
	expected = slices.Sorted(slices.Values(expected))
	if !slices.Equal(actual, expected) {
		t.Errorf("unexpected remotes for config key %q, wanted %v, was %v:", key, expected, actual)
	}
}

func expectActive(t *testing.T, path string, expected []string) {
	t.Helper()
	expectRemotesForConfigKey(t, path, activeKey, expected)
}

func expectArchived(t *testing.T, path string, expected []string) {
	t.Helper()
	expectRemotesForConfigKey(t, path, archivedKey, expected)
}

func expectDisabled(t *testing.T, path string, expected []string) {
	t.Helper()
	expectRemotesForConfigKey(t, path, disabledKey, expected)
}

func expectLocked(t *testing.T, path string, expected []string) {
	t.Helper()
	expectRemotesForConfigKey(t, path, lockedKey, expected)
}

func expectUnsupported(t *testing.T, path string, expected []string) {
	t.Helper()
	expectRemotesForConfigKey(t, path, unsupportedKey, expected)
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

func updateStubbedGitHubRepositories(t testing.TB, owner Owner, repos []repository) {
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
