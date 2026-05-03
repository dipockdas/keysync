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
		Long: `Prints all matching secrets as 'export KEY=VALUE' lines suitable for
shell eval or sourcing.

Global secrets are always included. If --project is provided, project-scoped
secrets are also exported and take precedence over globals with the same key.
If --env is also provided, environment-scoped secrets take highest precedence.

Usage:
  eval $(keysync export)
  eval $(keysync export --project my-app)
  eval $(keysync export --project my-app --env production)
  source <(keysync export --project my-app)`,
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
