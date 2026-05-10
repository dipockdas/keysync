// Package keysync provides access to secrets managed by keysync
// directly from the OS keychain, with no dependency on the keysync binary.
//
// Each platform uses its native keychain tooling:
//   - macOS:  security CLI (built-in)
//   - Linux:  secret-tool CLI (libsecret)
//   - Windows: wincred Go library (Win32 API)
//
// Usage:
//
//	import "github.com/dipockdas/keysync/clients/go"
//
//	dbURL, err := keysync.GetSecret("DATABASE_URL", "my-project")
package keysync

import (
	"fmt"
	"os"
)

// GetSecret retrieves a secret.
//
// Checks the environment variable identified by *key* first. If set, returns
// it immediately without touching the OS keychain. This is the primary path
// for both local development (where secrets are injected via
// `eval $(keysync export)`) and cloud deployments (where platforms inject
// environment variables directly).
//
// If the env var is not set, falls back to the OS keychain. When *project* is
// non-empty it checks project scope first, then global scope.
//
// Returns ErrNotFound if the secret doesn't exist in any checked scope.
func GetSecret(key, project string) (string, error) {
	// Primary path: check environment variable first.
	// In local dev the user runs `eval $(keysync export)` at shell startup;
	// in cloud/CI the platform injects env vars directly.
	if val, ok := os.LookupEnv(key); ok {
		return val, nil
	}

	// Try project scope first
	if project != "" {
		svc := serviceName("project", project)
		val, err := platformGet(svc, key)
		if err == nil {
			return val, nil
		}
		if !isNotFound(err) {
			return "", fmt.Errorf("project scope: %w", err)
		}
	}

	// Fall back to global scope
	svc := serviceName("global", "")
	val, err := platformGet(svc, key)
	if err != nil {
		if isNotFound(err) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("global scope: %w", err)
	}
	return val, nil
}

// ListSecrets returns all stored secret entries matching the given scope
// and project. Empty scope or project means "any."
func ListSecrets(scope, project string) ([]SecretEntry, error) {
	return platformList(scope, project)
}

// SecretEntry represents a single secret key with its scope and project.
type SecretEntry struct {
	Scope   string `json:"scope"`
	Project string `json:"project,omitempty"`
	Key     string `json:"key"`
}

// ErrNotFound is returned when a secret is not found.
var ErrNotFound = fmt.Errorf("secret not found")

// serviceName builds the keychain service name.
// Global:  "keysync/global"
// Project: "keysync/project/<name>"
func serviceName(scope, project string) string {
	if project == "" || scope == "global" {
		return "keysync/" + scope
	}
	return "keysync/" + scope + "/" + project
}

// parseServiceName splits a service name back into scope and project.
func parseServiceName(svc string) (scope, project string) {
	if len(svc) < 8 || svc[:8] != "keysync/" {
		return "global", ""
	}
	trimmed := svc[8:]
	for i := 0; i < len(trimmed); i++ {
		if trimmed[i] == '/' {
			scope = trimmed[:i]
			if scope != "project" {
				return scope, ""
			}
			return scope, trimmed[i+1:]
		}
	}
	return trimmed, ""
}

// Platform-specific variables set by build-tagged files.
var platformGet func(service, account string) (string, error)

var platformList func(scope, project string) ([]SecretEntry, error)

var isNotFound func(error) bool
