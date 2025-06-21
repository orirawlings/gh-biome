package cmd

import (
	cmdutil "github.com/orirawlings/gh-biome/internal/util/command"
	"github.com/spf13/cobra"
)

// remotesCmd represents the remotes command
var remotesCmd = &cobra.Command{
	Use:   "remotes",
	Short: "List remotes that have been discovered by the git biome",
	Long: `List remotes that have been discovered by the git biome. 
	
	A remote is a Git repository, owned by one of the owners that have been added to the biome. 
	Not all discovered remotes are eligible for fetching and/or pushing git data, so not all are
	configured as actual git remotes. But this command can list them, regardless.
	
	Use flag options to filter which categories of remotes to list.`,
	Args: cobra.NoArgs, // TODO (orirawlings): add support for filtering remotes by owners listed as positional arguments
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		b, err := load(ctx)
		if err != nil {
			return err
		}

		remotes, err := b.Remotes(ctx, remotesOptions.Categories()...)
		if err != nil {
			return err
		}

		for _, remote := range remotes {
			cmdutil.Println(cmd, remote)
		}
		return nil
	},
}

var (
	remotesOptions = newRemoteCategoryOptions(false)
)

func init() {
	rootCmd.AddCommand(remotesCmd)
	remotesOptions.AddFlags(remotesCmd.Flags())
}
