//go:build linux

package store

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// serviceAttrValue returns the service attribute for secret-tool.
// secret-tool uses "service" and "account" as the lookup attributes.
func serviceAttrValue(scope Scope, project, environment string) string {
	return serviceName(scope, project, environment)
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

func (l *LibsecretStore) Get(_ context.Context, scope Scope, project, environment, key string) (string, error) {
	svc := serviceAttrValue(scope, project, environment)
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

func (l *LibsecretStore) Set(_ context.Context, scope Scope, project, environment, key, value string) error {
	svc := serviceAttrValue(scope, project, environment)
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
		_ = l.index.add(SecretEntry{Scope: scope, Project: project, Environment: environment, Key: key})
	}
	return nil
}

func (l *LibsecretStore) Delete(_ context.Context, scope Scope, project, environment, key string) error {
	svc := serviceAttrValue(scope, project, environment)
	cmd := exec.Command("secret-tool", "clear", "service", svc, "account", accountName(key))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("secret-tool clear: %w: %s", err, strings.TrimSpace(string(out)))
	}

	if l.index != nil {
		_ = l.index.remove(scope, project, environment, key)
	}
	return nil
}

func (l *LibsecretStore) List(ctx context.Context, scope Scope, project, environment string) ([]SecretEntry, error) {
	if l.index == nil {
		return nil, nil
	}
	return l.index.list(scope, project, environment), nil
}

// rebuildIndex scans libsecret for keysync entries and populates the index.
// secret-tool search requires exact attribute matching, so we list generic
// keyring items and keep only services with a keysync/ prefix.
func (l *LibsecretStore) rebuildIndex() {
	if l.index == nil {
		return
	}
	existing := l.index.list("", "", "")
	if len(existing) > 0 {
		return
	}

	for _, entry := range l.scanKeysyncEntries() {
		_ = l.index.add(entry)
	}
}

func (l *LibsecretStore) scanKeysyncEntries() []SecretEntry {
	seen := make(map[string]struct{})
	var entries []SecretEntry
	add := func(batch []SecretEntry) {
		for _, entry := range batch {
			id := entry.Scope + "\x00" + entry.Project + "\x00" + entry.Environment + "\x00" + entry.Key
			if _, exists := seen[id]; exists {
				continue
			}
			seen[id] = struct{}{}
			entries = append(entries, entry)
		}
	}
	add(l.searchKeysyncEntries(secretToolSchemaAttribute, secretToolGenericSchema))
	add(l.searchKeysyncEntries("service", "keysync/global"))
	return entries
}

func (l *LibsecretStore) searchKeysyncEntries(attribute, value string) []SecretEntry {
	cmd := exec.Command("secret-tool", "search", "--all", "--unlock", attribute, value)
	out, err := cmd.CombinedOutput()
	if err != nil && len(out) == 0 {
		return nil
	}
	return parseSecretToolSearchOutput(string(out))
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

// keyIndex tracks which keys exist in the store for fast listing.
type keyIndex struct {
	mu   sync.RWMutex
	path string
	keys []SecretEntry
}

func loadKeyIndex() (*keyIndex, error) {
	return &keyIndex{path: ""}, nil
}

func (ki *keyIndex) add(entry SecretEntry) error {
	ki.mu.Lock()
	defer ki.mu.Unlock()
	// Deduplicate
	var filtered []SecretEntry
	for _, e := range ki.keys {
		if e.Scope == entry.Scope && e.Project == entry.Project && e.Environment == entry.Environment && e.Key == entry.Key {
			continue
		}
		filtered = append(filtered, e)
	}
	filtered = append(filtered, entry)
	ki.keys = filtered
	return nil
}

func (ki *keyIndex) remove(scope Scope, project, environment, key string) error {
	ki.mu.Lock()
	defer ki.mu.Unlock()
	var filtered []SecretEntry
	for _, e := range ki.keys {
		if e.Scope == scope && e.Project == project && e.Environment == environment && e.Key == key {
			continue
		}
		filtered = append(filtered, e)
	}
	ki.keys = filtered
	return nil
}

func (ki *keyIndex) list(scope Scope, project, environment string) []SecretEntry {
	ki.mu.RLock()
	defer ki.mu.RUnlock()
	var result []SecretEntry
	for _, e := range ki.keys {
		if (scope == "" || e.Scope == scope) &&
			(project == "" || e.Project == project) &&
			(environment == "" || e.Environment == environment) {
			result = append(result, e)
		}
	}
	return result
}
