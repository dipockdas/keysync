package commands

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/dipockdas/keysync/internal/store"
	"github.com/spf13/cobra"
)

func newRotateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rotate KEY [--project name]",
		Short: "Generate a new random value for a secret and update everywhere",
		Long: `Generates a cryptographically random value and updates the secret in
the local OS secret store, GitHub Secrets, and all deployment platforms.

The generated value is 32 bytes encoded as base64 (44 characters).

Usage:
  keysync rotate WEBHOOK_SECRET
  keysync rotate STRIPE_KEY --project my-app`,
		Args: cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			key := args[0]
			ctx := cobraCmd.Context()

			scope := store.ScopeGlobal
			proj := project
			if project != "" {
				scope = store.ScopeProject
			} else {
				proj = ""
			}

			// Generate random value
			newValue, err := generateSecret()
			if err != nil {
				return fmt.Errorf("generate secret: %w", err)
			}
			fmt.Printf("Generated new value for %s\n", key)

			// Update local store
			if err := secretSt.Set(ctx, scope, proj, key, newValue); err != nil {
				return fmt.Errorf("set local secret: %w", err)
			}
			fmt.Printf("  ✓ local store (%s)\n", scopeLabel(scope, proj))

			// Update GitHub
			if err := setGithubSecret(key, newValue); err != nil {
				fmt.Fprintf(os.Stderr, "  ✗ github: %v\n", err)
			} else {
				fmt.Println("  ✓ github")
			}

			fmt.Println("\nRotation complete.")
			return nil
		},
	}
}

// generateSecret creates a cryptographically random 32-byte value encoded as base64.
func generateSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand.Read: %w", err)
	}
	return base64.RawStdEncoding.EncodeToString(b), nil
}
