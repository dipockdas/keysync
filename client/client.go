package client

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// GetSecret retrieves a secret from the OS secret store via the keysync CLI.
// This is a bridge for non-Go languages or when the Go library isn't available.
//
// Resolution:
//  1. Project-scoped secret (if project is non-empty)
//  2. Global secret (fallback)
func GetSecret(project, key string) (string, error) {
	args := []string{"get", key, "--unmask"}
	if project != "" {
		args = append(args, "--project", project)
	}

	cmd := exec.Command("keysync", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("keysync get: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("keysync get: %w", err)
	}

	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return "", ErrNotFound
	}
	// Output format is KEY=VALUE, extract just the value
	if _, after, ok := strings.Cut(raw, "="); ok {
		return after, nil
	}
	return raw, nil
}

// GetSecretContext is context-aware version of GetSecret.
func GetSecretContext(ctx context.Context, project, key string) (string, error) {
	return GetSecret(project, key)
}
