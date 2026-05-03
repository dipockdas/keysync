package store

import (
	"context"
	"sync"
	"testing"
)

func TestServiceName(t *testing.T) {
	tests := []struct {
		name    string
		scope   Scope
		project string
		env     string
		want    string
	}{
		{"global no project", ScopeGlobal, "", "", "keysync/global"},
		{"global with project", ScopeGlobal, "my-app", "", "keysync/global"},
		{"project with name", ScopeProject, "my-app", "", "keysync/project/my-app"},
		{"project with env", ScopeProject, "my-app", "production", "keysync/project/my-app/env/production"},
		{"project empty name", ScopeProject, "", "", "keysync/project"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serviceName(tt.scope, tt.project, tt.env)
			if got != tt.want {
				t.Errorf("serviceName(%q, %q, %q) = %q, want %q", tt.scope, tt.project, tt.env, got, tt.want)
			}
		})
	}
}

func TestParseServiceName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantScope Scope
		wantProj  string
		wantEnv   string
	}{
		{"global", "keysync/global", ScopeGlobal, "", ""},
		{"project with name", "keysync/project/my-app", ScopeProject, "my-app", ""},
		{"project with env", "keysync/project/my-app/env/production", ScopeProject, "my-app", "production"},
		{"project with slashes", "keysync/project/my/deep/app", ScopeProject, "my/deep/app", ""},
		{"project with slashes and env", "keysync/project/my/deep/app/env/staging", ScopeProject, "my/deep/app", "staging"},
		{"project empty", "keysync/project/", ScopeProject, "", ""},
		{"just keysync", "keysync", ScopeGlobal, "", ""},
		{"unrecognized scope", "keysync/other/val", ScopeGlobal, "", ""},
		{"empty string", "", ScopeGlobal, "", ""},
		{"unprefixed", "other/global", ScopeGlobal, "", ""},
		{"keysync/ only", "keysync/", ScopeGlobal, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotScope, gotProj, gotEnv := parseServiceName(tt.input)
			if gotScope != tt.wantScope || gotProj != tt.wantProj || gotEnv != tt.wantEnv {
				t.Errorf("parseServiceName(%q) = (%q, %q, %q), want (%q, %q, %q)",
					tt.input, gotScope, gotProj, gotEnv, tt.wantScope, tt.wantProj, tt.wantEnv)
			}
		})
	}
}

func TestMemKey(t *testing.T) {
	tests := []struct {
		name    string
		scope   Scope
		project string
		env     string
		key     string
		want    string
	}{
		{"global", ScopeGlobal, "", "", "API_KEY", "global///API_KEY"},
		{"project", ScopeProject, "my-app", "", "DB_URL", "project/my-app//DB_URL"},
		{"project with env", ScopeProject, "my-app", "production", "DB_URL", "project/my-app/production/DB_URL"},
		{"global with project", ScopeGlobal, "my-app", "", "KEY", "global/my-app//KEY"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := memKey(tt.scope, tt.project, tt.env, tt.key)
			if got != tt.want {
				t.Errorf("memKey(%q, %q, %q, %q) = %q, want %q", tt.scope, tt.project, tt.env, tt.key, got, tt.want)
			}
		})
	}
}

