package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dipockdas/keysync/internal/config"
	"github.com/dipockdas/keysync/internal/platforms"
	"github.com/spf13/cobra"
)

var (
	pushPlatforms string
	pushDryRun    bool
	pushOnly      string
)

func newPushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push --project name [--env name] [--platforms vercel,railway,supabase]",
		Short: "Push secrets to GitHub Secrets and deployment platforms",
		Long: F(`Pushes secrets from the local OS secret store to {u}GitHub Secrets{/u}
and deployment platforms (Vercel, Railway, Supabase).

Provide either {c}--project{/c} or {c}--repo{/c} to identify which repo to push to:
  • {c}--project{/c} looks up the repo in {c}.keysync.json{/c}
  • {c}--repo{/c} uses the repo directly

{b}Which keys are pushed:{/b}
  • {c}globals{/c} — only global keys listed in the repo's {c}"globals"{/c} array
  • {c}project{/c} — project-wide keys (no environment) unless restricted (see below)
  • {c}environment{/c} — with {c}--env NAME{/c}, also keys for that environment (overrides project for same name)

{b}Restricting project keys:{/b}
  • {c}"secrets"{/c} in {c}.keysync.json{/c} — allowlist; only these key names are pushed
  • {c}"exclude"{/c} in {c}.keysync.json{/c} — never push these keys
  • {c}--only KEY1,KEY2{/c} — push only these keys for this run

Use {c}--dry-run{/c} to print the plan (key + scope source) without reading values from
the keychain or uploading.

Platform tokens are read from the OS secret store (global scope), falling back
to environment variables ({c}VERCEL_TOKEN{/c}, {c}RAILWAY_TOKEN{/c}, {c}SUPABASE_TOKEN{/c}).

{b}Examples:{/b}
  {c}keysync push --project my-app --dry-run{/c}
  {c}keysync push --project my-app --env production{/c}
  {c}keysync push --project my-app --only API_KEY,OAUTH_CLIENT_SECRET{/c}

{b}See also:{/b}
  {u}https://github.com/dipockdas/keysync/blob/main/docs/pushing-secrets.md{/u}`),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			ctx := cobraCmd.Context()

			repoKey, rc, proj, err := resolveRepoConfig()
			if err != nil {
				return err
			}

			if strings.ToLower(repoKey) == "dipockdas/keysync" {
				return fmt.Errorf("refusing to sync to the keysync repo itself\n\nPlease update .keysync.json with YOUR repository name (e.g., \"yourorg/yourrepo\")\nSee the example at: https://github.com/dipockdas/keysync#quick-start")
			}
			if strings.Contains(strings.ToUpper(repoKey), "YOUR_ORG") || strings.Contains(strings.ToUpper(repoKey), "YOUR_REPO") {
				return fmt.Errorf("please update .keysync.json with your actual repository name\n\nReplace \"YOUR_ORG/YOUR_REPO\" with your GitHub repository (e.g., \"yourorg/yourrepo\")")
			}

			var onlyKeys []string
			if pushOnly != "" {
				for _, k := range strings.Split(pushOnly, ",") {
					k = strings.TrimSpace(k)
					if k != "" {
						onlyKeys = append(onlyKeys, k)
					}
				}
			}

			plan, err := collectPushPlan(ctx, proj, effectiveEnv, rc.Globals, rc.Secrets, rc.Exclude, onlyKeys, !pushDryRun)
			if err != nil {
				return fmt.Errorf("collect secrets: %w", err)
			}
			secrets := planToSecrets(plan)
			if len(secrets) == 0 {
				if pushDryRun {
					printPushPlan(repoKey, proj, effectiveEnv, []string{"(no platforms — no keys to push)"}, plan, "")
				} else {
					fmt.Println("No secrets to push for this project (check globals, secrets allowlist, exclude, and --only).")
				}
				return nil
			}

			platformNames := pushPlatformNames(rc.Platforms)
			githubCfg := getPlatformConfigJSON(rc.Platforms, "github", repoKey)
			if githubCfg != "" {
				if err := platforms.ValidateGitHubConfig(githubCfg); err != nil {
					return err
				}
			}
			if pushDryRun {
				printPushPlan(repoKey, proj, effectiveEnv, platformNames, plan, githubCfg)
				return nil
			}

			fmt.Printf("Pushing %d secrets to repo %q (project: %s, env: %s)\n", len(secrets), repoKey, proj, effectiveEnv)

			var pushErrors int
			for _, name := range platformNames {
				name = strings.TrimSpace(name)
				platformCfg := getPlatformConfigJSON(rc.Platforms, name, repoKey)
				if platformCfg == "" {
					fmt.Fprintf(os.Stderr, "  SKIP: %s (not configured)\n", name)
					continue
				}

				p, err := platforms.Get(ctx, name, platformCfg, secretSt)
				if err != nil {
					fmt.Fprintf(os.Stderr, "  SKIP: %s (%v)\n", name, err)
					pushErrors++
					continue
				}

				fmt.Printf("  → %s\n", name)
				for key, value := range secrets {
					pushCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
					err := p.Upsert(pushCtx, key, value)
					cancel()
					if err != nil {
						fmt.Fprintf(os.Stderr, "    FAIL: %s: %v\n", key, err)
						pushErrors++
					} else if name == "github" {
						fmt.Printf("    ✓ %s (github %s)\n", key, platforms.GitHubKeyTarget(githubCfg, key))
					} else {
						fmt.Printf("    ✓ %s\n", key)
					}
				}
			}

			if pushErrors > 0 {
				fmt.Fprintf(os.Stderr, "\n%d error(s) during push.\n", pushErrors)
			} else {
				fmt.Println("\nPush complete.")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&pushPlatforms, "platforms", "", "comma-separated platforms (vercel,railway,supabase)")
	cmd.Flags().BoolVar(&pushDryRun, "dry-run", false, "print which keys would be pushed (with scope) without uploading")
	cmd.Flags().StringVar(&pushOnly, "only", "", "comma-separated keys to push for this run only")
	return cmd
}

