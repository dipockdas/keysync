package platforms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSupabaseUpsert_Success(t *testing.T) {
	var gotBody []supabaseSecret
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

	client := &SupabaseClient{
		token:   "test-token",
		ref:     "abc123",
		client:  ts.Client(),
		baseURL: ts.URL,
	}

	if err := client.Upsert(context.Background(), "MY_KEY", "my-value"); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	if len(gotBody) != 1 {
		t.Fatalf("got %d secrets, want 1", len(gotBody))
	}
	if gotBody[0].Name != "MY_KEY" {
		t.Errorf("Name = %q, want %q", gotBody[0].Name, "MY_KEY")
	}
	if gotBody[0].Value != "my-value" {
		t.Errorf("Value = %q, want %q", gotBody[0].Value, "my-value")
	}
}

func TestSupabaseUpsert_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer ts.Close()

	client := &SupabaseClient{
		token:   "bad-token",
		ref:     "abc123",
		client:  ts.Client(),
		baseURL: ts.URL,
	}

	err := client.Upsert(context.Background(), "MY_KEY", "val")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSupabaseNewClient_MissingToken(t *testing.T) {
	t.Setenv("SUPABASE_TOKEN", "")

	_, err := NewSupabaseClient(context.Background(), "abc123", nil)
	if err == nil {
		t.Fatal("expected error for missing SUPABASE_TOKEN, got nil")
	}
}

func TestSupabaseBulkUpsert(t *testing.T) {
	var gotBody []supabaseSecret
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	client := &SupabaseClient{
		token:   "test-token",
		ref:     "abc123",
		client:  ts.Client(),
		baseURL: ts.URL,
	}

	err := client.UpsertMap(context.Background(), map[string]string{"A": "1", "B": "2"})
	if err != nil {
		t.Fatalf("UpsertMap failed: %v", err)
	}

	if len(gotBody) != 2 {
		t.Fatalf("got %d secrets, want 2", len(gotBody))
	}
}

func TestSupabaseUpsert_Created(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	client := &SupabaseClient{
		token:   "test-token",
		ref:     "abc123",
		client:  ts.Client(),
		baseURL: ts.URL,
	}

	if err := client.Upsert(context.Background(), "KEY", "val"); err != nil {
		t.Fatalf("Upsert with 201 Created failed: %v", err)
	}
}
