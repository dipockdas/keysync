package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/dipockdas/keysync/internal/config"
	"github.com/dipockdas/keysync/internal/github"
	"github.com/dipockdas/keysync/internal/platforms"
	"github.com/dipockdas/keysync/internal/store"
	"github.com/spf13/cobra"
)

var syncPlatforms string

func newSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync --project name [--env name] [--platforms vercel,railway,supabase]",
		Short: "Push secrets to GitHub Secrets and deployment platforms",
		Long: F(`Pushes secrets from the local OS secret store to {u}GitHub Secrets{/u}
and deployment platforms (Vercel, Railway, Supabase).

Provide either {c}--project{/c} or {c}--repo{/c} to identify which repo to sync:
  • {c}--project{/c} looks up the repo in {c}.keysync.json{/c}
  • {c}--repo{/c} uses the repo directly

Secrets synced:
  • Project-scoped secrets — always included for the matching project
  • Environment-scoped secrets — included if {c}--env{/c} is provided
  • Global secrets — only those listed in the repo's {c}"globals"{/c} config

Platform tokens are read from the OS secret store (global scope), falling back
to environment variables ({c}VERCEL_TOKEN{/c}, {c}RAILWAY_TOKEN{/c}, {c}SUPABASE_TOKEN{/c}).

Use {c}--platforms{/c} to target specific platforms instead of all configured ones.

{b}Custom platforms:{/b} Implement the Platform interface (Name, Upsert) and register
via {c}platforms.Register(){/c}. See {c}internal/platforms/example_test.go{/c} for a
copyable template.

{b}Examples:{/b}
  {c}keysync sync --project my-app{/c}                              # repo from config
  {c}keysync sync --repo org/my-app{/c}                             # repo directly
  {c}keysync sync --project my-app --env production{/c}             # production env
  {c}keysync sync --project my-app --platforms vercel,railway{/c}   # specific platforms

{b}See also:{/b}
  Custom platform template: {u}https://github.com/dipockdas/keysync/blob/main/internal/platforms/example_test.go{/u}
  Syncing secrets: {u}https://github.com/dipockdas/keysync#syncing-secrets{/u}`),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			ctx := cobraCmd.Context()

			// Resolve repo and project from --project or --repo flag
			repoKey, project, globals, platformsCfg, err := resolveRepoConfig()
			if err != nil {
				return err
			}

			// Safety check: prevent syncing to the keysync repo itself
			if strings.ToLower(repoKey) == "dipockdas/keysync" {
				return fmt.Errorf("refusing to sync to the keysync repo itself\n\nPlease update .keysync.json with YOUR repository name (e.g., \"yourorg/yourrepo\")\nSee the example at: https://github.com/dipockdas/keysync#quick-start")
			}
			// Also prevent the placeholder value
			if strings.Contains(strings.ToUpper(repoKey), "YOUR_ORG") || strings.Contains(strings.ToUpper(repoKey), "YOUR_REPO") {
				return fmt.Errorf("please update .keysync.json with your actual repository name\n\nReplace \"YOUR_ORG/YOUR_REPO\" with your GitHub repository (e.g., \"yourorg/yourrepo\")")
			}

			// Read secrets from local store (only specified globals + project-scoped)
			secrets, err := collectSecrets(ctx, project, envFlag, globals)
			if err != nil {
				return fmt.Errorf("collect secrets: %w", err)
			}
			if len(secrets) == 0 {
				fmt.Println("No secrets found for this project.")
				return nil
			}

			fmt.Printf("Syncing %d secrets for repo %q (project: %s, env: %s)\n", len(secrets), repoKey, project, envFlag)

			// Determine which platforms to sync
			var platformNames []string
			if syncPlatforms != "" {
				platformNames = strings.Split(syncPlatforms, ",")
			} else {
				platformNames = configuredPlatforms(platformsCfg)
			}

			// Push to GitHub
			if err := syncToGitHub(repoKey, secrets); err != nil {
				fmt.Fprintf(os.Stderr, "warning: GitHub sync: %v\n", err)
			}

			// Sync to each platform
			var syncErrors int
			for _, name := range platformNames {
				name = strings.TrimSpace(name)
				platformCfg := getPlatformConfigJSON(platformsCfg, name)
				if platformCfg == "" {
					fmt.Fprintf(os.Stderr, "  SKIP: %s (not configured)\n", name)
					continue
				}

				p, err := platforms.Get(ctx, name, platformCfg, secretSt)
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

// resolveRepoConfig resolves the repo, project, globals, and platform config from
// --project or --repo flags. Returns the repo key, project name, globals list,
// platform config, or an error.
func resolveRepoConfig() (repoKey, projName string, globals []string, platformsCfg any, err error) {
	if project != "" {
		// Look up repo by project name
		repo, rc, ok := config.FindRepoByProject(cfg, project)
		if !ok {
			return "", "", nil, nil, fmt.Errorf("project %q not found in .keysync.json (add it under \"repos\")", project)
		}
		return repo, project, rc.Globals, rc.Platforms, nil
	}
	if repoFlag != "" {
		rc, ok := cfg.Repos[repoFlag]
		if !ok {
			return "", "", nil, nil, fmt.Errorf("repo %q not found in .keysync.json", repoFlag)
		}
		return repoFlag, rc.Project, rc.Globals, rc.Platforms, nil
	}
	return "", "", nil, nil, fmt.Errorf("either --project or --repo is required for sync")
}

// collectSecrets merges global (filtered), project, and environment-scoped secrets.
// Precedence: global (lowest) < project < project+env (highest).
// Only global keys listed in the globals slice are included.
func collectSecrets(ctx context.Context, project, env string, globals []string) (map[string]string, error) {
	secrets := make(map[string]string)

	// Global secrets — only those explicitly listed in the repo's globals config
	if len(globals) > 0 {
		globalSet := make(map[string]bool, len(globals))
		for _, g := range globals {
			globalSet[g] = true
		}
		globalEntries, err := secretSt.List(ctx, store.ScopeGlobal, "", "")
		if err != nil {
			return nil, fmt.Errorf("list global secrets: %w", err)
		}
		for _, e := range globalEntries {
			if globalSet[e.Key] {
				val, err := secretSt.Get(ctx, store.ScopeGlobal, "", "", e.Key)
				if err == nil {
					secrets[e.Key] = val
				}
			}
		}
	}

	// Project secrets (no specific environment)
	projectEntries, err := secretSt.List(ctx, store.ScopeProject, project, "")
	if err != nil {
		return nil, fmt.Errorf("list project secrets: %w", err)
	}
	for _, e := range projectEntries {
		val, err := secretSt.Get(ctx, store.ScopeProject, project, "", e.Key)
		if err == nil {
			secrets[e.Key] = val
		}
	}

	// Project + environment secrets (highest precedence)
	if env != "" {
		envEntries, err := secretSt.List(ctx, store.ScopeProject, project, env)
		if err != nil {
			return nil, fmt.Errorf("list environment secrets: %w", err)
		}
		for _, e := range envEntries {
			val, err := secretSt.Get(ctx, store.ScopeProject, project, env, e.Key)
			if err == nil {
				secrets[e.Key] = val
			}
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

// syncToGitHub pushes all secrets to GitHub Secrets for the given repo.
func syncToGitHub(repoKey string, secrets map[string]string) error {
	gh, err := github.NewClient(repoKey)
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