func pushPlatformNames(platformsCfg map[string]json.RawMessage) []string {
	if pushPlatforms != "" {
		names := strings.Split(pushPlatforms, ",")
		for i := range names {
			names[i] = strings.TrimSpace(names[i])
		}
		return names
	}
	names := []string{"github"}
	names = append(names, configuredPlatforms(platformsCfg)...)
	return names
}

// newSendCmd is an alias for newPushCmd for backward compatibility.
func newSendCmd() *cobra.Command {
	cmd := newPushCmd()
	cmd.Use = "send --project name [--env name] [--platforms vercel,railway,supabase]"
	return cmd
}

// resolveRepoConfig resolves the repo and repo config from --project or --repo flags.
func resolveRepoConfig() (repoKey string, rc config.RepoConfig, projName string, err error) {
	if project != "" {
		repo, repoCfg, ok := config.FindRepoByProject(cfg, project)
		if !ok {
			return "", config.RepoConfig{}, "", fmt.Errorf("project %q not found in .keysync.json (add it under \"repos\")", project)
		}
		return repo, *repoCfg, project, nil
	}
	if repoFlag != "" {
		repoCfg, ok := cfg.Repos[repoFlag]
		if !ok {
			return "", config.RepoConfig{}, "", fmt.Errorf("repo %q not found in .keysync.json", repoFlag)
		}
		return repoFlag, repoCfg, repoCfg.Project, nil
	}
	return "", config.RepoConfig{}, "", fmt.Errorf("either --project or --repo is required for push")
}

// collectSecrets merges global (filtered), project, and environment-scoped secrets.
// Deprecated for push filtering — use collectPushPlan. Kept for tests.
func collectSecrets(ctx context.Context, project, env string, globals []string) (map[string]string, error) {
	plan, err := collectPushPlan(ctx, project, env, globals, nil, nil, nil, true)
	if err != nil {
		return nil, err
	}
	return planToSecrets(plan), nil
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
func getPlatformConfigJSON(pc any, platformName string, repoKey string) string {
	if platformName == "github" {
		data, _ := json.Marshal(pc)
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			return ""
		}
		if cfg, ok := raw["github"]; ok {
			return string(cfg)
		}
		return fmt.Sprintf(`{"repo":"%s"}`, repoKey)
	}

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
