// Example demonstrating how to add a custom deployment platform.
//
// To add a new platform:
//  1. Copy this file as a starting point
//  2. Rename "example" to your platform name
//  3. Implement the Upsert method with your platform's API
//  4. Add your token key name (e.g. "MY_PLATFORM_TOKEN") to the constructor
//
// The init() function auto-registers the platform so keysync sync discovers it.

package platforms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dipockdas/keysync/internal/store"
)

// exampleConfig is the JSON config section from .keysync.json for this platform.
//
// In .keysync.json the user adds:
//
//	"projects": {
//	  "my-app": {
//	    "platforms": {
//	      "example": {
//	        "apiUrl": "https://api.example.com",
//	        "projectId": "proj_123"
//	      }
//	    }
//	  }
//	}
type exampleConfig struct {
	APIURL    string `json:"apiUrl"`
	ProjectID string `json:"projectId"`
}

// exampleClient implements the Platform interface for the "example" platform.
type exampleClient struct {
	token     string
	apiURL    string
	projectID string
	client    *http.Client
}

// newExampleClient is the constructor. It resolves the API token from:
//  1. The OS keychain (global secret "EXAMPLE_TOKEN")
//  2. The EXAMPLE_TOKEN environment variable (fallback)
//
// The secretSt parameter is injected by the sync command — your constructor
// just needs to accept it and call lookupToken.
func newExampleClient(ctx context.Context, apiURL, projectID string, secretSt store.Store) (*exampleClient, error) {
	token := lookupTokenCustom(ctx, secretSt, "EXAMPLE_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("EXAMPLE_TOKEN not set (store with: keysync set EXAMPLE_TOKEN=...)")
	}
	return &exampleClient{
		token:     token,
		apiURL:    apiURL,
		projectID: projectID,
		client:    http.DefaultClient,
	}, nil
}

// init registers the platform so keysync sync can find it by name.
// configJSON is the JSON object from .keysync.json under platforms.example.
func init() {
	Register("example", func(ctx context.Context, configJSON string, secretSt store.Store) (Platform, error) {
		var cfg exampleConfig
		if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
			return nil, fmt.Errorf("parse example config: %w", err)
		}
		return newExampleClient(ctx, cfg.APIURL, cfg.ProjectID, secretSt)
	})
}

func (e *exampleClient) Name() string { return "example" }

// Upsert sends a secret to the example platform's API.
// Follow the patterns in vercel.go, railway.go, or supabase.go for
// production details like response body error parsing and secret masking.
func (e *exampleClient) Upsert(ctx context.Context, key, value string) error {
	body := map[string]string{
		"projectId": e.projectID,
		"env":       key,
		"value":     value,
	}
	raw, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "PUT", e.apiURL+"/secrets", bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("example API %d", resp.StatusCode)
	}
	return nil
}

// lookupTokenCustom resolves a platform token from the keychain (global scope)
// with environment variable fallback. Unlike lookupToken (which uses the
// TokenEnvNames map), this lets you specify any env var name directly.
func lookupTokenCustom(ctx context.Context, secretSt store.Store, envName string) string {
	if secretSt != nil {
		token, err := secretSt.Get(ctx, "global", "", "", envName)
		if err == nil && token != "" {
			return token
		}
	}
	return osGetenv(envName)
}

// ----- Test demonstrating the full custom platform lifecycle -----

func TestExamplePlatform_FullFlow(t *testing.T) {
	// Isolate the registry so this test doesn't affect real platforms.
	saved := registry
	registry = map[string]func(ctx context.Context, configJSON string, secretSt store.Store) (Platform, error){}
	defer func() { registry = saved }()

	// Register the example platform (same pattern as the init() above).
	Register("example", func(ctx context.Context, configJSON string, secretSt store.Store) (Platform, error) {
		var cfg exampleConfig
		if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
			return nil, fmt.Errorf("parse config: %w", err)
		}
		return newExampleClient(ctx, cfg.APIURL, cfg.ProjectID, secretSt)
	})

	// Seed the keychain with a fake token via the in-memory store.
	secretSt := store.NewMemoryStore()
	secretSt.Set(context.Background(), store.ScopeGlobal, "", "", "EXAMPLE_TOKEN", "test-token-abc")

	// Start a test server that acts as the platform's API.
	var gotBody struct {
		ProjectID string `json:"projectId"`
		Key       string `json:"env"`
		Value     string `json:"value"`
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("method = %q, want PUT", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token-abc" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Config JSON as it would appear in .keysync.json.
	configJSON := fmt.Sprintf(`{"apiUrl": %q, "projectId": "proj_123"}`, ts.URL)

	// Retrieve the platform from the registry.
	p, err := Get(context.Background(), "example", configJSON, secretSt)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if p.Name() != "example" {
		t.Errorf("Name() = %q, want %q", p.Name(), "example")
	}

	// Upsert a secret.
	if err := p.Upsert(context.Background(), "DATABASE_URL", "postgres://prod:5432/db"); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Verify the test server received the right values.
	if gotBody.Key != "DATABASE_URL" {
		t.Errorf("key = %q, want %q", gotBody.Key, "DATABASE_URL")
	}
	if gotBody.Value != "postgres://prod:5432/db" {
		t.Errorf("value = %q, want %q", gotBody.Value, "postgres://prod:5432/db")
	}
	if gotBody.ProjectID != "proj_123" {
		t.Errorf("projectId = %q, want %q", gotBody.ProjectID, "proj_123")
	}
}

func TestExamplePlatform_MissingToken(t *testing.T) {
	// Override osGetenv to simulate no token available anywhere.
	orig := osGetenv
	osGetenv = func(string) string { return "" }
	defer func() { osGetenv = orig }()

	_, err := newExampleClient(context.Background(), "https://example.com", "proj_123", store.NewMemoryStore())
	if err == nil {
		t.Fatal("expected error for missing token, got nil")
	}
}
