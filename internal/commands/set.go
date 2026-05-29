package commands

import (
	"fmt"
	"strings"

	"github.com/dipockdas/keysync/internal/store"
	"github.com/spf13/cobra"
)

func newSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set KEY=value [--project name] [--env name]",
		Short: "Store a secret in the local OS secret store",
		Long: F(`Stores a secret in the local OS secret store (macOS Keychain, Linux libsecret,
Windows Credential Manager). This is a local-only operation — it does not push to
GitHub or deployment platforms. Use {c}keysync sync{/c} to push secrets upstream.

{b}Syntax:{/b}
  {c}keysync set KEY=VALUE{/c}        (use {c}={/c}, no space around it)
  {c}keysync set KEY=VALUE --project NAME --env ENV{/c}

If {c}--project{/c} is provided, the secret is scoped to that project.
With {c}--project{/c}, secrets default to the {c}dev{/c} environment unless you pass
{c}--env{/c} (e.g. {c}--env production{/c} for CI). Use {c}--env \"\"{/c} for project-wide
scope (no environment). Without {c}--project{/c}, secrets are stored {g}globally{/g}.

{b}Examples:{/b}
  {c}keysync set DATABASE_URL=postgres://user:pass@host/db{/c}            {g}# global{/g}
  {c}keysync set STRIPE_KEY=sk_live_xxx --project my-app{/c}              {g}# project{/g}
  {c}keysync set DB_URL=prod-url --project my-app --env production{/c}    {g}# project+env{/g}
  {c}keysync set DB_URL=staging-url --project my-app --env staging{/c}    {g}# project+env{/g}

{b}See also:{/b}
  {c}keysync sync --help{/c}
  Tutorial: {u}https://github.com/dipockdas/keysync#quick-start{/u}`),
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("requires KEY=value argument (e.g. keysync set DATABASE_URL=postgres://localhost/db)")
			}
			if len(args) > 1 {
				return fmt.Errorf("too many arguments — did you forget to quote? Use KEY=value syntax")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			kv := args[0]
			eq := strings.IndexByte(kv, '=')
			if eq < 1 {
				return fmt.Errorf("invalid format: use KEY=value")
			}
			key := kv[:eq]
			value := kv[eq+1:]

			if err := validateKeyName(key); err != nil {
				return err
			}

			scope := store.ScopeGlobal
			proj := ""
			env := ""
			if project != "" {
				scope = store.ScopeProject
				proj = project
				env = effectiveEnv
			}

			ctx := cmd.Context()

			// Write to local OS secret store
			if err := secretSt.Set(ctx, scope, proj, env, key, value); err != nil {
				return fmt.Errorf("set secret: %w", err)
			}
			fmt.Printf("Set %s/%s\n", scopeLabel(scope, proj), key)

			return nil
		},
	}
}

// validateKeyName checks that a secret key follows env-var naming conventions
// to prevent flag injection attacks when passing keys to CLI tools.
func validateKeyName(key string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	if len(key) > 256 {
		return fmt.Errorf("key too long (max 256 characters)")
	}
	for i, r := range key {
		if i == 0 && r >= '0' && r <= '9' {
			return fmt.Errorf("key cannot start with a digit: %q", key)
		}
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_') {
			return fmt.Errorf("invalid character %q in key %q (only A-Z, a-z, 0-9, and _ allowed)", r, key)
		}
	}
	return nil
}

func scopeLabel(scope store.Scope, proj string) string {
	if scope == store.ScopeGlobal || proj == "" {
		return "global"
	}
	return fmt.Sprintf("project/%s", proj)
}
