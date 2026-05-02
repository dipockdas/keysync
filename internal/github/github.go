package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Client wraps the `gh` CLI for GitHub Secrets operations.
type Client struct {
	repo string // "owner/repo"
}

// Secret represents a GitHub secret.
type Secret struct {
	Name string `json:"name"`
}

// NewClient creates a GitHub API client for the given repo.
// If repo is empty, it detects the repo from the current git remote.
func NewClient(repo string) (*Client, error) {
	if repo == "" {
		var err error
		repo, err = detectRepo()
		if err != nil {
			return nil, fmt.Errorf("detect repo: %w", err)
		}
	}
	return &Client{repo: repo}, nil
}

// detectRepo uses `gh repo view` to determine the current repo.
func detectRepo() (string, error) {
	// First check we're in a git repo
	if err := exec.Command("git", "rev-parse", "--git-dir").Run(); err != nil {
		return "", fmt.Errorf("not in a git repository (and no --repo flag provided)")
	}
	cmd := exec.Command("gh", "repo", "view", "--json", "nameWithOwner")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gh repo view: %w (is gh installed and authenticated?)", err)
	}
	var result struct {
		NameWithOwner string `json:"nameWithOwner"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return "", fmt.Errorf("parse gh output: %w", err)
	}
	if result.NameWithOwner == "" {
		return "", fmt.Errorf("could not determine GitHub repo")
	}
	return result.NameWithOwner, nil
}

// Set creates or updates a secret in GitHub.
func (c *Client) Set(name, value string) error {
	cmd := exec.Command("gh", "secret", "set", name,
		"--repo", c.repo,
	)
	cmd.Stdin = strings.NewReader(value)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh secret set: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// List returns all secret names for the repo.
func (c *Client) List() ([]string, error) {
	cmd := exec.Command("gh", "secret", "list",
		"--repo", c.repo,
		"--json", "name",
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh secret list: %w", err)
	}

	// gh returns JSON array of {"name": "SECRET_NAME"}
	var secrets []Secret
	if err := json.Unmarshal(out, &secrets); err != nil {
		return nil, fmt.Errorf("parse gh secret list: %w", err)
	}

	names := make([]string, len(secrets))
	for i, s := range secrets {
		names[i] = s.Name
	}
	return names, nil
}

// Get retrieves the value of a secret from GitHub.
// Note: GitHub's API doesn't return secret values. This uses `gh secret set` with a
// passthrough approach. For getting values, we use local store instead.
// This method is kept for the pull flow where we re-set locally.
func (c *Client) Get(name string) (string, error) {
	// gh doesn't support getting secret values directly for security reasons.
	// Secret values are only available via the API with proper auth if you have
	// the encryption key. For the pull flow, we use the local store as source.
	return "", fmt.Errorf("GitHub API does not expose secret values; use keysync pull to sync from local store")
}

// Delete removes a secret from GitHub.
func (c *Client) Delete(name string) error {
	cmd := exec.Command("gh", "secret", "delete", name,
		"--repo", c.repo,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh secret delete: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// Repo returns the repository name.
func (c *Client) Repo() string {
	return c.repo
}
