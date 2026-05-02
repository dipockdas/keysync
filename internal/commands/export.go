package commands

import (
	"fmt"
	"strings"

	"github.com/dipockdas/keysync/internal/store"
	"github.com/spf13/cobra"
)

func newExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export [--project name]",
		Short: "Export secrets as shell-exportable environment variables",
		Long: `Prints all matching secrets as 'export KEY=VALUE' lines suitable for
shell eval or sourcing.

Global secrets are always included. If --project is provided, project-scoped
secrets are also exported and take precedence over globals with the same key.

Usage:
  eval $(keysync export)
  eval $(keysync export --project my-app)
  source <(keysync export --project my-app)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Collect global secrets
			globalEntries, err := secretSt.List(ctx, store.ScopeGlobal, "")
			if err != nil {
				return fmt.Errorf("list global secrets: %w", err)
			}

			// Collect project secrets
			var projectEntries []store.SecretEntry
			if project != "" {
				projectEntries, err = secretSt.List(ctx, store.ScopeProject, project)
				if err != nil {
					return fmt.Errorf("list project secrets: %w", err)
				}
			}

			if len(globalEntries) == 0 && len(projectEntries) == 0 {
				return nil
			}

			// Build a map: key → value, starting with globals, overlaying project-scoped
			exported := make(map[string]string, len(globalEntries)+len(projectEntries))
			for _, e := range globalEntries {
				val, err := secretSt.Get(ctx, e.Scope, e.Project, e.Key)
				if err == nil {
					exported[e.Key] = val
				}
			}
			for _, e := range projectEntries {
				val, err := secretSt.Get(ctx, e.Scope, e.Project, e.Key)
				if err == nil {
					exported[e.Key] = val // project overrides global
				}
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
