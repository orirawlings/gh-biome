package cmd

import (
	cmdutil "github.com/orirawlings/gh-biome/internal/util/command"
	"github.com/spf13/cobra"
)

// headsCmd represents the heads command
var headsCmd = &cobra.Command{
	Use:   "heads",
	Short: "List git references for the primary branch of each remote",
	Long: `Print the HEAD git reference for each remote repository that has been added to the git biome. 
	
	The HEAD git references point to the tip of the primary branch of each remote repository, 
	typically the "main" or "master" branch, though it depends on how each repository is
	maintained.
	
	Many analyses need to inspect the latest content across all repositories. Therefore, the output of
	this command can be passed as arguments to any analysis step that requires a set of git references
	to operate upon.
	
	For example, to perform a text search across the current integrated version of each actively developed repository:
		
		gh biome heads | xargs git grep -i "search term"

	`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		b, err := load(ctx)
		if err != nil {
			return err
		}

		remotes, err := b.Remotes(ctx, headsOptions.Categories()...)
		if err != nil {
			return err
		}

		for _, remote := range remotes {
			cmdutil.Println(cmd, remote.Head())
		}
		return nil
	},
}

var (
	headsOptions = newRemoteCategoryOptions(true)
)

func init() {
	rootCmd.AddCommand(headsCmd)
	headsOptions.AddFlags(headsCmd.Flags())
}
