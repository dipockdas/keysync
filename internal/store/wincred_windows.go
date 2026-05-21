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
// Backward compatibility:
//   The parser accepts both v2 (tagged) and v1 (underscore-delimited) formats.
//   New credentials are always written in v2 format.
//   Legacy v1 format: keysync_<scope>_<project>_<environment>_<key>
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

// credTargetLegacy returns the credential target name in v1 underscore-delimited format
// Used as fallback when reading existing v1 credentials
func credTargetLegacy(scope Scope, project, environment, key string) string {
	if scope == ScopeGlobal {
		return fmt.Sprintf("keysync_global_%s", key)
	}
	// Project scope
	if environment == "" {
		return fmt.Sprintf("keysync_project_%s_%s", project, key)
	}
	return fmt.Sprintf("keysync_project_%s_%s_%s", project, environment, key)
}

// parseCredTarget parses both v2 (tagged) and v1 (legacy) formats
func parseCredTarget(target string) (Scope, string, string, string) {
	// Detect format
	if strings.HasPrefix(target, "keysync|") {
		return parseTaggedFormat(target)
	}

	// Legacy v1 format (underscore-delimited)
	// Also check for old "keysync_" prefix to handle v1
	if strings.HasPrefix(target, "keysync_") {
		return parseLegacyFormat(target)
	}

	return ScopeGlobal, "", "", ""
}

// parseTaggedFormat parses v2 format: keysync|s=...|p=...|e=...|k=...
func parseTaggedFormat(target string) (Scope, string, string, string) {
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

// parseLegacyFormat parses v1 format: keysync_scope_project_env_key
// This uses heuristics and is ambiguous for names with underscores,
// but provides backward compatibility for reading existing credentials.
func parseLegacyFormat(target string) (Scope, string, string, string) {
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
		key := strings.Join(parts[1:], "_") // Key may contain underscores
		return scope, "", "", key
	}

	// Project scope: keysync_project_PROJECT_KEY or keysync_project_PROJECT_ENV_KEY
	if len(parts) < 3 {
		return scope, "", "", ""
	}

	project := parts[1]

	// Heuristic: if 3 parts, no env; if 4+, parts[2] is env
	// This is ambiguous but maintains backward compatibility
	if len(parts) == 3 {
		key := parts[2]
		return scope, project, "", key
	}

	// 4+ parts: assume environment is parts[2], key is parts[3:]
	env := parts[2]
	key := strings.Join(parts[3:], "_")
	return scope, project, env, key
}

func (w *WincredStore) Get(_ context.Context, scope Scope, project, environment, key string) (string, error) {
	// Try v2 format first
	target := credTarget(scope, project, environment, key)
	cred, err := wincred.GetGenericCredential(target)
	if err == nil {
		return string(cred.CredentialBlob), nil
	}

	// Fall back to v1 format for backward compatibility
	targetLegacy := credTargetLegacy(scope, project, environment, key)
	cred, err = wincred.GetGenericCredential(targetLegacy)
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
	// Try v2 format first
	target := credTarget(scope, project, environment, key)
	cred, err := wincred.GetGenericCredential(target)
	if err != nil {
		// Fall back to v1 format for backward compatibility
		targetLegacy := credTargetLegacy(scope, project, environment, key)
		cred, err = wincred.GetGenericCredential(targetLegacy)
		if err != nil {
			return ErrNotFound
		}
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
		// Check for both v2 and v1 prefixes
		if !strings.HasPrefix(cred.TargetName, "keysync|") &&
			!strings.HasPrefix(cred.TargetName, "keysync_") {
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
