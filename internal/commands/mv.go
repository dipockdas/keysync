package commands

import (
	"fmt"

	"github.com/dipockdas/keysync/internal/store"
	"github.com/spf13/cobra"
)

var mvForce bool

func newMvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "mv KEY [--project name] [--env name] (--to-global | --to-project name [--to-env name]) [--force]",
		Aliases: []string{"move"},
		Short:   "Move a secret between scopes in the local OS secret store",
		SilenceUsage: true,
		Long: F(`Moves a secret from one scope to another in the local OS secret store
(macOS Keychain, Linux libsecret, Windows Credential Manager). This is a local-only
operation — it does not update GitHub Secrets or deployment platforms. Run
{c}keysync push{/c} afterward if you need upstream sync.

{b}Source scope{/b} (same rules as {c}keysync rm{/c}):
  • Omit {c}--project{/c} → global
  • {c}--project{/c} only → project scope
  • {c}--project{/c} + {c}--env{/c} → project + environment scope

{b}Destination scope{/b} (required):
  • {c}--to-global{/c} ({c}--to-g{/c}) → global
  • {c}--to-project NAME{/c} ({c}--to-p{/c}) → project scope
  • {c}--to-project NAME --to-env ENV{/c} ({c}--to-e{/c}) → project + environment scope

{b}Examples:{/b}
  {c}keysync mv DATABASE_URL --to-project my-app{/c}
  {c}keysync mv STRIPE_KEY -p my-app --to-global{/c}
  {c}keysync mv DB_URL -p my-app -e staging --to-p my-app --to-e production{/c}
  {c}keysync mv API_KEY -p old-app --to-p new-app --force{/c}

{b}See also:{/b}
  {c}keysync list --help{/c}
  {c}keysync rm --help{/c}`),
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("requires a KEY to move (e.g. keysync mv DATABASE_URL --to-project my-app)")
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

			toGlobal, toProject, toEnv, err := resolveMvDestFlags(cmd)
			if err != nil {
				return err
			}

			srcScope, srcProj, srcEnv := sourceScopeFromFlags()
			dstScope, dstProj, dstEnv := destScopeFromFlags(toGlobal, toProject, toEnv)

			if scopesEqual(srcScope, srcProj, srcEnv, dstScope, dstProj, dstEnv) {
				return fmt.Errorf("source and destination scope are the same")
			}

			ctx := cmd.Context()

			val, err := secretSt.Get(ctx, srcScope, srcProj, srcEnv, key)
			if err == store.ErrNotFound {
				return fmt.Errorf("secret %s not found in local store", secretPathLabel(srcScope, srcProj, srcEnv, key))
			}
			if err != nil {
				return fmt.Errorf("get secret: %w", err)
			}

			_, err = secretSt.Get(ctx, dstScope, dstProj, dstEnv, key)
			if err == nil && !mvForce {
				return fmt.Errorf("secret %s already exists at destination (use --force to overwrite)",
					secretPathLabel(dstScope, dstProj, dstEnv, key))
			}
			if err != nil && err != store.ErrNotFound {
				return fmt.Errorf("check destination: %w", err)
			}

			if err := secretSt.Set(ctx, dstScope, dstProj, dstEnv, key, val); err != nil {
				return fmt.Errorf("set destination: %w", err)
			}
			if err := secretSt.Delete(ctx, srcScope, srcProj, srcEnv, key); err != nil {
				_ = secretSt.Delete(ctx, dstScope, dstProj, dstEnv, key)
				return fmt.Errorf("delete source after move: %w", err)
			}

			fmt.Printf("Moved %s → %s\n",
				secretPathLabel(srcScope, srcProj, srcEnv, key),
				secretPathLabel(dstScope, dstProj, dstEnv, key))
			return nil
		},
	}

	cmd.Flags().BoolVar(&mvToGlobal, "to-global", false, "move to global scope")
	cmd.Flags().Bool("to-g", false, "alias for --to-global")
	cmd.Flags().StringVar(&mvToProject, "to-project", "", "destination project name")
	cmd.Flags().String("to-p", "", "alias for --to-project")
	cmd.Flags().StringVar(&mvToEnv, "to-env", "", "destination environment name (requires --to-project)")
	cmd.Flags().String("to-e", "", "alias for --to-env")
	cmd.Flags().BoolVarP(&mvForce, "force", "f", false, "overwrite if the key already exists at the destination")

	return cmd
}

var (
	mvToGlobal  bool
	mvToProject string
	mvToEnv     string
)

func resolveMvDestFlags(cmd *cobra.Command) (toGlobal bool, toProject, toEnv string, err error) {
	fs := cmd.Flags()

	toGlobal = mvToGlobal
	if g, _ := fs.GetBool("to-g"); g {
		toGlobal = true
	}

	toProject = mvToProject
	if p, _ := fs.GetString("to-p"); p != "" {
		if toProject != "" && toProject != p {
			return false, "", "", fmt.Errorf("conflicting values for --to-project and --to-p")
		}
		toProject = p
	}

	toEnv = mvToEnv
	if e, _ := fs.GetString("to-e"); e != "" {
		if toEnv != "" && toEnv != e {
			return false, "", "", fmt.Errorf("conflicting values for --to-env and --to-e")
		}
		toEnv = e
	}

	if toGlobal && toProject != "" {
		return false, "", "", fmt.Errorf("cannot use --to-global (--to-g) with --to-project (--to-p)")
	}
	if toEnv != "" && toProject == "" {
		return false, "", "", fmt.Errorf("--to-env (--to-e) requires --to-project (--to-p)")
	}
	if !toGlobal && toProject == "" {
		return false, "", "", fmt.Errorf("destination required: use --to-global (--to-g) or --to-project (--to-p)")
	}

	return toGlobal, toProject, toEnv, nil
}

func sourceScopeFromFlags() (store.Scope, string, string) {
	if project == "" {
		return store.ScopeGlobal, "", ""
	}
	return store.ScopeProject, project, effectiveEnv
}

func destScopeFromFlags(toGlobal bool, toProject, toEnv string) (store.Scope, string, string) {
	if toGlobal {
		return store.ScopeGlobal, "", ""
	}
	return store.ScopeProject, toProject, toEnv
}

func scopesEqual(aScope store.Scope, aProj, aEnv string, bScope store.Scope, bProj, bEnv string) bool {
	return aScope == bScope && aProj == bProj && aEnv == bEnv
}

// secretPathLabel returns a human-readable path for a secret (scope/key).
func secretPathLabel(scope store.Scope, project, environment, key string) string {
	if scope == store.ScopeGlobal {
		return fmt.Sprintf("global/%s", key)
	}
	if environment != "" {
		return fmt.Sprintf("project/%s/%s/%s", project, environment, key)
	}
	return fmt.Sprintf("project/%s/%s", project, key)
}
