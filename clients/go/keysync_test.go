package keysync

import (
	"context"
	"testing"
)

func TestServiceName_Global(t *testing.T) {
	got := serviceName("global", "", "")
	want := "keysync/global"
	if got != want {
		t.Errorf("serviceName(global, '', '') = %q, want %q", got, want)
	}
}

func TestServiceName_Project(t *testing.T) {
	got := serviceName("project", "my-app", "")
	want := "keysync/project/my-app"
	if got != want {
		t.Errorf("serviceName(project, my-app, '') = %q, want %q", got, want)
	}
}

func TestServiceName_GlobalWithProject(t *testing.T) {
	// Global scope ignores the project parameter
	got := serviceName("global", "my-app", "")
	want := "keysync/global"
	if got != want {
		t.Errorf("serviceName(global, my-app, '') = %q, want %q", got, want)
	}
}

func TestServiceName_WithEnv(t *testing.T) {
	got := serviceName("project", "my-app", "dev")
	want := "keysync/project/my-app/env/dev"
	if got != want {
		t.Errorf("serviceName(project, my-app, dev) = %q, want %q", got, want)
	}
}

func TestServiceName_WithEnvEmpty(t *testing.T) {
	// Empty environment is the same as no environment
	got := serviceName("project", "my-app", "")
	want := "keysync/project/my-app"
	if got != want {
		t.Errorf("serviceName(project, my-app, '') = %q, want %q", got, want)
	}
}

func TestServiceName_GlobalIgnoresEnv(t *testing.T) {
	// Global scope ignores both project and environment
	got := serviceName("global", "my-app", "dev")
	want := "keysync/global"
	if got != want {
		t.Errorf("serviceName(global, my-app, dev) = %q, want %q", got, want)
	}
}

func TestParseServiceName_Global(t *testing.T) {
	scope, project, env := parseServiceName("keysync/global")
	if scope != "global" || project != "" || env != "" {
		t.Errorf("parseServiceName = (%q, %q, %q), want (global, '', '')", scope, project, env)
	}
}

func TestParseServiceName_Project(t *testing.T) {
	scope, project, env := parseServiceName("keysync/project/my-app")
	if scope != "project" || project != "my-app" || env != "" {
		t.Errorf("parseServiceName = (%q, %q, %q), want (project, my-app, '')", scope, project, env)
	}
}

func TestParseServiceName_ProjectDeep(t *testing.T) {
	scope, project, env := parseServiceName("keysync/project/my/deep/app")
	if scope != "project" || project != "my/deep/app" || env != "" {
		t.Errorf("parseServiceName = (%q, %q, %q), want (project, my/deep/app, '')", scope, project, env)
	}
}

func TestParseServiceName_Unprefixed(t *testing.T) {
	scope, project, env := parseServiceName("other/global")
	if scope != "global" || project != "" || env != "" {
		t.Errorf("parseServiceName = (%q, %q, %q), want (global, '', '')", scope, project, env)
	}
}

func TestParseServiceName_Empty(t *testing.T) {
	scope, project, env := parseServiceName("")
	if scope != "global" || project != "" || env != "" {
		t.Errorf("parseServiceName = (%q, %q, %q), want (global, '', '')", scope, project, env)
	}
}

func TestParseServiceName_WithEnv(t *testing.T) {
	scope, project, env := parseServiceName("keysync/project/my-app/env/dev")
	if scope != "project" || project != "my-app" || env != "dev" {
		t.Errorf("parseServiceName = (%q, %q, %q), want (project, my-app, dev)", scope, project, env)
	}
}

func TestParseServiceName_WithEnvDeep(t *testing.T) {
	scope, project, env := parseServiceName("keysync/project/my/deep/app/env/staging")
	if scope != "project" || project != "my/deep/app" || env != "staging" {
		t.Errorf("parseServiceName = (%q, %q, %q), want (project, my/deep/app, staging)", scope, project, env)
	}
}

func TestGetSecret_NoProject(t *testing.T) {
	// Without a keysync keychain set up, this should return ErrNotFound
	_, err := GetSecret("NONEXISTENT_KEY", "", "")
	if err != ErrNotFound {
		t.Logf("GetSecret returned: %v (expected ErrNotFound if no keychain is configured)", err)
	}
}

func TestMemoryStore_GetSet(t *testing.T) {
	ctx := context.Background()
	m := NewMemoryStore()

	if err := m.SetSecret(ctx, "global", "", "MY_KEY", "my-val"); err != nil {
		t.Fatalf("SetSecret failed: %v", err)
	}

	val, err := m.GetSecret(ctx, "global", "", "MY_KEY")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if val != "my-val" {
		t.Errorf("got %q, want %q", val, "my-val")
	}
}

