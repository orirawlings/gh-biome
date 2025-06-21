package cmd

import (
	"github.com/spf13/cobra"
)

var (
	skipFetch bool
)

func init() {
	addCmd.Flags().BoolVar(&skipFetch, "skip-fetch", false, "Do not automatically fetch git references and objects from the owners' repositories.")
	rootCmd.AddCommand(addCmd)
}

var addCmd = &cobra.Command{
	Use:   "add <github-owner> [...]",
	Short: "Add GitHub user(s) or organization(s) to the git biome",
	Long: `
Add the given GitHub repository owner(s) to the git biome. An owner is a GitHub
user or organization. Each repository owned by the owner will be recorded and
added as a git remote. If an owner was previously added to the biome, the list
of remote repositories for that owner will be updated. All git objects and
references will be fetched from the owners' remotes.

<github-owner> is specified with the following format, where <host> is the GitHub
server name and <owner-name> is the name of the GitHub user or organziation within
the server. If <host> is omitted, "github.com" is assumed.

	[https://][<host>/]<owner-name>

Each of the owners' repositories will be configured as a git remote. All git
references are fetched from the remotes and stored under
refs/remotes/<remote-name>/, including refs/remotes/<remote-name>/tags/ and
refs/remotes/<remote-name>/pull/

<remote-name> uses the following format, based on the normalized specification
of the owner.

	<host>/<owner-name>/<repo-name>

Run 'git remote' to show a listing of all remotes added to the biome.
`,
	Example: `biome add orirawlings

biome add github.com/orirawlings

biome add https://github.com/orirawlings

biome add github.com/orirawlings github.com/git github.com/cli
`,
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
			cmd.PrintErrf("Adding %s...\n", owner)
		}

		// record owners in git config if not already present
		if err := b.AddOwners(ctx, owners); err != nil {
			return err
		}

		// update git remote configurations for all owners
		if err := b.UpdateRemotes(ctx); err != nil {
			return err
		}

		// fetch remotes
		if !skipFetch {
			if err := fetch(ctx, cmd, b, owners); err != nil {
				return err
			}
		}

		return nil
	},
}
