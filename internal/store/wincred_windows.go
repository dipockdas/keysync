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

// credTarget returns the credential target name.
// Format: "keysync_<scope>[_<project>[_<environment>]]"
func credTarget(scope Scope, project, environment string) string {
	if scope == ScopeGlobal {
		return fmt.Sprintf("keysync_%s", scope)
	}
	base := fmt.Sprintf("keysync_%s_%s", scope, project)
	if environment != "" {
		base += "_" + environment
	}
	return base
}

// parseCredTarget splits a target name back into scope, project, and environment.
// "keysync_global" → (global, "", "")
// "keysync_project_my-app" → (project, "my-app", "")
// "keysync_project_my-app_production" → (project, "my-app", "production")
func parseCredTarget(target string) (Scope, string, string) {
	trimmed := strings.TrimPrefix(target, "keysync_")
	parts := strings.SplitN(trimmed, "_", 3)
	if len(parts) == 0 {
		return ScopeGlobal, "", ""
	}
	scope := Scope(parts[0])
	if scope != ScopeGlobal && scope != ScopeProject {
		return ScopeGlobal, "", ""
	}
	if scope == ScopeGlobal {
		return scope, "", ""
	}
	if len(parts) < 2 {
		return scope, "", ""
	}
	project := parts[1]
	env := ""
	if len(parts) >= 3 {
		env = parts[2]
	}
	return scope, project, env
}

func (w *WincredStore) Get(_ context.Context, scope Scope, project, environment, key string) (string, error) {
	target := credTarget(scope, project, environment)
	cred, err := wincred.GetGenericCredential(target)
	if err != nil {
		return "", ErrNotFound
	}
	// Verify the account name matches (target might match but different key)
	if cred.UserName != key {
		return "", ErrNotFound
	}
	return string(cred.CredentialBlob), nil
}

func (w *WincredStore) Set(_ context.Context, scope Scope, project, environment, key, value string) error {
	target := credTarget(scope, project, environment)

	// Delete existing first to avoid duplicates
	existing, err := wincred.GetGenericCredential(target)
	if err == nil && existing.UserName == key {
		_ = existing.Delete()
	}

	cred := wincred.NewGenericCredential(target)
	cred.UserName = key
	cred.CredentialBlob = []byte(value)
	cred.Persist = wincred.PersistLocalMachine

	if err := cred.Write(); err != nil {
		return fmt.Errorf("wincred write: %w", err)
	}

	// Update cache
	w.mu.Lock()
	w.cache = append(w.cache, SecretEntry{Scope: scope, Project: project, Environment: environment, Key: key})
	w.mu.Unlock()
	return nil
}

func (w *WincredStore) Delete(_ context.Context, scope Scope, project, environment, key string) error {
	target := credTarget(scope, project, environment)
	cred, err := wincred.GetGenericCredential(target)
	if err != nil {
		return ErrNotFound
	}
	if cred.UserName != key {
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
		scope, project, env := parseCredTarget(cred.TargetName)
		w.cache = append(w.cache, SecretEntry{
			Scope:       scope,
			Project:     project,
			Environment: env,
			Key:         cred.UserName,
		})
	}
}
