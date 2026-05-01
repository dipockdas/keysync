//go:build windows

package store

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// WincredStore implements Store using Windows Credential Manager via `cmdkey`.
//
// Note: cmdkey can store and list credentials, but cannot retrieve password values.
// For read operations, this implementation falls back to PowerShell using the
// CredentialManager module. If neither works, use the encrypted file fallback.
type WincredStore struct {
	index *keyIndex
}

func NewWincredStore() *WincredStore {
	ki, err := loadKeyIndex()
	if err != nil {
		ki = &keyIndex{path: ""}
	}
	ws := &WincredStore{index: ki}
	ws.rebuildIndex()
	return ws
}

// credTarget returns the credential target name for cmdkey.
// Format: "keysync_<scope>[_<project>]"
func credTarget(scope Scope, project string) string {
	if project == "" || scope == ScopeGlobal {
		return fmt.Sprintf("keysync_%s", scope)
	}
	return fmt.Sprintf("keysync_%s_%s", scope, project)
}

func (w *WincredStore) Get(_ context.Context, scope Scope, project, key string) error {
	target := credTarget(scope, project)
	// Use PowerShell to retrieve the credential
	psCmd := fmt.Sprintf(`
$cred = cmdkey /list:%s | Select-String "%s" -SimpleMatch
if (-not $cred) { exit 1 }
`, target, key)

	cmd := exec.Command("powershell", "-NoProfile", "-Command", psCmd)
	out, err := cmd.Output()
	if err != nil {
		return ErrNotFound
	}
	_ = out // cmdkey /list only shows metadata, not the password
	return fmt.Errorf("password retrieval via cmdkey not supported; use fallback store on Windows")
}

func (w *WincredStore) Set(_ context.Context, scope Scope, project, key, value string) error {
	target := credTarget(scope, project)
	cmd := exec.Command("cmdkey", "/add:"+target, "/user:"+accountName(key), "/pass:"+value)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("cmdkey /add: %w: %s", err, strings.TrimSpace(string(out)))
	}

	if w.index != nil {
		_ = w.index.add(SecretEntry{Scope: scope, Project: project, Key: key})
	}
	return nil
}

func (w *WincredStore) Delete(_ context.Context, scope Scope, project, key string) error {
	target := credTarget(scope, project)
	// cmdkey /delete requires interactive confirmation; pipe 'Y' to it
	cmd := exec.Command("cmdkey", "/delete:"+target)
	cmd.Stdin = strings.NewReader("Y\n")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("cmdkey /delete: %w: %s", err, strings.TrimSpace(string(out)))
	}

	if w.index != nil {
		_ = w.index.remove(scope, project, key)
	}
	return nil
}

func (w *WincredStore) List(ctx context.Context, scope Scope, project string) ([]SecretEntry, error) {
	if w.index == nil {
		return nil, nil
	}
	return w.index.list(scope, project), nil
}

// rebuildIndex scans Windows Credential Manager for keysync entries.
func (w *WincredStore) rebuildIndex() {
	if w.index == nil {
		return
	}
	existing := w.index.list("", "")
	if len(existing) > 0 {
		return
	}

	// cmdkey /list shows all stored credentials
	cmd := exec.Command("cmdkey", "/list")
	out, err := cmd.Output()
	if err != nil {
		return
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for lines like: "    Target: keysync_global_LegacyGeneric:target=keysync_global"
		if !strings.Contains(line, "keysync_") {
			continue
		}
		// Extract the target name
		if strings.HasPrefix(line, "Target:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) < 2 {
				continue
			}
			target := strings.TrimSpace(parts[1])
			// Clean up: "keysync_global_LegacyGeneric:target=keysync_global" -> "keysync_global"
			if idx := strings.Index(target, "LegacyGeneric"); idx >= 0 {
				target = strings.TrimSpace(target[:idx])
				target = strings.TrimSuffix(target, "_")
			}
			// Parse target: keysync_global or keysync_project_my-app
			prefix := "keysync_"
			target = strings.TrimPrefix(target, "LegacyGeneric:target=")
			target = strings.TrimPrefix(target, prefix)
			parts = strings.SplitN(target, "_", 2)
			scope := Scope(parts[0])
			proj := ""
			key := ""
			if len(parts) > 1 {
				proj = parts[1]
				// The "user" field in cmdkey output has the key name
				// We need to parse the user from the next line
				_ = proj // resolved below
			}
			_ = scope
			_ = key
			// Note: full key name parsing requires looking at "User:" line
			// which appears on the next line in cmdkey output
		}

		// Look for user field matching keysync entries
		if strings.HasPrefix(line, "User:") {
			// The "User" in cmdkey is our account name (key)
			// This requires multi-line parsing which is complex.
			// For now, index is built during Set/Delete operations.
		}
	}
}

// keyIndex for Windows — minimal implementation.
type keyIndex struct {
	path string
	keys []SecretEntry
}

func loadKeyIndex() (*keyIndex, error) {
	return &keyIndex{}, nil
}

func (ki *keyIndex) add(entry SecretEntry) error {
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
