//go:build windows

package store

import (
	"context"
	"strings"
	"testing"

	"github.com/danieljoos/wincred"
)

// Test 1: Verify | and = are safe in Windows credential targets
func TestWincredTargetNameCharacters(t *testing.T) {
	testTarget := "keysync|s=global|p=|e=|k=TEST_CHAR_SAFETY"
	cred := wincred.NewGenericCredential(testTarget)
	cred.CredentialBlob = []byte("test_value")

	if err := cred.Write(); err != nil {
		t.Fatalf("Write with | and = failed: %v", err)
	}
	defer cred.Delete()

	retrieved, err := wincred.GetGenericCredential(testTarget)
	if err != nil {
		t.Fatalf("Get with | and = failed: %v", err)
	}

	if retrieved.TargetName != testTarget {
		t.Errorf("Target roundtrip failed: got %q, want %q", retrieved.TargetName, testTarget)
	}
}

// Test 2: New format round-trip
func TestCredTarget_NewFormat_RoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		scope Scope
		proj  string
		env   string
		key   string
	}{
		{"global", ScopeGlobal, "", "", "API_KEY"},
		{"project-simple", ScopeProject, "my-app", "", "DATABASE_URL"},
		{"project-underscores", ScopeProject, "my_app", "", "DATABASE_URL"},
		{"env-underscores", ScopeProject, "my_app", "prod_us", "STRIPE_KEY"},
		{"complex", ScopeProject, "api_v2", "prod_us_east_1", "AWS_SECRET_KEY"},
		{"special-chars", ScopeProject, "my:app", "env=test", "KEY|VALUE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := credTarget(tt.scope, tt.proj, tt.env, tt.key)
			scope, proj, env, key := parseCredTarget(target)

			if scope != tt.scope {
				t.Errorf("scope: got %v, want %v", scope, tt.scope)
			}
			if proj != tt.proj {
				t.Errorf("project: got %q, want %q", proj, tt.proj)
			}
			if env != tt.env {
				t.Errorf("env: got %q, want %q", env, tt.env)
			}
			if key != tt.key {
				t.Errorf("key: got %q, want %q", key, tt.key)
			}
		})
	}
}

// Test 3: End-to-end with WincredStore - underscores in all components
func TestWincredStore_UnderscoresInAllComponents(t *testing.T) {
	st := NewWincredStore()
	ctx := context.Background()

	// Clean up any test credentials from previous runs
	defer func() {
		st.Delete(ctx, ScopeProject, "my_app", "", "DATABASE_URL")
		st.Delete(ctx, ScopeProject, "my_app", "prod_us", "STRIPE_KEY")
	}()

	// Test underscores in project name
	if err := st.Set(ctx, ScopeProject, "my_app", "", "DATABASE_URL", "postgres://test"); err != nil {
		t.Fatalf("Set with project underscores failed: %v", err)
	}

	val, err := st.Get(ctx, ScopeProject, "my_app", "", "DATABASE_URL")
	if err != nil {
		t.Fatalf("Get with project underscores failed: %v", err)
	}
	if val != "postgres://test" {
		t.Errorf("value = %q, want %q", val, "postgres://test")
	}

	// Test underscores in environment name
	if err := st.Set(ctx, ScopeProject, "my_app", "prod_us", "STRIPE_KEY", "sk_test"); err != nil {
		t.Fatalf("Set with env underscores failed: %v", err)
	}

	val, err = st.Get(ctx, ScopeProject, "my_app", "prod_us", "STRIPE_KEY")
	if err != nil {
		t.Fatalf("Get with env underscores failed: %v", err)
	}
	if val != "sk_test" {
		t.Errorf("value = %q, want %q", val, "sk_test")
	}
}

