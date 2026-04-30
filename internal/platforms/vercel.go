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
	Register("vercel", newVercelFromConfig)
}

// VercelClient syncs secrets to Vercel Environment Variables.
type VercelClient struct {
	token     string
	projectID string
	targets   []string
}

// vercelConfig is the JSON config for Vercel from .keysync.json.
type vercelConfig struct {
	ProjectID string   `json:"projectId"`
	Target    []string `json:"target,omitempty"`
}

func newVercelFromConfig(configJSON string) (Platform, error) {
	var cfg vercelConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, fmt.Errorf("parse vercel config: %w", err)
	}
	return NewVercelClient(cfg.ProjectID, cfg.Target)
}

// NewVercelClient creates a Vercel client. Requires VERCEL_TOKEN env var.
func NewVercelClient(projectID string, targets []string) (*VercelClient, error) {
	token := os.Getenv("VERCEL_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("VERCEL_TOKEN not set")
	}
	if len(targets) == 0 {
		targets = []string{"production", "preview"}
	}
	return &VercelClient{
		token:     token,
		projectID: projectID,
		targets:   targets,
	}, nil
}

func (v *VercelClient) Name() string { return "vercel" }

// vercelEnvRequest is the request body for creating/updating env vars.
type vercelEnvRequest struct {
	Key    string   `json:"key"`
	Value  string   `json:"value"`
	Target []string `json:"target"`
	Type   string   `json:"type"`
}

// Upsert creates or updates an environment variable in Vercel.
func (v *VercelClient) Upsert(key, value string) error {
	body := vercelEnvRequest{
		Key:    key,
		Value:  value,
		Target: v.targets,
		Type:   "encrypted",
	}
	raw, _ := json.Marshal(body)

	url := fmt.Sprintf("https://api.vercel.com/v9/projects/%s/env", v.projectID)
	req, err := http.NewRequest("POST", url, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+v.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("vercel API %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	if err := json.Unmarshal(respBody, &result); err == nil && result.Error != nil {
		return fmt.Errorf("vercel error: %s", result.Error.Message)
	}

	return nil
}

// BulkUpsert sends multiple env vars in a single request.
func (v *VercelClient) BulkUpsert(envs map[string]string) error {
	for key, value := range envs {
		if err := v.Upsert(key, value); err != nil {
			return fmt.Errorf("upsert %s: %w", key, err)
		}
	}
	return nil
}
