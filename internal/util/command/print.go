package command

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Println is a utility function to print to the command's output, or standard
// output if not set.
func Println(cmd *cobra.Command, args ...any) {
	fmt.Fprintln(cmd.OutOrStdout(), args...)
}
