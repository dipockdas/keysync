package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// FallbackStore stores secrets in an encrypted JSON file at ~/.config/keysync/store.json.
// Used on headless Linux or when D-Bus is unavailable.
// NOTE: Encryption (via crypto package) will be added in Phase 2.
// For now, the file is permission-guarded (0600).
type FallbackStore struct {
	mu       sync.RWMutex
	filePath string
	data     map[string]string // key = "scope/project/key"
}

func NewFallbackStore() (*FallbackStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("home dir: %w", err)
	}
	dir := filepath.Join(home, ".config", "keysync")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}
	fp := filepath.Join(dir, "store.json")
	s := &FallbackStore{
		filePath: fp,
		data:     make(map[string]string),
	}
	// Load existing data
	if _, err := os.Stat(fp); err == nil {
		raw, err := os.ReadFile(fp)
		if err != nil {
			return nil, fmt.Errorf("read store: %w", err)
		}
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &s.data); err != nil {
				return nil, fmt.Errorf("parse store: %w", err)
			}
		}
	}
	return s, nil
}

func (f *FallbackStore) save() error {
	raw, err := json.MarshalIndent(f.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal store: %w", err)
	}
	return os.WriteFile(f.filePath, raw, 0600)
}

func (f *FallbackStore) Get(_ context.Context, scope Scope, project, key string) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	k := memKey(scope, project, key)
	v, ok := f.data[k]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}

func (f *FallbackStore) Set(_ context.Context, scope Scope, project, key, value string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data[memKey(scope, project, key)] = value
	return f.save()
}

func (f *FallbackStore) Delete(_ context.Context, scope Scope, project, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	k := memKey(scope, project, key)
	if _, ok := f.data[k]; !ok {
		return ErrNotFound
	}
	delete(f.data, k)
	return f.save()
}

func (f *FallbackStore) List(_ context.Context, scope Scope, project string) ([]SecretEntry, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	prefix := memKey(scope, project, "")
	var entries []SecretEntry
	for k := range f.data {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		parts := strings.SplitN(k, "/", 3)
		if len(parts) < 3 {
			continue
		}
		entries = append(entries, SecretEntry{
			Scope:   Scope(parts[0]),
			Project: parts[1],
			Key:     parts[2],
		})
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

// FilePath returns the path to the store file.
func (f *FallbackStore) FilePath() string {
	return f.filePath
}
