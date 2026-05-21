//go:build windows

package store

import (
	"context"
	"testing"
)

// TestWincredStore_MultipleSecretsInSameScope verifies that multiple secrets
// can coexist in the same scope/project/environment. This is the test for
// Finding 3 of the audit report.
func TestWincredStore_MultipleSecretsInSameScope(t *testing.T) {
	st := NewWincredStore()
	ctx := context.Background()

	// Clean up any existing test secrets
	defer func() {
		st.Delete(ctx, ScopeProject, "test-app", "", "DATABASE_URL")
		st.Delete(ctx, ScopeProject, "test-app", "", "STRIPE_KEY")
		st.Delete(ctx, ScopeProject, "test-app", "", "SENDGRID_KEY")
	}()

	// Store multiple secrets in the same scope
	if err := st.Set(ctx, ScopeProject, "test-app", "", "DATABASE_URL", "postgres://localhost/testdb"); err != nil {
		t.Fatalf("Set DATABASE_URL failed: %v", err)
	}
	if err := st.Set(ctx, ScopeProject, "test-app", "", "STRIPE_KEY", "sk_test_123456"); err != nil {
		t.Fatalf("Set STRIPE_KEY failed: %v", err)
	}
	if err := st.Set(ctx, ScopeProject, "test-app", "", "SENDGRID_KEY", "SG.abcdef"); err != nil {
		t.Fatalf("Set SENDGRID_KEY failed: %v", err)
	}

	// List should return all 3 secrets
	entries, err := st.List(ctx, ScopeProject, "test-app", "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Verify all 3 secrets are listed
	found := make(map[string]bool)
	for _, e := range entries {
		found[e.Key] = true
	}

	if !found["DATABASE_URL"] {
		t.Error("DATABASE_URL not found - Finding 3 NOT FIXED")
	}
	if !found["STRIPE_KEY"] {
		t.Error("STRIPE_KEY not found - Finding 3 NOT FIXED")
	}
	if !found["SENDGRID_KEY"] {
		t.Error("SENDGRID_KEY not found - Finding 3 NOT FIXED")
	}

	if len(entries) < 3 {
		t.Errorf("List returned %d entries, want at least 3 - Finding 3 NOT FIXED", len(entries))
	}

	// Get each secret individually and verify values
	val, err := st.Get(ctx, ScopeProject, "test-app", "", "DATABASE_URL")
	if err != nil {
		t.Errorf("Get DATABASE_URL failed: %v", err)
	} else if val != "postgres://localhost/testdb" {
		t.Errorf("DATABASE_URL = %q, want %q", val, "postgres://localhost/testdb")
	}

	val, err = st.Get(ctx, ScopeProject, "test-app", "", "STRIPE_KEY")
	if err != nil {
		t.Errorf("Get STRIPE_KEY failed: %v", err)
	} else if val != "sk_test_123456" {
		t.Errorf("STRIPE_KEY = %q, want %q", val, "sk_test_123456")
	}

	val, err = st.Get(ctx, ScopeProject, "test-app", "", "SENDGRID_KEY")
	if err != nil {
		t.Errorf("Get SENDGRID_KEY failed: %v", err)
	} else if val != "SG.abcdef" {
		t.Errorf("SENDGRID_KEY = %q, want %q", val, "SG.abcdef")
	}

	// Verify secrets don't overwrite each other
	// Update one secret
	if err := st.Set(ctx, ScopeProject, "test-app", "", "DATABASE_URL", "postgres://updated"); err != nil {
		t.Fatalf("Update DATABASE_URL failed: %v", err)
	}

	// Verify the other secrets are unchanged
	val, err = st.Get(ctx, ScopeProject, "test-app", "", "STRIPE_KEY")
	if err != nil {
		t.Errorf("Get STRIPE_KEY after update failed: %v", err)
	} else if val != "sk_test_123456" {
		t.Errorf("STRIPE_KEY changed after updating DATABASE_URL - Finding 3 NOT FIXED")
	}

	// Verify the updated secret has new value
	val, err = st.Get(ctx, ScopeProject, "test-app", "", "DATABASE_URL")
	if err != nil {
		t.Errorf("Get DATABASE_URL after update failed: %v", err)
	} else if val != "postgres://updated" {
		t.Errorf("DATABASE_URL = %q, want %q", val, "postgres://updated")
	}
}

// TestWincredStore_GlobalScope verifies multiple secrets work in global scope.
func TestWincredStore_GlobalScope(t *testing.T) {
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

// TestWincredStore_EnvironmentScope verifies multiple secrets work in project+env scope.
func TestWincredStore_EnvironmentScope(t *testing.T) {
	st := NewWincredStore()
	ctx := context.Background()

	// Clean up
	defer func() {
		st.Delete(ctx, ScopeProject, "test-app", "production", "API_KEY")
		st.Delete(ctx, ScopeProject, "test-app", "production", "DB_URL")
		st.Delete(ctx, ScopeProject, "test-app", "staging", "API_KEY")
	}()

	// Store multiple secrets in production environment
	if err := st.Set(ctx, ScopeProject, "test-app", "production", "API_KEY", "prod-key"); err != nil {
		t.Fatalf("Set production API_KEY failed: %v", err)
	}
	if err := st.Set(ctx, ScopeProject, "test-app", "production", "DB_URL", "prod-db"); err != nil {
		t.Fatalf("Set production DB_URL failed: %v", err)
	}

	// Store secret in staging environment (same key name, different env)
	if err := st.Set(ctx, ScopeProject, "test-app", "staging", "API_KEY", "staging-key"); err != nil {
		t.Fatalf("Set staging API_KEY failed: %v", err)
	}

	// List production secrets
	prodEntries, err := st.List(ctx, ScopeProject, "test-app", "production")
	if err != nil {
		t.Fatalf("List production failed: %v", err)
	}

	// Verify production has 2 secrets
	if len(prodEntries) < 2 {
		t.Errorf("production has %d secrets, want at least 2", len(prodEntries))
	}

	// Verify staging has its secret
	stagingEntries, err := st.List(ctx, ScopeProject, "test-app", "staging")
	if err != nil {
		t.Fatalf("List staging failed: %v", err)
	}

	if len(stagingEntries) < 1 {
		t.Errorf("staging has %d secrets, want at least 1", len(stagingEntries))
	}

	// Verify production API_KEY has correct value
	val, err := st.Get(ctx, ScopeProject, "test-app", "production", "API_KEY")
	if err != nil {
		t.Errorf("Get production API_KEY failed: %v", err)
	} else if val != "prod-key" {
		t.Errorf("production API_KEY = %q, want %q", val, "prod-key")
	}

	// Verify staging API_KEY has different value
	val, err = st.Get(ctx, ScopeProject, "test-app", "staging", "API_KEY")
	if err != nil {
		t.Errorf("Get staging API_KEY failed: %v", err)
	} else if val != "staging-key" {
		t.Errorf("staging API_KEY = %q, want %q", val, "staging-key")
	}
}

// TestWincredStore_DeleteOneSecret verifies deleting one secret doesn't affect others.
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

	// List should return 2 entries
	entries, err := st.List(ctx, ScopeProject, "test-app", "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	count := 0
	for _, e := range entries {
		if e.Project == "test-app" && e.Environment == "" {
			count++
		}
	}

	if count != 2 {
		t.Errorf("List returned %d entries for test-app, want 2", count)
	}
}

// TestCredTarget verifies the credential target format includes the key.
func TestCredTarget(t *testing.T) {
	tests := []struct {
		name        string
		scope       Scope
		project     string
		environment string
		key         string
		want        string
	}{
		{
			name:  "global scope",
			scope: ScopeGlobal,
			key:   "API_KEY",
			want:  "keysync_global_API_KEY",
		},
		{
			name:    "project scope",
			scope:   ScopeProject,
			project: "my-app",
			key:     "DATABASE_URL",
			want:    "keysync_project_my-app_DATABASE_URL",
		},
		{
			name:        "project + environment scope",
			scope:       ScopeProject,
			project:     "my-app",
			environment: "production",
			key:         "STRIPE_KEY",
			want:        "keysync_project_my-app_production_STRIPE_KEY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := credTarget(tt.scope, tt.project, tt.environment, tt.key)
			if got != tt.want {
				t.Errorf("credTarget() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestParseCredTarget verifies credential target parsing extracts the key correctly.
func TestParseCredTarget(t *testing.T) {
	tests := []struct {
		name        string
		target      string
		wantScope   Scope
		wantProject string
		wantEnv     string
		wantKey     string
	}{
		{
			name:      "global scope",
			target:    "keysync_global_API_KEY",
			wantScope: ScopeGlobal,
			wantKey:   "API_KEY",
		},
		{
			name:        "project scope",
			target:      "keysync_project_my-app_DATABASE_URL",
			wantScope:   ScopeProject,
			wantProject: "my-app",
			wantKey:     "DATABASE_URL",
		},
		{
			name:        "project + environment scope",
			target:      "keysync_project_my-app_production_STRIPE_KEY",
			wantScope:   ScopeProject,
			wantProject: "my-app",
			wantEnv:     "production",
			wantKey:     "STRIPE_KEY",
		},
		{
			name:      "key with underscores",
			target:    "keysync_global_MY_COMPLEX_KEY_NAME",
			wantScope: ScopeGlobal,
			wantKey:   "MY_COMPLEX_KEY_NAME",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotScope, gotProject, gotEnv, gotKey := parseCredTarget(tt.target)
			if gotScope != tt.wantScope {
				t.Errorf("scope = %q, want %q", gotScope, tt.wantScope)
			}
			if gotProject != tt.wantProject {
				t.Errorf("project = %q, want %q", gotProject, tt.wantProject)
			}
			if gotEnv != tt.wantEnv {
				t.Errorf("env = %q, want %q", gotEnv, tt.wantEnv)
			}
			if gotKey != tt.wantKey {
				t.Errorf("key = %q, want %q", gotKey, tt.wantKey)
			}
		})
	}
}
