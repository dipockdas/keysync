package platforms

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/dipockdas/keysync/internal/config"
	"github.com/dipockdas/keysync/internal/store"
)

// TestFirstPartyConfigs validates that the shipped first-party platform configs
// in docs/platform-configs/ are valid and can be loaded by the generic engine.
func TestFirstPartyConfigs(t *testing.T) {
	// Find project root (go up from internal/platforms to project root)
	root := filepath.Join("..", "..")
	configsDir := filepath.Join(root, "docs", "platform-configs")

	testCases := []struct {
		name         string
		file         string
		expectedType string
		expectedVars []string // Required template_vars
	}{
		{
			name:         "Vercel",
			file:         "vercel.json",
			expectedType: "http",
			expectedVars: []string{"PROJECT_ID"},
		},
		{
			name:         "Railway",
			file:         "railway.json",
			expectedType: "http",
			expectedVars: []string{"SERVICE_ID", "ENVIRONMENT"},
		},
		{
			name:         "Supabase",
			file:         "supabase.json",
			expectedType: "http",
			expectedVars: []string{"REF"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Read the config file
			configPath := filepath.Join(configsDir, tc.file)
			data, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("Failed to read %s: %v", configPath, err)
			}

			// Parse as GenericPlatformConfig
			var cfg config.GenericPlatformConfig
			if err := json.Unmarshal(data, &cfg); err != nil {
				t.Fatalf("Failed to parse %s: %v", tc.file, err)
			}

			// Validate type
			if cfg.Type != tc.expectedType {
				t.Errorf("type = %q, want %q", cfg.Type, tc.expectedType)
			}

			// Validate required fields for HTTP type
			if cfg.Type == "http" {
				if cfg.Endpoint == "" {
					t.Error("endpoint is required for HTTP type")
				}
				if cfg.Method == "" {
					t.Error("method is required for HTTP type")
				}
				if cfg.TokenEnv == "" {
					t.Error("token_env is required")
				}
			}

			// Validate template_vars contains expected fields
			for _, varName := range tc.expectedVars {
				if _, ok := cfg.TemplateVars[varName]; !ok {
					t.Errorf("template_vars missing required field %q", varName)
				}
			}

			// Validate body is present (all first-party configs need bodies)
			if cfg.Body == nil {
				t.Error("body is required")
			}

			// Try to create a platform instance (this validates the config is usable)
			// Note: This will fail on token lookup, but should parse the config successfully
			secretSt := store.NewMemoryStore()
			secretSt.Set(context.Background(), store.ScopeGlobal, "", "", cfg.TokenEnv, "test_token_for_validation")

			platform, err := NewGeneric(context.Background(), tc.name, string(data), secretSt)
			if err != nil {
				t.Fatalf("NewGeneric failed for valid config: %v", err)
			}

			if platform.Name() != tc.name {
				t.Errorf("platform.Name() = %q, want %q", platform.Name(), tc.name)
			}
		})
	}
}

// TestFirstPartyConfigs_BodyStructures validates the body structures
// match what the platform APIs expect
func TestFirstPartyConfigs_BodyStructures(t *testing.T) {
	root := filepath.Join("..", "..")
	configsDir := filepath.Join(root, "docs", "platform-configs")

	t.Run("Vercel body structure", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(configsDir, "vercel.json"))
		if err != nil {
			t.Fatalf("Failed to read vercel.json: %v", err)
		}

		var cfg config.GenericPlatformConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		// Validate body structure
		bodyMap, ok := cfg.Body.(map[string]interface{})
		if !ok {
			t.Fatalf("body should be an object, got %T", cfg.Body)
		}

		// Check required fields
		if bodyMap["key"] != "{KEY}" {
			t.Errorf("body.key = %v, want {KEY}", bodyMap["key"])
		}
		if bodyMap["value"] != "{VALUE}" {
			t.Errorf("body.value = %v, want {VALUE}", bodyMap["value"])
		}
		if bodyMap["type"] != "encrypted" {
			t.Errorf("body.type = %v, want encrypted", bodyMap["type"])
		}

		// Validate target is an array (not a template string)
		target, ok := bodyMap["target"].([]interface{})
		if !ok {
			t.Fatalf("body.target should be an array, got %T", bodyMap["target"])
		}
		if len(target) == 0 {
			t.Error("body.target array should not be empty")
		}
	})

	t.Run("Railway body structure", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(configsDir, "railway.json"))
		if err != nil {
			t.Fatalf("Failed to read railway.json: %v", err)
		}

		var cfg config.GenericPlatformConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		// Validate body structure (GraphQL)
		bodyMap, ok := cfg.Body.(map[string]interface{})
		if !ok {
			t.Fatalf("body should be an object, got %T", cfg.Body)
		}

		if _, ok := bodyMap["query"]; !ok {
			t.Error("body missing 'query' field for GraphQL")
		}
		if _, ok := bodyMap["variables"]; !ok {
			t.Error("body missing 'variables' field for GraphQL")
		}
	})

	t.Run("Supabase body structure", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(configsDir, "supabase.json"))
		if err != nil {
			t.Fatalf("Failed to read supabase.json: %v", err)
		}

		var cfg config.GenericPlatformConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		// Validate body is an array
		bodyArray, ok := cfg.Body.([]interface{})
		if !ok {
			t.Fatalf("body should be an array, got %T", cfg.Body)
		}
		if len(bodyArray) != 1 {
			t.Errorf("body array length = %d, want 1", len(bodyArray))
		}
	})
}
