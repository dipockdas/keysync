package commands

import (
	"fmt"
	"strings"

	"github.com/dipockdas/keysync/internal/store"
	"github.com/spf13/cobra"
)

func newExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export [--project name] [--env name]",
		Short: "Export secrets as shell-exportable environment variables",
		Long: F(`Prints all matching secrets as {c}export KEY=VALUE{/c} lines suitable for
shell eval or sourcing. Useful for loading secrets into your environment
without a .env file.

Global secrets are always included. If {c}--project{/c} is provided, project-scoped
secrets are also exported and take precedence over globals with the same key.
If {c}--env{/c} is also provided, environment-scoped secrets take highest precedence.

{b}Resolution order:{/b} global < project < project+env

{b}Examples:{/b}
  {c}eval $(keysync export){/c}                                    # all global secrets
  {c}eval $(keysync export --project my-app){/c}                   # project + global
  {c}eval $(keysync export --project my-app --env production){/c}  # env + project + global
  {c}source <(keysync export --project my-app){/c}                 # source directly

{b}See also:{/b}
  {c}keysync get --help{/c}
  {c}keysync list --help{/c}`),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Collect all secrets with precedence: global < project < project+env
			exported := make(map[string]string)

			// Global secrets (lowest precedence)
			globalEntries, err := secretSt.List(ctx, store.ScopeGlobal, "", "")
			if err != nil {
				return fmt.Errorf("list global secrets: %w", err)
			}
			for _, e := range globalEntries {
				val, err := secretSt.Get(ctx, e.Scope, e.Project, e.Environment, e.Key)
				if err == nil {
					exported[e.Key] = val
				}
			}

			if project != "" {
				// Project-scoped secrets (no specific environment)
				projEntries, err := secretSt.List(ctx, store.ScopeProject, project, "")
				if err != nil {
					return fmt.Errorf("list project secrets: %w", err)
				}
				for _, e := range projEntries {
					val, err := secretSt.Get(ctx, e.Scope, e.Project, e.Environment, e.Key)
					if err == nil {
						exported[e.Key] = val
					}
				}

				// Project + environment scoped secrets (highest precedence)
				if envFlag != "" {
					envEntries, err := secretSt.List(ctx, store.ScopeProject, project, envFlag)
					if err != nil {
						return fmt.Errorf("list environment secrets: %w", err)
					}
					for _, e := range envEntries {
						val, err := secretSt.Get(ctx, e.Scope, e.Project, e.Environment, e.Key)
						if err == nil {
							exported[e.Key] = val
						}
					}
				}
			}

			if len(exported) == 0 {
				return nil
			}

			// Output export statements with values single-quoted for shell safety
			for key, value := range exported {
				fmt.Printf("export %s=%s\n", key, shellQuote(value))
			}
			return nil
		},
	}
}

// shellQuote wraps a string in single quotes, escaping any embedded single quotes
// for safe use in POSIX shell eval.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
