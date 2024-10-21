package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List GitHub user(s) or organization(s) that have been added to the git biome.",
	Long: `
List GitHub repository owner(s) that have been added to the git biome. An owner
is a GitHub user or organization.
`,
	Args:    cobra.NoArgs,
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		b, err := load(ctx)
		if err != nil {
			return err
		}

		owners, err := b.Owners(ctx)
		if err != nil {
			return err
		}
		for _, owner := range owners {
			fmt.Println(owner)
		}

		return nil
	},
}
