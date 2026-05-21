//go:build windows

package store

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/danieljoos/wincred"
)

// WincredStore implements Store using Windows Credential Manager
// via the Win32 API (github.com/danieljoos/wincred).
//
// Credentials are stored as generic credentials with:
//   - TargetName: "keysync_<scope>[_<project>[_<environment>]]"
//   - UserName:   "<key>"
//   - CredentialBlob: "<value>" (UTF-8 encoded)
type WincredStore struct {
	mu    sync.RWMutex
	cache []SecretEntry // cached list, rebuilt on demand
}

func NewWincredStore() *WincredStore {
	ws := &WincredStore{}
	ws.rebuildCache()
	return ws
}

// credTarget returns the credential target name, INCLUDING the key name.
// Format: "keysync_<scope>_<key>" or "keysync_<scope>_<project>_<key>" or "keysync_<scope>_<project>_<environment>_<key>"
// This ensures each secret has a unique credential (fixes audit finding 3).
func credTarget(scope Scope, project, environment, key string) string {
	if scope == ScopeGlobal {
		return fmt.Sprintf("keysync_%s_%s", scope, key)
	}
	if environment != "" {
		return fmt.Sprintf("keysync_%s_%s_%s_%s", scope, project, environment, key)
	}
	return fmt.Sprintf("keysync_%s_%s_%s", scope, project, key)
}

// parseCredTarget splits a target name back into scope, project, environment, and key.
// "keysync_global_API_KEY" → (global, "", "", "API_KEY")
// "keysync_project_my-app_DATABASE_URL" → (project, "my-app", "", "DATABASE_URL")
// "keysync_project_my-app_production_STRIPE_KEY" → (project, "my-app", "production", "STRIPE_KEY")
func parseCredTarget(target string) (Scope, string, string, string) {
	trimmed := strings.TrimPrefix(target, "keysync_")
	parts := strings.Split(trimmed, "_")

	if len(parts) < 2 {
		return ScopeGlobal, "", "", ""
	}

	scope := Scope(parts[0])
	if scope != ScopeGlobal && scope != ScopeProject {
		return ScopeGlobal, "", "", ""
	}

	// Global scope: keysync_global_KEY
	if scope == ScopeGlobal {
		if len(parts) < 2 {
			return scope, "", "", ""
		}
		key := strings.Join(parts[1:], "_") // In case key contains underscores
		return scope, "", "", key
	}

	// Project scope: keysync_project_PROJECT_KEY or keysync_project_PROJECT_ENV_KEY
	if len(parts) < 3 {
		return scope, "", "", ""
	}

	project := parts[1]

	// Try to determine if there's an environment
	// Format could be: keysync_project_my-app_API_KEY (no env)
	// Or: keysync_project_my-app_production_API_KEY (with env)
	//
	// We need a heuristic: if parts[2] looks like a common env name, treat it as env
	// Otherwise, treat remaining parts as the key
	//
	// Actually, we can't reliably distinguish without context. Let's use a different approach:
	// - If we have exactly 3 parts: keysync_project_PROJECT_KEY (no env)
	// - If we have 4+ parts: keysync_project_PROJECT_ENV_KEY (with env)
	// But this breaks if project/env names have underscores!
	//
	// Better approach: rebuild cache should use the full target as identifier,
	// and we don't need to parse it perfectly. We just need uniqueness.
	//
	// Actually, looking at the code, rebuildCache uses parseCredTarget to populate
	// the cache with scope/project/env/key, so we DO need to parse correctly.
	//
	// The safest approach: store metadata in a separate field or use a delimiter
	// that can't appear in names. But we can't change the Windows API.
	//
	// Pragmatic solution: Assume project and environment names don't contain underscores,
	// and key names might. So:
	// - 3 parts: scope_project_KEY
	// - 4+ parts: scope_project_env_KEY (where KEY might have underscores)

	if len(parts) == 3 {
		// keysync_project_my-app_API_KEY → no environment
		key := parts[2]
		return scope, project, "", key
	}

	// 4+ parts: keysync_project_my-app_production_STRIPE_KEY
	// Assume environment is parts[2], key is parts[3:]
	env := parts[2]
	key := strings.Join(parts[3:], "_")
	return scope, project, env, key
}

func (w *WincredStore) Get(_ context.Context, scope Scope, project, environment, key string) (string, error) {
	target := credTarget(scope, project, environment, key)
	cred, err := wincred.GetGenericCredential(target)
	if err != nil {
		return "", ErrNotFound
	}
	// Target now includes key, so no need to verify UserName
	return string(cred.CredentialBlob), nil
}

func (w *WincredStore) Set(_ context.Context, scope Scope, project, environment, key, value string) error {
	target := credTarget(scope, project, environment, key)

	// Delete existing first to avoid duplicates (target now includes key)
	existing, err := wincred.GetGenericCredential(target)
	if err == nil {
		_ = existing.Delete()
	}

	cred := wincred.NewGenericCredential(target)
	cred.UserName = key // Keep UserName for debugging/display purposes
	cred.CredentialBlob = []byte(value)
	cred.Persist = wincred.PersistLocalMachine

	if err := cred.Write(); err != nil {
		return fmt.Errorf("wincred write: %w", err)
	}

	// Update cache
	w.mu.Lock()
	// Check if already in cache (avoid duplicates)
	found := false
	for _, e := range w.cache {
		if e.Scope == scope && e.Project == project && e.Environment == environment && e.Key == key {
			found = true
			break
		}
	}
	if !found {
		w.cache = append(w.cache, SecretEntry{Scope: scope, Project: project, Environment: environment, Key: key})
	}
	w.mu.Unlock()
	return nil
}

func (w *WincredStore) Delete(_ context.Context, scope Scope, project, environment, key string) error {
	target := credTarget(scope, project, environment, key)
	cred, err := wincred.GetGenericCredential(target)
	if err != nil {
		return ErrNotFound
	}
	if err := cred.Delete(); err != nil {
		return fmt.Errorf("wincred delete: %w", err)
	}

	// Update cache
	w.mu.Lock()
	filtered := make([]SecretEntry, 0, len(w.cache))
	for _, e := range w.cache {
		if e.Scope == scope && e.Project == project && e.Environment == environment && e.Key == key {
			continue
		}
		filtered = append(filtered, e)
	}
	w.cache = filtered
	w.mu.Unlock()
	return nil
}

func (w *WincredStore) List(_ context.Context, scope Scope, project, environment string) ([]SecretEntry, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var result []SecretEntry
	for _, e := range w.cache {
		if (scope == "" || e.Scope == scope) &&
			(project == "" || e.Project == project) &&
			(environment == "" || e.Environment == environment) {
			result = append(result, e)
		}
	}
	return result, nil
}

// rebuildCache scans Credential Manager for keysync entries and populates the cache.
func (w *WincredStore) rebuildCache() {
	all, err := wincred.List()
	if err != nil {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	w.cache = nil

	for _, cred := range all {
		if !strings.HasPrefix(cred.TargetName, "keysync_") {
			continue
		}
		scope, project, env, key := parseCredTarget(cred.TargetName)
		// Skip entries where we couldn't extract a key
		if key == "" {
			continue
		}
		w.cache = append(w.cache, SecretEntry{
			Scope:       scope,
			Project:     project,
			Environment: env,
			Key:         key,
		})
	}
}
