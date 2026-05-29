package commands

import (
	"context"
	"testing"

	"github.com/dipockdas/keysync/internal/store"
)

// TestSet_DevScope_WhenEnvDefaults verifies that project-scoped set uses dev when --env is omitted.
func TestSet_DevScope_WhenEnvDefaults(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"
	effectiveEnv = "dev"

	ctx := context.Background()

	if err := secretSt.Set(ctx, store.ScopeProject, project, effectiveEnv, "API_KEY", "dev-value"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, err := secretSt.Get(ctx, store.ScopeProject, project, "dev", "API_KEY")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "dev-value" {
		t.Errorf("Get returned %q, want dev-value", val)
	}

	_, err = secretSt.Get(ctx, store.ScopeProject, project, "", "API_KEY")
	if err == nil {
		t.Error("secret found in project-wide scope when it should be in dev environment scope")
	}
}

// TestSet_ProjectScope_WhenEnvExplicitlyEmpty verifies --env "" stores project-wide (no environment).
func TestSet_ProjectScope_WhenEnvExplicitlyEmpty(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"
	effectiveEnv = ""

	ctx := context.Background()

	if err := secretSt.Set(ctx, store.ScopeProject, project, effectiveEnv, "API_KEY", "project-value"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, err := secretSt.Get(ctx, store.ScopeProject, project, "", "API_KEY")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "project-value" {
		t.Errorf("Get returned %q, want project-value", val)
	}
}

// TestSet_EnvScope_WhenEnvProvided verifies that specifying --env stores to project+env scope.
func TestSet_EnvScope_WhenEnvProvided(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"
	effectiveEnv = "staging"

	ctx := context.Background()

	if err := secretSt.Set(ctx, store.ScopeProject, project, effectiveEnv, "API_KEY", "staging-value"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, err := secretSt.Get(ctx, store.ScopeProject, project, "staging", "API_KEY")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "staging-value" {
		t.Errorf("Get returned %q, want staging-value", val)
	}
}

// TestGet_ExplicitEnvBeforeProject verifies --env dev is resolved before project-wide scope.
func TestGet_ExplicitEnvBeforeProject(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"

	ctx := context.Background()

	secretSt.Set(ctx, store.ScopeProject, project, "", "DB_URL", "postgres://localhost")
	secretSt.Set(ctx, store.ScopeProject, project, "dev", "DB_URL", "postgres://dev")

	val, err := secretSt.Get(ctx, store.ScopeProject, project, "dev", "DB_URL")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "postgres://dev" {
		t.Errorf("Get returned %q, want postgres://dev", val)
	}
}

// TestPush_DevScope_WhenEnvDefaults verifies default env includes dev + project-wide keys only.
func TestPush_DevScope_WhenEnvDefaults(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"

	ctx := context.Background()

	secretSt.Set(ctx, store.ScopeProject, project, "", "PROJECT_SECRET", "value1")
	secretSt.Set(ctx, store.ScopeProject, project, "dev", "DEV_SECRET", "value2")
	secretSt.Set(ctx, store.ScopeProject, project, "production", "PROD_SECRET", "value3")

	secrets, err := collectSecrets(ctx, project, "dev", nil)
	if err != nil {
		t.Fatalf("collectSecrets failed: %v", err)
	}

	if _, ok := secrets["PROJECT_SECRET"]; !ok {
		t.Error("PROJECT_SECRET not included")
	}
	if _, ok := secrets["DEV_SECRET"]; !ok {
		t.Error("DEV_SECRET not included for default dev env")
	}
	if _, ok := secrets["PROD_SECRET"]; ok {
		t.Error("PROD_SECRET must not be included when env defaults to dev")
	}
}

// TestList_DevAndProject_WhenEnvDefaults verifies list shows dev + project-wide keys only.
func TestList_DevAndProject_WhenEnvDefaults(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"

	ctx := context.Background()

	secretSt.Set(ctx, store.ScopeProject, project, "", "PROJECT_KEY", "v1")
	secretSt.Set(ctx, store.ScopeProject, project, "dev", "DEV_KEY", "v2")
	secretSt.Set(ctx, store.ScopeProject, project, "production", "PROD_KEY", "v3")

	entries, err := collectPushEntries(ctx, project, "dev", nil)
	if err != nil {
		t.Fatalf("collectPushEntries failed: %v", err)
	}

	found := make(map[string]bool)
	for _, e := range entries {
		found[e.Key] = true
	}

	if !found["PROJECT_KEY"] || !found["DEV_KEY"] {
		t.Errorf("missing expected keys, got %v", found)
	}
	if found["PROD_KEY"] {
		t.Error("PROD_KEY must not appear when env defaults to dev")
	}
}

// TestScopeResolution_Precedence verifies global < project < project+env.
func TestScopeResolution_Precedence(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"

	ctx := context.Background()

	secretSt.Set(ctx, store.ScopeGlobal, "", "", "SHARED", "global-value")
	secretSt.Set(ctx, store.ScopeProject, project, "", "SHARED", "project-value")
	secretSt.Set(ctx, store.ScopeProject, project, "production", "SHARED", "production-value")

	secrets, _ := collectSecrets(ctx, project, "dev", []string{"SHARED"})
	if secrets["SHARED"] != "project-value" {
		t.Errorf("with default dev env, got %q, want project-value", secrets["SHARED"])
	}

	secrets, _ = collectSecrets(ctx, project, "production", []string{"SHARED"})
	if secrets["SHARED"] != "production-value" {
		t.Errorf("with --env production, got %q, want production-value", secrets["SHARED"])
	}
}
