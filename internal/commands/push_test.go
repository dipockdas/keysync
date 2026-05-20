package commands

import (
	"context"
	"testing"

	"github.com/dipockdas/keysync/internal/config"
	"github.com/dipockdas/keysync/internal/store"
)

// ---------------------------------------------------------------------------
// resolveRepoConfig tests
// ---------------------------------------------------------------------------

func TestResolveRepoConfig_ByProject(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"

	repoKey, projName, globals, _, err := resolveRepoConfig()
	if err != nil {
		t.Fatalf("resolveRepoConfig failed: %v", err)
	}
	if repoKey != "test/repo" {
		t.Errorf("repoKey = %q, want %q", repoKey, "test/repo")
	}
	if projName != "test-app" {
		t.Errorf("projName = %q, want %q", projName, "test-app")
	}
	if globals != nil {
		t.Errorf("globals = %v, want nil", globals)
	}
}

func TestResolveRepoConfig_ByProjectWithGlobals(t *testing.T) {
	defer setupTest(t)()
	cfg = &config.Config{
		Repos: map[string]config.RepoConfig{
			"myorg/my-app": {
				Project: "my-app",
				Globals: []string{"STRIPE_KEY", "SENDGRID_KEY"},
			},
		},
	}
	project = "my-app"

	_, _, globals, _, err := resolveRepoConfig()
	if err != nil {
		t.Fatalf("resolveRepoConfig failed: %v", err)
	}
	if len(globals) != 2 {
		t.Fatalf("expected 2 globals, got %d", len(globals))
	}
	if globals[0] != "STRIPE_KEY" {
		t.Errorf("globals[0] = %q, want %q", globals[0], "STRIPE_KEY")
	}
}

func TestResolveRepoConfig_ProjectNotFound(t *testing.T) {
	defer setupTest(t)()
	project = "nonexistent"

	_, _, _, _, err := resolveRepoConfig()
	if err == nil {
		t.Fatal("expected error for nonexistent project, got nil")
	}
}

func TestResolveRepoConfig_ByRepoFlag(t *testing.T) {
	defer setupTest(t)()
	project = ""
	repoFlag = "test/repo"

	repoKey, projName, _, _, err := resolveRepoConfig()
	if err != nil {
		t.Fatalf("resolveRepoConfig failed: %v", err)
	}
	if repoKey != "test/repo" {
		t.Errorf("repoKey = %q, want %q", repoKey, "test/repo")
	}
	if projName != "test-app" {
		t.Errorf("projName = %q, want %q", projName, "test-app")
	}
}

func TestResolveRepoConfig_RepoFlagNotFound(t *testing.T) {
	defer setupTest(t)()
	project = ""
	repoFlag = "unknown/repo"

	_, _, _, _, err := resolveRepoConfig()
	if err == nil {
		t.Fatal("expected error for nonexistent repo, got nil")
	}
}

func TestResolveRepoConfig_NeitherFlag(t *testing.T) {
	defer setupTest(t)()
	project = ""
	repoFlag = ""

	_, _, _, _, err := resolveRepoConfig()
	if err == nil {
		t.Fatal("expected error when neither flag is set, got nil")
	}
}

// ---------------------------------------------------------------------------
// collectSecrets tests
// ---------------------------------------------------------------------------

func TestCollectSecrets_GlobalFiltered(t *testing.T) {
	defer setupTest(t)()
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeGlobal, "", "", "GLOBAL_KEYS", "global-val")
	secretSt.Set(ctx, store.ScopeGlobal, "", "", "OTHER_KEY", "other-val")

	secrets, err := collectSecrets(ctx, "test-app", "", []string{"GLOBAL_KEYS"})
	if err != nil {
		t.Fatalf("collectSecrets failed: %v", err)
	}
	if len(secrets) != 1 {
		t.Fatalf("expected 1 secret, got %d", len(secrets))
	}
	if secrets["GLOBAL_KEYS"] != "global-val" {
		t.Errorf("GLOBAL_KEYS = %q, want %q", secrets["GLOBAL_KEYS"], "global-val")
	}
}

func TestCollectSecrets_ProjectOverridesGlobal(t *testing.T) {
	defer setupTest(t)()
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeGlobal, "", "", "SHARED", "global-val")
	secretSt.Set(ctx, store.ScopeProject, "test-app", "", "SHARED", "proj-val")

	secrets, err := collectSecrets(ctx, "test-app", "", []string{"SHARED"})
	if err != nil {
		t.Fatalf("collectSecrets failed: %v", err)
	}
	if secrets["SHARED"] != "proj-val" {
		t.Errorf("SHARED = %q, want %q", secrets["SHARED"], "proj-val")
	}
}

func TestCollectSecrets_EnvOverridesProject(t *testing.T) {
	defer setupTest(t)()
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeProject, "test-app", "", "SHARED", "proj-val")
	secretSt.Set(ctx, store.ScopeProject, "test-app", "production", "SHARED", "env-val")

	secrets, err := collectSecrets(ctx, "test-app", "production", nil)
	if err != nil {
		t.Fatalf("collectSecrets failed: %v", err)
	}
	if secrets["SHARED"] != "env-val" {
		t.Errorf("SHARED = %q, want %q", secrets["SHARED"], "env-val")
	}
}

func TestCollectSecrets_NoGlobalsFilter(t *testing.T) {
	defer setupTest(t)()
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeGlobal, "", "", "G_KEY", "gv")
	secretSt.Set(ctx, store.ScopeProject, "test-app", "", "P_KEY", "pv")

	// With nil/empty globals, global secrets should not be included
	secrets, err := collectSecrets(ctx, "test-app", "", nil)
	if err != nil {
		t.Fatalf("collectSecrets failed: %v", err)
	}
	if _, ok := secrets["G_KEY"]; ok {
		t.Error("global key included without globals filter")
	}
	if secrets["P_KEY"] != "pv" {
		t.Errorf("P_KEY = %q, want %q", secrets["P_KEY"], "pv")
	}
}

