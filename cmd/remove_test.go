package cmd

import (
	"context"
	"testing"
)

func init() {
	removeCmd.SetContext(context.Background())
	pushInContext(removeCmd)
}

func TestRemoveCmd_Execute(t *testing.T) {

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

	t.Run("remove", func(t *testing.T) {
		setup(t)
		rootCmd.SetArgs([]string{"remove", github_com_orirawlings.String()})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("unexpected error executing command: %v", err)
		}
	})

	t.Run("rm", func(t *testing.T) {
		setup(t)
		rootCmd.SetArgs([]string{"rm", github_com_orirawlings.String()})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("unexpected error executing command: %v", err)
		}
	})
}