// Test 5: Cache rebuild with new format
func TestWincredStore_CacheRebuildAfterSet(t *testing.T) {
	st := NewWincredStore()
	ctx := context.Background()

	// Store secrets with underscores
	testCases := []struct {
		project string
		env     string
		key     string
		value   string
	}{
		{"my_app", "", "DATABASE_URL", "postgres://test1"},
		{"my_app", "prod_us", "DATABASE_URL", "postgres://test2"},
		{"api_v2", "", "STRIPE_KEY", "sk_test"},
	}

	// Clean up at end
	defer func() {
		for _, tc := range testCases {
			st.Delete(ctx, ScopeProject, tc.project, tc.env, tc.key)
		}
	}()

	for _, tc := range testCases {
		if err := st.Set(ctx, ScopeProject, tc.project, tc.env, tc.key, tc.value); err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	// Force cache rebuild
	st.rebuildCache()

	// Verify all entries are correctly parsed
	for _, tc := range testCases {
		entries, err := st.List(ctx, ScopeProject, tc.project, tc.env)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}

		found := false
		for _, e := range entries {
			if e.Key == tc.key && e.Project == tc.project && e.Environment == tc.env {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("After rebuild: key %q in project %q env %q not found", tc.key, tc.project, tc.env)
		}
	}
}

// Test 6: Character length limit (256 chars)
func TestCredTarget_CharacterLimit(t *testing.T) {
	// Windows credential targets max out at 256 characters
	// Test that we handle this gracefully

	// Create reasonably long names (not extreme, but realistic)
	longProject := strings.Repeat("a", 50)
	longEnv := strings.Repeat("b", 50)
	longKey := strings.Repeat("c", 50)

	target := credTarget(ScopeProject, longProject, longEnv, longKey)

	if len(target) > 256 {
		t.Errorf("Target length %d exceeds Windows 256-char limit", len(target))
	}

	// Verify round-trip
	_, proj, env, key := parseCredTarget(target)
	if proj != longProject || env != longEnv || key != longKey {
		t.Errorf("Long name round-trip failed")
	}
}

// Test 7: Complex scenario - project, env, and key all with underscores
func TestWincredStore_ComplexUnderscoreScenario(t *testing.T) {
	st := NewWincredStore()
	ctx := context.Background()

	// Project with underscores, env with underscores, key with underscores
	project := "my_api_v2"
	env := "prod_us_east_1"
	key := "AWS_SECRET_ACCESS_KEY"
	value := "super_secret_value"

	defer st.Delete(ctx, ScopeProject, project, env, key)

	if err := st.Set(ctx, ScopeProject, project, env, key, value); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Rebuild cache and verify
	st.rebuildCache()

	got, err := st.Get(ctx, ScopeProject, project, env, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got != value {
		t.Errorf("got %q, want %q", got, value)
	}

	// Verify List returns correct scope metadata
	entries, err := st.List(ctx, ScopeProject, project, env)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("List returned no entries")
	}

	found := false
	for _, e := range entries {
		if e.Key == key {
			found = true
			if e.Project != project {
				t.Errorf("project = %q, want %q", e.Project, project)
			}
			if e.Environment != env {
				t.Errorf("environment = %q, want %q", e.Environment, env)
			}
			break
		}
	}
	if !found {
		t.Errorf("Key %q not found in list", key)
	}
}

// Test 8: Multiple secrets in same scope (Finding 3 verification)
func TestWincredStore_MultipleSecretsInSameScope(t *testing.T) {
	st := NewWincredStore()
	ctx := context.Background()

	project := "test_app"
	secrets := map[string]string{
		"DATABASE_URL":  "postgres://test",
		"STRIPE_KEY":    "sk_test_123",
		"SENDGRID_KEY":  "SG.test456",
		"AWS_ACCESS_ID": "AKIA_TEST",
	}

	defer func() {
		for key := range secrets {
			st.Delete(ctx, ScopeProject, project, "", key)
		}
	}()

	// Store all secrets
	for key, value := range secrets {
		if err := st.Set(ctx, ScopeProject, project, "", key, value); err != nil {
			t.Fatalf("Set %s failed: %v", key, err)
		}
	}

	// List should return all 4
	entries, err := st.List(ctx, ScopeProject, project, "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Count how many test_app secrets we found
	count := 0
	for _, e := range entries {
		if e.Project == project {
			count++
		}
	}

	if count < len(secrets) {
		t.Errorf("List returned %d entries for project %q, want at least %d", count, project, len(secrets))
	}

	// Get each individually
	for key, expectedValue := range secrets {
		val, err := st.Get(ctx, ScopeProject, project, "", key)
		if err != nil {
			t.Errorf("Get(%s) failed: %v", key, err)
		}
		if val != expectedValue {
			t.Errorf("Get(%s) = %q, want %q", key, val, expectedValue)
		}
	}
}

// Test 9: Encoding edge cases
func TestEncodeComponent_EdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		wantDiff bool // Should encoding produce different output?
	}{
		{"simple", false},      // No special chars
		{"my-app", false},      // Hyphens OK
		{"my_app", false},      // Underscores NOT encoded (unreserved)
		{"my|app", true},       // Pipe encoded
		{"my=app", true},       // Equals encoded
		{"", false},            // Empty OK
		{"123", false},         // Numbers OK
		{"my.app", false},      // Period OK
		{"my app", true},       // Space encoded
		{"my:app", true},       // Colon encoded
		{"api_v2_prod", false}, // Multiple underscores NOT encoded
	}

	for _, tt := range tests {
		encoded := encodeComponent(tt.input)
		decoded, err := decodeComponent(encoded)
		if err != nil {
			t.Errorf("decodeComponent(%q) failed: %v", encoded, err)
		}
		if decoded != tt.input {
			t.Errorf("round-trip failed: %q → %q → %q", tt.input, encoded, decoded)
		}

		isDifferent := encoded != tt.input
		if isDifferent != tt.wantDiff {
			t.Errorf("encodeComponent(%q) = %q, encoding changed=%v, want changed=%v",
				tt.input, encoded, isDifferent, tt.wantDiff)
		}
	}
}

