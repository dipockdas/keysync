package commands

import (
	"fmt"

	"github.com/dipockdas/keysync/internal/github"
)

// setGithubSecret writes a secret to GitHub Secrets.
// Used by rotate (which intentionally updates everywhere).
func setGithubSecret(key, value string) error {
	gh, err := github.NewClient(repoFlag)
	if err != nil {
		return fmt.Errorf("github client: %w", err)
	}
	return gh.Set(key, value)
}
