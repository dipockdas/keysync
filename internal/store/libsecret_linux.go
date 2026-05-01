//go:build linux

package store

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// serviceAttrValue returns the service attribute for secret-tool.
// secret-tool uses "service" and "account" as the lookup attributes.
func serviceAttrValue(scope Scope, project string) string {
	return serviceName(scope, project)
}

// LibsecretStore implements Store using Linux libsecret (`secret-tool` CLI).
type LibsecretStore struct {
	index *keyIndex
}

func NewLibsecretStore() *LibsecretStore {
	ki, err := loadKeyIndex()
	if err != nil {
		ki = &keyIndex{path: ""}
	}
	ls := &LibsecretStore{index: ki}
	ls.rebuildIndex()
	return ls
}

func (l *LibsecretStore) Get(_ context.Context, scope Scope, project, key string) (string, error) {
	svc := serviceAttrValue(scope, project)
	cmd := exec.Command("secret-tool", "lookup", "service", svc, "account", accountName(key))
	out, err := cmd.Output()
	if err != nil {
		if isSecretToolNotFound(err) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("secret-tool lookup: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (l *LibsecretStore) Set(_ context.Context, scope Scope, project, key, value string) error {
	svc := serviceAttrValue(scope, project)
	cmd := exec.Command("secret-tool", "store",
		"--label="+svc,
		"service", svc,
		"account", accountName(key),
	)
	cmd.Stdin = strings.NewReader(value + "\n")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("secret-tool store: %w: %s", err, strings.TrimSpace(string(out)))
	}

	if l.index != nil {
		_ = l.index.add(SecretEntry{Scope: scope, Project: project, Key: key})
	}
	return nil
}

func (l *LibsecretStore) Delete(_ context.Context, scope Scope, project, key string) error {
	svc := serviceAttrValue(scope, project)
	cmd := exec.Command("secret-tool", "clear", "service", svc, "account", accountName(key))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("secret-tool clear: %w: %s", err, strings.TrimSpace(string(out)))
	}

	if l.index != nil {
		_ = l.index.remove(scope, project, key)
	}
	return nil
}

func (l *LibsecretStore) List(ctx context.Context, scope Scope, project string) ([]SecretEntry, error) {
	if l.index == nil {
		return nil, nil
	}
	return l.index.list(scope, project), nil
}

// rebuildIndex scans libsecret for keysync entries and populates the index.
// Uses `secret-tool search service keysync` to find all matching entries.
func (l *LibsecretStore) rebuildIndex() {
	if l.index == nil {
		return
	}
	existing := l.index.list("", "")
	if len(existing) > 0 {
		return
	}

	// secret-tool search outputs lines like:
	//   service = keysync/global
	//   account = MY_KEY
	//   password = <value>
	// with a blank line between entries.
	cmd := exec.Command("secret-tool", "search", "service", "keysync")
	out, err := cmd.Output()
	if err != nil {
		return // no results or tool not available
	}

	lines := strings.Split(string(out), "\n")
	var currentSvc, currentAcct string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			// Blank line = end of entry
			if currentSvc != "" && currentAcct != "" && strings.HasPrefix(currentSvc, "keysync/") {
				scope, project := parseServiceName(currentSvc)
				_ = l.index.add(SecretEntry{Scope: scope, Project: project, Key: currentAcct})
			}
			currentSvc = ""
			currentAcct = ""
			continue
		}
		if strings.HasPrefix(line, "service") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				currentSvc = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "account") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				currentAcct = strings.TrimSpace(parts[1])
			}
		}
	}
	// Handle last entry if no trailing blank line
	if currentSvc != "" && currentAcct != "" && strings.HasPrefix(currentSvc, "keysync/") {
		scope, project := parseServiceName(currentSvc)
		_ = l.index.add(SecretEntry{Scope: scope, Project: project, Key: currentAcct})
	}
}

// isSecretToolNotFound checks if secret-tool returned "not found".
func isSecretToolNotFound(err error) bool {
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode() == 1 // secret-tool returns 1 when not found (stderr is empty)
	}
	return false
}

// Ensure keyIndex methods compile on Linux (types defined in keychain_darwin.go).
// On Linux, these types need to be defined here too or in a shared file.
//
// keyIndex is defined per-platform since it's used by both darwin and linux stores.
// It uses build-tagged files, so it's not available here unless shared.
// We use a local implementation instead.

// keyIndex tracks which keys exist in the store for fast listing.
type keyIndex struct {
	mu   syncStore
	path string
	keys []SecretEntry
}

type syncStore struct{}

func (s *syncStore) Lock()    {}
func (s *syncStore) Unlock()  {}
func (s *syncStore) RLock()   {}
func (s *syncStore) RUnlock() {}

func loadKeyIndex() (*keyIndex, error) {
	return &keyIndex{path: ""}, nil
}

func (ki *keyIndex) add(entry SecretEntry) error {
	// Deduplicate
	var filtered []SecretEntry
	for _, e := range ki.keys {
		if e.Scope == entry.Scope && e.Project == entry.Project && e.Key == entry.Key {
			continue
		}
		filtered = append(filtered, e)
	}
	filtered = append(filtered, entry)
	ki.keys = filtered
	return nil
}

func (ki *keyIndex) remove(scope Scope, project, key string) error {
	var filtered []SecretEntry
	for _, e := range ki.keys {
		if e.Scope == scope && e.Project == project && e.Key == key {
			continue
		}
		filtered = append(filtered, e)
	}
	ki.keys = filtered
	return nil
}

func (ki *keyIndex) list(scope Scope, project string) []SecretEntry {
	var result []SecretEntry
	for _, e := range ki.keys {
		if (scope == "" || e.Scope == scope) &&
			(project == "" || e.Project == project) {
			result = append(result, e)
		}
	}
	return result
}
