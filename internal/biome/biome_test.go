package biome

import (
	"bytes"
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
		b := initBiome(t, ctx, path, true)
		if b.Path() != path {
			t.Fatalf("expected biome path %q, got %q", path, b.Path())
		}

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
		b := load(t, ctx, path, true)
		if b.Path() != path {
			t.Fatalf("expected biome path %q, got %q", path, b.Path())
		}
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
	if shouldSucceed {
		testutil.Check(t, err)
	} else {
		testutil.ExpectError(t, err)
	}
	return b
}

func load(t testing.TB, ctx context.Context, path string, shouldSucceed bool) Biome {
	stubGitHub(t)
	b, err := Load(ctx, path, biomeOptions()...)
	if shouldSucceed {
		testutil.Check(t, err)
	} else {
		testutil.ExpectError(t, err)
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

	// removing an owner should be idempotent
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
	testutil.ExpectError(t, err)
}

func TestBiome_UpdateRemotes(t *testing.T) {
	ctx := context.Background()
	path := t.TempDir()
	b := initBiome(t, ctx, path, true)

	commitID := createCommitFor(t, ctx, path, []string{
		barRemoteCfg.Head,
		archivedRemoteCfg.Head,
	})

	// Add github.com/orirawlings
	addOwners(t, ctx, b, github_com_orirawlings)
	testutil.Check(t, b.UpdateRemotes(ctx))
	expectGitRemotes(t, ctx, b, []Remote{
		barRemote,
		archivedRemote,
		headlessRemote,
	})
	expectBiomeRemotes(t, ctx, b, []Remote{
		barRemote,
		archivedRemote,
		lockedRemote,
		disabledRemote,
		headlessRemote,
		dotPrefixRemote,
	})
	expectActive(t, ctx, b, []Remote{
		barRemote,
		headlessRemote,
	})
	expectArchived(t, ctx, b, []Remote{
		archivedRemote,
	})
	expectDisabled(t, ctx, b, []Remote{
		disabledRemote,
	})
	expectLocked(t, ctx, b, []Remote{
		lockedRemote,
	})
	expectUnsupported(t, ctx, b, []Remote{
		dotPrefixRemote,
	})
	expectGitRemoteGroups(t, path, map[string][]string{
		github_com_orirawlings.RemoteGroup(): {
			barRemote.Name,
			archivedRemote.Name,
			headlessRemote.Name,
		},
	})
	expectRefs(t, ctx, path, []string{
		fmt.Sprintf(`%s commit refs/remotes/github.com/orirawlings/archived/HEAD %s`, commitID, archivedRemoteCfg.Head),
		fmt.Sprintf(`%s commit %s `, commitID, archivedRemoteCfg.Head),
		fmt.Sprintf(`%s commit refs/remotes/github.com/orirawlings/bar/HEAD %s`, commitID, barRemoteCfg.Head),
		fmt.Sprintf(`%s commit %s `, commitID, barRemoteCfg.Head),
	})

	// should be idempotent
	testutil.Check(t, b.UpdateRemotes(ctx))
	expectGitRemotes(t, ctx, b, []Remote{
		barRemote,
		archivedRemote,
		headlessRemote,
	})
	expectBiomeRemotes(t, ctx, b, []Remote{
		barRemote,
		archivedRemote,
		lockedRemote,
		disabledRemote,
		headlessRemote,
		dotPrefixRemote,
	})
	expectActive(t, ctx, b, []Remote{
		barRemote,
		headlessRemote,
	})
	expectArchived(t, ctx, b, []Remote{
		archivedRemote,
	})
	expectDisabled(t, ctx, b, []Remote{
		disabledRemote,
	})
	expectLocked(t, ctx, b, []Remote{
		lockedRemote,
	})
	expectUnsupported(t, ctx, b, []Remote{
		dotPrefixRemote,
	})
	expectGitRemoteGroups(t, path, map[string][]string{
		github_com_orirawlings.RemoteGroup(): {
			barRemote.Name,
			archivedRemote.Name,
			headlessRemote.Name,
		},
	})
	expectRefs(t, ctx, path, []string{
		fmt.Sprintf(`%s commit refs/remotes/github.com/orirawlings/archived/HEAD %s`, commitID, archivedRemoteCfg.Head),
		fmt.Sprintf(`%s commit %s `, commitID, archivedRemoteCfg.Head),
		fmt.Sprintf(`%s commit refs/remotes/github.com/orirawlings/bar/HEAD %s`, commitID, barRemoteCfg.Head),
		fmt.Sprintf(`%s commit %s `, commitID, barRemoteCfg.Head),
	})

	// Add github.com/cli, github.com/git, github.com/kubernetes, my.github.biz/foobar
	addOwners(t, ctx, b, github_com_cli, github_com_git, github_com_kubernetes, my_github_biz_foobar)
	testutil.Check(t, b.UpdateRemotes(ctx))
	expectGitRemotes(t, ctx, b, []Remote{
		githubCLICLIRemote,
		githubGitGitRemote,
		githubKubernetesCommunityRemote,
		githubKubernetesKubernetesRemote,
		barRemote,
		archivedRemote,
		headlessRemote,
		myGithubBizFoobarBazbizRemote,
	})
	expectBiomeRemotes(t, ctx, b, []Remote{
		githubCLICLIRemote,
		githubGitGitRemote,
		githubKubernetesCommunityRemote,
		githubKubernetesKubernetesRemote,
		barRemote,
		archivedRemote,
		disabledRemote,
		lockedRemote,
		headlessRemote,
		dotPrefixRemote,
		myGithubBizFoobarBazbizRemote,
	})
	expectActive(t, ctx, b, []Remote{
		githubCLICLIRemote,
		githubGitGitRemote,
		githubKubernetesCommunityRemote,
		githubKubernetesKubernetesRemote,
		barRemote,
		headlessRemote,
		myGithubBizFoobarBazbizRemote,
	})
	expectArchived(t, ctx, b, []Remote{
		archivedRemote,
	})
	expectDisabled(t, ctx, b, []Remote{
		disabledRemote,
	})
	expectLocked(t, ctx, b, []Remote{
		lockedRemote,
	})
	expectUnsupported(t, ctx, b, []Remote{
		dotPrefixRemote,
	})
	expectGitRemoteGroups(t, path, map[string][]string{
		github_com_cli.RemoteGroup(): {
			githubCLICLIRemote.Name,
		},
		github_com_git.RemoteGroup(): {
			githubGitGitRemote.Name,
		},
		github_com_kubernetes.RemoteGroup(): {
			githubKubernetesCommunityRemote.Name,
			githubKubernetesKubernetesRemote.Name,
		},
		github_com_orirawlings.RemoteGroup(): {
			barRemote.Name,
			archivedRemote.Name,
			headlessRemote.Name,
		},
		my_github_biz_foobar.RemoteGroup(): {
			myGithubBizFoobarBazbizRemote.Name,
		},
	})
	expectRefs(t, ctx, path, []string{
		fmt.Sprintf(`%s commit refs/remotes/github.com/orirawlings/archived/HEAD %s`, commitID, archivedRemoteCfg.Head),
		fmt.Sprintf(`%s commit %s `, commitID, archivedRemoteCfg.Head),
		fmt.Sprintf(`%s commit refs/remotes/github.com/orirawlings/bar/HEAD %s`, commitID, barRemoteCfg.Head),
		fmt.Sprintf(`%s commit %s `, commitID, barRemoteCfg.Head),
	})

	// Remove all github.com/orirawlings repos except github.com/orirawlings/bar
	updateStubbedGitHubRepositories(t, github_com_orirawlings, []repository{
		github_com_orirawlings_bar,
	})
	testutil.Check(t, b.UpdateRemotes(ctx))
	expectGitRemotes(t, ctx, b, []Remote{
		githubCLICLIRemote,
		githubGitGitRemote,
		githubKubernetesCommunityRemote,
		githubKubernetesKubernetesRemote,
		barRemote,
		myGithubBizFoobarBazbizRemote,
	})
	expectBiomeRemotes(t, ctx, b, []Remote{
		githubCLICLIRemote,
		githubGitGitRemote,
		githubKubernetesCommunityRemote,
		githubKubernetesKubernetesRemote,
		barRemote,
		myGithubBizFoobarBazbizRemote,
	})
	expectActive(t, ctx, b, []Remote{
		githubCLICLIRemote,
		githubGitGitRemote,
		githubKubernetesCommunityRemote,
		githubKubernetesKubernetesRemote,
		barRemote,
		myGithubBizFoobarBazbizRemote,
	})
	expectArchived(t, ctx, b, nil)
	expectDisabled(t, ctx, b, nil)
	expectLocked(t, ctx, b, nil)
	expectUnsupported(t, ctx, b, nil)
	expectGitRemoteGroups(t, path, map[string][]string{
		github_com_cli.RemoteGroup(): {
			githubCLICLIRemote.Name,
		},
		github_com_git.RemoteGroup(): {
			githubGitGitRemote.Name,
		},
		github_com_kubernetes.RemoteGroup(): {
			githubKubernetesCommunityRemote.Name,
			githubKubernetesKubernetesRemote.Name,
		},
		github_com_orirawlings.RemoteGroup(): {
			barRemote.Name,
		},
		my_github_biz_foobar.RemoteGroup(): {
			myGithubBizFoobarBazbizRemote.Name,
		},
	})
	expectRefs(t, ctx, path, []string{
		fmt.Sprintf(`%s commit refs/remotes/github.com/orirawlings/bar/HEAD %s`, commitID, barRemoteCfg.Head),
		fmt.Sprintf(`%s commit %s `, commitID, barRemoteCfg.Head),
	})

	removeOwners(t, ctx, b, github_com_orirawlings, github_com_kubernetes)
	testutil.Check(t, b.UpdateRemotes(ctx))
	expectGitRemotes(t, ctx, b, []Remote{
		githubCLICLIRemote,
		githubGitGitRemote,
		myGithubBizFoobarBazbizRemote,
	})
	expectBiomeRemotes(t, ctx, b, []Remote{
		githubCLICLIRemote,
		githubGitGitRemote,
		myGithubBizFoobarBazbizRemote,
	})
	expectActive(t, ctx, b, []Remote{
		githubCLICLIRemote,
		githubGitGitRemote,
		myGithubBizFoobarBazbizRemote,
	})
	expectArchived(t, ctx, b, nil)
	expectDisabled(t, ctx, b, nil)
	expectLocked(t, ctx, b, nil)
	expectUnsupported(t, ctx, b, nil)
	expectGitRemoteGroups(t, path, map[string][]string{
		github_com_cli.RemoteGroup(): {
			githubCLICLIRemote.Name,
		},
		github_com_git.RemoteGroup(): {
			githubGitGitRemote.Name,
		},
		my_github_biz_foobar.RemoteGroup(): {
			myGithubBizFoobarBazbizRemote.Name,
		},
	})
	expectRefs(t, ctx, path, nil)
}

func TestBiome_Remotes(t *testing.T) {
	ctx := context.Background()
	path := t.TempDir()
	b := initBiome(t, ctx, path, true)

	// querying without categories should fail
	_, err := b.Remotes(ctx)
	testutil.ExpectError(t, err)

	expectGitRemotes(t, ctx, b, nil)
	expectBiomeRemotes(t, ctx, b, nil)
	expectActive(t, ctx, b, nil)
	expectArchived(t, ctx, b, nil)
	expectDisabled(t, ctx, b, nil)
	expectLocked(t, ctx, b, nil)
	expectUnsupported(t, ctx, b, nil)

	// Adding owners shouldn't cause remotes to be updated
	addOwners(t, ctx, b, github_com_orirawlings, github_com_cli, github_com_git, github_com_kubernetes, my_github_biz_foobar)
	expectGitRemotes(t, ctx, b, nil)
	expectBiomeRemotes(t, ctx, b, nil)
	expectActive(t, ctx, b, nil)
	expectArchived(t, ctx, b, nil)
	expectDisabled(t, ctx, b, nil)
	expectLocked(t, ctx, b, nil)
	expectUnsupported(t, ctx, b, nil)

	// Updating remotes should cause remotes to be added
	testutil.Check(t, b.UpdateRemotes(ctx))
	expectGitRemotes(t, ctx, b, []Remote{
		githubCLICLIRemote,
		githubGitGitRemote,
		githubKubernetesCommunityRemote,
		githubKubernetesKubernetesRemote,
		barRemote,
		archivedRemote,
		headlessRemote,
		myGithubBizFoobarBazbizRemote,
	})
	expectBiomeRemotes(t, ctx, b, []Remote{
		githubCLICLIRemote,
		githubGitGitRemote,
		githubKubernetesCommunityRemote,
		githubKubernetesKubernetesRemote,
		barRemote,
		archivedRemote,
		disabledRemote,
		lockedRemote,
		headlessRemote,
		dotPrefixRemote,
		myGithubBizFoobarBazbizRemote,
	})
	expectActive(t, ctx, b, []Remote{
		githubCLICLIRemote,
		githubGitGitRemote,
		githubKubernetesCommunityRemote,
		githubKubernetesKubernetesRemote,
		barRemote,
		headlessRemote,
		myGithubBizFoobarBazbizRemote,
	})
	expectArchived(t, ctx, b, []Remote{
		archivedRemote,
	})
	expectDisabled(t, ctx, b, []Remote{
		disabledRemote,
	})
	expectLocked(t, ctx, b, []Remote{
		lockedRemote,
	})
	expectUnsupported(t, ctx, b, []Remote{
		dotPrefixRemote,
	})

	// querying without categories should fail
	_, err = b.Remotes(ctx)
	testutil.ExpectError(t, err)
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

func expectGitRemotes(t *testing.T, ctx context.Context, b Biome, expected []Remote) {
	t.Helper()
	slices.SortFunc(expected, func(a, b Remote) int {
		return strings.Compare(a.Name, b.Name)
	})
	var expectedNames []string
	for _, r := range expected {
		expectedNames = append(expectedNames, r.Name)
	}
	out := strings.TrimSpace(testutil.Execute(t, "git", "-C", b.Path(), "remote"))
	var gitRemotes []string
	if out != "" {
		gitRemotes = strings.Split(out, "\n")
	}
	if !slices.Equal(gitRemotes, expectedNames) {
		t.Errorf("unexpected git remotes, wanted %v, was %v:", expected, gitRemotes)
	}
	remotes, err := b.Remotes(ctx, FetchableRemoteCategories...)
	testutil.Check(t, err)
	if !slices.Equal(remotes, expected) {
		t.Errorf("unexpected biome remotes, wanted %v, was %v:", expected, remotes)
	}
}

func expectBiomeRemotes(t *testing.T, ctx context.Context, b Biome, expected []Remote) {
	t.Helper()
	slices.SortFunc(expected, func(a, b Remote) int {
		return strings.Compare(a.Name, b.Name)
	})
	remotes, err := b.Remotes(ctx, AllRemoteCategories...)
	testutil.Check(t, err)
	if !slices.Equal(remotes, expected) {
		t.Errorf("unexpected biome remotes, wanted %v, was %v:", expected, remotes)
	}
}

func expectGitRemoteGroups(t *testing.T, path string, expected map[string][]string) {
	t.Helper()
	for group, values := range expected {
		expectRemotesForConfigKey(t, path, "remotes."+group, values)
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

func expectCategory(t *testing.T, ctx context.Context, b Biome, category RemoteCategory, expected []Remote) {
	t.Helper()
	remotes, err := b.Remotes(ctx, category)
	testutil.Check(t, err)
	if !slices.Equal(remotes, expected) {
		t.Errorf("unexpected remotes for category %q, wanted %v, was %v:", category, expected, remotes)
	}
	var expectedNames []string
	for _, r := range expected {
		expectedNames = append(expectedNames, r.Name)
	}
	key := strings.Join([]string{section, remotesSubsection, string(category)}, ".")
	expectRemotesForConfigKey(t, b.Path(), key, expectedNames)
}

func expectActive(t *testing.T, ctx context.Context, b Biome, expected []Remote) {
	t.Helper()
	expectCategory(t, ctx, b, Active, expected)
}

func expectArchived(t *testing.T, ctx context.Context, b Biome, expected []Remote) {
	t.Helper()
	expectCategory(t, ctx, b, Archived, expected)
}

func expectDisabled(t *testing.T, ctx context.Context, b Biome, expected []Remote) {
	t.Helper()
	expectCategory(t, ctx, b, Disabled, expected)
}

func expectLocked(t *testing.T, ctx context.Context, b Biome, expected []Remote) {
	t.Helper()
	expectCategory(t, ctx, b, Locked, expected)
}

func expectUnsupported(t *testing.T, ctx context.Context, b Biome, expected []Remote) {
	t.Helper()
	expectCategory(t, ctx, b, Unsupported, expected)
}

func expectRefs(t *testing.T, ctx context.Context, path string, expected []string) {
	t.Helper()
	cmd := exec.CommandContext(ctx, "git", "-C", path, "for-each-ref", "--format=%(objectname) %(objecttype) %(refname) %(symref)")
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			t.Errorf("unexpected error exec'ing %q: %v\n\nstdout:\n%s\n\nstderr:\n%s\n", cmd, err, out, exitErr.Stderr)
		} else {
			t.Errorf("unexpected error exec'ing %q: %v", cmd, err)
		}
	}
	actual := strings.Split(string(out), "\n")
	if actual[len(actual)-1] == "" {
		actual = actual[:len(actual)-1]
	}
	if !slices.Equal(actual, expected) {
		t.Errorf("expected:\n%s\nwas:\n%s", strings.Join(expected, "\n"), strings.Join(actual, "\n"))
	}
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
			BodyString(fmt.Sprintf(`{"query":"query OwnerRepositories($endCursor:String$owner:String!){repositoryOwner(login: $owner){repositories(first: 100, after: $endCursor, affiliations: [OWNER]){nodes{isDisabled,isArchived,isLocked,url,defaultBranchRef{name,prefix}},pageInfo{hasNextPage,endCursor}}}}","variables":{"endCursor":null,"owner":%q}}`, o.Name())).
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

// createCommitFor the given refs, to ensure that they resolve to a real git object
func createCommitFor(t testing.TB, ctx context.Context, path string, refs []string) string {
	t.Helper()
	cmd := exec.CommandContext(ctx, "git", "-C", path, "hash-object", "-t", "commit", "-w", "--stdin")
	cmd.Stdin = bytes.NewReader([]byte(`tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904
author A <a@example.com> 0 +0000
committer C <c@example.com> 0 +0000

initial commit
`))
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			t.Errorf("unexpected error exec'ing %q: %v\n\nstdout:\n%s\n\nstderr:\n%s\n", cmd, err, out, exitErr.Stderr)
		} else {
			t.Errorf("unexpected error exec'ing %q: %v", cmd, err)
		}
	}
	commitID := string(bytes.TrimSpace(out))
	w, err := newRefUpdater(ctx, path)
	testutil.Check(t, err)
	for _, ref := range refs {
		_, err := fmt.Fprintf(w, "update %s %s\n", ref, commitID)
		testutil.Check(t, err)
	}
	testutil.Check(t, w.Close())
	return commitID
}
