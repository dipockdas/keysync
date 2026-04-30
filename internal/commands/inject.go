package commands

import (
	"fmt"
	"os"
	"sort"

	"github.com/dipockdas/keysync/internal/github"
	"github.com/dipockdas/keysync/internal/store"
	"github.com/spf13/cobra"
)

var (
	injectFormat string
	ciMode       bool
)

func newInjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inject [--project name] [--format dotenv|exports]",
		Short: "Generate .env.local or shell exports from local secrets",
		Long: `Reads all secrets for a project (global + project-scoped) and outputs them
in the requested format.

Formats:
  dotenv   — KEY=value format suitable for .env.local (DEFAULT)
  exports  — export KEY=value format for eval

Usage:
  keysync inject --project my-app > .env.local
  eval $(keysync inject --project my-app --format exports)
  keysync inject --ci --format dotenv > .env.test`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if ciMode {
				return injectCI(cmd)
			}

			if project == "" {
				return fmt.Errorf("--project is required for inject")
			}

			// Collect global secrets
			globalSecrets, err := secretSt.List(ctx, store.ScopeGlobal, "")
			if err != nil {
				return fmt.Errorf("list global secrets: %w", err)
			}

			// Collect project secrets (overrides)
			projectSecrets, err := secretSt.List(ctx, store.ScopeProject, project)
			if err != nil {
				return fmt.Errorf("list project secrets: %w", err)
			}

			// Build merged map: global first, then project overrides
			merged := make(map[string]string)
			for _, e := range globalSecrets {
				val, err := secretSt.Get(ctx, store.ScopeGlobal, "", e.Key)
				if err == nil {
					merged[e.Key] = val
				}
			}
			for _, e := range projectSecrets {
				val, err := secretSt.Get(ctx, store.ScopeProject, project, e.Key)
				if err == nil {
					merged[e.Key] = val
				}
			}

			if len(merged) == 0 {
				fmt.Fprintln(os.Stderr, "No secrets found.")
				return nil
			}

			// Sort keys for deterministic output
			keys := make([]string, 0, len(merged))
			for k := range merged {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, k := range keys {
				switch injectFormat {
				case "exports":
					fmt.Printf("export %s=%s\n", k, merged[k])
				default:
					fmt.Printf("%s=%s\n", k, merged[k])
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&injectFormat, "format", "dotenv", "output format: dotenv or exports")
	cmd.Flags().BoolVar(&ciMode, "ci", false, "read from GitHub Secrets instead of OS store")
	return cmd
}

// injectCI reads secrets from GitHub Actions environment variables for CI.
// In GitHub Actions, secrets are injected as environment variables.
// This reads the secret names from GitHub and looks for their values in env vars.
func injectCI(_ *cobra.Command) error {
	if project == "" {
		return fmt.Errorf("--project is required for CI inject")
	}

	gh, err := github.NewClient(repoFlag)
	if err != nil {
		return fmt.Errorf("github client: %w", err)
	}

	secretNames, err := gh.List()
	if err != nil {
		return fmt.Errorf("list github secrets: %w", err)
	}

	merged := make(map[string]string)
	for _, name := range secretNames {
		// Check env var directly (GitHub injects secrets as env vars)
		val := os.Getenv(name)
		if val != "" {
			merged[name] = val
		}
	}

	if len(merged) == 0 {
		fmt.Fprintln(os.Stderr, "No secrets found in CI environment.")
		return nil
	}

	keys := make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		switch injectFormat {
		case "exports":
			fmt.Printf("export %s=%s\n", k, merged[k])
		default:
			fmt.Printf("%s=%s\n", k, merged[k])
		}
	}
	return nil
}

