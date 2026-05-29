package commands

import (
	"fmt"
	"os"

	"github.com/dipockdas/keysync/internal/github"
	"github.com/dipockdas/keysync/internal/store"
	"github.com/spf13/cobra"
)

func newPullCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pull [--project name] [--env name]",
		Short: "Reconcile local secrets with GitHub Secret names",
		Long: F(`Lists all secrets in GitHub and checks whether they exist in the local
OS secret store. Secrets that exist on GitHub but are missing locally are reported.

Note: GitHub's API does not expose secret values. This command reconciles
secret names only. To populate missing secrets, use {c}keysync set KEY=value{/c}.

If {c}--project{/c} is provided, the check includes project and environment scopes.
If {c}--env{/c} is also provided, environment-scoped secrets are checked too.

{b}Examples:{/b}
  {c}keysync pull{/c}
  {c}keysync pull --project my-app{/c}
  {c}keysync pull --project my-app --env staging{/c}

{b}See also:{/b}
  {c}keysync set --help{/c}
  {c}keysync push --help{/c}`),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			gh, err := github.NewClient(repoFlag)
			if err != nil {
				return fmt.Errorf("github client: %w", err)
			}

			secretNames, err := gh.List()
			if err != nil {
				return fmt.Errorf("list github secrets: %w", err)
			}

			if len(secretNames) == 0 {
				fmt.Println("No secrets found in GitHub.")
				return nil
			}

			fmt.Printf("Found %d secrets in GitHub (%s)\n", len(secretNames), gh.Repo())

			var missing int
			for _, name := range secretNames {
				// Check local store for this secret
				_, err := secretSt.Get(ctx, store.ScopeGlobal, "", "", name)
				if err == store.ErrNotFound {
					// Also check project scope if specified
					if project != "" {
						_, err = secretSt.Get(ctx, store.ScopeProject, project, "", name)
						if err == store.ErrNotFound {
							// Check project + env scope
							if env := envForGet(cmd); env != "" {
								_, err = secretSt.Get(ctx, store.ScopeProject, project, env, name)
								if err == store.ErrNotFound {
									fmt.Printf("  MISSING: %s (not found locally)\n", name)
									missing++
								} else if err == nil {
									fmt.Printf("  OK:      %s (project/%s/%s scope)\n", name, project, env)
								}
							} else {
								fmt.Printf("  MISSING: %s (not found locally)\n", name)
								missing++
							}
						} else if err == nil {
							fmt.Printf("  OK:      %s (project/%s scope)\n", name, project)
						}
					} else {
						fmt.Printf("  MISSING: %s (not found locally)\n", name)
						missing++
					}
				} else if err == nil {
					fmt.Printf("  OK:      %s (global scope)\n", name)
				}
			}

			if missing > 0 {
				fmt.Fprintf(os.Stderr, "\n%d secrets are missing locally. Use 'keysync set KEY=value' to add them.\n", missing)
			} else {
				fmt.Println("\nAll secrets are present in local store.")
			}

			return nil
		},
	}
}
