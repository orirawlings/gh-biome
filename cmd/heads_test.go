package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func init() {
	headsCmd.SetContext(context.Background())
	pushInContext(headsCmd)
}

func TestHeadsCmd_Execute(t *testing.T) {
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

	for _, run := range []struct {
		flags    []string
		expected []string
	}{
		{
			expected: []string{
				"refs/remotes/github.com/cli/cli/HEAD",
				"refs/remotes/github.com/orirawlings/bar/HEAD",
				"refs/remotes/github.com/orirawlings/headless/HEAD",
				"refs/remotes/my.github.biz/foobar/bazbiz/HEAD",
			},
		},
		{
			flags: []string{
				"--all",
			},
			expected: []string{
				"refs/remotes/github.com/cli/cli/HEAD",
				"refs/remotes/github.com/orirawlings/archived/HEAD",
				"refs/remotes/github.com/orirawlings/bar/HEAD",
				"refs/remotes/github.com/orirawlings/headless/HEAD",
				"refs/remotes/my.github.biz/foobar/bazbiz/HEAD",
			},
		},
		{
			flags: []string{
				"--active",
			},
			expected: []string{
				"refs/remotes/github.com/cli/cli/HEAD",
				"refs/remotes/github.com/orirawlings/bar/HEAD",
				"refs/remotes/github.com/orirawlings/headless/HEAD",
				"refs/remotes/my.github.biz/foobar/bazbiz/HEAD",
			},
		},
		{
			flags: []string{
				"--archived",
			},
			expected: []string{
				"refs/remotes/github.com/orirawlings/archived/HEAD",
			},
		},
		{
			flags: []string{
				"--active",
				"--archived",
			},
			expected: []string{
				"refs/remotes/github.com/cli/cli/HEAD",
				"refs/remotes/github.com/orirawlings/archived/HEAD",
				"refs/remotes/github.com/orirawlings/bar/HEAD",
				"refs/remotes/github.com/orirawlings/headless/HEAD",
				"refs/remotes/my.github.biz/foobar/bazbiz/HEAD",
			},
		},
	} {
		t.Run(strings.Join(run.flags, " "), func(t *testing.T) {
			buf := new(bytes.Buffer)
			headsCmd.SetOut(buf)
			t.Cleanup(func() {
				headsCmd.SetOut(nil)
				headsOptions.Reset()
			})
			rootCmd.SetArgs(append([]string{"heads"}, run.flags...))
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("unexpected error executing command: %v", err)
			}
			expected := strings.Join(run.expected, "\n") + "\n"
			if buf.String() != expected {
				t.Errorf("expected %q, got %q", expected, buf.String())
			}
		})
	}
}
