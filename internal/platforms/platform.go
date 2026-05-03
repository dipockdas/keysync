package platforms

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"

	"github.com/dipockdas/keysync/internal/store"
)

// sanitizeResponseBody masks secret values in API response bodies before including them in error messages.
// This prevents secrets from leaking via error messages if a platform echoes request payloads in its response.
func sanitizeResponseBody(body []byte) string {
	s := string(body)
	// Mask "value":"<anything>" patterns in JSON responses
	re := regexp.MustCompile(`"value"\s*:\s*"[^"]+"`)
	s = re.ReplaceAllString(s, `"value":"***MASKED***"`)
	// Mask general key=value patterns for sensitive field names
	re2 := regexp.MustCompile(`(?i)(secret|key|token|password|api_key)=[^\s&"]+`)
	s = re2.ReplaceAllString(s, `$1=***MASKED***`)
	// Truncate to first 512 chars to avoid huge error messages
	if len(s) > 512 {
		s = s[:512] + "..."
	}
	return s
}

// HTTPClient is an interface for making HTTP requests.
// It matches *http.Client so the real client can be used in production,
// and *httptest.Server.Client() can be used in tests.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Platform defines the interface for syncing secrets to a deployment platform.
// Each platform (Vercel, Railway, Supabase, etc.) implements this interface.
type Platform interface {
	// Name returns the platform name (e.g., "vercel", "railway").
	Name() string

	// Upsert creates or updates a single secret on the platform.
	Upsert(key, value string) error
}

// registry holds all registered platform constructors.
// Constructors receive a context, config JSON, and the secret store for token lookup.
var registry = map[string]func(ctx context.Context, configJSON string, secretSt store.Store) (Platform, error){}

// Register adds a platform constructor to the registry.
func Register(name string, fn func(ctx context.Context, configJSON string, secretSt store.Store) (Platform, error)) {
	registry[name] = fn
}

// Get creates a platform instance by name with the given JSON config and secret store.
// The store is used to look up the API token (e.g., VERCEL_TOKEN) as a global secret,
// falling back to environment variables.
func Get(ctx context.Context, name, configJSON string, secretSt store.Store) (Platform, error) {
	fn, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown platform: %s (available: %v)", name, available())
	}
	return fn(ctx, configJSON, secretSt)
}

// List returns all registered platform names.
func List() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	return names
}

func available() []string {
	return List()
}

// TokenEnvNames maps platform names to their environment variable and store key names.
var TokenEnvNames = map[string]string{
	"vercel":   "VERCEL_TOKEN",
	"railway":  "RAILWAY_TOKEN",
	"supabase": "SUPABASE_TOKEN",
}

// lookupToken resolves a platform API token from the secret store first,
// falling back to the environment variable.
func lookupToken(ctx context.Context, secretSt store.Store, platformName string) string {
	envName, ok := TokenEnvNames[platformName]
	if !ok {
		return ""
	}

	// Try the secret store first (global scope)
	if secretSt != nil {
		token, err := secretSt.Get(ctx, "global", "", "", envName)
		if err == nil && token != "" {
			return token
		}
	}

	// Fall back to env var
	return osGetenv(envName)
}

// osGetenv is a variable so tests can override it.
var osGetenv = getenv

func getenv(key string) string {
	return os.Getenv(key)
}
