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
//	dbURL, err := keysync.GetSecret("DATABASE_URL", "my-project", "production")
package keysync

import (
	"fmt"
	"os"
	"strings"
)

// GetSecret retrieves a secret.
//
// Checks the environment variable identified by *key* first. If set, returns
// it immediately without touching the OS keychain. This is the primary path
// for both local development (where secrets are injected via
// `eval $(keysync export)`) and cloud deployments (where platforms inject
// environment variables directly).
//
// If the env var is not set, checks the OS keychain in this order:
//   - project-scoped secret with environment (if both parameters are provided)
//   - project-scoped secret without environment (if project is provided)
//   - global-scoped secret
//
// Returns ErrNotFound if the secret doesn't exist in any checked scope.
func GetSecret(key, project, environment string) (string, error) {
	// Primary path: check environment variable first.
	// In local dev the user runs `eval $(keysync export)` at shell startup;
	// in cloud/CI the platform injects env vars directly.
	if val, ok := os.LookupEnv(key); ok {
		return val, nil
	}

	// If environment is provided, try project scope with environment first
	if project != "" && environment != "" {
		svc := serviceName("project", project, environment)
		val, err := platformGet(svc, key)
		if err == nil {
			return val, nil
		}
		if !isNotFound(err) {
			return "", fmt.Errorf("read env-scoped secret: %w", err)
		}
	}

	// Try project scope (no environment)
	if project != "" {
		svc := serviceName("project", project, "")
		val, err := platformGet(svc, key)
		if err == nil {
			return val, nil
		}
		if !isNotFound(err) {
			return "", fmt.Errorf("read project-scoped secret: %w", err)
		}
	}

	// Fall back to global scope
	svc := serviceName("global", "", "")
	val, err := platformGet(svc, key)
	if err != nil {
		if isNotFound(err) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("read global secret: %w", err)
	}
	return val, nil
}

// ListSecrets returns all stored secret entries matching the given scope,
// project, and environment. Empty scope, project, or environment means "any."
func ListSecrets(scope, project, environment string) ([]SecretEntry, error) {
	return platformList(scope, project, environment)
}

// SecretEntry represents a single secret key with its scope, project, and environment.
type SecretEntry struct {
	Scope       string `json:"scope"`
	Project     string `json:"project,omitempty"`
	Environment string `json:"environment,omitempty"`
	Key         string `json:"key"`
}

// ErrNotFound is returned when a secret is not found.
var ErrNotFound = fmt.Errorf("secret not found")

// serviceName builds the keychain service name.
// Global:        "keysync/global"
// Project:       "keysync/project/<name>"
// Project+Env:   "keysync/project/<name>/env/<env>"
func serviceName(scope, project, environment string) string {
	if scope == "global" || project == "" {
		return "keysync/" + scope
	}
	name := "keysync/" + scope + "/" + project
	if environment != "" {
		name += "/env/" + environment
	}
	return name
}

// parseServiceName splits a service name into scope, project, and environment.
// e.g. "keysync/project/myapp/env/dev" → ("project", "myapp", "dev")
//      "keysync/project/myapp"       → ("project", "myapp", "")
//      "keysync/global"             → ("global", "", "")
func parseServiceName(svc string) (scope, project, environment string) {
	if len(svc) < 8 || svc[:8] != "keysync/" {
		return "global", "", ""
	}
	trimmed := svc[8:]
	for i := 0; i < len(trimmed); i++ {
		if trimmed[i] == '/' {
			scope = trimmed[:i]
			rest := trimmed[i+1:]
			if scope != "project" {
				return scope, "", ""
			}
			// Look for the last "/env/" suffix to extract environment
			envIdx := strings.LastIndex(rest, "/env/")
			if envIdx >= 0 {
				project = rest[:envIdx]
				environment = rest[envIdx+len("/env/"):]
			} else {
				project = rest
			}
			return scope, project, environment
		}
	}
	return trimmed, "", ""
}

// Platform-specific variables set by build-tagged files.
var platformGet func(service, account string) (string, error)

var platformList func(scope, project, environment string) ([]SecretEntry, error)

var isNotFound func(error) bool
