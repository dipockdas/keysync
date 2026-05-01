package keysync

import (
	"context"
	"testing"
)

func TestServiceName_Global(t *testing.T) {
	got := serviceName("global", "")
	want := "keysync/global"
	if got != want {
		t.Errorf("serviceName(global, '') = %q, want %q", got, want)
	}
}

func TestServiceName_Project(t *testing.T) {
	got := serviceName("project", "my-app")
	want := "keysync/project/my-app"
	if got != want {
		t.Errorf("serviceName(project, my-app) = %q, want %q", got, want)
	}
}

func TestServiceName_GlobalWithProject(t *testing.T) {
	// Global scope ignores the project parameter
	got := serviceName("global", "my-app")
	want := "keysync/global"
	if got != want {
		t.Errorf("serviceName(global, my-app) = %q, want %q", got, want)
	}
}

func TestParseServiceName_Global(t *testing.T) {
	scope, project := parseServiceName("keysync/global")
	if scope != "global" || project != "" {
		t.Errorf("parseServiceName = (%q, %q), want (global, '')", scope, project)
	}
}

func TestParseServiceName_Project(t *testing.T) {
	scope, project := parseServiceName("keysync/project/my-app")
	if scope != "project" || project != "my-app" {
		t.Errorf("parseServiceName = (%q, %q), want (project, my-app)", scope, project)
	}
}

func TestParseServiceName_ProjectDeep(t *testing.T) {
	scope, project := parseServiceName("keysync/project/my/deep/app")
	if scope != "project" || project != "my/deep/app" {
		t.Errorf("parseServiceName = (%q, %q), want (project, my/deep/app)", scope, project)
	}
}

func TestParseServiceName_Unprefixed(t *testing.T) {
	scope, project := parseServiceName("other/global")
	if scope != "global" || project != "" {
		t.Errorf("parseServiceName = (%q, %q), want (global, '')", scope, project)
	}
}

func TestParseServiceName_Empty(t *testing.T) {
	scope, project := parseServiceName("")
	if scope != "global" || project != "" {
		t.Errorf("parseServiceName = (%q, %q), want (global, '')", scope, project)
	}
}

func TestGetSecret_NoProject(t *testing.T) {
	// Without a keysync keychain set up, this should return ErrNotFound
	_, err := GetSecret("NONEXISTENT_KEY", "")
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
