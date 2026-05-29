package platforms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dipockdas/keysync/internal/store"
)

// DEPRECATED: This built-in Railway implementation is deprecated in favor of
// declarative generic configs. It will be removed in keysync 2.0.
//
// To migrate, use the generic HTTP platform configuration. See:
//   docs/platform-configs/railway.json
//
// The generic config provides the same functionality with better timeout
// support and maintainability. Legacy configs (without "type": "http") will
// continue to work via this implementation until keysync 2.0.
//
// Introduced: keysync 0.x
// Deprecated: keysync 1.0
// Removal planned: keysync 2.0

func init() {
	Register("railway", newRailwayFromConfig)
}

// RailwayClient syncs secrets to Railway using the GraphQL API.
type RailwayClient struct {
	token       string
	environment string
	service     string
	client      HTTPClient
	baseURL     string // default: https://backboard.railway.app/graphql/v2
}

// railwayConfig is the JSON config for Railway from .keysync.json.
type railwayConfig struct {
	Environment string `json:"environment,omitempty"`
	Service     string `json:"service,omitempty"`
}

func newRailwayFromConfig(ctx context.Context, configJSON string, secretSt store.Store) (Platform, error) {
	var cfg railwayConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, fmt.Errorf("parse railway config: %w", err)
	}
	return NewRailwayClient(ctx, cfg.Environment, cfg.Service, secretSt)
}

// NewRailwayClient creates a Railway client. The token is read from the secret store
// (global scope, key "RAILWAY_TOKEN"), falling back to the RAILWAY_TOKEN env var.
func NewRailwayClient(ctx context.Context, environment, service string, secretSt store.Store) (*RailwayClient, error) {
	token := lookupToken(ctx, secretSt, "railway")
	if token == "" {
		return nil, fmt.Errorf("RAILWAY_TOKEN not set (store it with: keysync set RAILWAY_TOKEN=...)")
	}
	return &RailwayClient{
		token:       token,
		environment: environment,
		service:     service,
		client: &http.Client{
			Timeout: 30 * time.Second, // Match generic platform timeout
		},
		baseURL: "https://backboard.railway.app/graphql/v2",
	}, nil
}

func (r *RailwayClient) Name() string { return "railway" }

// Upsert sets an environment variable in Railway.
func (r *RailwayClient) Upsert(ctx context.Context, key, value string) error {
	mutation := `mutation($input: VariableUpsertInput!) {
		variableUpsert(input: $input) {
			id
			name
		}
	}`

	vars := map[string]any{
		"input": map[string]any{
			"name":        key,
			"value":       value,
			"projectId":   r.service,
			"environment": r.environment,
		},
	}

	body := map[string]any{
		"query":     mutation,
		"variables": vars,
	}
	raw, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", r.baseURL, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("railway API %d: %s", resp.StatusCode, sanitizeResponseBody(respBody))
	}

	var gqlResponse struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(respBody, &gqlResponse); err == nil && len(gqlResponse.Errors) > 0 {
		return fmt.Errorf("railway error: %s", gqlResponse.Errors[0].Message)
	}

	return nil
}

// BulkUpsert sends multiple env vars to Railway.
func (r *RailwayClient) BulkUpsert(envs map[string]string) error {
	ctx := context.Background() // TODO: Accept context as parameter in future
	for key, value := range envs {
		if err := r.Upsert(ctx, key, value); err != nil {
			return fmt.Errorf("upsert %s: %w", key, err)
		}
	}
	return nil
}
