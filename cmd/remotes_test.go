package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func init() {
	remotesCmd.SetContext(context.Background())
	pushInContext(remotesCmd)
}

func TestRemotesCmd_Execute(t *testing.T) {
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
				"github.com/cli/cli",
				"github.com/orirawlings/bar",
				"github.com/orirawlings/headless",
				"my.github.biz/foobar/bazbiz",
			},
		},
		{
			flags: []string{
				"--all",
			},
			expected: []string{
				"github.com/cli/cli",
				"github.com/orirawlings/.github",
				"github.com/orirawlings/archived",
				"github.com/orirawlings/bar",
				"github.com/orirawlings/disabled",
				"github.com/orirawlings/headless",
				"github.com/orirawlings/locked",
				"my.github.biz/foobar/bazbiz",
			},
		},
		{
			flags: []string{
				"--active",
			},
			expected: []string{
				"github.com/cli/cli",
				"github.com/orirawlings/bar",
				"github.com/orirawlings/headless",
				"my.github.biz/foobar/bazbiz",
			},
		},
		{
			flags: []string{
				"--archived",
			},
			expected: []string{
				"github.com/orirawlings/archived",
			},
		},
		{
			flags: []string{
				"--disabled",
			},
			expected: []string{
				"github.com/orirawlings/disabled",
			},
		},
		{
			flags: []string{
				"--locked",
			},
			expected: []string{
				"github.com/orirawlings/locked",
			},
		},
		{
			flags: []string{
				"--unsupported",
			},
			expected: []string{
				"github.com/orirawlings/.github",
			},
		},
		{
			flags: []string{
				"--archived",
				"--disabled",
				"--locked",
				"--unsupported",
			},
			expected: []string{
				"github.com/orirawlings/.github",
				"github.com/orirawlings/archived",
				"github.com/orirawlings/disabled",
				"github.com/orirawlings/locked",
			},
		},
		{
			flags: []string{
				"--active",
				"--disabled",
				"--locked",
				"--unsupported",
			},
			expected: []string{
				"github.com/cli/cli",
				"github.com/orirawlings/.github",
				"github.com/orirawlings/bar",
				"github.com/orirawlings/disabled",
				"github.com/orirawlings/headless",
				"github.com/orirawlings/locked",
				"my.github.biz/foobar/bazbiz",
			},
		},
		{
			flags: []string{
				"--active",
				"--archived",
				"--locked",
				"--unsupported",
			},
			expected: []string{
				"github.com/cli/cli",
				"github.com/orirawlings/.github",
				"github.com/orirawlings/archived",
				"github.com/orirawlings/bar",
				"github.com/orirawlings/headless",
				"github.com/orirawlings/locked",
				"my.github.biz/foobar/bazbiz",
			},
		},
		{
			flags: []string{
				"--active",
				"--archived",
				"--disabled",
				"--unsupported",
			},
			expected: []string{
				"github.com/cli/cli",
				"github.com/orirawlings/.github",
				"github.com/orirawlings/archived",
				"github.com/orirawlings/bar",
				"github.com/orirawlings/disabled",
				"github.com/orirawlings/headless",
				"my.github.biz/foobar/bazbiz",
			},
		},
		{
			flags: []string{
				"--active",
				"--archived",
				"--disabled",
				"--locked",
			},
			expected: []string{
				"github.com/cli/cli",
				"github.com/orirawlings/archived",
				"github.com/orirawlings/bar",
				"github.com/orirawlings/disabled",
				"github.com/orirawlings/headless",
				"github.com/orirawlings/locked",
				"my.github.biz/foobar/bazbiz",
			},
		},
		{
			flags: []string{
				"--active",
				"--archived",
				"--disabled",
				"--locked",
				"--unsupported",
			},
			expected: []string{
				"github.com/cli/cli",
				"github.com/orirawlings/.github",
				"github.com/orirawlings/archived",
				"github.com/orirawlings/bar",
				"github.com/orirawlings/disabled",
				"github.com/orirawlings/headless",
				"github.com/orirawlings/locked",
				"my.github.biz/foobar/bazbiz",
			},
		},
	} {
		t.Run(strings.Join(run.flags, " "), func(t *testing.T) {
			buf := new(bytes.Buffer)
			remotesCmd.SetOut(buf)
			t.Cleanup(func() {
				remotesCmd.SetOut(nil)
				remotesOptions.Reset()
			})
			rootCmd.SetArgs(append([]string{"remotes"}, run.flags...))
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
