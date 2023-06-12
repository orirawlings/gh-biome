package cmd

import (
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
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
