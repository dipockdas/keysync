package platforms

import (
	"context"
	"encoding/json"

	"github.com/dipockdas/keysync/internal/github"
	"github.com/dipockdas/keysync/internal/store"
)

// GitHubPlatform wraps the github.Client to implement the Platform interface.
type GitHubPlatform struct {
	client *github.Client
}

// GitHubConfig holds the GitHub platform configuration.
type GitHubConfig struct {
	Repo string `json:"repo"` // Repository name (owner/repo)
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

	return &GitHubPlatform{client: client}, nil
}

// Name returns the platform name.
func (g *GitHubPlatform) Name() string {
	return "github"
}

// Upsert creates or updates a secret in GitHub Secrets.
func (g *GitHubPlatform) Upsert(ctx context.Context, key, value string) error {
	// Note: github.Client.Set doesn't support context yet, but we accept it for interface compliance
	_ = ctx
	return g.client.Set(key, value)
}

func init() {
	Register("github", NewGitHub)
}
