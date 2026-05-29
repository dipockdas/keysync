//go:build windows

package store

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/danieljoos/wincred"
)

// Windows Credential Target Format Specification (v2)
//
// Keysync stores secrets in Windows Credential Manager using a tagged-field format
// with percent-encoded values. This format is deterministic, reversible, and supports
// special characters (underscores, spaces, symbols) in all components.
//
// Format:
//   keysync|s=<scope>|p=<project>|e=<environment>|k=<key>
//
// Where:
//   - Separator: | (pipe character, ASCII 124)
//   - Field separator: = (equals, ASCII 61)
//   - Value encoding: url.QueryEscape (Go stdlib, RFC 3986-based)
//   - Empty values: field present with empty value (e.g., "e=" for no environment)
//
// Encoding rules (per url.QueryEscape):
//   - Unreserved characters (A-Z, a-z, 0-9, -, ., _, ~) are NOT encoded
//   - Separators (|, =) are percent-encoded: | → %7C, = → %3D
//   - Spaces are encoded as + (not %20)
//   - All other characters are percent-encoded
//
// Examples:
//   Global scope:
//     keysync|s=global|p=|e=|k=API_KEY
//
//   Project scope (hyphens not encoded):
//     keysync|s=project|p=my-app|e=|k=DATABASE_URL
//
//   Project scope (underscores not encoded):
//     keysync|s=project|p=my_app|e=|k=DATABASE_URL
//
//   Project+Environment (underscores not encoded):
//     keysync|s=project|p=api_v2|e=prod_us_east|k=STRIPE_KEY
//
//   Special characters (pipes and equals encoded):
//     keysync|s=project|p=my%7Capp|e=env%3Dtest|k=KEY
//
//
// Character limit:
//   Windows credential targets have a 256-character maximum. Very long project,
//   environment, or key names may exceed this limit. Percent encoding adds ~40%
//   overhead for names with many special characters.
//
// Wire format version: 2
// Introduced: keysync 1.0.0
// Stability: This format is considered stable and will be supported indefinitely.

// WincredStore implements Store using Windows Credential Manager
// via the Win32 API (github.com/danieljoos/wincred).
type WincredStore struct {
	mu    sync.RWMutex
	cache []SecretEntry // cached list, rebuilt on demand
}

func NewWincredStore() *WincredStore {
	ws := &WincredStore{}
	ws.rebuildCache()
	return ws
}

// encodeComponent percent-encodes a component for safe inclusion in target name
func encodeComponent(s string) string {
	if s == "" {
		return ""
	}
	// Use url.QueryEscape which implements RFC 3986 percent encoding
	return url.QueryEscape(s)
}

// decodeComponent percent-decodes a component
func decodeComponent(s string) (string, error) {
	return url.QueryUnescape(s)
}

// credTarget returns the credential target name in v2 tagged format
func credTarget(scope Scope, project, environment, key string) string {
	return fmt.Sprintf("keysync|s=%s|p=%s|e=%s|k=%s",
		scope,
		encodeComponent(project),
		encodeComponent(environment),
		encodeComponent(key))
}

// parseCredTarget parses v2 tagged format: keysync|s=...|p=...|e=...|k=...
func parseCredTarget(target string) (Scope, string, string, string) {
	// Only accept v2 format
	if !strings.HasPrefix(target, "keysync|") {
		return ScopeGlobal, "", "", ""
	}

	// Remove "keysync|" prefix
	trimmed := strings.TrimPrefix(target, "keysync|")

	// Split into field=value pairs
	fields := make(map[string]string)
	for _, pair := range strings.Split(trimmed, "|") {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			fields[parts[0]] = parts[1]
		}
	}

	// Extract and decode scope (not encoded, just a fixed value)
	scope := Scope(fields["s"])

	// Decode project, environment, key
	project, err := decodeComponent(fields["p"])
	if err != nil {
		// If decode fails, use raw value (should not happen with valid data)
		project = fields["p"]
	}

	env, err := decodeComponent(fields["e"])
	if err != nil {
		env = fields["e"]
	}

	key, err := decodeComponent(fields["k"])
	if err != nil {
		key = fields["k"]
	}

	return scope, project, env, key
}

func (w *WincredStore) Get(_ context.Context, scope Scope, project, environment, key string) (string, error) {
	target := credTarget(scope, project, environment, key)
	cred, err := wincred.GetGenericCredential(target)
	if err != nil {
		return "", ErrNotFound
	}
	return string(cred.CredentialBlob), nil
}

func (w *WincredStore) Set(_ context.Context, scope Scope, project, environment, key, value string) error {
	target := credTarget(scope, project, environment, key)

	// Delete existing first to avoid duplicates
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
		// Only check for v2 format
		if !strings.HasPrefix(cred.TargetName, "keysync|") {
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
