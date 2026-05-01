package platforms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

func init() {
	Register("supabase", newSupabaseFromConfig)
}

// SupabaseClient syncs secrets to Supabase using the Management API.
type SupabaseClient struct {
	token   string
	ref     string
	client  HTTPClient
	baseURL string // default: https://api.supabase.com
}

// supabaseConfig is the JSON config for Supabase from .keysync.json.
type supabaseConfig struct {
	Ref string `json:"ref"`
}

func newSupabaseFromConfig(configJSON string) (Platform, error) {
	var cfg supabaseConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, fmt.Errorf("parse supabase config: %w", err)
	}
	return NewSupabaseClient(cfg.Ref)
}

// NewSupabaseClient creates a Supabase client. Requires SUPABASE_TOKEN env var.
func NewSupabaseClient(ref string) (*SupabaseClient, error) {
	token := os.Getenv("SUPABASE_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("SUPABASE_TOKEN not set")
	}
	return &SupabaseClient{
		token:   token,
		ref:     ref,
		client:  http.DefaultClient,
		baseURL: "https://api.supabase.com",
	}, nil
}

func (s *SupabaseClient) Name() string { return "supabase" }

type supabaseSecret struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Upsert sets a secret in Supabase.
func (s *SupabaseClient) Upsert(key, value string) error {
	return s.BulkUpsert([]supabaseSecret{{Name: key, Value: value}})
}

// BulkUpsert sends multiple secrets to Supabase in a single request.
func (s *SupabaseClient) BulkUpsert(secrets []supabaseSecret) error {
	raw, _ := json.Marshal(secrets)

	url := fmt.Sprintf("%s/v1/projects/%s/secrets", s.baseURL, s.ref)
	req, err := http.NewRequest("POST", url, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("supabase API %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// UpsertMap sends multiple env vars to Supabase.
func (s *SupabaseClient) UpsertMap(envs map[string]string) error {
	secrets := make([]supabaseSecret, 0, len(envs))
	for key, value := range envs {
		secrets = append(secrets, supabaseSecret{Name: key, Value: value})
	}
	return s.BulkUpsert(secrets)
}
