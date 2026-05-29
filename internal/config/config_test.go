package config

import (
	"encoding/json"
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

	// Create platform configs as JSON
	vercelJSON, _ := json.Marshal(VercelConfig{
		ProjectID: "vercel-proj-123",
		Target:    []string{"production"},
	})
	railwayJSON, _ := json.Marshal(RailwayConfig{
		Environment: "production",
		Service:     "my-service",
	})
	supabaseJSON, _ := json.Marshal(SupabaseConfig{
		Ref: "supabase-ref-456",
	})

	// Create a config with a repo
	cfg := &Config{
		Repos: map[string]RepoConfig{
			"myorg/my-app": {
				Project: "my-app",
				Globals: []string{"STRIPE_KEY"},
				Platforms: map[string]json.RawMessage{
					"vercel":   vercelJSON,
					"railway":  railwayJSON,
					"supabase": supabaseJSON,
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

	// Verify platform configs
	if _, ok := rc.Platforms["vercel"]; !ok {
		t.Error("vercel platform config missing")
	}
	if _, ok := rc.Platforms["supabase"]; !ok {
		t.Error("supabase platform config missing")
	}

	// Verify Vercel config can be unmarshaled
	var vc VercelConfig
	if err := json.Unmarshal(rc.Platforms["vercel"], &vc); err != nil {
		t.Fatalf("failed to unmarshal vercel config: %v", err)
	}
	if vc.ProjectID != "vercel-proj-123" {
		t.Errorf("vercel ProjectID = %q, want %q", vc.ProjectID, "vercel-proj-123")
	}

	// Verify Supabase config can be unmarshaled
	var sc SupabaseConfig
	if err := json.Unmarshal(rc.Platforms["supabase"], &sc); err != nil {
		t.Fatalf("failed to unmarshal supabase config: %v", err)
	}
	if sc.Ref != "supabase-ref-456" {
		t.Errorf("supabase Ref = %q, want %q", sc.Ref, "supabase-ref-456")
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
			"org/alpha": {Project: "alpha", Globals: []string{"GLOBAL_A"}, Platforms: make(map[string]json.RawMessage)},
			"org/beta":  {Project: "beta", Globals: []string{"GLOBAL_B"}, Platforms: make(map[string]json.RawMessage)},
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

// TestGenericPlatformPreservation verifies that unknown platform entries
// (like cloudflare, gitlab, etc.) are preserved during JSON unmarshal.
// This is the critical test for Finding 1 of the audit report.
func TestGenericPlatformPreservation(t *testing.T) {
	input := `{
		"repos": {
			"test/repo": {
				"project": "test",
				"platforms": {
					"vercel": {"projectId": "prj_123"},
					"cloudflare": {
						"type": "cli",
						"command": "wrangler secret put {KEY}",
						"stdin": "{VALUE}",
						"token_env": "CLOUDFLARE_API_TOKEN"
					},
					"gitlab": {
						"type": "http",
						"endpoint": "https://gitlab.com/api/v4/projects/{PROJECT_ID}/variables",
						"method": "POST",
						"headers": {"PRIVATE-TOKEN": "{GITLAB_TOKEN}"},
						"body": {"key": "{KEY}", "value": "{VALUE}", "masked": true},
						"token_env": "GITLAB_TOKEN",
						"config": {"PROJECT_ID": "12345678"}
					}
				}
			}
		}
	}`

	var cfg Config
	err := json.Unmarshal([]byte(input), &cfg)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	platforms := cfg.Repos["test/repo"].Platforms

	// Verify Vercel preserved (hardcoded platform)
	if _, ok := platforms["vercel"]; !ok {
		t.Error("vercel platform not preserved")
	}

	// Verify Cloudflare preserved (generic CLI platform)
	if _, ok := platforms["cloudflare"]; !ok {
		t.Fatal("cloudflare platform not preserved - Finding 1 NOT FIXED")
	}

	// Verify GitLab preserved (generic HTTP platform)
	if _, ok := platforms["gitlab"]; !ok {
		t.Fatal("gitlab platform not preserved - Finding 1 NOT FIXED")
	}

	// Verify Cloudflare config is valid GenericPlatformConfig
	var cloudflare GenericPlatformConfig
	if err := json.Unmarshal(platforms["cloudflare"], &cloudflare); err != nil {
		t.Fatalf("cloudflare config invalid: %v", err)
	}
	if cloudflare.Type != "cli" {
		t.Errorf("cloudflare type = %q, want %q", cloudflare.Type, "cli")
	}
	if cloudflare.Command != "wrangler secret put {KEY}" {
		t.Errorf("cloudflare command = %q, want %q", cloudflare.Command, "wrangler secret put {KEY}")
	}
	if cloudflare.TokenEnv != "CLOUDFLARE_API_TOKEN" {
		t.Errorf("cloudflare token_env = %q, want %q", cloudflare.TokenEnv, "CLOUDFLARE_API_TOKEN")
	}

	// Verify GitLab config is valid GenericPlatformConfig
	var gitlab GenericPlatformConfig
	if err := json.Unmarshal(platforms["gitlab"], &gitlab); err != nil {
		t.Fatalf("gitlab config invalid: %v", err)
	}
	if gitlab.Type != "http" {
		t.Errorf("gitlab type = %q, want %q", gitlab.Type, "http")
	}
	if gitlab.Method != "POST" {
		t.Errorf("gitlab method = %q, want %q", gitlab.Method, "POST")
	}
	if gitlab.Config["PROJECT_ID"] != "12345678" {
		t.Errorf("gitlab PROJECT_ID = %q, want %q", gitlab.Config["PROJECT_ID"], "12345678")
	}

	// Verify Vercel config is still valid
	var vercel VercelConfig
	if err := json.Unmarshal(platforms["vercel"], &vercel); err != nil {
		t.Fatalf("vercel config invalid: %v", err)
	}
	if vercel.ProjectID != "prj_123" {
		t.Errorf("vercel projectId = %q, want %q", vercel.ProjectID, "prj_123")
	}
}
