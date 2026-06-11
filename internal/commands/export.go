package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/dipockdas/keysync/internal/store"
	"github.com/spf13/cobra"
)

func newExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export [KEY] [--project name] [--env name]",
		Short: "Export secrets as shell-exportable environment variables",
		Long: F(`Prints secrets as {c}export KEY=VALUE{/c} lines for {c}eval{/c} or {c}source{/c}.
Same scope rules as {c}keysync get{/c}.

{b}With KEY{/b} (recommended for scripts):
  Resolves one secret (index lookup, then one keychain read) and prints a single line.

{b}Without KEY{/b}:
  Exports every matching secret (one keychain read per key). Use for full local dev setup.

{b}Resolution order{/b} (when {c}--project{/c} is provided, same as {c}get{/c}):
  1. Project + environment-scoped secret (only if {c}--env{/c} is passed)
  2. Project-scoped secret (no environment)
  3. Global secret

Without {c}--project{/c}, only global scope is included (unless {c}KEY{/c} is given).

{b}Examples:{/b}
  {c}eval $(keysync export API_KEY){/c}                               {g}# one secret{/g}
  {c}keysync export DATABASE_URL -p my-app{/c}                        {g}# one secret, project scope{/g}
  {c}eval $(keysync export -p my-app){/c}                             {g}# all project + global{/g}
  {c}eval $(keysync export -p my-app --env production){/c}            {g}# env + project + global{/g}
  {c}source <(keysync export -p my-app){/c}                          {g}# source directly{/g}

On macOS, use a signed binary ({c}make build-signed{/c}) and run {c}keysync trust{/c} once
after install to avoid repeated keychain password prompts.

{b}See also:{/b}
  {c}keysync get --help{/c}
  {c}keysync list --help{/c}`),
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 2 {
				return fmt.Errorf("accepts at most one KEY (e.g. keysync export API_KEY)")
			}
			if len(args) == 2 && !mightBeTrailingProjectArg(args) {
				return fmt.Errorf("accepts at most one KEY (e.g. keysync export API_KEY)")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			args = commandArgs(args)

			if len(args) == 1 {
				return exportOne(ctx, cmd, args[0])
			}

			exported := make(map[string]string)

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

				if explicitEnv := envForExport(cmd); explicitEnv != "" {
					envEntries, err := secretSt.List(ctx, store.ScopeProject, project, explicitEnv)
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

			return printExports(exported)
		},
	}
}

func exportOne(ctx context.Context, cmd *cobra.Command, key string) error {
	explicitEnv := envForExport(cmd)
	scope, proj, env, found := locateSecret(ctx, key, project, explicitEnv)
	if !found {
		fmt.Fprintf(os.Stderr, "secret not found: %s\n", key)
		os.Exit(1)
	}
	val, err := secretSt.Get(ctx, scope, proj, env, key)
	if err == store.ErrNotFound {
		fmt.Fprintf(os.Stderr, "secret not found: %s\n", key)
		os.Exit(1)
	}
	if err != nil {
		return fmt.Errorf("get secret: %w", err)
	}
	return printExports(map[string]string{key: val})
}

func printExports(exported map[string]string) error {
	if len(exported) == 0 {
		return nil
	}
	for key, value := range exported {
		fmt.Printf("export %s=%s\n", key, shellQuote(value))
	}
	return nil
}

// shellQuote wraps a string in single quotes, escaping any embedded single quotes
// for safe use in POSIX shell eval.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
