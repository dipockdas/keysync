package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// execCommand is overridable for testing.
var execCommand = exec.Command

// SetExecCommandForTesting replaces the exec.Command function used internally by
// the github package. It returns a restore function. Only use this in tests.
func SetExecCommandForTesting(fn func(name string, arg ...string) *exec.Cmd) func() {
	orig := execCommand
	execCommand = fn
	return func() { execCommand = orig }
}

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
			return nil, err
		}
	}
	return &Client{repo: repo}, nil
}

// detectRepo uses `gh repo view` to determine the current repo.
func detectRepo() (string, error) {
	// First check we're in a git repo
	if err := execCommand("git", "rev-parse", "--git-dir").Run(); err != nil {
		return "", fmt.Errorf("could not detect repo: not inside a git repository. Run from your project directory or use --repo owner/name")
	}
	cmd := execCommand("gh", "repo", "view", "--json", "nameWithOwner")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("could not detect repo: 'gh repo view' failed — is gh installed and authenticated? Use --repo owner/name to set it explicitly")
	}
	var result struct {
		NameWithOwner string `json:"nameWithOwner"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return "", fmt.Errorf("could not detect repo: failed to parse 'gh repo view' output: %w", err)
	}
	if result.NameWithOwner == "" {
		return "", fmt.Errorf("could not detect repo: 'gh repo view' returned empty. Use --repo owner/name to set it explicitly")
	}
	return result.NameWithOwner, nil
}

// Set creates or updates a secret in GitHub Actions secrets.
func (c *Client) Set(name, value string) error {
	return c.runGhWithStdin(value, "secret", "set", name, "--repo", c.repo)
}

// SetVariable creates or updates a GitHub Actions variable (non-secret).
func (c *Client) SetVariable(name, value string) error {
	return c.runGhWithStdin(value, "variable", "set", name, "--repo", c.repo)
}

func (c *Client) runGhWithStdin(value string, args ...string) error {
	cmd := execCommand("gh", args...)
	cmd.Stdin = strings.NewReader(value)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		op := "gh"
		if len(args) > 0 {
			op = fmt.Sprintf("gh %s %s", args[0], args[1])
		}
		return fmt.Errorf("%s: %w: %s", op, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// List returns all secret names for the repo.
func (c *Client) List() ([]string, error) {
	cmd := execCommand("gh", "secret", "list",
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
	cmd := execCommand("gh", "secret", "delete", name,
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
