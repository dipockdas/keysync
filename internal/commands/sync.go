package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/dipockdas/keysync/internal/github"
	"github.com/dipockdas/keysync/internal/platforms"
	"github.com/dipockdas/keysync/internal/store"
	"github.com/spf13/cobra"
)

var syncPlatforms string

func newSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync [--project name] [--platforms vercel,railway,supabase]",
		Short: "Push secrets to deployment platforms",
		Long: `Pushes secrets from the local store to deployment platforms.

The --platforms flag filters which platforms to sync. By default, all configured platforms are synced.
Platform tokens must be set as environment variables:
  VERCEL_TOKEN, RAILWAY_TOKEN, SUPABASE_TOKEN

To add a new platform, implement the Platform interface and register it.

Usage:
  keysync sync --project my-app
  keysync sync --project my-app --platforms vercel,railway`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			ctx := cobraCmd.Context()

			if project == "" {
				return fmt.Errorf("--project is required for sync")
			}

			// Look up project config
			projCfg, ok := cfg.Projects[project]
			if !ok {
				return fmt.Errorf("project %q not found in .keysync.json", project)
			}

			// Read secrets from local store
			secrets, err := collectSecrets(ctx, project)
			if err != nil {
				return fmt.Errorf("collect secrets: %w", err)
			}
			if len(secrets) == 0 {
				fmt.Println("No secrets found for this project.")
				return nil
			}

			fmt.Printf("Syncing %d secrets for project %q\n", len(secrets), project)

			// Determine which platforms to sync
			var platformNames []string
			if syncPlatforms != "" {
				platformNames = strings.Split(syncPlatforms, ",")
			} else {
				platformNames = configuredPlatforms(projCfg.Platforms)
			}

			if len(platformNames) == 0 {
				fmt.Println("No platforms configured for this project.")
				return nil
			}

			// Also push to GitHub
			if err := syncToGitHub(secrets); err != nil {
				fmt.Fprintf(os.Stderr, "warning: GitHub sync: %v\n", err)
			}

			// Sync to each platform
			var syncErrors int
			for _, name := range platformNames {
				name = strings.TrimSpace(name)
				platformCfg := getPlatformConfigJSON(projCfg.Platforms, name)
				if platformCfg == "" {
					fmt.Fprintf(os.Stderr, "  SKIP: %s (not configured)\n", name)
					continue
				}

				p, err := platforms.Get(name, platformCfg)
				if err != nil {
					fmt.Fprintf(os.Stderr, "  SKIP: %s (%v)\n", name, err)
					syncErrors++
					continue
				}

				fmt.Printf("  → %s\n", name)
				for key, value := range secrets {
					if err := p.Upsert(key, value); err != nil {
						fmt.Fprintf(os.Stderr, "    FAIL: %s: %v\n", key, err)
						syncErrors++
					} else {
						fmt.Printf("    ✓ %s\n", key)
					}
				}
			}

			if syncErrors > 0 {
				fmt.Fprintf(os.Stderr, "\n%d error(s) during sync.\n", syncErrors)
			} else {
				fmt.Println("\nSync complete.")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&syncPlatforms, "platforms", "", "comma-separated platforms (vercel,railway,supabase)")
	return cmd
}

// collectSecrets merges global and project-scoped secrets into one map.
func collectSecrets(ctx context.Context, project string) (map[string]string, error) {
	secrets := make(map[string]string)

	// Global secrets
	globalEntries, err := secretSt.List(ctx, store.ScopeGlobal, "")
	if err != nil {
		return nil, fmt.Errorf("list global secrets: %w", err)
	}
	for _, e := range globalEntries {
		val, err := secretSt.Get(ctx, store.ScopeGlobal, "", e.Key)
		if err == nil {
			secrets[e.Key] = val
		}
	}

	// Project secrets (override global)
	projectEntries, err := secretSt.List(ctx, store.ScopeProject, project)
	if err != nil {
		return nil, fmt.Errorf("list project secrets: %w", err)
	}
	for _, e := range projectEntries {
		val, err := secretSt.Get(ctx, store.ScopeProject, project, e.Key)
		if err == nil {
			secrets[e.Key] = val
		}
	}

	return secrets, nil
}

// configuredPlatforms returns the list of platform names that have config.
func configuredPlatforms(pc any) []string {
	var names []string
	data, _ := json.Marshal(pc)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	for name, cfg := range raw {
		if len(cfg) > 0 && string(cfg) != "{}" && string(cfg) != "null" {
			// Check if the config has the required fields by trying to parse
			var v any
			if err := json.Unmarshal(cfg, &v); err == nil && v != nil {
				names = append(names, name)
			}
		}
	}
	return names
}

// getPlatformConfigJSON returns the JSON config string for a platform.
func getPlatformConfigJSON(pc any, platformName string) string {
	data, _ := json.Marshal(pc)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return ""
	}
	cfg, ok := raw[platformName]
	if !ok {
		return ""
	}
	return string(cfg)
}

// syncToGitHub pushes all secrets to GitHub Secrets.
func syncToGitHub(secrets map[string]string) error {
	gh, err := github.NewClient(repoFlag)
	if err != nil {
		return err
	}
	fmt.Printf("  → github (%s)\n", gh.Repo())
	for key, value := range secrets {
		if err := gh.Set(key, value); err != nil {
			fmt.Fprintf(os.Stderr, "    FAIL: %s: %v\n", key, err)
		} else {
			fmt.Printf("    ✓ %s\n", key)
		}
	}
	return nil
}
