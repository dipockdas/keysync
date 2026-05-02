package commands

import (
	"fmt"
	"os"

	"github.com/dipockdas/keysync/internal/store"
	"github.com/spf13/cobra"
)

var getUnmask bool

func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get KEY [--project name]",
		Short: "Retrieve a secret from the local OS secret store",
		Long: `Reads a secret from the local OS secret store.

Resolution order:
  1. Project-scoped secret (if --project is provided)
  2. Global secret (fallback)

Use --unmask to display the key name alongside the value (KEY=VALUE format)
for human-readable verification.

Usage:
  keysync get DATABASE_URL
  keysync get STRIPE_KEY --project my-app
  keysync get DATABASE_URL --unmask`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			ctx := cmd.Context()

			// Try project scope first
			if project != "" {
				val, err := secretSt.Get(ctx, store.ScopeProject, project, key)
				if err == nil {
					if getUnmask {
						fmt.Printf("%s=%s", key, val)
					} else {
						fmt.Print(val)
					}
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
			if getUnmask {
				fmt.Printf("%s=%s", key, val)
			} else {
				fmt.Print(val)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&getUnmask, "unmask", false, "show key=value format (for human verification)")
	return cmd
}
