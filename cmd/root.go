package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ubergit",
	Short: "Manage a local git repository with many disjoint commit graphs fetched from many independent remotes.",
	Long: `
Manage a local git repository with many disjoint commit graphs fetched from many
independent remotes. This allows all git objects from all the repos to be stored
in a common local git database for bulk analysis and queries.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		pushInContext(cmd)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type cmdValueKey struct{}

func pushInContext(cmd *cobra.Command) {
	cmd.SetContext(context.WithValue(cmd.Context(), cmdValueKey{}, cmd))
}

func commandFrom(ctx context.Context) *cobra.Command {
	v := ctx.Value(cmdValueKey{})
	result, ok := v.(*cobra.Command)
	if !ok {
		return nil
	}
	return result
}
