package platforms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRailwayUpsert_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q", r.Header.Get("Content-Type"))
		}

		// Verify the request body
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		query, ok := body["query"].(string)
		if !ok || !strings.Contains(query, "variableUpsert") {
			t.Errorf("query missing variableUpsert mutation")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"variableUpsert":{"id":"123","name":"MY_KEY"}}}`))
	}))
	defer ts.Close()

	client := &RailwayClient{
		token:       "test-token",
		environment: "production",
		service:     "svc_abc",
		client:      ts.Client(),
		baseURL:     ts.URL,
	}

	if err := client.Upsert(context.Background(), "MY_KEY", "my-value"); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}
}

func TestRailwayUpsert_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"errors":[{"message":"invalid token"}]}`))
	}))
	defer ts.Close()

	client := &RailwayClient{
		token:       "bad-token",
		environment: "production",
		service:     "svc_abc",
		client:      ts.Client(),
		baseURL:     ts.URL,
	}

	err := client.Upsert(context.Background(), "MY_KEY", "val")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRailwayUpsert_GraphQLError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"errors":[{"message":"variable already exists"}]}`))
	}))
	defer ts.Close()

	client := &RailwayClient{
		token:       "test-token",
		environment: "production",
		service:     "svc_abc",
		client:      ts.Client(),
		baseURL:     ts.URL,
	}

	err := client.Upsert(context.Background(), "MY_KEY", "val")
	if err == nil {
		t.Fatal("expected error for GraphQL error, got nil")
	}
}

func TestRailwayNewClient_MissingToken(t *testing.T) {
	t.Setenv("RAILWAY_TOKEN", "")

	_, err := NewRailwayClient(context.Background(), "production", "svc_abc", nil)
	if err == nil {
		t.Fatal("expected error for missing RAILWAY_TOKEN, got nil")
	}
}

func TestRailwayBulkUpsert(t *testing.T) {
	var callCount int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"variableUpsert":{"id":"123","name":"KEY"}}}`))
	}))
	defer ts.Close()

	client := &RailwayClient{
		token:       "test-token",
		environment: "production",
		service:     "svc_abc",
		client:      ts.Client(),
		baseURL:     ts.URL,
	}

	err := client.BulkUpsert(map[string]string{"A": "1", "B": "2"})
	if err != nil {
		t.Fatalf("BulkUpsert failed: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}
