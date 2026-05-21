package platforms

import (
	"context"
	"testing"

	"github.com/dipockdas/keysync/internal/store"
)

func TestRegisterAndGet(t *testing.T) {
	// Save and restore registry
	saved := registry
	registry = map[string]func(ctx context.Context, configJSON string, secretSt store.Store) (Platform, error){}
	defer func() { registry = saved }()

	Register("test", func(ctx context.Context, configJSON string, secretSt store.Store) (Platform, error) {
		return &testPlatform{name: configJSON}, nil
	})

	p, err := Get(context.Background(), "test", "my-config", nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if p.Name() != "my-config" {
		t.Errorf("Name() = %q, want %q", p.Name(), "my-config")
	}
}

func TestGet_Unknown(t *testing.T) {
	_, err := Get(context.Background(), "nonexistent", "", nil)
	if err == nil {
		t.Fatal("expected error for unknown platform, got nil")
	}
}

func TestList(t *testing.T) {
	// Save and restore registry
	saved := registry
	registry = map[string]func(ctx context.Context, configJSON string, secretSt store.Store) (Platform, error){}
	defer func() { registry = saved }()

	Register("a", func(context.Context, string, store.Store) (Platform, error) { return &testPlatform{}, nil })
	Register("b", func(context.Context, string, store.Store) (Platform, error) { return &testPlatform{}, nil })

	names := List()
	if len(names) != 2 {
		t.Errorf("List() returned %d names, want 2", len(names))
	}
	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	if !found["a"] || !found["b"] {
		t.Errorf("List() = %v, missing expected entries", names)
	}
}

type testPlatform struct {
	name string
}

func (t *testPlatform) Name() string {
	if t.name == "" {
		return "test"
	}
	return t.name
}

func (t *testPlatform) Upsert(ctx context.Context, key, value string) error {
	return nil
}

// ---------------------------------------------------------------------------
// sanitizeResponseBody tests
// ---------------------------------------------------------------------------

func TestSanitizeResponseBody_JSONValueMasking(t *testing.T) {
	input := []byte(`{"key":"MY_SECRET","value":"super-secret-value","type":"encrypted"}`)
	got := sanitizeResponseBody(input)
	if contains(got, "super-secret-value") {
		t.Errorf("response body still contains secret value: %s", got)
	}
	if !contains(got, "***MASKED***") {
		t.Errorf("response body missing MASKED marker: %s", got)
	}
}

func TestSanitizeResponseBody_NoSecrets(t *testing.T) {
	input := []byte(`{"status":"ok","project":"my-app"}`)
	got := sanitizeResponseBody(input)
	// Should remain unchanged
	if got != string(input) {
		t.Errorf("sanitized = %q, want %q", got, string(input))
	}
}

func TestSanitizeResponseBody_EmptyBody(t *testing.T) {
	got := sanitizeResponseBody([]byte{})
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestSanitizeResponseBody_URLParamMasking(t *testing.T) {
	input := []byte(`error: secret=mysupersecret&other=param`)
	got := sanitizeResponseBody(input)
	if contains(got, "mysupersecret") {
		t.Errorf("response body still contains secret value: %s", got)
	}
	if !contains(got, "secret=***MASKED***") {
		t.Errorf("response body missing MASKED marker: %s", got)
	}
}

func TestSanitizeResponseBody_Truncation(t *testing.T) {
	// Build body > 512 chars with no secrets
	body := make([]byte, 600)
	for i := range body {
		body[i] = 'x'
	}
	got := sanitizeResponseBody(body)
	if len(got) > 520 {
		t.Errorf("sanitized body length = %d, want <= 520 (512 + ...)", len(got))
	}
}

func TestSanitizeResponseBody_BelowTruncationThreshold(t *testing.T) {
	body := []byte(`{"key":"val"}`)
	got := sanitizeResponseBody(body)
	if contains(got, "...") {
		t.Errorf("body was truncated but is under threshold: %s", got)
	}
}

func TestSanitizeResponseBody_MultipleSensitiveFields(t *testing.T) {
	input := []byte(`error: token=abc123&key=def456&password=ghi789&api_key=jkl012`)
	got := sanitizeResponseBody(input)
	if contains(got, "abc123") || contains(got, "def456") {
		t.Errorf("sensitive values still present: %s", got)
	}
	if !contains(got, "token=***MASKED***") {
		t.Errorf("token not masked: %s", got)
	}
	if !contains(got, "key=***MASKED***") {
		t.Errorf("key not masked: %s", got)
	}
	if !contains(got, "password=***MASKED***") {
		t.Errorf("password not masked: %s", got)
	}
	if !contains(got, "api_key=***MASKED***") {
		t.Errorf("api_key not masked: %s", got)
	}
}

// ---------------------------------------------------------------------------
// lookupToken tests
// ---------------------------------------------------------------------------

func TestLookupToken_FromStore(t *testing.T) {
	ctx := context.Background()
	secretSt := store.NewMemoryStore()
	secretSt.Set(ctx, store.ScopeGlobal, "", "", "VERCEL_TOKEN", "store-token-abc")

	token := lookupToken(ctx, secretSt, "vercel")
	if token != "store-token-abc" {
		t.Errorf("lookupToken = %q, want %q", token, "store-token-abc")
	}
}

func TestLookupToken_FromEnvVarFallback(t *testing.T) {
	// Override osGetenv so no env var is set
	orig := osGetenv
	osGetenv = func(key string) string {
		if key == "VERCEL_TOKEN" {
			return "env-token-xyz"
		}
		return ""
	}
	defer func() { osGetenv = orig }()

	token := lookupToken(context.Background(), nil, "vercel")
	if token != "env-token-xyz" {
		t.Errorf("lookupToken = %q, want %q", token, "env-token-xyz")
	}
}

func TestLookupToken_StoreTakesPrecedence(t *testing.T) {
	// Even when env var is set, store should take precedence
	orig := osGetenv
	osGetenv = func(key string) string {
		if key == "VERCEL_TOKEN" {
			return "env-token"
		}
		return ""
	}
	defer func() { osGetenv = orig }()

	ctx := context.Background()
	secretSt := store.NewMemoryStore()
	secretSt.Set(ctx, store.ScopeGlobal, "", "", "VERCEL_TOKEN", "store-token")

	token := lookupToken(ctx, secretSt, "vercel")
	if token != "store-token" {
		t.Errorf("lookupToken = %q, want %q (store should take precedence)", token, "store-token")
	}
}

func TestLookupToken_EmptyStoreValueFallsBack(t *testing.T) {
	orig := osGetenv
	osGetenv = func(key string) string {
		if key == "RAILWAY_TOKEN" {
			return "railway-env-token"
		}
		return ""
	}
	defer func() { osGetenv = orig }()

	// Store has the key but with empty value — should fall through to env
	ctx := context.Background()
	secretSt := store.NewMemoryStore()
	secretSt.Set(ctx, store.ScopeGlobal, "", "", "RAILWAY_TOKEN", "")

	token := lookupToken(ctx, secretSt, "railway")
	if token != "railway-env-token" {
		t.Errorf("lookupToken = %q, want %q (empty store value should fallback)", token, "railway-env-token")
	}
}

func TestLookupToken_NeitherStoreNorEnv(t *testing.T) {
	orig := osGetenv
	osGetenv = func(string) string { return "" }
	defer func() { osGetenv = orig }()

	token := lookupToken(context.Background(), store.NewMemoryStore(), "vercel")
	if token != "" {
		t.Errorf("lookupToken = %q, want empty", token)
	}
}

func TestLookupToken_UnknownPlatform(t *testing.T) {
	token := lookupToken(context.Background(), nil, "nonexistent")
	if token != "" {
		t.Errorf("lookupToken = %q, want empty", token)
	}
}

func TestLookupToken_StoreGetError(t *testing.T) {
	orig := osGetenv
	osGetenv = func(key string) string {
		if key == "SUPABASE_TOKEN" {
			return "supabase-env-token"
		}
		return ""
	}
	defer func() { osGetenv = orig }()

	// Store is nil so Get will not be called — should fall back to env
	token := lookupToken(context.Background(), nil, "supabase")
	if token != "supabase-env-token" {
		t.Errorf("lookupToken = %q, want %q", token, "supabase-env-token")
	}
}

// contains is a small helper for substring check.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

// containsStr is the actual check split to avoid inline allocation lint noise.
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
