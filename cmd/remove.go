package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(removeCmd)
}

var removeCmd = &cobra.Command{
	Use:   "remove <github-owner> [...]",
	Short: "Remove GitHub user(s) or organization(s) from the git biome",
	Long: `
Remove the given GitHub repository owner(s) from the git biome. An owner is a
GitHub user or organization. Each repository owned by the owner will be from
the configured git remotes. The list of remote repositories for any remaining
owners will be updated.

<github-owner> is specified with the following format, where <host> is the GitHub
server name and <owner-name> is the name of the GitHub user or organziation within
the server. If <host> is omitted, "github.com" is assumed.

	[https://][<host>/]<owner-name>

Each of the owners' repositories will be removed from the git remotes.
`,
	Example: `biome remove orirawlings

biome remove github.com/orirawlings

biome remove https://github.com/orirawlings

biome remove github.com/orirawlings github.com/git github.com/cli
`,
	Aliases: []string{"rm"},
	Args: cobra.MatchAll(
		cobra.MinimumNArgs(1),
		validOwnerRefs,
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		b, err := load(ctx)
		if err != nil {
			return err
		}

		owners, err := parseOwners(args)
		if err != nil {
			return err
		}
		for _, owner := range owners {
			cmd.PrintErrf("Removing %s...\n", owner)
		}

		// remove owners/users in git config if already present
		if err := b.RemoveOwners(ctx, owners); err != nil {
			return err
		}

		// update git remote configurations for all owners
		if err := b.UpdateRemotes(ctx); err != nil {
			return err
		}

		return nil
	},
}
