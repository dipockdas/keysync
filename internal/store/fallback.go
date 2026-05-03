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

	"github.com/dipockdas/keysync/internal/crypto"
)

// FallbackStore stores secrets in an encrypted JSON file at ~/.config/keysync/store.json.
// Used on headless Linux or when D-Bus is unavailable.
// The file is encrypted using NaCl box (Curve25519 + XSalsa20-Poly1305).
// The encryption key is stored alongside at ~/.config/keysync/key (0600 permissions).
type FallbackStore struct {
	mu       sync.RWMutex
	filePath string
	keyPath  string
	box      *crypto.SealedBox
	data     map[string]string // key = "scope/project/env/key"
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
	return newFallbackStore(filepath.Join(dir, "store.json"))
}

// newFallbackStore creates a FallbackStore at the given file path.
// This is exposed for testing; use NewFallbackStore for production.
func newFallbackStore(fp string) (*FallbackStore, error) {
	keyPath := filepath.Join(filepath.Dir(fp), "key")

	// Load or generate encryption key
	box, err := loadOrGenerateKey(keyPath)
	if err != nil {
		return nil, fmt.Errorf("encryption key: %w", err)
	}

	s := &FallbackStore{
		filePath: fp,
		keyPath:  keyPath,
		box:      box,
		data:     make(map[string]string),
	}

	// Load existing data
	if _, err := os.Stat(fp); err == nil {
		raw, err := os.ReadFile(fp)
		if err != nil {
			return nil, fmt.Errorf("read store: %w", err)
		}
		if len(raw) > 0 {
			// Try to decrypt first
			decrypted, err := box.Decrypt(raw)
			if err != nil {
				// Maybe it's plaintext JSON from a previous version — migrate
				if raw[0] == '{' {
					if err := json.Unmarshal(raw, &s.data); err != nil {
						return nil, fmt.Errorf("parse store: %w", err)
					}
					// Re-save encrypted to migrate
					if err := s.save(); err != nil {
						return nil, fmt.Errorf("migrate to encrypted: %w", err)
					}
					return s, nil
				}
				return nil, fmt.Errorf("decrypt store: %w", err)
			}
			if err := json.Unmarshal(decrypted, &s.data); err != nil {
				return nil, fmt.Errorf("parse store: %w", err)
			}
		}
	}
	return s, nil
}

// loadOrGenerateKey loads an existing encryption key or generates a new one.
func loadOrGenerateKey(keyPath string) (*crypto.SealedBox, error) {
	raw, err := os.ReadFile(keyPath)
	if err == nil {
		key, err := crypto.BytesToKey(raw)
		if err != nil {
			return nil, err
		}
		return crypto.NewSealedBoxFromSecret(key), nil
	}

	if !os.IsNotExist(err) {
		return nil, err
	}

	// Generate a new key
	key, err := crypto.GenerateRandomKey()
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(keyPath, crypto.KeyToBytes(key), 0600); err != nil {
		return nil, err
	}

	return crypto.NewSealedBoxFromSecret(key), nil
}

func (f *FallbackStore) save() error {
	raw, err := json.MarshalIndent(f.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal store: %w", err)
	}
	encrypted, err := f.box.Encrypt(raw)
	if err != nil {
		return fmt.Errorf("encrypt store: %w", err)
	}
	return os.WriteFile(f.filePath, encrypted, 0600)
}

func (f *FallbackStore) Get(_ context.Context, scope Scope, project, environment, key string) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	k := memKey(scope, project, environment, key)
	v, ok := f.data[k]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}

func (f *FallbackStore) Set(_ context.Context, scope Scope, project, environment, key, value string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data[memKey(scope, project, environment, key)] = value
	return f.save()
}

func (f *FallbackStore) Delete(_ context.Context, scope Scope, project, environment, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	k := memKey(scope, project, environment, key)
	if _, ok := f.data[k]; !ok {
		return ErrNotFound
	}
	delete(f.data, k)
	return f.save()
}

func (f *FallbackStore) List(_ context.Context, scope Scope, project, environment string) ([]SecretEntry, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var entries []SecretEntry
	for k := range f.data {
		parts := strings.SplitN(k, "/", 4)
		if len(parts) < 4 {
			continue
		}
		entryScope := Scope(parts[0])
		entryProject := parts[1]
		entryEnv := parts[2]
		entryKey := parts[3]
		if (scope == "" || entryScope == scope) &&
			(project == "" || entryProject == project) &&
			(environment == "" || entryEnv == environment) {
			entries = append(entries, SecretEntry{
				Scope:       entryScope,
				Project:     entryProject,
				Environment: entryEnv,
				Key:         entryKey,
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
		if entries[i].Environment != entries[j].Environment {
			return entries[i].Environment < entries[j].Environment
		}
		return entries[i].Key < entries[j].Key
	})
	return entries, nil
}

// FilePath returns the path to the store file.
func (f *FallbackStore) FilePath() string {
	return f.filePath
}
