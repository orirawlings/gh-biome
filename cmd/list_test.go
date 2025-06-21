package cmd

import (
	"bytes"
	"context"
	"testing"
)

func init() {
	listCmd.SetContext(context.Background())
	pushInContext(listCmd)
}

func TestListCmd_Execute(t *testing.T) {
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

	expected := `github.com/cli
github.com/orirawlings
my.github.biz/foobar
`

	for _, name := range []string{
		"list",
		"ls",
	} {
		t.Run(name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			listCmd.SetOut(buf)
			t.Cleanup(func() {
				listCmd.SetOut(nil)
			})
			rootCmd.SetArgs([]string{name})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("unexpected error executing command: %v", err)
			}
			if buf.String() != expected {
				t.Errorf("expected %q, got %q", expected, buf.String())
			}
		})
	}
}
