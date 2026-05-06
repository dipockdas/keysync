package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}
	if cfg.Repos == nil {
		t.Fatal("DefaultConfig.Repos is nil")
	}
	if len(cfg.Repos) != 0 {
		t.Errorf("DefaultConfig.Repos has %d entries, want 0", len(cfg.Repos))
	}
}

func TestLoadSaveConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".keysync.json")

	// Create a config with a repo
	cfg := &Config{
		Repos: map[string]RepoConfig{
			"myorg/my-app": {
				Project: "my-app",
				Globals: []string{"STRIPE_KEY"},
				Platforms: PlatformConfig{
					Vercel: &VercelConfig{
						ProjectID: "vercel-proj-123",
						Target:    []string{"production"},
					},
					Railway: &RailwayConfig{
						Environment: "production",
						Service:     "my-service",
					},
					Supabase: &SupabaseConfig{
						Ref: "supabase-ref-456",
					},
				},
			},
		},
	}

	if err := SaveConfig(cfg, path); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("SaveConfig did not create file")
	}

	// Load it back
	loaded, loadedPath, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if loadedPath != path {
		t.Errorf("LoadConfig returned path %q, want %q", loadedPath, path)
	}

	// Verify contents
	rc, ok := loaded.Repos["myorg/my-app"]
	if !ok {
		t.Fatal("loaded config missing repo 'myorg/my-app'")
	}
	if rc.Project != "my-app" {
		t.Errorf("project = %q, want %q", rc.Project, "my-app")
	}
	if len(rc.Globals) != 1 || rc.Globals[0] != "STRIPE_KEY" {
		t.Errorf("globals = %v, want [STRIPE_KEY]", rc.Globals)
	}
	if rc.Platforms.Vercel == nil || rc.Platforms.Vercel.ProjectID != "vercel-proj-123" {
		t.Errorf("Vercel config mismatch: %+v", rc.Platforms.Vercel)
	}
	if rc.Platforms.Supabase == nil || rc.Platforms.Supabase.Ref != "supabase-ref-456" {
		t.Errorf("Supabase config mismatch: %+v", rc.Platforms.Supabase)
	}
}

func TestLoadConfig_NotFound(t *testing.T) {
	dir := t.TempDir() // no .keysync.json here

	cfg, path, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if path != "" {
		t.Errorf("expected empty path, got %q", path)
	}
	if cfg == nil {
		t.Fatal("expected default config, got nil")
	}
	if len(cfg.Repos) != 0 {
		t.Errorf("expected empty repos, got %d", len(cfg.Repos))
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".keysync.json")
	os.WriteFile(path, []byte("{invalid json"), 0644)

	_, _, err := LoadConfig(dir)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestFindConfig_CurrentDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".keysync.json")
	os.WriteFile(path, []byte("{}"), 0644)

	found, err := findConfig(dir)
	if err != nil {
		t.Fatalf("findConfig failed: %v", err)
	}
	if found != path {
		t.Errorf("findConfig returned %q, want %q", found, path)
	}
}

func TestFindConfig_ParentDir(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "sub", "nested")
	os.MkdirAll(child, 0755)

	// Create config in parent
	configPath := filepath.Join(parent, ".keysync.json")
	os.WriteFile(configPath, []byte("{}"), 0644)

	found, err := findConfig(child)
	if err != nil {
		t.Fatalf("findConfig failed: %v", err)
	}
	if found != configPath {
		t.Errorf("findConfig returned %q, want %q", found, configPath)
	}
}

func TestFindConfig_NotFound(t *testing.T) {
	dir := t.TempDir()

	found, err := findConfig(dir)
	if err != nil {
		t.Fatalf("findConfig failed: %v", err)
	}
	if found != "" {
		t.Errorf("findConfig returned %q, want empty", found)
	}
}

func TestDefaultConfigPath(t *testing.T) {
	want := filepath.Join("/some/dir", ".keysync.json")
	got := DefaultConfigPath("/some/dir")
	if got != want {
		t.Errorf("DefaultConfigPath = %q, want %q", got, want)
	}
}

func TestLoadConfig_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".keysync.json")
	os.WriteFile(path, []byte{}, 0644)

	_, _, err := LoadConfig(dir)
	if err == nil {
		t.Fatal("expected error for empty file, got nil")
	}
}

func TestFindRepoByProject(t *testing.T) {
	cfg := &Config{
		Repos: map[string]RepoConfig{
			"org/alpha": {Project: "alpha", Globals: []string{"GLOBAL_A"}},
			"org/beta":  {Project: "beta", Globals: []string{"GLOBAL_B"}},
		},
	}

	repo, rc, ok := FindRepoByProject(cfg, "beta")
	if !ok {
		t.Fatal("FindRepoByProject(beta) returned false")
	}
	if repo != "org/beta" {
		t.Errorf("repo = %q, want %q", repo, "org/beta")
	}
	if rc.Globals[0] != "GLOBAL_B" {
		t.Errorf("globals = %v, want [GLOBAL_B]", rc.Globals)
	}

	_, _, ok = FindRepoByProject(cfg, "nonexistent")
	if ok {
		t.Fatal("FindRepoByProject(nonexistent) returned true")
	}
}
