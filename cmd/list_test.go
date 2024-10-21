package cmd

import (
	"context"
	"testing"
)

func init() {
	listCmd.SetContext(context.Background())
	pushInContext(listCmd)
}

func TestListCmd_Execute(t *testing.T) {
	initBiome(t)

	t.Run("list", func(t *testing.T) {
		rootCmd.SetArgs([]string{"list"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("unexpected error executing command: %v", err)
		}
	})

	t.Run("ls", func(t *testing.T) {
		rootCmd.SetArgs([]string{"ls"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("unexpected error executing command: %v", err)
		}
	})
}
