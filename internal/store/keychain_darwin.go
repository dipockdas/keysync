//go:build darwin

package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// indexFilePath returns the path to the keysync key index file.
func indexFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "keysync")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "index.json"), nil
}

// keyIndex tracks which keys exist in the keychain for fast listing.
type keyIndex struct {
	mu   sync.RWMutex
	path string
	keys []SecretEntry
}

func loadKeyIndex() (*keyIndex, error) {
	path, err := indexFilePath()
	if err != nil {
		return nil, err
	}
	ki := &keyIndex{path: path}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ki, nil
		}
		return nil, err
	}
	_ = json.Unmarshal(raw, &ki.keys)
	return ki, nil
}

func (ki *keyIndex) add(entry SecretEntry) error {
	ki.mu.Lock()
	defer ki.mu.Unlock()
	// Deduplicate: remove existing entry with same scope/project/environment/key
	filtered := make([]SecretEntry, 0, len(ki.keys))
	for _, e := range ki.keys {
		if e.Scope == entry.Scope && e.Project == entry.Project && e.Environment == entry.Environment && e.Key == entry.Key {
			continue
		}
		filtered = append(filtered, e)
	}
	filtered = append(filtered, entry)
	ki.keys = filtered
	return ki.save()
}

func (ki *keyIndex) remove(scope Scope, project, environment, key string) error {
	ki.mu.Lock()
	defer ki.mu.Unlock()
	filtered := make([]SecretEntry, 0, len(ki.keys))
	for _, e := range ki.keys {
		if e.Scope == scope && e.Project == project && e.Environment == environment && e.Key == key {
			continue
		}
		filtered = append(filtered, e)
	}
	ki.keys = filtered
	return ki.save()
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

func (ki *keyIndex) save() error {
	raw, err := json.MarshalIndent(ki.keys, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ki.path, raw, 0600)
}

// KeychainStore implements Store using the macOS `security` CLI.
type KeychainStore struct {
	index *keyIndex
}

func NewKeychainStore() *KeychainStore {
	ki, err := loadKeyIndex()
	if err != nil {
		ki = &keyIndex{path: ""}
	}
	ks := &KeychainStore{index: ki}
	// Rebuild index from keychain if empty (handles migration from before index existed)
	ks.rebuildIndex()
	return ks
}

// rebuildIndex scans the keychain for keysync entries and adds any that are missing from the index.
func (k *KeychainStore) rebuildIndex() {
	if k.index == nil {
		return
	}
	existing := k.index.list("", "", "")
	if len(existing) > 0 {
		return // index already populated
	}

	out, err := exec.Command("security", "dump-keychain").Output()
	if err != nil {
		return
	}

	records := strings.Split(string(out), "\nkeychain:")
	for _, rec := range records {
		rec = strings.TrimSpace(rec)
		if rec == "" || !strings.Contains(rec, `class: "genp"`) {
			continue
		}
		// Find service name
		svc := findAttrValue(rec, "svce")
		if !strings.HasPrefix(svc, "keysync/") {
			continue
		}
		scope, project, env := parseServiceName(svc)
		acct := findAttrValue(rec, "acct")
		if acct != "" {
			_ = k.index.add(SecretEntry{Scope: scope, Project: project, Environment: env, Key: acct})
		}
	}
}

// findAttrValue extracts an attribute value from a dump-keychain record.
func findAttrValue(record, attrName string) string {
	idx := strings.Index(record, fmt.Sprintf(`"%s"`, attrName))
	if idx < 0 {
		return ""
	}
	after := record[idx+len(attrName)+2:]
	eqIdx := strings.IndexByte(after, '=')
	if eqIdx < 0 {
		return ""
	}
	val := strings.TrimSpace(after[eqIdx+1:])
	if val == "<NULL>" {
		return ""
	}
	// Take just the quoted value (first token)
	if strings.HasPrefix(val, `"`) {
		end := strings.IndexByte(val[1:], '"')
		if end >= 0 {
			return val[1 : end+1]
		}
	}
	return strings.Trim(val, `"`)
}

func (k *KeychainStore) Get(_ context.Context, scope Scope, project, environment, key string) (string, error) {
	svc := serviceName(scope, project, environment)
	cmd := exec.Command("security", "find-generic-password",
		"-s", svc,
		"-a", accountName(key),
		"-w", // output only the password
	)
	out, err := cmd.Output()
	if err != nil {
		if isNotFound(err) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("security find-generic-password: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (k *KeychainStore) Set(_ context.Context, scope Scope, project, environment, key, value string) error {
	svc := serviceName(scope, project, environment)

	// Delete existing first to avoid duplicates
	_ = exec.Command("security", "delete-generic-password",
		"-s", svc,
		"-a", accountName(key),
	).Run()

	// Use -w flag to read password from stdin (avoid exposing value in process list).
	// The security CLI reads stdin twice — once for the password and once for confirmation —
	// so we send the value twice.
	cmd := exec.Command("security", "add-generic-password",
		"-s", svc,
		"-a", accountName(key),
		"-U", // allow update (though we already deleted)
		"-w", // read password from stdin
	)
	cmd.Stdin = strings.NewReader(value + "\n" + value + "\n")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("security add-generic-password: %w: %s", err, strings.TrimSpace(string(out)))
	}

	// Update index
	if k.index != nil {
		_ = k.index.add(SecretEntry{Scope: scope, Project: project, Environment: environment, Key: key})
	}
	return nil
}

func (k *KeychainStore) Delete(_ context.Context, scope Scope, project, environment, key string) error {
	svc := serviceName(scope, project, environment)
	cmd := exec.Command("security", "delete-generic-password",
		"-s", svc,
		"-a", accountName(key),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		if isNotFound(err) {
			return ErrNotFound
		}
		return fmt.Errorf("security delete-generic-password: %w: %s", err, strings.TrimSpace(string(out)))
	}

	// Update index
	if k.index != nil {
		_ = k.index.remove(scope, project, environment, key)
	}
	return nil
}

func (k *KeychainStore) List(ctx context.Context, scope Scope, project, environment string) ([]SecretEntry, error) {
	if k.index == nil {
		return nil, nil
	}
	return k.index.list(scope, project, environment), nil
}

// isNotFound checks if the error is a "not found" from the security CLI.
func isNotFound(err error) bool {
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode() == 44 // security CLI returns 44 for "item not found"
	}
	return false
}
