//go:build linux

package keysync

import (
	"fmt"
	"os/exec"
	"strings"
)

func init() {
	isNotFound = linuxIsNotFound
	platformGet = linuxGet
	platformList = linuxList
}

// linuxGet retrieves a secret from libsecret via the secret-tool CLI.
func linuxGet(service, account string) (string, error) {
	cmd := exec.Command("secret-tool", "lookup",
		"service", service,
		"account", account,
	)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("secret-tool lookup: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// linuxList lists all keysync secrets by searching libsecret.
func linuxList(scope, project string) ([]SecretEntry, error) {
	cmd := exec.Command("secret-tool", "search", "service", "keysync")
	out, err := cmd.Output()
	if err != nil {
		return nil, nil // secret-tool not available or no results
	}

	lines := strings.Split(string(out), "\n")
	var entries []SecretEntry
	var currentSvc, currentAcct string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if currentSvc != "" && currentAcct != "" && strings.HasPrefix(currentSvc, "keysync/") {
				entryScope, entryProject := parseServiceName(currentSvc)
				if (scope == "" || entryScope == scope) &&
					(project == "" || entryProject == project) {
					entries = append(entries, SecretEntry{
						Scope:   entryScope,
						Project: entryProject,
						Key:     currentAcct,
					})
				}
			}
			currentSvc = ""
			currentAcct = ""
			continue
		}
		if strings.HasPrefix(line, "service") {
			parts := splitStr(line, "=", 2)
			if len(parts) == 2 {
				currentSvc = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "account") {
			parts := splitStr(line, "=", 2)
			if len(parts) == 2 {
				currentAcct = strings.TrimSpace(parts[1])
			}
		}
	}
	// Handle last entry if no trailing blank line
	if currentSvc != "" && currentAcct != "" && strings.HasPrefix(currentSvc, "keysync/") {
		entryScope, entryProject := parseServiceName(currentSvc)
		if (scope == "" || entryScope == scope) &&
			(project == "" || entryProject == project) {
			entries = append(entries, SecretEntry{
				Scope:   entryScope,
				Project: entryProject,
				Key:     currentAcct,
			})
		}
	}
	return entries, nil
}

func linuxIsNotFound(err error) bool {
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode() == 1 // secret-tool returns 1 when not found
	}
	return false
}

// splitStr splits a string by sep, returning at most n parts.
func splitStr(s, sep string, n int) []string {
	var result []string
	for i := 0; i < n-1; i++ {
		idx := strings.Index(s, sep)
		if idx < 0 {
			break
		}
		result = append(result, s[:idx])
		s = s[idx+len(sep):]
	}
	result = append(result, s)
	return result
}
