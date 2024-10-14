package cmd

import (
	"fmt"

	pb "github.com/orirawlings/gh-biome/internal/config/protobuf"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func init() {
	rootCmd.AddCommand(configEditHelperCmd)
}

var configEditHelperCmd = &cobra.Command{
	Use:   "config-edit-helper <callback-unix-socket> <config-file>",
	Short: "A GIT_EDITOR implementation to use with 'git config edit'.",
	Long: `
A GIT_EDITOR implementation to use with 'git config edit'.

This command calls back to a edit server listening at the given unix socke
with the name of the git config file that needs to be edited.
`,
	Hidden: true,
	Args:   cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := grpc.NewClient(fmt.Sprintf("unix:%s", args[0]), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return err
		}
		defer conn.Close()
		c := pb.NewEditorClient(conn)
		_, err = c.Edit(cmd.Context(), &pb.EditRequest{
			Path: args[1],
		})
		return err
	},
}