func TestCollectSecrets_EmptyStore(t *testing.T) {
	defer setupTest(t)()
	ctx := context.Background()

	secrets, err := collectSecrets(ctx, "test-app", "", []string{"GLOBAL_KEYS"})
	if err != nil {
		t.Fatalf("collectSecrets failed: %v", err)
	}
	if len(secrets) != 0 {
		t.Errorf("expected 0 secrets, got %d", len(secrets))
	}
}

func TestCollectSecrets_FullMerge(t *testing.T) {
	defer setupTest(t)()
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeGlobal, "", "", "GLOBAL1", "gv1")
	secretSt.Set(ctx, store.ScopeGlobal, "", "", "GLOBAL2", "gv2")
	secretSt.Set(ctx, store.ScopeProject, "test-app", "", "PROJ_KEY", "proj-val")
	secretSt.Set(ctx, store.ScopeProject, "test-app", "staging", "PROJ_KEY", "staging-val")
	secretSt.Set(ctx, store.ScopeProject, "test-app", "staging", "ENV_ONLY", "env-val")

	secrets, err := collectSecrets(ctx, "test-app", "staging", []string{"GLOBAL1"})
	if err != nil {
		t.Fatalf("collectSecrets failed: %v", err)
	}

	// GLOBAL1 should be included (in globals filter)
	if secrets["GLOBAL1"] != "gv1" {
		t.Errorf("GLOBAL1 = %q, want %q", secrets["GLOBAL1"], "gv1")
	}
	// GLOBAL2 should NOT be included (not in globals filter)
	if _, ok := secrets["GLOBAL2"]; ok {
		t.Error("GLOBAL2 included but not in globals filter")
	}
	// PROJ_KEY should have the staging value (env overrides project)
	if secrets["PROJ_KEY"] != "staging-val" {
		t.Errorf("PROJ_KEY = %q, want %q", secrets["PROJ_KEY"], "staging-val")
	}
	// ENV_ONLY should be present
	if secrets["ENV_ONLY"] != "env-val" {
		t.Errorf("ENV_ONLY = %q, want %q", secrets["ENV_ONLY"], "env-val")
	}
}

// ---------------------------------------------------------------------------
// configuredPlatforms tests
// ---------------------------------------------------------------------------

func TestConfiguredPlatforms_Single(t *testing.T) {
	pc := config.PlatformConfig{
		Vercel: &config.VercelConfig{
			ProjectID: "proj_abc",
			Target:    []string{"production"},
		},
	}
	names := configuredPlatforms(pc)
	if len(names) != 1 || names[0] != "vercel" {
		t.Errorf("configuredPlatforms = %v, want [vercel]", names)
	}
}

func TestConfiguredPlatforms_Multiple(t *testing.T) {
	pc := config.PlatformConfig{
		Vercel: &config.VercelConfig{
			ProjectID: "proj_abc",
		},
		Supabase: &config.SupabaseConfig{
			Ref: "ref_xyz",
		},
	}
	names := configuredPlatforms(pc)
	if len(names) != 2 {
		t.Fatalf("configuredPlatforms = %v, want 2 entries", names)
	}
	found := make(map[string]bool)
	for _, n := range names {
		found[n] = true
	}
	if !found["vercel"] {
		t.Error("missing vercel")
	}
	if !found["supabase"] {
		t.Error("missing supabase")
	}
}

func TestConfiguredPlatforms_None(t *testing.T) {
	pc := config.PlatformConfig{}
	names := configuredPlatforms(pc)
	if len(names) != 0 {
		t.Errorf("configuredPlatforms = %v, want empty", names)
	}
}

func TestConfiguredPlatforms_Nil(t *testing.T) {
	names := configuredPlatforms(nil)
	if names != nil {
		t.Errorf("configuredPlatforms = %v, want nil", names)
	}
}

func TestConfiguredPlatforms_ExcludesEmpty(t *testing.T) {
	pc := config.PlatformConfig{
		Vercel:  &config.VercelConfig{ProjectID: "proj_abc"},
		Railway: nil, // nil should be excluded
	}
	names := configuredPlatforms(pc)
	if len(names) != 1 || names[0] != "vercel" {
		t.Errorf("configuredPlatforms = %v, want [vercel]", names)
	}
}

// ---------------------------------------------------------------------------
// getPlatformConfigJSON tests
// ---------------------------------------------------------------------------

func TestGetPlatformConfigJSON_Existing(t *testing.T) {
	pc := config.PlatformConfig{
		Vercel: &config.VercelConfig{
			ProjectID: "proj_abc",
			Target:    []string{"production"},
		},
	}
	json := getPlatformConfigJSON(pc, "vercel", "test/repo")
	if json == "" {
		t.Fatal("getPlatformConfigJSON returned empty for vercel")
	}
}

func TestGetPlatformConfigJSON_Missing(t *testing.T) {
	pc := config.PlatformConfig{
		Vercel: &config.VercelConfig{ProjectID: "proj_abc"},
	}
	json := getPlatformConfigJSON(pc, "railway", "test/repo")
	if json != "" {
		t.Errorf("getPlatformConfigJSON = %q, want empty", json)
	}
}

func TestGetPlatformConfigJSON_NilConfig(t *testing.T) {
	json := getPlatformConfigJSON(nil, "vercel", "test/repo")
	if json != "" {
		t.Errorf("getPlatformConfigJSON = %q, want empty", json)
	}
}
