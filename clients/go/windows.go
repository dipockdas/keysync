//go:build windows

package keysync

import (
	"fmt"
	"strings"
	"sync"

	"github.com/danieljoos/wincred"
)

var (
	cacheMu sync.RWMutex
	entCache []SecretEntry // cached list, rebuilt on demand
)

func init() {
	isNotFound = windowsIsNotFound
	platformGet = windowsGet
	platformList = windowsList
	rebuildCache()
}

// credTarget builds the credential target name for Windows Credential Manager.
// Format: "keysync_<scope>" or "keysync_<scope>_<project>"
func credTarget(scope, project string) string {
	if project == "" || scope == "global" {
		return fmt.Sprintf("keysync_%s", scope)
	}
	return fmt.Sprintf("keysync_%s_%s", scope, project)
}

// parseCredTarget splits a Windows target name back into scope and project.
func parseCredTarget(target string) (scope, project string) {
	trimmed := strings.TrimPrefix(target, "keysync_")
	parts := strings.SplitN(trimmed, "_", 2)
	if len(parts) == 0 {
		return "global", ""
	}
	scope = parts[0]
	if scope != "global" && scope != "project" {
		return "global", ""
	}
	if len(parts) < 2 {
		return scope, ""
	}
	return scope, parts[1]
}

// windowsGet retrieves a secret from Windows Credential Manager.
func windowsGet(service, account string) (string, error) {
	// Convert service name "keysync/global" to target "keysync_global"
	target := targetFromService(service)
	cred, err := wincred.GetGenericCredential(target)
	if err != nil {
		return "", ErrNotFound
	}
	if cred.UserName != account {
		return "", ErrNotFound
	}
	return string(cred.CredentialBlob), nil
}

// windowsList lists all keysync secrets from the cached Credential Manager entries.
func windowsList(scope, project string) ([]SecretEntry, error) {
	cacheMu.RLock()
	defer cacheMu.RUnlock()

	var result []SecretEntry
	for _, e := range entCache {
		if (scope == "" || e.Scope == scope) &&
			(project == "" || e.Project == project) {
			result = append(result, e)
		}
	}
	return result, nil
}

func windowsIsNotFound(err error) bool {
	return err == ErrNotFound
}

// rebuildCache scans Credential Manager for keysync entries.
func rebuildCache() {
	all, err := wincred.List()
	if err != nil {
		return
	}

	cacheMu.Lock()
	defer cacheMu.Unlock()
	entCache = nil

	for _, cred := range all {
		if !strings.HasPrefix(cred.TargetName, "keysync_") {
			continue
		}
		entryScope, entryProject := parseCredTarget(cred.TargetName)
		entCache = append(entCache, SecretEntry{
			Scope:   entryScope,
			Project: entryProject,
			Key:     cred.UserName,
		})
	}
}

// targetFromService converts "keysync/global" → "keysync_global"
// and "keysync/project/my-app" → "keysync_project_my-app"
func targetFromService(service string) string {
	if strings.HasPrefix(service, "keysync/") {
		return "keysync_" + strings.ReplaceAll(service[8:], "/", "_")
	}
	return "keysync_" + service
}
