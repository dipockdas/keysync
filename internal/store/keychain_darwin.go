//go:build darwin && cgo

package store

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation
#pragma GCC diagnostic ignored "-Wdeprecated-declarations"
#include <CoreFoundation/CoreFoundation.h>
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

static OSStatus ks_unlock_login_keychain(const char *keychainPath, UInt32 passwordLen, const void *password) {
    if (keychainPath == NULL || passwordLen == 0) {
        return errSecSuccess;
    }
    SecKeychainRef keychain = NULL;
    OSStatus status = SecKeychainOpen(keychainPath, &keychain);
    if (status != errSecSuccess) {
        return status;
    }
    status = SecKeychainUnlock(keychain, passwordLen, password, TRUE);
    CFRelease(keychain);
    return status;
}

// ks_find_item_ref locates a generic password item without decrypting its value.
static int ks_find_item_ref(const char *svc, const char *acct, SecKeychainItemRef *itemOut) {
    CFStringRef service = CFStringCreateWithCString(NULL, svc, kCFStringEncodingUTF8);
    CFStringRef account = CFStringCreateWithCString(NULL, acct, kCFStringEncodingUTF8);
    if (service == NULL || account == NULL) {
        if (service != NULL) CFRelease(service);
        if (account != NULL) CFRelease(account);
        return (int)errSecAllocate;
    }

    const void *keys[] = {
        kSecClass, kSecAttrService, kSecAttrAccount, kSecReturnRef, kSecMatchLimit,
    };
    const void *values[] = {
        kSecClassGenericPassword, service, account, kCFBooleanTrue, kSecMatchLimitOne,
    };
    CFDictionaryRef query = CFDictionaryCreate(
        NULL, keys, values, 5,
        &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
    CFRelease(service);
    CFRelease(account);
    if (query == NULL) {
        return (int)errSecAllocate;
    }

    CFTypeRef result = NULL;
    OSStatus status = SecItemCopyMatching(query, &result);
    CFRelease(query);
    if (status != errSecSuccess) {
        return (int)status;
    }
    *itemOut = (SecKeychainItemRef)result;
    return (int)errSecSuccess;
}

static int ks_trust_app_on_item_ref(SecKeychainItemRef item, SecTrustedApplicationRef app) {
    SecAccessRef access = NULL;
    OSStatus status = SecKeychainItemCopyAccess(item, &access);
    if (status != errSecSuccess) {
        return (int)status;
    }

    CFArrayRef aclList = NULL;
    status = SecAccessCopyACLList(access, &aclList);
    if (status != errSecSuccess) {
        CFRelease(access);
        return (int)status;
    }

    OSStatus lastStatus = errSecItemNotFound;
    CFIndex count = CFArrayGetCount(aclList);
    for (CFIndex i = 0; i < count; i++) {
        SecACLRef acl = (SecACLRef)CFArrayGetValueAtIndex(aclList, i);
        CFArrayRef appList = NULL;
        CFStringRef desc = NULL;
        CSSM_ACL_KEYCHAIN_PROMPT_SELECTOR prompt;
        status = SecACLCopySimpleContents(acl, &appList, &desc, &prompt);
        if (status != errSecSuccess) {
            continue;
        }
        if (appList == NULL) {
            if (desc != NULL) {
                CFRelease(desc);
            }
            continue;
        }

        CFIndex appCount = CFArrayGetCount(appList);
        CFMutableArrayRef newApps = CFArrayCreateMutableCopy(NULL, appCount + 1, appList);
        if (newApps == NULL) {
            CFRelease(appList);
            if (desc != NULL) {
                CFRelease(desc);
            }
            continue;
        }
        CFArrayAppendValue(newApps, app);

        status = SecACLSetSimpleContents(acl, newApps, desc, &prompt);
        CFRelease(newApps);
        CFRelease(appList);
        if (desc != NULL) {
            CFRelease(desc);
        }
        if (status == errSecSuccess) {
            lastStatus = errSecSuccess;
        }
    }

    if (lastStatus == errSecSuccess) {
        status = SecKeychainItemSetAccess(item, access);
        if (status != errSecSuccess) {
            lastStatus = status;
        }
    }

    CFRelease(aclList);
    CFRelease(access);
    return (int)lastStatus;
}

typedef struct {
    SecTrustedApplicationRef app;
} ks_trust_session;

static int ks_trust_session_begin(ks_trust_session *session, const char *appPath) {
    memset(session, 0, sizeof(*session));
    return (int)SecTrustedApplicationCreateFromPath(appPath, &session->app);
}

static void ks_trust_session_end(ks_trust_session *session) {
    if (session->app != NULL) {
        CFRelease(session->app);
        session->app = NULL;
    }
}

static int ks_trust_session_item(
    ks_trust_session *session,
    const char *svc,
    const char *acct
) {
    SecKeychainItemRef item = NULL;
    int status = ks_find_item_ref(svc, acct, &item);
    if (status != errSecSuccess) {
        return status;
    }
    status = ks_trust_app_on_item_ref(item, session->app);
    CFRelease(item);
    return status;
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

	"golang.org/x/term"
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
	// Do not call trustBinaryForItem here — macOS prompts for the login keychain password on
	// every ACL change. New items trust the creating binary by default; use keysync trust to
	// repair partition lists after upgrading the keysync binary.
	return nil
}

// resolveExecutablePath returns the absolute, symlink-resolved path to this binary.
func resolveExecutablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

// keychainPartitionList returns the partition list for set-generic-password-partition-list.
// Signed binaries include team ID (TN3127). Unsigned builds rely on application ACL trust instead.
func keychainPartitionList(exe string) string {
	const base = "apple-tool:,apple:,codesign:"
	if tid := teamIDFromExecutable(exe); tid != "" {
		return base + ",teamid:" + tid
	}
	return base
}

// teamIDFromExecutable reads TeamIdentifier from codesign -dv output (empty if unsigned).
func teamIDFromExecutable(exe string) string {
	out, err := exec.Command("codesign", "-dv", exe).CombinedOutput()
	if err != nil {
		return ""
	}
	return parseTeamIdentifier(string(out))
}

func parseTeamIdentifier(codesignOutput string) string {
	for line := range strings.SplitSeq(codesignOutput, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "TeamIdentifier=") {
			tid := strings.TrimPrefix(line, "TeamIdentifier=")
			if tid != "" && tid != "not set" {
				return tid
			}
		}
	}
	return ""
}

