package platforms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dipockdas/keysync/internal/store"
)

// TestGenericEngine_VercelEquivalence proves generic config produces
// identical HTTP requests to built-in Vercel client
func TestGenericEngine_VercelEquivalence(t *testing.T) {
	var gotRequest *http.Request
	var gotBody map[string]any

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRequest = r
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"created": true}`))
	}))
	defer ts.Close()

	// Generic config
	configJSON := `{
		"type": "http",
		"endpoint": "` + ts.URL + `/v9/projects/{PROJECT_ID}/env",
		"method": "POST",
		"token_env": "VERCEL_TOKEN",
		"headers": {
			"Authorization": "Bearer {TOKEN}",
			"Content-Type": "application/json"
		},
		"body": {"key": "{KEY}", "value": "{VALUE}", "target": ["production"], "type": "encrypted"},
		"template_vars": {
			"PROJECT_ID": "prj_test123"
		}
	}`

	secretSt := store.NewMemoryStore()
	secretSt.Set(context.Background(), store.ScopeGlobal, "", "", "VERCEL_TOKEN", "test_token_abc")

	platform, err := NewGeneric(context.Background(), "vercel", configJSON, secretSt)
	if err != nil {
		t.Fatalf("NewGeneric failed: %v", err)
	}

	err = platform.Upsert(context.Background(), "MY_KEY", "my_value")
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Verify HTTP request matches built-in client behavior
	if gotRequest.Method != "POST" {
		t.Errorf("method = %s, want POST", gotRequest.Method)
	}

	if gotRequest.URL.Path != "/v9/projects/prj_test123/env" {
		t.Errorf("path = %s, want /v9/projects/prj_test123/env", gotRequest.URL.Path)
	}

	if auth := gotRequest.Header.Get("Authorization"); auth != "Bearer test_token_abc" {
		t.Errorf("Authorization = %s, want Bearer test_token_abc", auth)
	}

	// Verify body fields (JSON field order is not guaranteed)
	if gotBody["key"] != "MY_KEY" {
		t.Errorf("key = %v, want MY_KEY", gotBody["key"])
	}
	if gotBody["value"] != "my_value" {
		t.Errorf("value = %v, want my_value", gotBody["value"])
	}
	if gotBody["type"] != "encrypted" {
		t.Errorf("type = %v, want encrypted", gotBody["type"])
	}
	target, ok := gotBody["target"].([]interface{})
	if !ok || len(target) != 1 || target[0] != "production" {
		t.Errorf("target = %v, want [production]", gotBody["target"])
	}
}

// TestGenericEngine_RailwayEquivalence proves Railway GraphQL works via generic
func TestGenericEngine_RailwayEquivalence(t *testing.T) {
	var gotRequest *http.Request
	var gotBody map[string]any

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRequest = r
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"variableUpsert":{"id":"123","name":"MY_KEY"}}}`))
	}))
	defer ts.Close()

	configJSON := `{
		"type": "http",
		"endpoint": "` + ts.URL + `",
		"method": "POST",
		"token_env": "RAILWAY_TOKEN",
		"headers": {
			"Authorization": "Bearer {TOKEN}",
			"Content-Type": "application/json"
		},
		"body": {
			"query": "mutation($input: VariableUpsertInput!) { variableUpsert(input: $input) { id name } }",
			"variables": {
				"input": {
					"name": "{KEY}",
					"value": "{VALUE}",
					"projectId": "{SERVICE_ID}",
					"environment": "{ENVIRONMENT}"
				}
			}
		},
		"template_vars": {
			"SERVICE_ID": "svc_test123",
			"ENVIRONMENT": "production"
		}
	}`

	secretSt := store.NewMemoryStore()
	secretSt.Set(context.Background(), store.ScopeGlobal, "", "", "RAILWAY_TOKEN", "railway_token_xyz")

	platform, err := NewGeneric(context.Background(), "railway", configJSON, secretSt)
	if err != nil {
		t.Fatalf("NewGeneric failed: %v", err)
	}

	err = platform.Upsert(context.Background(), "MY_KEY", "my_value")
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Verify GraphQL structure
	if gotRequest.Method != "POST" {
		t.Errorf("method = %s, want POST", gotRequest.Method)
	}

	if _, ok := gotBody["query"]; !ok {
		t.Error("GraphQL query missing")
	}

	vars, ok := gotBody["variables"].(map[string]any)
	if !ok {
		t.Fatal("variables missing")
	}

	input, ok := vars["input"].(map[string]any)
	if !ok {
		t.Fatal("input missing")
	}

	if input["name"] != "MY_KEY" {
		t.Errorf("name = %v, want MY_KEY", input["name"])
	}
	if input["value"] != "my_value" {
		t.Errorf("value = %v, want my_value", input["value"])
	}
	if input["projectId"] != "svc_test123" {
		t.Errorf("projectId = %v, want svc_test123", input["projectId"])
	}
}

// TestGenericEngine_SupabaseEquivalence proves Supabase array payload works
func TestGenericEngine_SupabaseEquivalence(t *testing.T) {
	var gotRequest *http.Request
	var gotBody []map[string]string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRequest = r
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	configJSON := `{
		"type": "http",
		"endpoint": "` + ts.URL + `/v1/projects/{REF}/secrets",
		"method": "POST",
		"token_env": "SUPABASE_TOKEN",
		"headers": {
			"Authorization": "Bearer {TOKEN}",
			"Content-Type": "application/json"
		},
		"body": [{"name": "{KEY}", "value": "{VALUE}"}],
		"template_vars": {
			"REF": "abc123def456"
		}
	}`

	secretSt := store.NewMemoryStore()
	secretSt.Set(context.Background(), store.ScopeGlobal, "", "", "SUPABASE_TOKEN", "supabase_token_xyz")

	platform, err := NewGeneric(context.Background(), "supabase", configJSON, secretSt)
	if err != nil {
		t.Fatalf("NewGeneric failed: %v", err)
	}

	err = platform.Upsert(context.Background(), "MY_KEY", "my_value")
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Verify array payload
	if gotRequest.URL.Path != "/v1/projects/abc123def456/secrets" {
		t.Errorf("path = %s", gotRequest.URL.Path)
	}

	if len(gotBody) != 1 {
		t.Fatalf("body length = %d, want 1", len(gotBody))
	}

	if gotBody[0]["name"] != "MY_KEY" {
		t.Errorf("name = %s, want MY_KEY", gotBody[0]["name"])
	}
	if gotBody[0]["value"] != "my_value" {
		t.Errorf("value = %s, want my_value", gotBody[0]["value"])
	}
}
