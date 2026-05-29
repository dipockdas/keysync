package commands

import (
	"fmt"
	"text/tabwriter"

	"github.com/dipockdas/keysync/internal/store"
	"github.com/spf13/cobra"
)

var listUnmask bool

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list [--project name] [--env name]",
		Aliases: []string{"ls"},
		Short: "List all managed secrets",
		Long: F(`Lists all secrets in the local OS secret store.

Without {c}--project{/c}, all secrets across every project and scope are shown.
With {c}--project{/c}, only global secrets and secrets for that project are shown.
With {c}--project{/c}, {c}--env{/c} defaults to {c}dev{/c} (project-wide + dev keys). Use {c}--env production{/c} for CI secrets.

Use {c}--unmask{/c} to also display secret values (for verification purposes).

{b}Examples:{/b}
  {c}keysync list{/c}  (alias: {c}ls{/c})                 # all secrets (every project)
  {c}keysync list --project my-app{/c}                   # project + global only
  {c}keysync list --project my-app --env production{/c}  # specific env
  {c}keysync list --unmask{/c}                           # show values

{b}See also:{/b}
  {c}keysync get --help{/c}
  Tutorial: {u}https://github.com/dipockdas/keysync#quick-start{/u}`),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			var all []store.SecretEntry
			if project != "" {
				// Show project-scoped + global secrets for the specified project
				projectEntries, err := collectPushEntries(ctx, project, effectiveEnv, nil)
				if err != nil {
					return fmt.Errorf("list project secrets: %w", err)
				}
				globalEntries, err := secretSt.List(ctx, store.ScopeGlobal, "", "")
				if err != nil {
					return fmt.Errorf("list global secrets: %w", err)
				}
				all = append(globalEntries, projectEntries...)
			} else {
				// No --project: list ALL secrets across every project and scope
				var err error
				all, err = secretSt.List(ctx, "", "", "")
				if err != nil {
					return fmt.Errorf("list secrets: %w", err)
				}
			}

			if len(all) == 0 {
				fmt.Println("No secrets found.")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			if listUnmask {
				fmt.Fprintln(w, "SCOPE\tKEY\tVALUE")
				for _, e := range all {
					label := scopeDisplayLabel(e)
					val, err := secretSt.Get(ctx, e.Scope, e.Project, e.Environment, e.Key)
					displayVal := "***"
					if err == nil {
						displayVal = val
					}
					fmt.Fprintf(w, "%s\t%s\t%s\n", label, e.Key, displayVal)
				}
			} else {
				fmt.Fprintln(w, "SCOPE\tKEY")
				for _, e := range all {
					label := scopeDisplayLabel(e)
					fmt.Fprintf(w, "%s\t%s\n", label, e.Key)
				}
			}
			w.Flush()
			return nil
		},
	}

	cmd.Flags().BoolVar(&listUnmask, "unmask", false, "show secret values alongside key names")
	return cmd
}

// scopeDisplayLabel returns a human-readable scope label for a SecretEntry.
func scopeDisplayLabel(e store.SecretEntry) string {
	label := string(e.Scope)
	if e.Project != "" {
		label = fmt.Sprintf("%s/%s", e.Scope, e.Project)
	}
	if e.Environment != "" {
		label = fmt.Sprintf("%s/%s", label, e.Environment)
	}
	return label
}
