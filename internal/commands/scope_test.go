package commands

import (
	"context"
	"testing"

	"github.com/dipockdas/keysync/internal/store"
)

// TestSet_ProjectScope_WhenEnvOmitted verifies that omitting --env stores to
// project scope, not env scope. This is the test for Finding 2 of the audit report.
func TestSet_ProjectScope_WhenEnvOmitted(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"
	envFlag = "" // Explicitly empty (simulates omitting --env)

	ctx := context.Background()

	// Store a secret without --env flag
	if err := secretSt.Set(ctx, store.ScopeProject, project, envFlag, "API_KEY", "test-value"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Verify it's stored in project scope (no environment)
	val, err := secretSt.Get(ctx, store.ScopeProject, project, "", "API_KEY")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "test-value" {
		t.Errorf("Get returned %q, want %q", val, "test-value")
	}

	// Verify it's NOT in a "dev" environment scope
	_, err = secretSt.Get(ctx, store.ScopeProject, project, "dev", "API_KEY")
	if err == nil {
		t.Error("secret found in 'dev' environment scope when it should be in project scope - Finding 2 NOT FIXED")
	}
}

// TestSet_EnvScope_WhenEnvProvided verifies that specifying --env stores to
// project+env scope, not project scope.
func TestSet_EnvScope_WhenEnvProvided(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"
	envFlag = "staging"

	ctx := context.Background()

	// Store a secret with --env staging
	if err := secretSt.Set(ctx, store.ScopeProject, project, envFlag, "API_KEY", "staging-value"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Verify it's stored in project+env scope
	val, err := secretSt.Get(ctx, store.ScopeProject, project, "staging", "API_KEY")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "staging-value" {
		t.Errorf("Get returned %q, want %q", val, "staging-value")
	}

	// Verify it's NOT in project scope (no environment)
	_, err = secretSt.Get(ctx, store.ScopeProject, project, "", "API_KEY")
	if err == nil {
		t.Error("secret found in project scope when it should be in staging environment scope")
	}
}

// TestGet_ProjectScope_WhenEnvOmitted verifies that omitting --env reads from
// project scope, not env scope.
func TestGet_ProjectScope_WhenEnvOmitted(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"
	envFlag = ""

	ctx := context.Background()

	// Store secret in project scope
	if err := secretSt.Set(ctx, store.ScopeProject, project, "", "DB_URL", "postgres://localhost"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Store different value in "dev" environment scope
	if err := secretSt.Set(ctx, store.ScopeProject, project, "dev", "DB_URL", "postgres://dev"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get without --env should return project scope, not dev scope
	val, err := secretSt.Get(ctx, store.ScopeProject, project, envFlag, "DB_URL")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "postgres://localhost" {
		t.Errorf("Get returned %q, expected project scope value 'postgres://localhost' - Finding 2 NOT FIXED", val)
	}
}

// TestPush_ProjectScope_WhenEnvOmitted verifies that omitting --env when pushing
// includes project-scoped secrets, not env-scoped secrets.
func TestPush_ProjectScope_WhenEnvOmitted(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"
	envFlag = "" // Explicitly empty

	ctx := context.Background()

	// Store secret in project scope
	if err := secretSt.Set(ctx, store.ScopeProject, project, "", "PROJECT_SECRET", "value1"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Store secret in "dev" environment scope
	if err := secretSt.Set(ctx, store.ScopeProject, project, "dev", "DEV_SECRET", "value2"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Collect secrets without --env (should only get project scope)
	secrets, err := collectSecrets(ctx, project, envFlag, nil)
	if err != nil {
		t.Fatalf("collectSecrets failed: %v", err)
	}

	// Should include project secret
	if _, ok := secrets["PROJECT_SECRET"]; !ok {
		t.Error("PROJECT_SECRET not included - should be in project scope")
	}

	// Should NOT include dev-scoped secret
	if _, ok := secrets["DEV_SECRET"]; ok {
		t.Error("DEV_SECRET included when --env not specified - Finding 2 NOT FIXED")
	}
}

// TestList_ShowsAllEnvironments_WhenEnvOmitted verifies that omitting --env when
// listing shows ALL environments (wildcard behavior). This is correct because users
// typically want to see all secrets for a project when listing.
func TestList_ShowsAllEnvironments_WhenEnvOmitted(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"
	envFlag = ""

	ctx := context.Background()

	// Store secrets in different scopes
	secretSt.Set(ctx, store.ScopeProject, project, "", "PROJECT_KEY", "v1")
	secretSt.Set(ctx, store.ScopeProject, project, "dev", "DEV_KEY", "v2")
	secretSt.Set(ctx, store.ScopeProject, project, "staging", "STAGING_KEY", "v3")

	// List without --env should show ALL environments (wildcard)
	entries, err := secretSt.List(ctx, store.ScopeProject, project, envFlag)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Should include all three keys
	found := make(map[string]bool)
	for _, e := range entries {
		found[e.Key] = true
	}

	if !found["PROJECT_KEY"] {
		t.Error("PROJECT_KEY not found - should show project scope")
	}
	if !found["DEV_KEY"] {
		t.Error("DEV_KEY not found - should show dev environment")
	}
	if !found["STAGING_KEY"] {
		t.Error("STAGING_KEY not found - should show staging environment")
	}

	if len(entries) != 3 {
		t.Errorf("got %d entries, want 3 (all environments)", len(entries))
	}
}

// TestScopeResolution_Precedence verifies the correct scope precedence:
// global (lowest) < project < project+env (highest)
func TestScopeResolution_Precedence(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"

	ctx := context.Background()

	// Store same key in all three scopes
	secretSt.Set(ctx, store.ScopeGlobal, "", "", "SHARED", "global-value")
	secretSt.Set(ctx, store.ScopeProject, project, "", "SHARED", "project-value")
	secretSt.Set(ctx, store.ScopeProject, project, "production", "SHARED", "production-value")

	// Without --env, should get project value (overrides global)
	envFlag = ""
	secrets, _ := collectSecrets(ctx, project, envFlag, []string{"SHARED"})
	if secrets["SHARED"] != "project-value" {
		t.Errorf("without --env, got %q, want project-value", secrets["SHARED"])
	}

	// With --env production, should get production value (overrides project)
	envFlag = "production"
	secrets, _ = collectSecrets(ctx, project, envFlag, []string{"SHARED"})
	if secrets["SHARED"] != "production-value" {
		t.Errorf("with --env production, got %q, want production-value", secrets["SHARED"])
	}
}
