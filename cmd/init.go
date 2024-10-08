package cmd

import (
	"fmt"
	"os/exec"

	"github.com/orirawlings/gh-biome/internal/config"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init [<directory>]",
	Short: "Initialize a new git-biome in the given directory.",
	Long: `
Initialize a new git-biome in the given directory.

This will initialize a new, bare git repo in the directory with configuration settings tuned for git-biome support.
Register the git repo for incremental maintenance and starts the maintenance schedule in the background.
`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		ctx := cmd.Context()

		// TODO (orirawlings): Fail gracefully if reftable is not available in the user's version of git.
		gitInitCmd := exec.CommandContext(ctx, "git", "init", "--bare", "--ref-format=reftable", path)
		if out, err := gitInitCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("could not init git repo: %q: %w\n\n%s", gitInitCmd.String(), err, out)
		}

		c := config.New(path)
		if err := c.Init(ctx); err != nil {
			return err
		}

		// fetch.parallel Specifies the maximal number of fetch operations to
		// be run in parallel at a time (submodules, or remotes when the
		// --multiple option of git-fetch(1) is in effect).
		// A value of 0 will give some reasonable default. If unset, it
		// defaults to 1.
		if err := setGitConfig(path, "fetch.parallel", "0"); err != nil {
			return err
		}

		maintenanceStartCmd := exec.Command("git", "-C", path, "maintenance", "start")
		if _, err := maintenanceStartCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("could not %q: %w", maintenanceStartCmd.String(), err)
		}

		return nil
	},
}

// setGitConfig executes git-config in the repo at the given directory to assign
// local configuration key/value pair.
func setGitConfig(dir, key, value string) error {
	cmd := exec.Command("git", "-C", dir, "config", "set", "--local", key, value)
	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("could not %q: %w", cmd.String(), err)
	}
	return nil
}