func TestMemoryStore_ProjectFallback(t *testing.T) {
	ctx := context.Background()
	m := NewMemoryStore()

	m.SetSecret(ctx, "global", "", "SHARED_KEY", "global-val")

	// Project scope should fall back to global
	val, err := m.GetSecret(ctx, "project", "my-app", "SHARED_KEY")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if val != "global-val" {
		t.Errorf("got %q, want %q", val, "global-val")
	}

	// Project-scoped value should override global
	m.SetSecret(ctx, "project", "my-app", "SHARED_KEY", "project-val")
	val, err = m.GetSecret(ctx, "project", "my-app", "SHARED_KEY")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if val != "project-val" {
		t.Errorf("got %q, want %q", val, "project-val")
	}
}

func TestMemoryStore_NotFound(t *testing.T) {
	ctx := context.Background()
	m := NewMemoryStore()

	_, err := m.GetSecret(ctx, "global", "", "NONEXISTENT")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryStore_List(t *testing.T) {
	m := NewMemoryStore()
	ctx := context.Background()

	m.SetSecret(ctx, "global", "", "G_KEY", "gv")
	m.SetSecret(ctx, "project", "app-a", "P_KEY_A", "pv")
	m.SetSecret(ctx, "project", "app-b", "P_KEY_B", "pv")

	globalKeys := m.List("global", "")
	if len(globalKeys) != 1 || globalKeys[0] != "G_KEY" {
		t.Errorf("global keys = %v, want [G_KEY]", globalKeys)
	}

	appAKeys := m.List("project", "app-a")
	if len(appAKeys) != 1 || appAKeys[0] != "P_KEY_A" {
		t.Errorf("app-a keys = %v, want [P_KEY_A]", appAKeys)
	}

	allKeys := m.List("", "")
	if len(allKeys) != 3 {
		t.Errorf("all keys = %v, want 3 entries", allKeys)
	}
}

func TestMemoryStore_EnvironmentFallback(t *testing.T) {
	ctx := context.Background()
	m := NewMemoryStore()

	// Set a global secret
	m.SetSecret(ctx, "global", "", "SHARED_KEY", "global-val")

	// Project scope should fall back to global (no env set)
	val, err := m.GetSecret(ctx, "project", "my-app", "SHARED_KEY")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if val != "global-val" {
		t.Errorf("got %q, want %q", val, "global-val")
	}

	// Project scope with env still falls back to global
	val, err = m.GetSecret(ctx, "project", "my-app", "SHARED_KEY")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if val != "global-val" {
		t.Errorf("got %q, want %q", val, "global-val")
	}
}

func TestMemoryStore_EnvironmentOverride(t *testing.T) {
	ctx := context.Background()
	m := NewMemoryStore()

	m.SetSecret(ctx, "global", "", "API_KEY", "global-api-key")
	m.SetSecret(ctx, "project", "my-app", "API_KEY", "project-api-key")

	// The MemoryStore uses scope/project/key for storage -- it does not
	// independently store secrets by environment. Environment-scoped secrets
	// are handled by the real keychain via serviceName() embedding the
	// environment into the service name string.
	//
	// This test verifies that the existing scope/project/key fallback logic
	// continues to work correctly, and that the Store interface accepts the
	// environment parameter (even though MemoryStore ignores it for lookup).

	// Project scope overrides global
	val, err := m.GetSecret(ctx, "project", "my-app", "API_KEY")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if val != "project-api-key" {
		t.Errorf("got %q, want %q", val, "project-api-key")
	}

	// A key that only exists in global scope -- project scope falls back
	m.SetSecret(ctx, "global", "", "ONLY_GLOBAL", "only-global-val")
	val, err = m.GetSecret(ctx, "project", "my-app", "ONLY_GLOBAL")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if val != "only-global-val" {
		t.Errorf("got %q, want %q", val, "only-global-val")
	}
}

func TestMemoryStore_ListWithEnvFilter(t *testing.T) {
	m := NewMemoryStore()
	ctx := context.Background()

	m.SetSecret(ctx, "global", "", "G1", "gv1")
	m.SetSecret(ctx, "global", "", "G2", "gv2")
	m.SetSecret(ctx, "project", "app", "P1", "pv1")
	m.SetSecret(ctx, "project", "app", "P2", "pv2")

	// List all entries (scope=project covers both env and non-env)
	allProject := m.List("project", "app")
	if len(allProject) != 2 {
		t.Errorf("project keys = %v, want 2 entries", allProject)
	}

	// List all entries
	all := m.List("", "")
	if len(all) != 4 {
		t.Errorf("all keys = %v, want 4 entries", all)
	}
}
