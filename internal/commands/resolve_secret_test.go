package commands

import (
	"context"
	"testing"

	"github.com/dipockdas/keysync/internal/store"
)

func TestLocateSecret_ProjectWideBeforeGlobal(t *testing.T) {
	defer setupTest(t)()
	ctx := context.Background()
	project := "test-app"

	secretSt.Set(ctx, store.ScopeGlobal, "", "", "SHARED", "global-val")
	secretSt.Set(ctx, store.ScopeProject, project, "", "SHARED", "proj-val")

	scope, proj, env, found := locateSecret(ctx, "SHARED", project, "")
	if !found {
		t.Fatal("not found")
	}
	if scope != store.ScopeProject || proj != project || env != "" {
		t.Errorf("got scope=%s project=%s env=%q", scope, proj, env)
	}
}

func TestLocateSecret_SkipsOtherEnvironmentsWithoutExplicitEnv(t *testing.T) {
	defer setupTest(t)()
	ctx := context.Background()
	project := "test-app"

	secretSt.Set(ctx, store.ScopeProject, project, "production", "PROD_ONLY", "prod")

	_, _, _, found := locateSecret(ctx, "PROD_ONLY", project, "")
	if found {
		t.Error("should not match production env without --env")
	}
}

func TestLocateSecret_ExplicitEnv(t *testing.T) {
	defer setupTest(t)()
	ctx := context.Background()
	project := "test-app"

	secretSt.Set(ctx, store.ScopeProject, project, "", "DB_URL", "project-val")
	secretSt.Set(ctx, store.ScopeProject, project, "production", "DB_URL", "prod-val")

	scope, _, env, found := locateSecret(ctx, "DB_URL", project, "production")
	if !found || scope != store.ScopeProject || env != "production" {
		t.Fatalf("got scope=%s env=%q found=%v", scope, env, found)
	}
}

func TestLocateSecret_FallsBackToGlobal(t *testing.T) {
	defer setupTest(t)()
	ctx := context.Background()
	project := "test-app"

	secretSt.Set(ctx, store.ScopeGlobal, "", "", "GLOBAL_ONLY", "g")
	secretSt.Set(ctx, store.ScopeProject, project, "production", "GLOBAL_ONLY", "prod")

	scope, _, env, found := locateSecret(ctx, "GLOBAL_ONLY", project, "")
	if !found || scope != store.ScopeGlobal || env != "" {
		t.Errorf("got scope=%s env=%q found=%v", scope, env, found)
	}
}

func TestGetCmd_OneKeychainRead(t *testing.T) {
	defer setupTest(t)()
	ctx := context.Background()
	project = "test-app"

	secretSt.Set(ctx, store.ScopeGlobal, "", "", "MY_KEY", "global-val")
	secretSt.Set(ctx, store.ScopeProject, project, "", "MY_KEY", "proj-val")

	orig := secretSt
	reads := 0
	secretSt = getCountingStore{Store: orig, onGet: func() { reads++ }}
	t.Cleanup(func() { secretSt = orig })

	cmd := newGetCmd()
	getUnmask = true
	t.Cleanup(func() { getUnmask = false })

	_, _, err := captureCommand(cmd, []string{"MY_KEY"})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if reads != 1 {
		t.Errorf("keychain Get calls = %d, want 1", reads)
	}
}
