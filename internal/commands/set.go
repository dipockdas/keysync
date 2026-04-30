package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/dipockdas/keysync/internal/github"
	"github.com/dipockdas/keysync/internal/store"
	"github.com/spf13/cobra"
)

func newSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set KEY=value [--project name]",
		Short: "Store a secret in the local OS secret store",
		Long: `Stores a secret in the local OS secret store (macOS Keychain, Linux libsecret, etc.).

If --project is provided, the secret is scoped to that project.
Otherwise, it is stored as a global secret (available to all projects).

Usage:
  keysync set DATABASE_URL=postgres://user:pass@host/db    # global scope
  keysync set STRIPE_KEY=sk_live_xxx --project my-app      # project scope`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kv := args[0]
			eq := strings.IndexByte(kv, '=')
			if eq < 1 {
				return fmt.Errorf("invalid format: use KEY=value")
			}
			key := kv[:eq]
			value := kv[eq+1:]

			if key == "" {
				return fmt.Errorf("key cannot be empty")
			}

			scope := store.ScopeGlobal
			proj := project
			if project != "" {
				scope = store.ScopeProject
			} else {
				proj = ""
			}

			ctx := cmd.Context()

			// Write to local OS secret store
			if err := secretSt.Set(ctx, scope, proj, key, value); err != nil {
				return fmt.Errorf("set secret: %w", err)
			}
			fmt.Printf("Set %s/%s\n", scopeLabel(scope, proj), key)

			// Also write to GitHub Secrets (best-effort)
			if err := setGithubSecret(key, value); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to write to GitHub: %v\n", err)
			}

			return nil
		},
	}
}

// setGithubSecret writes a secret to GitHub Secrets.
func setGithubSecret(key, value string) error {
	gh, err := github.NewClient(repoFlag)
	if err != nil {
		return fmt.Errorf("github client: %w", err)
	}
	return gh.Set(key, value)
}

func scopeLabel(scope store.Scope, proj string) string {
	if scope == store.ScopeGlobal || proj == "" {
		return "global"
	}
	return fmt.Sprintf("project/%s", proj)
}
