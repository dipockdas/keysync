package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/dipockdas/keysync/internal/config"
)

// TestGenericPlatformEndToEnd verifies that generic platforms survive the full
// load → configuredPlatforms → getPlatformConfigJSON → platforms.Get() flow.
// This is the end-to-end test for Finding 1 of the audit report.
func TestGenericPlatformEndToEnd(t *testing.T) {
	// Create a temporary .keysync.json with generic platforms
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".keysync.json")

	// Marshal platform configs
	vercelJSON, _ := json.Marshal(config.VercelConfig{ProjectID: "prj_123"})
	cloudflareJSON, _ := json.Marshal(config.GenericPlatformConfig{
		Type:     "cli",
		Command:  "wrangler secret put {KEY}",
		Stdin:    "{VALUE}",
		TokenEnv: "CLOUDFLARE_API_TOKEN",
	})
	gitlabJSON, _ := json.Marshal(config.GenericPlatformConfig{
		Type:     "http",
		Endpoint: "https://gitlab.com/api/v4/projects/{PROJECT_ID}/variables",
		Method:   "POST",
		Headers:  map[string]string{"PRIVATE-TOKEN": "{GITLAB_TOKEN}"},
		Body:     map[string]interface{}{"key": "{KEY}", "value": "{VALUE}", "masked": true},
		TokenEnv: "GITLAB_TOKEN",
		Config:   map[string]string{"PROJECT_ID": "12345678"},
	})

	cfg := &config.Config{
		Repos: map[string]config.RepoConfig{
			"test/repo": {
				Project: "test-app",
				Globals: []string{"STRIPE_KEY"},
				Platforms: map[string]json.RawMessage{
					"vercel":     vercelJSON,
					"cloudflare": cloudflareJSON,
					"gitlab":     gitlabJSON,
				},
			},
		},
	}

	if err := config.SaveConfig(cfg, configPath); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Load the config back
	loaded, _, err := config.LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify the repo exists
	rc, ok := loaded.Repos["test/repo"]
	if !ok {
		t.Fatal("repo 'test/repo' not found after load")
	}

	// Test configuredPlatforms() - should return all 3 platforms
	platformsCfg := rc.Platforms
	platformNames := configuredPlatforms(platformsCfg)

	// Verify all platforms are detected
	found := make(map[string]bool)
	for _, name := range platformNames {
		found[name] = true
	}

	if !found["vercel"] {
		t.Error("vercel not in configuredPlatforms output")
	}
	if !found["cloudflare"] {
		t.Fatal("cloudflare not in configuredPlatforms output - Finding 1 NOT FIXED")
	}
	if !found["gitlab"] {
		t.Fatal("gitlab not in configuredPlatforms output - Finding 1 NOT FIXED")
	}

	// Test getPlatformConfigJSON() for each platform
	vercelConfig := getPlatformConfigJSON(platformsCfg, "vercel", "test/repo")
	if vercelConfig == "" {
		t.Error("getPlatformConfigJSON returned empty for vercel")
	}

	cloudflareConfig := getPlatformConfigJSON(platformsCfg, "cloudflare", "test/repo")
	if cloudflareConfig == "" {
		t.Fatal("getPlatformConfigJSON returned empty for cloudflare - Finding 1 NOT FIXED")
	}

	gitlabConfig := getPlatformConfigJSON(platformsCfg, "gitlab", "test/repo")
	if gitlabConfig == "" {
		t.Fatal("getPlatformConfigJSON returned empty for gitlab - Finding 1 NOT FIXED")
	}

	// Verify the returned JSON is valid GenericPlatformConfig
	var cfCfg config.GenericPlatformConfig
	if err := json.Unmarshal([]byte(cloudflareConfig), &cfCfg); err != nil {
		t.Fatalf("cloudflare config not valid JSON: %v", err)
	}
	if cfCfg.Type != "cli" {
		t.Errorf("cloudflare type = %q, want %q", cfCfg.Type, "cli")
	}

	var glCfg config.GenericPlatformConfig
	if err := json.Unmarshal([]byte(gitlabConfig), &glCfg); err != nil {
		t.Fatalf("gitlab config not valid JSON: %v", err)
	}
	if glCfg.Type != "http" {
		t.Errorf("gitlab type = %q, want %q", glCfg.Type, "http")
	}

	// Clean up
	os.RemoveAll(dir)
}
