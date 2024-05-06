package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/go-git/go-git/v5/plumbing/format/config"
)

func init() {
	rootCmd.AddCommand(addRemotesEditorCmd)
}

var addRemotesEditorCmd = &cobra.Command{
	Use:    "add-remotes-editor <remotes-data-file-path> <git-config-file-path>",
	Short:  "A git config editor that adds many remotes, given a data file that describes remotes that should be configured. This is faster than invoking `git config` or `git remote add` for many remotes individually.",
	Args:   cobra.ExactArgs(2),
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// load remote data
		var remotes remotes
		data, err := os.Open(args[0])
		if err != nil {
			return err
		}
		if err := remotes.load(data); err != nil {
			return fmt.Errorf("unable to load remotes data: %w", err)
		}

		// load current git config
		configPath := args[1]
		configFile, err := os.Open(configPath)
		if err != nil {
			return err
		}
		defer configFile.Close()
		cfg := config.New()
		config.NewDecoder(configFile).Decode(cfg)

		// merge desired remotes data with current git config
		updateConfig(cmd.Context(), cfg, remotes)

		// save git config
		w, err := os.Create(configPath)
		if err != nil {
			return err
		}
		defer w.Close()
		return config.NewEncoder(w).Encode(cfg)
	},
}

func updateConfig(ctx context.Context, cfg *config.Config, remotes map[string]remote) {
	for _, r := range remotes {
		refspec, err := r.FetchRefspec()
		if err != nil {
			cmd := commandFrom(ctx)
			fmt.Fprintf(cmd.OutOrStdout(), "Skipping configuring %s: %v", r.Name, err)
			cfg.RemoveSubsection("remote", r.Name)
			continue
		}
		if r.Disabled {
			cfg.RemoveSubsection("remote", r.Name)
			continue
		}
		cfg.SetOption("remote", r.Name, "url", r.FetchURL)
		cfg.SetOption("remote", r.Name, "fetch", refspec)
		cfg.SetOption("remote", r.Name, "archived", strconv.FormatBool(r.Archived))
		cfg.SetOption("remote", r.Name, "tagOpt", "--no-tags")
	}
}
