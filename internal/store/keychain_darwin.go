//go:build darwin && cgo

package store

/*
#cgo LDFLAGS: -framework Security
#pragma GCC diagnostic ignored "-Wdeprecated-declarations"
#include <Security/Security.h>
#include <string.h>

static int ks_add(const char *svc, const char *acct, const char *val) {
    return SecKeychainAddGenericPassword(
        NULL, (UInt32)strlen(svc), svc,
        (UInt32)strlen(acct), acct,
        (UInt32)strlen(val), val,
        NULL);
}

static int ks_find(const char *svc, const char *acct, UInt32 *pwlen, void **pwdata, SecKeychainItemRef *item) {
    return SecKeychainFindGenericPassword(
        NULL, (UInt32)strlen(svc), svc,
        (UInt32)strlen(acct), acct,
        pwlen, pwdata, item);
}
*/
import "C"

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"
)

// macOS Security framework error codes.
const (
	errSecSuccess       = 0
	errSecDuplicateItem = -25299
	errSecItemNotFound  = -25300
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

// remove deletes an entry from the index. Returns true if an entry was actually removed.
func (ki *keyIndex) remove(scope Scope, project, environment, key string) bool {
	ki.mu.Lock()
	defer ki.mu.Unlock()
	filtered := make([]SecretEntry, 0, len(ki.keys))
	removed := false
	for _, e := range ki.keys {
		if e.Scope == scope && e.Project == project && e.Environment == environment && e.Key == key {
			removed = true
			continue
		}
		filtered = append(filtered, e)
	}
	if !removed {
		return false
	}
	ki.keys = filtered
	_ = ki.save()
	return true
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

// KeychainStore implements Store using the macOS Security framework API.
type KeychainStore struct {
	index *keyIndex
}

func NewKeychainStore() *KeychainStore {
	ki, err := loadKeyIndex()
	if err != nil {
		ki = &keyIndex{path: ""}
	}
	ks := &KeychainStore{index: ki}
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
		return
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
	if strings.HasPrefix(val, `"`) {
		end := strings.IndexByte(val[1:], '"')
		if end >= 0 {
			return val[1 : end+1]
		}
	}
	return strings.Trim(val, `"`)
}

// Get retrieves a secret from the macOS keychain using the Security framework API.
func (k *KeychainStore) Get(_ context.Context, scope Scope, project, environment, key string) (string, error) {
	svcName := serviceName(scope, project, environment)
	acctName := accountName(key)

	cSvc := C.CString(svcName)
	cAcct := C.CString(acctName)
	defer C.free(unsafe.Pointer(cSvc))
	defer C.free(unsafe.Pointer(cAcct))

	var pwlen C.UInt32
	var pwdata unsafe.Pointer
	var itemRef C.SecKeychainItemRef

	status := C.ks_find(cSvc, cAcct, &pwlen, &pwdata, &itemRef)
	if int(status) == errSecItemNotFound {
		return "", ErrNotFound
	}
	if int(status) != errSecSuccess {
		return "", fmt.Errorf("keychain get: OSStatus %d", int(status))
	}

	val := C.GoStringN((*C.char)(pwdata), C.int(pwlen))
	C.SecKeychainItemFreeContent(nil, pwdata)
	C.CFRelease(C.CFTypeRef(itemRef))
	return val, nil
}

// Set stores a secret in the macOS keychain using the Security framework API.
func (k *KeychainStore) Set(_ context.Context, scope Scope, project, environment, key, value string) error {
	svcName := serviceName(scope, project, environment)
	acctName := accountName(key)

	cSvc := C.CString(svcName)
	cAcct := C.CString(acctName)
	cVal := C.CString(value)
	defer C.free(unsafe.Pointer(cSvc))
	defer C.free(unsafe.Pointer(cAcct))
	defer C.free(unsafe.Pointer(cVal))

	status := C.ks_add(cSvc, cAcct, cVal)
	if int(status) == errSecDuplicateItem {
		if err := k.Delete(nil, scope, project, environment, key); err != nil {
			return fmt.Errorf("delete before re-add: %w", err)
		}
		status = C.ks_add(cSvc, cAcct, cVal)
	}
	if int(status) != errSecSuccess {
		return fmt.Errorf("keychain set: OSStatus %d", int(status))
	}

	if k.index != nil {
		_ = k.index.add(SecretEntry{Scope: scope, Project: project, Environment: environment, Key: key})
	}
	return nil
}

// Delete removes a secret from the macOS keychain using the Security framework API.
func (k *KeychainStore) Delete(_ context.Context, scope Scope, project, environment, key string) error {
	svcName := serviceName(scope, project, environment)
	acctName := accountName(key)

	cSvc := C.CString(svcName)
	cAcct := C.CString(acctName)
	defer C.free(unsafe.Pointer(cSvc))
	defer C.free(unsafe.Pointer(cAcct))

	var pwlen C.UInt32
	var pwdata unsafe.Pointer
	var itemRef C.SecKeychainItemRef

	status := C.ks_find(cSvc, cAcct, &pwlen, &pwdata, &itemRef)
	if int(status) == errSecItemNotFound {
		// Key not in keychain, but may be a stale index entry. Clean up index.
		if k.index != nil {
			if removed := k.index.remove(scope, project, environment, key); removed {
				return nil // removed stale index entry
			}
		}
		return ErrNotFound
	}
	if int(status) != errSecSuccess {
		return fmt.Errorf("keychain delete (find): OSStatus %d", int(status))
	}

	C.SecKeychainItemFreeContent(nil, pwdata)
	status = C.SecKeychainItemDelete(itemRef)
	C.CFRelease(C.CFTypeRef(itemRef))

	if int(status) != errSecSuccess {
		return fmt.Errorf("keychain delete: OSStatus %d", int(status))
	}

	if k.index != nil {
		_ = k.index.remove(scope, project, environment, key)
	}
	return nil
}

// List returns all matching secret entries from the key index.
func (k *KeychainStore) List(ctx context.Context, scope Scope, project, environment string) ([]SecretEntry, error) {
	if k.index == nil {
		return nil, nil
	}
	return k.index.list(scope, project, environment), nil
}
