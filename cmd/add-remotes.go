package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(addRemotesCmd)
}

var addRemotesCmd = &cobra.Command{
	Use:   "add-remotes <github-user-or-org> [...]",
	Short: "Add all GitHub repositories of a given user or organization as separate git remotes on the current local git repository.",
	Long: `
Add all GitHub repositories of a given user or organization as separate git
remotes on the current local git repository. Remotes are added with a special
fetch refspec. All references on the remote are retrieved and stored under
refs/remotes/<remote-name>/, including refs/remotes/<remote-name>/tags/ and
refs/remotes/<remote-name>/pull/. This enables analyses of all objects reachable
from any reference on the remote.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var editor strings.Builder
		editor.WriteString("gh ubergit add-remotes-editor")
		for _, arg := range args {
			fmt.Fprintf(&editor, " %q", arg)
		}
		os.Setenv("GIT_EDITOR", editor.String())
		gitConfigCmd := exec.Command("git", "config", "--edit")
		gitConfigCmd.Stderr = os.Stderr
		gitConfigCmd.Stdout = os.Stdout
		return gitConfigCmd.Run()
	},
}
