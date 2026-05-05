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
		Use:   "rotate KEY [--project name] [--env name]",
		Short: "Generate a new random value for a secret and update everywhere",
		Long: F(`Generates a cryptographically random value (32 bytes, base64-encoded,
44 characters) and updates the secret everywhere: local OS secret store,
{u}GitHub Secrets{/u}, and all deployment platforms.

Use this to rotate API keys, webhook secrets, or any credential that
may have been compromised. The new value is generated using {c}crypto/rand{/c}.

If {c}--project{/c} is provided, the rotation targets a project-scoped secret.
If {c}--env{/c} is also provided, it targets a specific environment scope.

{b}Examples:{/b}
  {c}keysync rotate WEBHOOK_SECRET{/c}                           # global scope
  {c}keysync rotate STRIPE_KEY --project my-app{/c}              # project scope
  {c}keysync rotate DB_PASSWORD --project my-app --env prod{/c}  # env scope

{b}See also:{/b}
  {c}keysync set --help{/c}`),
		Args: cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			key := args[0]
			if err := validateKeyName(key); err != nil {
				return err
			}
			ctx := cobraCmd.Context()

			scope := store.ScopeGlobal
			proj := project
			env := ""
			if project != "" {
				scope = store.ScopeProject
				env = envFlag
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
			if err := secretSt.Set(ctx, scope, proj, env, key, newValue); err != nil {
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