func TestMemoryStore_SetAndGet(t *testing.T) {
	ctx := context.Background()
	m := NewMemoryStore()

	// Set a global secret
	if err := m.Set(ctx, ScopeGlobal, "", "", "GLOBAL_KEY", "global-val"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	// Set a project secret
	if err := m.Set(ctx, ScopeProject, "my-app", "", "PROJ_KEY", "proj-val"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	// Set a project+env secret
	if err := m.Set(ctx, ScopeProject, "my-app", "production", "ENV_KEY", "env-val"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get global
	val, err := m.Get(ctx, ScopeGlobal, "", "", "GLOBAL_KEY")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "global-val" {
		t.Errorf("got %q, want %q", val, "global-val")
	}

	// Get project
	val, err = m.Get(ctx, ScopeProject, "my-app", "", "PROJ_KEY")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "proj-val" {
		t.Errorf("got %q, want %q", val, "proj-val")
	}

	// Get project+env
	val, err = m.Get(ctx, ScopeProject, "my-app", "production", "ENV_KEY")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "env-val" {
		t.Errorf("got %q, want %q", val, "env-val")
	}
}

func TestMemoryStore_GetNotFound(t *testing.T) {
	ctx := context.Background()
	m := NewMemoryStore()

	_, err := m.Get(ctx, ScopeGlobal, "", "", "NONEXISTENT")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	ctx := context.Background()
	m := NewMemoryStore()

	m.Set(ctx, ScopeGlobal, "", "", "DEL_KEY", "val")
	if err := m.Delete(ctx, ScopeGlobal, "", "", "DEL_KEY"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	// Verify it's gone
	_, err := m.Get(ctx, ScopeGlobal, "", "", "DEL_KEY")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestMemoryStore_DeleteNotFound(t *testing.T) {
	ctx := context.Background()
	m := NewMemoryStore()

	err := m.Delete(ctx, ScopeGlobal, "", "", "NONEXISTENT")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryStore_Overwrite(t *testing.T) {
	ctx := context.Background()
	m := NewMemoryStore()

	m.Set(ctx, ScopeGlobal, "", "", "KEY", "first")
	m.Set(ctx, ScopeGlobal, "", "", "KEY", "second")

	val, err := m.Get(ctx, ScopeGlobal, "", "", "KEY")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "second" {
		t.Errorf("got %q, want %q", val, "second")
	}
}

func TestMemoryStore_List(t *testing.T) {
	ctx := context.Background()
	m := NewMemoryStore()

	entries := []struct {
		scope   Scope
		project string
		env     string
		key     string
	}{
		{ScopeGlobal, "", "", "AAA"},
		{ScopeGlobal, "", "", "BBB"},
		{ScopeProject, "app-a", "", "KEY_1"},
		{ScopeProject, "app-a", "", "KEY_2"},
		{ScopeProject, "app-b", "", "KEY_1"},
		{ScopeProject, "app-a", "production", "ENV_KEY"},
	}
	for _, e := range entries {
		m.Set(ctx, e.scope, e.project, e.env, e.key, "val")
	}

	tests := []struct {
		name    string
		scope   Scope
		project string
		env     string
		want    int
	}{
		{"all", "", "", "", 6},
		{"global only", ScopeGlobal, "", "", 2},
		{"project app-a", ScopeProject, "app-a", "", 3},
		{"project app-a production", ScopeProject, "app-a", "production", 1},
		{"project app-b", ScopeProject, "app-b", "", 1},
		{"nonexistent project", ScopeProject, "other", "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := m.List(ctx, tt.scope, tt.project, tt.env)
			if err != nil {
				t.Fatalf("List failed: %v", err)
			}
			if len(result) != tt.want {
				t.Errorf("List returned %d entries, want %d", len(result), tt.want)
			}
		})
	}
}

func TestMemoryStore_ListSorted(t *testing.T) {
	ctx := context.Background()
	m := NewMemoryStore()

	m.Set(ctx, ScopeProject, "z-app", "", "A_KEY", "v")
	m.Set(ctx, ScopeGlobal, "", "", "Z_KEY", "v")
	m.Set(ctx, ScopeProject, "a-app", "", "B_KEY", "v")
	m.Set(ctx, ScopeGlobal, "", "", "A_KEY", "v")

	result, err := m.List(ctx, "", "", "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(result) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(result))
	}

	// Expect: global/A_KEY < global/Z_KEY < project/a-app/B_KEY < project/z-app/A_KEY
	expected := []SecretEntry{
		{Scope: ScopeGlobal, Project: "", Environment: "", Key: "A_KEY"},
		{Scope: ScopeGlobal, Project: "", Environment: "", Key: "Z_KEY"},
		{Scope: ScopeProject, Project: "a-app", Environment: "", Key: "B_KEY"},
		{Scope: ScopeProject, Project: "z-app", Environment: "", Key: "A_KEY"},
	}
	for i, exp := range expected {
		if result[i] != exp {
			t.Errorf("result[%d] = %+v, want %+v", i, result[i], exp)
		}
	}
}

func TestMemoryStore_Concurrency(t *testing.T) {
	ctx := context.Background()
	m := NewMemoryStore()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := string(rune('A' + n))
			m.Set(ctx, ScopeGlobal, "", "", key, "val")
			m.Get(ctx, ScopeGlobal, "", "", key)
			m.List(ctx, "", "", "")
		}(i)
	}
	wg.Wait()
}
