package commands

import (
	"fmt"

	"github.com/dipockdas/keysync/internal/store"
	"github.com/spf13/cobra"
)

func newRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm KEY [--project name] [--env name]",
		Short: "Delete a secret from the local OS secret store",
		SilenceUsage: true,
		Long: F(`Deletes a secret from the local OS secret store (macOS Keychain, Linux libsecret,
Windows Credential Manager). This is a local-only operation — it does not remove
secrets from GitHub or deployment platforms.

{br}Tip:{/br} Use {c}keysync list{/c} to find the key name and scope before deleting.

{b}Examples:{/b}
  {c}keysync rm K1{/c}                                    {g}# delete global key{/g}
  {c}keysync rm OLLAMA_API_KEY --project agents{/c}        {g}# delete project key{/g}
  {c}keysync rm DB_URL --project my-app --env staging{/c}  {g}# delete env-scoped key{/g}

{b}See also:{/b}
  {c}keysync list --help{/c}
  {c}keysync set --help{/c}`),
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("requires a KEY to delete (e.g. keysync rm OLD_SECRET)")
			}
			if len(args) > 2 {
				return fmt.Errorf("accepts only one KEY at a time")
			}
			if len(args) == 2 && !mightBeTrailingProjectArg(args) {
				return fmt.Errorf("accepts only one KEY at a time")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			args = commandArgs(args)
			key := args[0]

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

			if err := secretSt.Delete(ctx, scope, proj, env, key); err != nil {
				if err == store.ErrNotFound {
					return fmt.Errorf("secret %s/%s not found in local store", scopeLabel(scope, proj), key)
				}
				return fmt.Errorf("delete secret: %w", err)
			}

			fmt.Printf("Deleted %s/%s\n", scopeLabel(scope, proj), key)
			return nil
		},
	}
}
