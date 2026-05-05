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
		Use:   "list [--project name] [--env name]",
		Short: "List all managed secrets",
		Long: F(`Lists all secrets in the local OS secret store.

If {c}--project{/c} is provided, only secrets for that project are shown (alongside
global secrets). If {c}--env{/c} is also provided, only secrets for that environment are shown.
Global secrets are included in all listings.

Use {c}--unmask{/c} to also display secret values (for verification purposes).

{b}Examples:{/b}
  {c}keysync list{/c}                                    # all secrets
  {c}keysync list --project my-app{/c}                   # project + global
  {c}keysync list --project my-app --env production{/c}  # env + project + global
  {c}keysync list --unmask{/c}                           # show values

{b}See also:{/b}
  {c}keysync get --help{/c}
  Tutorial: {u}https://github.com/dipockdas/keysync#quick-start{/u}`),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Always list global secrets
			globalEntries, err := secretSt.List(ctx, store.ScopeGlobal, "", "")
			if err != nil {
				return fmt.Errorf("list global secrets: %w", err)
			}

			// List project secrets
			var projectEntries []store.SecretEntry
			if project != "" {
				projectEntries, err = secretSt.List(ctx, store.ScopeProject, project, envFlag)
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
