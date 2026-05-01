package keysync

import (
	"context"
	"sort"
	"strings"
	"sync"
)

// MemoryStore is an in-memory implementation of Store for testing.
// Use this in unit tests to avoid OS keychain dependencies.
//
// Example:
//
//	store := keysync.NewMemoryStore()
//	store.SetSecret(ctx, "global", "", "DATABASE_URL", "postgres://test:test@localhost/db")
//	val, _ := store.GetSecret(ctx, "project", "my-app", "DATABASE_URL")
type MemoryStore struct {
	mu   sync.RWMutex
	data map[string]string
}

// NewMemoryStore creates an empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: make(map[string]string)}
}

func (m *MemoryStore) memKey(scope, project, key string) string {
	return strings.Join([]string{scope, project, key}, "/")
}

// GetSecret retrieves a secret from memory.
// If not found in project scope, falls back to global scope.
func (m *MemoryStore) GetSecret(_ context.Context, scope, project, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Try requested scope first
	if v, ok := m.data[m.memKey(scope, project, key)]; ok {
		return v, nil
	}

	// Fall back to global
	if scope != "global" {
		if v, ok := m.data[m.memKey("global", "", key)]; ok {
			return v, nil
		}
	}

	return "", ErrNotFound
}

// SetSecret stores a secret in memory.
func (m *MemoryStore) SetSecret(_ context.Context, scope, project, key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[m.memKey(scope, project, key)] = value
	return nil
}

// List returns all keys for a given scope/project combination.
// Empty scope and project returns all keys.
func (m *MemoryStore) List(scope, project string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var keys []string
	for k := range m.data {
		parts := strings.SplitN(k, "/", 3)
		if len(parts) != 3 {
			continue
		}
		entryScope, entryProject, entryKey := parts[0], parts[1], parts[2]
		if (scope == "" || entryScope == scope) &&
			(project == "" || entryProject == project) {
			keys = append(keys, entryKey)
		}
	}
	sort.Strings(keys)
	return keys
}
