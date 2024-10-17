package biome

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/orirawlings/gh-biome/internal/config"
	testutil "github.com/orirawlings/gh-biome/internal/util/testing"
	"gopkg.in/h2non/gock.v1"
)

func TestInit(t *testing.T) {
	ctx := context.Background()

	path := t.TempDir()

	t.Cleanup(func() {
		testutil.Execute(t, "git", "-C", path, "maintenance", "unregister")
	})

	t.Run("new biome", func(t *testing.T) {
		initBiome(t, ctx, path, true)

		testutil.Execute(t, "git", "-C", path, "fsck")
		if refFormat := strings.TrimSpace(testutil.Execute(t, "git", "-C", path, "rev-parse", "--show-ref-format")); refFormat != "reftable" {
			t.Errorf("expected reftable format for references, but was %q", refFormat)
		}
		if fetchParallel := getGitConfig(t, path, "fetch.parallel"); fetchParallel != "0" {
			t.Errorf("expected parallel fetch to be enabled, but was: %q", fetchParallel)
		}
		if autoMaintenanceEnabled := getGitConfig(t, path, "maintenance.auto"); autoMaintenanceEnabled != "false" {
			t.Errorf("expected auto maintenance to be disabled, but was not")
		}
		if maintenanceStrategy := getGitConfig(t, path, "maintenance.strategy"); maintenanceStrategy != "incremental" {
			t.Errorf("expected incremental maintenance strategy, but was: %q", maintenanceStrategy)
		}
	})

	t.Run("existing biome", func(t *testing.T) {
		// assert that Init is idempotent
		initBiome(t, ctx, path, true)
	})

	t.Run("existing repo with bad biome version", func(t *testing.T) {
		path := testutil.TempRepo(t)
		testutil.Execute(t, "git", "-C", path, "config", "set", "--local", versionKey, "foobar")
		initBiome(t, ctx, path, false)
	})
}

func getGitConfig(t *testing.T, dir, key string) string {
	return strings.TrimSpace(testutil.Execute(t, "git", "-C", dir, "config", "get", "--local", key))
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
	stubGitHub(t)
	b, err := Init(ctx, path, biomeOptions(t)...)
	if shouldSucceed && err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if !shouldSucceed && err == nil {
		t.Fatalf("expected error, but initialized successfully")
	}
	return b
}

func load(t testing.TB, ctx context.Context, path string, shouldSucceed bool) Biome {
	stubGitHub(t)
	b, err := Load(ctx, path, biomeOptions(t)...)
	if shouldSucceed && err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if !shouldSucceed && err == nil {
		t.Fatalf("expected biome to be invalid, but loaded successfully")
	}
	return b
}

func TestBiome_Owners(t *testing.T) {
	ctx := context.Background()
	b := newBiome(t, ctx)
	owners, err := b.Owners(ctx)
	testutil.Check(t, err)
	if len(owners) != 0 {
		t.Errorf("expected zero owners in biome, but was: %d", len(owners))
	}
	addOwners(t, ctx, b, "github.com/orirawlings")
	expectOwners(t, ctx, b, []Owner{
		{
			host: "github.com",
			name: "orirawlings",
		},
	})
	addOwners(t, ctx, b, "github.com/orirawlings", "github.com/kubernetes")
	expectOwners(t, ctx, b, []Owner{
		{
			host: "github.com",
			name: "kubernetes",
		},
		{
			host: "github.com",
			name: "orirawlings",
		},
	})
	addOwners(t, ctx, b, "github.com/git", "github.com/cli")
	expectOwners(t, ctx, b, []Owner{
		{
			host: "github.com",
			name: "cli",
		},
		{
			host: "github.com",
			name: "git",
		},
		{
			host: "github.com",
			name: "kubernetes",
		},
		{
			host: "github.com",
			name: "orirawlings",
		},
	})
	addOwners(t, ctx, b, "github.com/git", "github.com/cli")
	expectOwners(t, ctx, b, []Owner{
		{
			host: "github.com",
			name: "cli",
		},
		{
			host: "github.com",
			name: "git",
		},
		{
			host: "github.com",
			name: "kubernetes",
		},
		{
			host: "github.com",
			name: "orirawlings",
		},
	})
}

func newBiome(t testing.TB, ctx context.Context) Biome {
	return initBiome(t, ctx, t.TempDir(), true)
}

func biomeOptions(t testing.TB) []BiomeOption {
	_, thisFilePath, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine current source file path")
	}
	projectSourceDir := filepath.Join(filepath.Dir(thisFilePath), "../..")
	biomePath := filepath.Join(t.TempDir(), "biome")
	testutil.Execute(t, "go", "build", "-o", biomePath, projectSourceDir)
	return []BiomeOption{
		EditorOptions(config.HelperCommand(fmt.Sprintf("%s config-edit-helper", biomePath))),
	}
}

func addOwners(t *testing.T, ctx context.Context, b Biome, ownerRefs ...string) {
	var owners []Owner
	for _, ownerRef := range ownerRefs {
		owner, err := ParseOwner(ownerRef)
		testutil.Check(t, err)
		owners = append(owners, owner)
	}
	testutil.Check(t, b.AddOwners(ctx, owners))
}

func expectOwners(t *testing.T, ctx context.Context, b Biome, expected []Owner) {
	owners, err := b.Owners(ctx)
	testutil.Check(t, err)
	if !slices.Equal(owners, expected) {
		t.Errorf("unexpected owners: wanted %v, was %v", expected, owners)
	}
}

func stubGitHub(t testing.TB) {
	const testGHConfig = `
hosts:
  github.com:
    user: user1
    oauth_token: abc123
  mygithub.biz:
    user: bizuser1
    oauth_token: def456
`
	t.Helper()
	testutil.StubGHConfig(t, testGHConfig)
	t.Cleanup(gock.Off)
	// gock.Observe(gock.DumpRequest)

	for owner, id := range map[string]string{
		"orirawlings": "MDQ6VXNlcjU3MjEz",
		"kubernetes":  "MDEyOk9yZ2FuaXphdGlvbjEzNjI5NDA4",
		"git":         "MDEyOk9yZ2FuaXphdGlvbjE4MTMz",
		"cli":         "MDEyOk9yZ2FuaXphdGlvbjU5NzA0NzEx",
	} {
		gock.New("https://api.github.com").
			Post("/graphql").
			MatchHeader("Authorization", "token abc123").
			BodyString(fmt.Sprintf(`{"query":"query RepositoryOwner($login:String!){repositoryOwner(login: $owner){id}}","variables":{"login":"%s"}}`, owner)).
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
			`, id))
	}
}
