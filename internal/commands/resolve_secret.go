package commands

import (
	"context"

	"github.com/dipockdas/keysync/internal/store"
)

// locateSecret uses the key index (no keychain reads) to find where a key lives,
// following the same precedence as get: explicit env → project-wide → global.
func locateSecret(ctx context.Context, key, proj, explicitEnv string) (scope store.Scope, project, environment string, found bool) {
	if proj != "" {
		if explicitEnv != "" {
			if scope, project, environment, found = findInList(ctx, store.ScopeProject, proj, explicitEnv, key); found {
				return
			}
		}
		if scope, project, environment, found = findProjectWide(ctx, proj, key); found {
			return
		}
	}
	return findInList(ctx, store.ScopeGlobal, "", "", key)
}

func findProjectWide(ctx context.Context, proj, key string) (store.Scope, string, string, bool) {
	entries, err := secretSt.List(ctx, store.ScopeProject, proj, "")
	if err != nil {
		return "", "", "", false
	}
	for _, e := range entries {
		if e.Key == key && e.Environment == "" {
			return e.Scope, e.Project, e.Environment, true
		}
	}
	return "", "", "", false
}

func findInList(ctx context.Context, scope store.Scope, proj, env, key string) (store.Scope, string, string, bool) {
	entries, err := secretSt.List(ctx, scope, proj, env)
	if err != nil {
		return "", "", "", false
	}
	for _, e := range entries {
		if e.Key == key {
			if env != "" && e.Environment != env {
				continue
			}
			if env == "" && scope == store.ScopeProject && e.Environment != "" {
				continue
			}
			return e.Scope, e.Project, e.Environment, true
		}
	}
	return "", "", "", false
}
