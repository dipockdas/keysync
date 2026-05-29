package platforms

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dipockdas/keysync/internal/github"
	"github.com/dipockdas/keysync/internal/store"
)

// GitHubPlatform wraps the github.Client to implement the Platform interface.
type GitHubPlatform struct {
	client    *github.Client
	variables map[string]bool
}

// GitHubConfig holds the GitHub platform configuration.
type GitHubConfig struct {
	Repo      string   `json:"repo"` // owner/repo
	Secrets   []string `json:"secrets,omitempty"`   // optional documentation; all non-variable keys use secrets API
	Variables []string `json:"variables,omitempty"` // keys pushed via gh variable set (Actions variables)
}

// NewGitHub creates a new GitHub platform instance.
func NewGitHub(ctx context.Context, configJSON string, secretSt store.Store) (Platform, error) {
	var cfg GitHubConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, err
	}

	client, err := github.NewClient(cfg.Repo)
	if err != nil {
		return nil, err
	}

	variables := make(map[string]bool, len(cfg.Variables))
	for _, k := range cfg.Variables {
		if k != "" {
			variables[k] = true
		}
	}

	return &GitHubPlatform{client: client, variables: variables}, nil
}

// Name returns the platform name.
func (g *GitHubPlatform) Name() string {
	return "github"
}

// Upsert creates or updates a GitHub Actions secret or variable.
func (g *GitHubPlatform) Upsert(ctx context.Context, key, value string) error {
	_ = ctx
	if g.variables[key] {
		return g.client.SetVariable(key, value)
	}
	return g.client.Set(key, value)
}

// KeyTarget returns how a key would be stored on GitHub (for dry-run output).
func GitHubKeyTarget(configJSON, key string) string {
	var cfg GitHubConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return "secret"
	}
	for _, k := range cfg.Variables {
		if k == key {
			return "variable"
		}
	}
	return "secret"
}

// ValidateGitHubConfig returns an error if a key is listed as both secret and variable.
func ValidateGitHubConfig(configJSON string) error {
	var cfg GitHubConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return err
	}
	secretSet := make(map[string]bool, len(cfg.Secrets))
	for _, k := range cfg.Secrets {
		secretSet[k] = true
	}
	for _, k := range cfg.Variables {
		if secretSet[k] {
			return fmt.Errorf("key %q is listed in both github.secrets and github.variables", k)
		}
	}
	return nil
}

func init() {
	Register("github", NewGitHub)
}
