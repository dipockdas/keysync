package commands

import (
	"fmt"
	"text/tabwriter"

	"github.com/dipockdas/keysync/internal/store"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list [--project name]",
		Short: "List all managed secrets",
		Long: `Lists all secrets in the local OS secret store.

If --project is provided, only secrets for that project are shown.
Global secrets are always included.

Usage:
  keysync list
  keysync list --project my-app`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Always list global secrets
			globalEntries, err := secretSt.List(ctx, store.ScopeGlobal, "")
			if err != nil {
				return fmt.Errorf("list global secrets: %w", err)
			}

			// List project secrets
			var projectEntries []store.SecretEntry
			if project != "" {
				projectEntries, err = secretSt.List(ctx, store.ScopeProject, project)
				if err != nil {
					return fmt.Errorf("list project secrets: %w", err)
				}
			}

			all := append(globalEntries, projectEntries...)
			if len(all) == 0 {
				fmt.Println("No secrets found.")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "SCOPE\tKEY")
			for _, e := range all {
				label := string(e.Scope)
				if e.Project != "" {
					label = fmt.Sprintf("%s/%s", e.Scope, e.Project)
				}
				fmt.Fprintf(w, "%s\t%s\n", label, e.Key)
			}
			w.Flush()
			return nil
		},
	}
}
