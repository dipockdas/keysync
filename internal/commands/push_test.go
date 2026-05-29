package commands

import (
	"context"
	"encoding/json"
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

	repoKey, rc, projName, err := resolveRepoConfig()
	if err != nil {
		t.Fatalf("resolveRepoConfig failed: %v", err)
	}
	if repoKey != "test/repo" {
		t.Errorf("repoKey = %q, want %q", repoKey, "test/repo")
	}
	if projName != "test-app" {
		t.Errorf("projName = %q, want %q", projName, "test-app")
	}
	if rc.Globals != nil {
		t.Errorf("globals = %v, want nil", rc.Globals)
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

	_, rc, _, err := resolveRepoConfig()
	if err != nil {
		t.Fatalf("resolveRepoConfig failed: %v", err)
	}
	if len(rc.Globals) != 2 {
		t.Fatalf("expected 2 globals, got %d", len(rc.Globals))
	}
	if rc.Globals[0] != "STRIPE_KEY" {
		t.Errorf("globals[0] = %q, want %q", rc.Globals[0], "STRIPE_KEY")
	}
}

func TestResolveRepoConfig_ProjectNotFound(t *testing.T) {
	defer setupTest(t)()
	project = "nonexistent"

	_, _, _, err := resolveRepoConfig()
	if err == nil {
		t.Fatal("expected error for nonexistent project, got nil")
	}
}

func TestResolveRepoConfig_ByRepoFlag(t *testing.T) {
	defer setupTest(t)()
	project = ""
	repoFlag = "test/repo"

	repoKey, _, projName, err := resolveRepoConfig()
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

	_, _, _, err := resolveRepoConfig()
	if err == nil {
		t.Fatal("expected error for nonexistent repo, got nil")
	}
}

func TestResolveRepoConfig_NeitherFlag(t *testing.T) {
	defer setupTest(t)()
	project = ""
	repoFlag = ""

	_, _, _, err := resolveRepoConfig()
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
	vercelJSON, _ := json.Marshal(config.VercelConfig{
		ProjectID: "proj_abc",
		Target:    []string{"production"},
	})
	pc := map[string]json.RawMessage{
		"vercel": vercelJSON,
	}
	names := configuredPlatforms(pc)
	if len(names) != 1 || names[0] != "vercel" {
		t.Errorf("configuredPlatforms = %v, want [vercel]", names)
	}
}

func TestConfiguredPlatforms_Multiple(t *testing.T) {
	vercelJSON, _ := json.Marshal(config.VercelConfig{
		ProjectID: "proj_abc",
	})
	supabaseJSON, _ := json.Marshal(config.SupabaseConfig{
		Ref: "ref_xyz",
	})
	pc := map[string]json.RawMessage{
		"vercel":   vercelJSON,
		"supabase": supabaseJSON,
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
	pc := make(map[string]json.RawMessage)
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
	vercelJSON, _ := json.Marshal(config.VercelConfig{ProjectID: "proj_abc"})
	pc := map[string]json.RawMessage{
		"vercel":  vercelJSON,
		"railway": json.RawMessage("null"), // null should be excluded
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
	vercelJSON, _ := json.Marshal(config.VercelConfig{
		ProjectID: "proj_abc",
		Target:    []string{"production"},
	})
	pc := map[string]json.RawMessage{
		"vercel": vercelJSON,
	}
	result := getPlatformConfigJSON(pc, "vercel", "test/repo")
	if result == "" {
		t.Fatal("getPlatformConfigJSON returned empty for vercel")
	}
}

func TestGetPlatformConfigJSON_Missing(t *testing.T) {
	vercelJSON, _ := json.Marshal(config.VercelConfig{ProjectID: "proj_abc"})
	pc := map[string]json.RawMessage{
		"vercel": vercelJSON,
	}
	result := getPlatformConfigJSON(pc, "railway", "test/repo")
	if result != "" {
		t.Errorf("getPlatformConfigJSON = %q, want empty", result)
	}
}

func TestGetPlatformConfigJSON_NilConfig(t *testing.T) {
	result := getPlatformConfigJSON(nil, "vercel", "test/repo")
	if result != "" {
		t.Errorf("getPlatformConfigJSON = %q, want empty", result)
	}
}

func TestCollectPushPlan_Exclude(t *testing.T) {
	defer setupTest(t)()
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeProject, "test-app", "", "PUBLIC_KEY", "ok")
	secretSt.Set(ctx, store.ScopeProject, "test-app", "", "LOCAL_DEV_TOKEN", "local-pat")

	plan, err := collectPushPlan(ctx, "test-app", "", nil, nil, []string{"LOCAL_DEV_TOKEN"}, nil, true)
	if err != nil {
		t.Fatalf("collectPushPlan: %v", err)
	}
	secrets := planToSecrets(plan)
	if _, ok := secrets["LOCAL_DEV_TOKEN"]; ok {
		t.Error("LOCAL_DEV_TOKEN should be excluded")
	}
	if secrets["PUBLIC_KEY"] != "ok" {
		t.Errorf("PUBLIC_KEY = %q", secrets["PUBLIC_KEY"])
	}
}

func TestCollectPushPlan_SecretsAllowlist(t *testing.T) {
	defer setupTest(t)()
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeGlobal, "", "", "GLOBAL1", "g1")
	secretSt.Set(ctx, store.ScopeProject, "test-app", "", "PROJ_A", "a")
	secretSt.Set(ctx, store.ScopeProject, "test-app", "", "PROJ_B", "b")

	plan, err := collectPushPlan(ctx, "test-app", "", []string{"GLOBAL1"}, []string{"GLOBAL1", "PROJ_A"}, nil, nil, true)
	if err != nil {
		t.Fatalf("collectPushPlan: %v", err)
	}
	secrets := planToSecrets(plan)
	if len(secrets) != 2 {
		t.Fatalf("secrets = %v, want 2 keys", secrets)
	}
	if _, ok := secrets["PROJ_B"]; ok {
		t.Error("PROJ_B should not be in allowlist")
	}
}

func TestListCmd_HasLsAlias(t *testing.T) {
	cmd := newListCmd()
	found := false
	for _, a := range cmd.Aliases {
		if a == "ls" {
			found = true
			break
		}
	}
	if !found {
		t.Error("list command should have ls alias")
	}
}

func TestCollectPushPlan_EnvSkipsOtherEnvironments(t *testing.T) {
	defer setupTest(t)()
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeProject, "test-app", "", "SHARED", "project-val")
	secretSt.Set(ctx, store.ScopeProject, "test-app", "dev", "SHARED", "dev-val")
	secretSt.Set(ctx, store.ScopeProject, "test-app", "production", "SHARED", "prod-val")
	secretSt.Set(ctx, store.ScopeProject, "test-app", "production", "PROD_ONLY", "prod-only")

	plan, err := collectPushPlan(ctx, "test-app", "production", nil, nil, nil, nil, true)
	if err != nil {
		t.Fatalf("collectPushPlan: %v", err)
	}
	secrets := planToSecrets(plan)
	if secrets["SHARED"] != "prod-val" {
		t.Errorf("SHARED = %q, want prod-val (production should override project)", secrets["SHARED"])
	}
	if secrets["PROD_ONLY"] != "prod-only" {
		t.Errorf("PROD_ONLY = %q", secrets["PROD_ONLY"])
	}
	if len(secrets) != 2 {
		t.Errorf("secrets = %v, want 2 keys (not dev)", secrets)
	}
}

