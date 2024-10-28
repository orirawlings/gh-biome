package cmd

import (
	"context"
	"testing"
)

func init() {
	fetchCmd.SetContext(context.Background())
	pushInContext(fetchCmd)
}

func TestFetchCmd_Execute(t *testing.T) {

	setup := func(t *testing.T) {
		t.Helper()
		initBiome(t)
		stubGitHub(t)
		rootCmd.SetArgs([]string{
			"add",
			"--skip-fetch",
			github_com_cli.String(),
			github_com_orirawlings.String(),
			my_github_biz_foobar.String(),
		})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("unexpected error executing command: %v", err)
		}
	}

	t.Run("no arguments", func(t *testing.T) {
		t.Skip("TODO (orirawlings): Need to stub git fetching for remotes")
		setup(t)
		rootCmd.SetArgs([]string{
			"fetch",
		})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("unexpected error executing command: %v", err)
		}
	})

	t.Run("with arguments", func(t *testing.T) {
		t.Skip("TODO (orirawlings): Need to stub git fetching for remotes")
		setup(t)

		rootCmd.SetArgs([]string{
			"fetch",
			github_com_orirawlings.String(),
		})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("unexpected error executing command: %v", err)
		}
	})
}
