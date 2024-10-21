package cmd

import (
	"fmt"

	"github.com/orirawlings/gh-biome/internal/biome"

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
		if _, err := biome.Init(cmd.Context(), path, biomeOptions...); err != nil {
			return fmt.Errorf("failed to initialize biome: %w", err)
		}
		_, err := fmt.Fprintf(cmd.OutOrStderr(), "git biome initialized in %s\n", path)
		return err
	},
}