// Test 10: Format detection
func TestParseCredTarget_FormatDetection(t *testing.T) {
	tests := []struct {
		name   string
		target string
		valid  bool
	}{
		{"v2-global", "keysync|s=global|p=|e=|k=API_KEY", true},
		{"v2-project", "keysync|s=project|p=myapp|e=|k=KEY", true},
		{"invalid-old-format", "keysync_global_API_KEY", false},
		{"invalid-random", "something_else", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scope, _, _, key := parseCredTarget(tt.target)

			if tt.valid {
				// Valid v2 formats should extract both scope and key
				if (scope != ScopeGlobal && scope != ScopeProject) || key == "" {
					t.Errorf("parseCredTarget(%q) should return valid scope and non-empty key", tt.target)
				}
			} else {
				// Invalid formats should return empty key
				if key != "" {
					t.Errorf("parseCredTarget(%q) should return empty key for invalid format, got %q", tt.target, key)
				}
			}
		})
	}
}

// Test 11: Global scope with new format
func TestWincredStore_GlobalScope_NewFormat(t *testing.T) {
	st := NewWincredStore()
	ctx := context.Background()

	// Clean up
	defer func() {
		st.Delete(ctx, ScopeGlobal, "", "", "GLOBAL_KEY_1")
		st.Delete(ctx, ScopeGlobal, "", "", "GLOBAL_KEY_2")
	}()

	// Store multiple global secrets
	if err := st.Set(ctx, ScopeGlobal, "", "", "GLOBAL_KEY_1", "value1"); err != nil {
		t.Fatalf("Set GLOBAL_KEY_1 failed: %v", err)
	}
	if err := st.Set(ctx, ScopeGlobal, "", "", "GLOBAL_KEY_2", "value2"); err != nil {
		t.Fatalf("Set GLOBAL_KEY_2 failed: %v", err)
	}

	// List global secrets
	entries, err := st.List(ctx, ScopeGlobal, "", "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Verify both are present
	found := make(map[string]bool)
	for _, e := range entries {
		if e.Scope == ScopeGlobal {
			found[e.Key] = true
		}
	}

	if !found["GLOBAL_KEY_1"] || !found["GLOBAL_KEY_2"] {
		t.Errorf("global secrets not found: %v", found)
	}
}

// Test 12: Delete doesn't affect other secrets
func TestWincredStore_DeleteOneSecret(t *testing.T) {
	st := NewWincredStore()
	ctx := context.Background()

	// Clean up
	defer func() {
		st.Delete(ctx, ScopeProject, "test-app", "", "KEY_A")
		st.Delete(ctx, ScopeProject, "test-app", "", "KEY_B")
		st.Delete(ctx, ScopeProject, "test-app", "", "KEY_C")
	}()

	// Store 3 secrets
	st.Set(ctx, ScopeProject, "test-app", "", "KEY_A", "value-a")
	st.Set(ctx, ScopeProject, "test-app", "", "KEY_B", "value-b")
	st.Set(ctx, ScopeProject, "test-app", "", "KEY_C", "value-c")

	// Delete KEY_B
	if err := st.Delete(ctx, ScopeProject, "test-app", "", "KEY_B"); err != nil {
		t.Fatalf("Delete KEY_B failed: %v", err)
	}

	// Verify KEY_B is gone
	_, err := st.Get(ctx, ScopeProject, "test-app", "", "KEY_B")
	if err != ErrNotFound {
		t.Errorf("KEY_B should be deleted, got err: %v", err)
	}

	// Verify KEY_A and KEY_C still exist
	val, err := st.Get(ctx, ScopeProject, "test-app", "", "KEY_A")
	if err != nil {
		t.Errorf("Get KEY_A failed: %v", err)
	} else if val != "value-a" {
		t.Errorf("KEY_A = %q, want %q", val, "value-a")
	}

	val, err = st.Get(ctx, ScopeProject, "test-app", "", "KEY_C")
	if err != nil {
		t.Errorf("Get KEY_C failed: %v", err)
	} else if val != "value-c" {
		t.Errorf("KEY_C = %q, want %q", val, "value-c")
	}
}
