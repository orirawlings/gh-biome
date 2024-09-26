package cmd

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

const (
	// biomeVersionKey is a git config key that indicates what version of biome
	// configuration settings are used in the repo.
	biomeVersionKey = "biome.version"

	// biomeV1 is the first version of biome configuration settings tha are used
	// in a repo.
	biomeV1 = "1"
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
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}

		// TODO (orirawlings): Fail gracefully if reftable is not available in the user's version of git.
		gitInitCmd := exec.Command("git", "init", "--bare", "--ref-format=reftable", dir)
		if out, err := gitInitCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("could not init git repo: %q: %w\n\n%s", gitInitCmd.String(), err, out)
		}

		if err := setGitConfigs(
			dir,
			biomeVersionKey, biomeV1,

			// fetch.parallel Specifies the maximal number of fetch operations
			// to be run in parallel at a time (submodules, or remotes when the
			// --multiple option of git-fetch(1) is in effect).
			// A value of 0 will give some reasonable default. If unset, it
			// defaults to 1.
			"fetch.parallel", "0",
		); err != nil {
			return err
		}

		maintenanceStartCmd := exec.Command("git", "-C", dir, "maintenance", "start")
		if _, err := maintenanceStartCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("could not %q: %w", maintenanceStartCmd.String(), err)
		}

		return nil
	},
}

// setGitConfigs sets multiple git config keys in the git repo at the given
// directory. Configurations are provided as a sequence of key and value pairs.
func setGitConfigs(dir string, keyValues ...string) error {
	if len(keyValues)%2 != 0 {
		panic("key value parameters must be specified as sequence of pairs")
	}
	for i := 0; i < len(keyValues); i += 2 {
		if err := setGitConfig(dir, keyValues[i], keyValues[i+1]); err != nil {
			return err
		}
	}
	return nil
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
