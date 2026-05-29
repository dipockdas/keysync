package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/dipockdas/keysync/internal/platforms"
	"github.com/dipockdas/keysync/internal/store"
)

// pushPlanEntry describes one key considered for push.
type pushPlanEntry struct {
	Key      string
	Value    string
	Source   string
	Included bool
	SkipNote string
}

func pushSourceLabel(scope store.Scope, project, env string) string {
	if scope == store.ScopeGlobal {
		return "global"
	}
	if env != "" {
		return fmt.Sprintf("project/%s/%s", project, env)
	}
	return fmt.Sprintf("project/%s", project)
}

func entrySource(e store.SecretEntry) string {
	return pushSourceLabel(e.Scope, e.Project, e.Environment)
}

// collectPushEntries lists candidate secrets for push without reading values.
// When env is set, only project-wide keys (no environment) and that environment are included.
func collectPushEntries(ctx context.Context, project, env string, globals []string) ([]store.SecretEntry, error) {
	merged := make(map[string]store.SecretEntry)

	add := func(entries []store.SecretEntry) {
		for _, e := range entries {
			merged[e.Key] = e
		}
	}

	if len(globals) > 0 {
		globalSet := make(map[string]bool, len(globals))
		for _, g := range globals {
			globalSet[g] = true
		}
		globalEntries, err := secretSt.List(ctx, store.ScopeGlobal, "", "")
		if err != nil {
			return nil, fmt.Errorf("list global secrets: %w", err)
		}
		var selected []store.SecretEntry
		for _, e := range globalEntries {
			if globalSet[e.Key] {
				selected = append(selected, e)
			}
		}
		add(selected)
	}

	projectEntries, err := secretSt.List(ctx, store.ScopeProject, project, "")
	if err != nil {
		return nil, fmt.Errorf("list project secrets: %w", err)
	}
	if env == "" {
		add(projectEntries)
	} else {
		var projectWide []store.SecretEntry
		for _, e := range projectEntries {
			if e.Environment == "" {
				projectWide = append(projectWide, e)
			}
		}
		add(projectWide)

		envEntries, err := secretSt.List(ctx, store.ScopeProject, project, env)
		if err != nil {
			return nil, fmt.Errorf("list environment secrets: %w", err)
		}
		add(envEntries)
	}

	out := make([]store.SecretEntry, 0, len(merged))
	for _, e := range merged {
		out = append(out, e)
	}
	return out, nil
}

// collectPushPlan merges keychain secrets and applies push filters from config/flags.
// When loadValues is false (--dry-run), only key names and scopes are read from the index;
// secret values are not fetched from the OS keychain.
func collectPushPlan(ctx context.Context, project, env string, globals, allowlist, exclude, only []string, loadValues bool) ([]pushPlanEntry, error) {
	entries, err := collectPushEntries(ctx, project, env, globals)
	if err != nil {
		return nil, err
	}

	allowSet := stringSet(allowlist)
	excludeSet := stringSet(exclude)
	onlySet := stringSet(only)

	plan := make([]pushPlanEntry, 0, len(entries))
	for _, e := range entries {
		entry := pushPlanEntry{
			Key:      e.Key,
			Source:   entrySource(e),
			Included: true,
		}

		if len(onlySet) > 0 {
			if !onlySet[e.Key] {
				entry.Included = false
				entry.SkipNote = "not in --only list"
			}
		} else if len(allowSet) > 0 {
			if !allowSet[e.Key] {
				entry.Included = false
				entry.SkipNote = "not in secrets allowlist"
			}
		}

		if excludeSet[e.Key] {
			entry.Included = false
			entry.SkipNote = "excluded in .keysync.json"
		}

		if loadValues && entry.Included {
			val, err := secretSt.Get(ctx, e.Scope, e.Project, e.Environment, e.Key)
			if err != nil {
				return nil, fmt.Errorf("get %s: %w", e.Key, err)
			}
			entry.Value = val
		}

		plan = append(plan, entry)
	}

	return plan, nil
}

func stringSet(items []string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, s := range items {
		s = strings.TrimSpace(s)
		if s != "" {
			m[s] = true
		}
	}
	return m
}

func planToSecrets(plan []pushPlanEntry) map[string]string {
	out := make(map[string]string)
	for _, e := range plan {
		if e.Included {
			out[e.Key] = e.Value
		}
	}
	return out
}

func printPushPlan(repoKey, project, env string, platformNames []string, plan []pushPlanEntry, githubCfgJSON string) {
	envLabel := env
	if envLabel == "" {
		envLabel = "(none — project scope only for env keys)"
	}
	fmt.Printf("Push plan for %q (project: %s, env: %s)\n", repoKey, project, envLabel)
	fmt.Println("(dry-run: key names and scopes only — values not read from keychain)")
	fmt.Println()

	included := 0
	for _, e := range plan {
		if e.Included {
			included++
		}
	}
	for _, platform := range platformNames {
		fmt.Printf("  %s:\n", platform)
		any := false
		for _, e := range plan {
			if e.Included {
				line := fmt.Sprintf("%-28s  %s", e.Key, e.Source)
				if platform == "github" && githubCfgJSON != "" {
					line += fmt.Sprintf("  → github %s", platforms.GitHubKeyTarget(githubCfgJSON, e.Key))
				}
				fmt.Printf("    %s\n", line)
				any = true
			}
		}
		if !any {
			fmt.Println("    (no keys)")
		}
	}

	skipped := 0
	for _, e := range plan {
		if !e.Included {
			skipped++
		}
	}
	if skipped > 0 {
		fmt.Fprintf(os.Stderr, "\nSkipped %d key(s) (not pushed):\n", skipped)
		for _, e := range plan {
			if !e.Included {
				fmt.Fprintf(os.Stderr, "  %-28s  %s  — %s\n", e.Key, e.Source, e.SkipNote)
			}
		}
	}

	if included == 0 {
		fmt.Println("\nNo keys would be pushed.")
	} else {
		fmt.Printf("\nWould push %d secret value(s) to %d platform(s).\n", len(planToSecrets(plan)), len(platformNames))
	}
}
