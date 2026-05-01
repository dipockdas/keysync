package store

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Scope defines the namespace scope for a secret.
type Scope string

const (
	ScopeGlobal  Scope = "global"
	ScopeProject Scope = "project"
)

// Store is the interface for OS-native secret storage.
// Implementations: macOS Keychain, Linux libsecret, Windows Credential Manager, encrypted file fallback.
type Store interface {
	// Get retrieves a secret value. Returns ErrNotFound if not present.
	Get(ctx context.Context, scope Scope, project, key string) (string, error)
	// Set stores a secret value.
	Set(ctx context.Context, scope Scope, project, key, value string) error
	// Delete removes a secret.
	Delete(ctx context.Context, scope Scope, project, key string) error
	// List returns all keys for the given scope/project combination.
	// If project is "", returns all keys across all projects for that scope.
	List(ctx context.Context, scope Scope, project string) ([]SecretEntry, error)
}

// SecretEntry represents a single secret in list output.
type SecretEntry struct {
	Scope   Scope  `json:"scope"`
	Project string `json:"project,omitempty"`
	Key     string `json:"key"`
}

// ErrNotFound is returned when a secret is not found in the store.
var ErrNotFound = fmt.Errorf("secret not found")

// serviceName builds the keychain service name from scope and project.
// Format: "keysync/<scope>[/<project>]"
func serviceName(scope Scope, project string) string {
	if project == "" || scope == ScopeGlobal {
		return fmt.Sprintf("keysync/%s", scope)
	}
	return fmt.Sprintf("keysync/%s/%s", scope, project)
}

// accountName builds the keychain account name for a key.
func accountName(key string) string {
	return key
}

// parseServiceName splits a service name into scope and project.
// "keysync/global" → (global, "")
// "keysync/project/my-app" → (project, "my-app")
func parseServiceName(svc string) (Scope, string) {
	trimmed := strings.TrimPrefix(svc, "keysync/")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) == 0 {
		return ScopeGlobal, ""
	}
	scope := Scope(parts[0])
	if scope != ScopeGlobal && scope != ScopeProject {
		return ScopeGlobal, ""
	}
	if len(parts) < 2 {
		return scope, ""
	}
	return scope, parts[1]
}

// MemoryStore is an in-memory implementation of Store for testing.
type MemoryStore struct {
	mu   sync.RWMutex
	data map[string]string // key = "scope/project/key"
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: make(map[string]string)}
}

func memKey(scope Scope, project, key string) string {
	return fmt.Sprintf("%s/%s/%s", scope, project, key)
}

func (m *MemoryStore) Get(_ context.Context, scope Scope, project, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.data[memKey(scope, project, key)]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}

func (m *MemoryStore) Set(_ context.Context, scope Scope, project, key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[memKey(scope, project, key)] = value
	return nil
}

func (m *MemoryStore) Delete(_ context.Context, scope Scope, project, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := memKey(scope, project, key)
	if _, ok := m.data[k]; !ok {
		return ErrNotFound
	}
	delete(m.data, k)
	return nil
}

func (m *MemoryStore) List(_ context.Context, scope Scope, project string) ([]SecretEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var entries []SecretEntry
	for k := range m.data {
		parts := strings.SplitN(k, "/", 3)
		if len(parts) < 3 {
			continue
		}
		entryScope := Scope(parts[0])
		entryProject := parts[1]
		entryKey := parts[2]
		if (scope == "" || entryScope == scope) &&
			(project == "" || entryProject == project) {
			entries = append(entries, SecretEntry{
				Scope:   entryScope,
				Project: entryProject,
				Key:     entryKey,
			})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Scope != entries[j].Scope {
			return entries[i].Scope < entries[j].Scope
		}
		if entries[i].Project != entries[j].Project {
			return entries[i].Project < entries[j].Project
		}
		return entries[i].Key < entries[j].Key
	})
	return entries, nil
}
