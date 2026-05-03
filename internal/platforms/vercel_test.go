package platforms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVercelUpsert_Success(t *testing.T) {
	var gotBody vercelEnvRequest
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
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	client := &VercelClient{
		token:     "test-token",
		projectID: "proj_abc",
		targets:   []string{"production"},
		client:    ts.Client(),
		baseURL:   ts.URL,
	}

	if err := client.Upsert("MY_KEY", "my-value"); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	if gotBody.Key != "MY_KEY" {
		t.Errorf("Key = %q, want %q", gotBody.Key, "MY_KEY")
	}
	if gotBody.Value != "my-value" {
		t.Errorf("Value = %q, want %q", gotBody.Value, "my-value")
	}
	if gotBody.Type != "encrypted" {
		t.Errorf("Type = %q, want %q", gotBody.Type, "encrypted")
	}
}

func TestVercelUpsert_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"message":"project not found"}}`))
	}))
	defer ts.Close()

	client := &VercelClient{
		token:     "test-token",
		projectID: "bad-project",
		targets:   []string{"production"},
		client:    ts.Client(),
		baseURL:   ts.URL,
	}

	err := client.Upsert("MY_KEY", "val")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestVercelUpsert_ErrorInResponseBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"error":{"message":"rate limit exceeded"}}`))
	}))
	defer ts.Close()

	client := &VercelClient{
		token:     "test-token",
		projectID: "proj_abc",
		targets:   []string{"production"},
		client:    ts.Client(),
		baseURL:   ts.URL,
	}

	err := client.Upsert("MY_KEY", "val")
	if err == nil {
		t.Fatal("expected error for error in response body, got nil")
	}
}

func TestVercelNewClient_MissingToken(t *testing.T) {
	t.Setenv("VERCEL_TOKEN", "")

	_, err := NewVercelClient(context.Background(), "proj_abc", nil, nil)
	if err == nil {
		t.Fatal("expected error for missing VERCEL_TOKEN, got nil")
	}
}

func TestVercelBulkUpsert(t *testing.T) {
	var callCount int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	client := &VercelClient{
		token:     "test-token",
		projectID: "proj_abc",
		targets:   []string{"production"},
		client:    ts.Client(),
		baseURL:   ts.URL,
	}

	err := client.BulkUpsert(map[string]string{"A": "1", "B": "2"})
	if err != nil {
		t.Fatalf("BulkUpsert failed: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}
