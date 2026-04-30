package commands

import (
	"fmt"
	"os"

	"github.com/dipockdas/keysync/internal/store"
	"github.com/spf13/cobra"
)

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get KEY [--project name]",
		Short: "Retrieve a secret from the local OS secret store",
		Long: `Reads a secret from the local OS secret store.

Resolution order:
  1. Project-scoped secret (if --project is provided)
  2. Global secret (fallback)

Usage:
  keysync get DATABASE_URL
  keysync get STRIPE_KEY --project my-app`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			ctx := cmd.Context()

			// Try project scope first
			if project != "" {
				val, err := secretSt.Get(ctx, store.ScopeProject, project, key)
				if err == nil {
					fmt.Print(val)
					return nil
				}
				if err != store.ErrNotFound {
					return fmt.Errorf("get secret: %w", err)
				}
				// Fall through to global
			}

			// Try global scope
			val, err := secretSt.Get(ctx, store.ScopeGlobal, "", key)
			if err == store.ErrNotFound {
				fmt.Fprintf(os.Stderr, "secret not found: %s\n", key)
				os.Exit(1)
			}
			if err != nil {
				return fmt.Errorf("get secret: %w", err)
			}
			fmt.Print(val)
			return nil
		},
	}
}
