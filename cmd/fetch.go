package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(fetchCmd)
}

var fetchCmd = &cobra.Command{
	Use:   "fetch [<github-owner> ...]",
	Short: "Fetch git remotes for GitHub user(s) or organization(s) added to the git biome",
	Long: `
Update configured git remotes for all owners previously added to the biome and
fetch them. If owners are specified as arguments, only fetch remotes for those
owners. An owner is a GitHub user or organization. All git objects and
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
`,
	Example: `biome fetch

biome fetch orirawlings

biome fetch github.com/orirawlings

biome fetch https://github.com/orirawlings

biome fetch github.com/orirawlings github.com/git github.com/cli
`,
	Args: validOwnerRefs,
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
		if err := validateOwnersPresent(ctx, b, owners); err != nil {
			return err
		}

		cmd.PrintErrln("Updating git remote configurations...")

		// update git remote configurations for all owners
		if err := b.UpdateRemotes(ctx); err != nil {
			return err
		}

		// fetch remotes
		return fetch(ctx, cmd, b, owners)
	},
}