// loginKeychainPath returns the path to the user's login keychain, or "" if not found.
func loginKeychainPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	for _, name := range []string{"login.keychain-db", "login.keychain"} {
		p := filepath.Join(home, "Library", "Keychains", name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// readLoginKeychainPassword prompts once for the login keychain password (not stored).
const trustProgressBarWidth = 24

// formatTrustProgress renders an inline progress bar for keysync trust.
func formatTrustProgress(done, total int) string {
	if total <= 0 {
		return "  0/0 trusted"
	}
	if done > total {
		done = total
	}
	filled := (done * trustProgressBarWidth) / total
	bar := strings.Repeat("█", filled) + strings.Repeat("░", trustProgressBarWidth-filled)
	return fmt.Sprintf("  %s  %d/%d trusted", bar, done, total)
}

type trustProgress struct {
	total      int
	isTerminal bool
}

func newTrustProgress(total int) *trustProgress {
	return &trustProgress{
		total:      total,
		isTerminal: term.IsTerminal(int(os.Stderr.Fd())),
	}
}

func (p *trustProgress) update(done int) {
	if !p.isTerminal {
		return
	}
	fmt.Fprint(os.Stderr, "\r"+formatTrustProgress(done, p.total))
}

func (p *trustProgress) finish(done int) {
	if p.isTerminal {
		fmt.Fprint(os.Stderr, "\r"+formatTrustProgress(done, p.total)+"\n")
		return
	}
	fmt.Fprintf(os.Stderr, "%s\n", formatTrustProgress(done, p.total))
}

func readLoginKeychainPassword() ([]byte, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return nil, fmt.Errorf("keysync trust must be run in an interactive terminal")
	}
	fmt.Fprint(os.Stderr, "Login keychain password (used once for trust, not stored): ")
	pw, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return nil, err
	}
	return pw, nil
}

func trustBinaryForItemOnKeychain(keychain, partitions, service, account string, keychainPassword []byte) error {
	args := []string{
		"set-generic-password-partition-list",
		"-s", service,
		"-a", account,
		"-S", partitions,
	}
	if len(keychainPassword) > 0 {
		args = append(args, "-k", string(keychainPassword))
	}
	if keychain != "" {
		args = append(args, keychain)
	}
	return exec.Command("security", args...).Run()
}