func TestCollectPushPlan_ExcludeSkipsKeychainGet(t *testing.T) {
	defer setupTest(t)()
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeProject, "test-app", "", "KEEP", "ok")
	secretSt.Set(ctx, store.ScopeProject, "test-app", "", "SKIP_ME", "should-not-read")

	orig := secretSt
	reads := 0
	secretSt = getCountingStore{Store: orig, onGet: func() { reads++ }}
	t.Cleanup(func() { secretSt = orig })

	plan, err := collectPushPlan(ctx, "test-app", "", nil, nil, []string{"SKIP_ME"}, nil, true)
	if err != nil {
		t.Fatalf("collectPushPlan: %v", err)
	}
	if _, ok := planToSecrets(plan)["KEEP"]; !ok {
		t.Fatalf("plan = %v", plan)
	}
	if reads != 1 {
		t.Errorf("keychain Get calls = %d, want 1 (excluded key must not be read)", reads)
	}
}

type getCountingStore struct {
	store.Store
	onGet func()
}

func (s getCountingStore) Get(ctx context.Context, scope store.Scope, project, environment, key string) (string, error) {
	s.onGet()
	return s.Store.Get(ctx, scope, project, environment, key)
}

func TestCollectPushPlan_DryRunSkipsValues(t *testing.T) {
	defer setupTest(t)()
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeProject, "test-app", "", "API_KEY", "secret-value")

	orig := secretSt
	reads := 0
	secretSt = getCountingStore{Store: orig, onGet: func() { reads++ }}
	t.Cleanup(func() { secretSt = orig })

	plan, err := collectPushPlan(ctx, "test-app", "", nil, nil, nil, nil, false)
	if err != nil {
		t.Fatalf("collectPushPlan: %v", err)
	}
	if len(plan) != 1 {
		t.Fatalf("plan len = %d, want 1", len(plan))
	}
	if plan[0].Value != "" {
		t.Errorf("dry-run Value = %q, want empty", plan[0].Value)
	}
	if reads != 0 {
		t.Errorf("keychain Get calls = %d, want 0 on dry-run", reads)
	}
}

func TestCollectPushPlan_OnlyFlag(t *testing.T) {
	defer setupTest(t)()
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeProject, "test-app", "", "A", "1")
	secretSt.Set(ctx, store.ScopeProject, "test-app", "", "B", "2")

	plan, err := collectPushPlan(ctx, "test-app", "", nil, nil, nil, []string{"A"}, true)
	if err != nil {
		t.Fatalf("collectPushPlan: %v", err)
	}
	secrets := planToSecrets(plan)
	if len(secrets) != 1 || secrets["A"] != "1" {
		t.Errorf("secrets = %v", secrets)
	}
}
