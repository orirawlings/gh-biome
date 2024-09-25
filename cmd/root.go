package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "biome",
	Short: "Store many git repos fetched from independent remotes in a single local git repo.",
	Long: `
Store many git repos fetched from independent remotes in a single local git
repo, a.k.a. a "git biome". By storing all git objects and references from
many repos in a common database, we enable fast bulk analysis and querying
across all repos.

This tool helps manage the initialization, configuration, and maintenance of
the local git biome repo.`,
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