type keychainTrustSession struct {
	c C.ks_trust_session
}

func unlockLoginKeychain(keychain string, keychainPassword []byte) error {
	if keychain == "" || len(keychainPassword) == 0 {
		return nil
	}
	cKeychain := C.CString(keychain)
	defer C.free(unsafe.Pointer(cKeychain))
	status := C.ks_unlock_login_keychain(
		cKeychain,
		C.UInt32(len(keychainPassword)),
		unsafe.Pointer(&keychainPassword[0]),
	)
	if int(status) != errSecSuccess {
		return fmt.Errorf("unlock login keychain: OSStatus %d", int(status))
	}
	return nil
}

func beginKeychainTrustSession(appPath string) (*keychainTrustSession, error) {
	cApp := C.CString(appPath)
	defer C.free(unsafe.Pointer(cApp))

	session := &keychainTrustSession{}
	status := C.ks_trust_session_begin(&session.c, cApp)
	if int(status) != errSecSuccess {
		return nil, fmt.Errorf("keychain trust session: OSStatus %d", int(status))
	}
	return session, nil
}

func (s *keychainTrustSession) trustItem(service, account string) error {
	cSvc := C.CString(service)
	cAcct := C.CString(account)
	defer C.free(unsafe.Pointer(cSvc))
	defer C.free(unsafe.Pointer(cAcct))

	status := C.ks_trust_session_item(&s.c, cSvc, cAcct)
	if int(status) == errSecItemNotFound {
		return ErrNotFound
	}
	if int(status) != errSecSuccess {
		return fmt.Errorf("keychain application trust: OSStatus %d", int(status))
	}
	return nil
}

func (s *keychainTrustSession) close() {
	C.ks_trust_session_end(&s.c)
}

// RepairTrust re-applies keychain ACLs for every indexed secret (no value reads).
// Prompts once for the login keychain password, then updates all indexed items.
func (k *KeychainStore) RepairTrust() (succeeded, failed int, err error) {
	if k.index == nil {
		return 0, 0, nil
	}
	entries := k.index.list("", "", "")
	if len(entries) == 0 {
		return 0, 0, nil
	}

	exe, err := resolveExecutablePath()
	if err != nil {
		return 0, len(entries), err
	}
	teamID := teamIDFromExecutable(exe)
	signed := teamID != ""

	fmt.Fprintf(os.Stderr, "Trusting binary: %s\n", exe)
	if signed {
		fmt.Fprintf(os.Stderr, "Signed build (team %s) — updating partition lists (no per-key prompts).\n", teamID)
	} else {
		fmt.Fprintf(os.Stderr, "Unsigned build — updating application ACLs (use make build-signed to avoid this).\n")
	}

	keychainPassword, err := readLoginKeychainPassword()
	if err != nil {
		return 0, 0, err
	}
	defer func() {
		for i := range keychainPassword {
			keychainPassword[i] = 0
		}
	}()

	partitions := keychainPartitionList(exe)
	keychain := loginKeychainPath()

	if err := unlockLoginKeychain(keychain, keychainPassword); err != nil {
		return 0, len(entries), err
	}

	var session *keychainTrustSession
	if !signed {
		session, err = beginKeychainTrustSession(exe)
		if err != nil {
			return 0, len(entries), err
		}
		defer session.close()
	}

	progress := newTrustProgress(len(entries))
	for i, e := range entries {
		svc := serviceName(e.Scope, e.Project, e.Environment)
		acct := accountName(e.Key)

		if signed {
			if err := trustBinaryForItemOnKeychain(keychain, partitions, svc, acct, keychainPassword); err != nil {
				failed++
			} else {
				succeeded++
			}
		} else {
			// Unsigned: partition list alone does not grant access; add this binary to item ACLs.
			_ = trustBinaryForItemOnKeychain(keychain, partitions, svc, acct, keychainPassword)
			if err := session.trustItem(svc, acct); err != nil {
				failed++
			} else {
				succeeded++
			}
		}
		progress.update(i + 1)
	}
	progress.finish(len(entries))
	return succeeded, failed, nil
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
