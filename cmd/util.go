package cmd

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/orirawlings/gh-biome/internal/biome"
	"github.com/spf13/cobra"
)

// validOwnerRefs ensures that command line arguments are valid owner references.
func validOwnerRefs(_ *cobra.Command, args []string) error {
	for _, owner := range args {
		if _, err := biome.ParseOwner(owner); err != nil {
			return err
		}
	}
	return nil
}

// validateOwnersPresent ensures that all owners are currently members of the biome.
func validateOwnersPresent(ctx context.Context, b biome.Biome, owners []biome.Owner) error {
	validOwners, err := b.Owners(ctx)
	if err != nil {
		// ignore
		return nil
	}
	var errs []error
	for _, owner := range owners {
		var valid bool
		for _, validOwner := range validOwners {
			if validOwner == owner {
				valid = true
				break
			}
		}
		if !valid {
			errs = append(errs, fmt.Errorf("owner was not added to the biome: %s", owner))
		}
	}
	return errors.Join(errs...)
}

// parseOwners from command line arguments.
func parseOwners(args []string) ([]biome.Owner, error) {
	var owners []biome.Owner
	var errs []error
	for _, owner := range args {
		owner, err := biome.ParseOwner(owner)
		owners = append(owners, owner)
		errs = append(errs, err)
	}
	return owners, errors.Join(errs...)
}

// fetch git remotes for the given owners (or all remotes if no owners given)
// in the git repo in the current directory.
func fetch(ctx context.Context, cmd *cobra.Command, owners []biome.Owner) error {
	fetchArgs := []string{"-C", ".", "fetch"}
	if len(owners) == 0 {
		fetchArgs = append(fetchArgs, "--all")
	} else {
		fetchArgs = append(fetchArgs, "--multiple")
		for _, owner := range owners {
			fetchArgs = append(fetchArgs, owner.RemoteGroup())
		}
	}
	c := exec.CommandContext(ctx, "git", fetchArgs...)
	c.Stdout = cmd.OutOrStdout()
	c.Stderr = cmd.ErrOrStderr()
	if err := c.Run(); err != nil {
		return fmt.Errorf("could not %q: %w", c, err)
	}
	return nil
}
