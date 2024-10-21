package cmd

import (
	"context"
	"testing"
)

func init() {
	addCmd.SetContext(context.Background())
	pushInContext(addCmd)
}

func TestAddCmd_Execute(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		t.Skip("TODO (orirawlings): Need to stub git fetching for remotes")
		initBiome(t)
		stubGitHub(t)
		rootCmd.SetArgs([]string{
			"add",
			github_com_cli.String(),
			github_com_orirawlings.String(),
			my_github_biz_foobar.String(),
		})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("unexpected error executing command: %v", err)
		}

	})

	t.Run("--skip-fetch", func(t *testing.T) {
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
	})
}
