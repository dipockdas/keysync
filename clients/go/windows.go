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
// Format: "keysync_<scope>" or "keysync_<scope>_<project>" or "keysync_<scope>_<project>_<env>"
func credTarget(scope, project, environment string) string {
	if project == "" || scope == "global" {
		return fmt.Sprintf("keysync_%s", scope)
	}
	if environment == "" {
		return fmt.Sprintf("keysync_%s_%s", scope, project)
	}
	return fmt.Sprintf("keysync_%s_%s_%s", scope, project, environment)
}

// parseCredTarget splits a Windows target name back into scope, project, and environment.
// e.g. "keysync_project_myapp_dev" → ("project", "myapp", "dev")
//      "keysync_project_myapp"     → ("project", "myapp", "")
//      "keysync_global"            → ("global", "", "")
func parseCredTarget(target string) (scope, project, environment string) {
	trimmed := strings.TrimPrefix(target, "keysync_")
	parts := strings.SplitN(trimmed, "_", 2)
	if len(parts) == 0 {
		return "global", "", ""
	}
	scope = parts[0]
	if scope != "global" && scope != "project" {
		return "global", "", ""
	}
	if len(parts) < 2 {
		return scope, "", ""
	}
	// Split the remainder by _; if there are multiple segments, the last is environment
	restParts := strings.Split(parts[1], "_")
	if len(restParts) > 1 {
		environment = restParts[len(restParts)-1]
		project = strings.Join(restParts[:len(restParts)-1], "_")
		return scope, project, environment
	}
	return scope, parts[1], ""
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
func windowsList(scope, project, environment string) ([]SecretEntry, error) {
	cacheMu.RLock()
	defer cacheMu.RUnlock()

	var result []SecretEntry
	for _, e := range entCache {
		if (scope == "" || e.Scope == scope) &&
			(project == "" || e.Project == project) &&
			(environment == "" || e.Environment == environment) {
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
		entryScope, entryProject, entryEnvironment := parseCredTarget(cred.TargetName)
		entCache = append(entCache, SecretEntry{
			Scope:       entryScope,
			Project:     entryProject,
			Environment: entryEnvironment,
			Key:         cred.UserName,
		})
	}
}

// targetFromService converts service names to Windows credential target names.
// "keysync/global"             → "keysync_global"
// "keysync/project/my-app"     → "keysync_project_my-app"
// "keysync/project/myapp/env/dev" → "keysync_project_myapp_dev"
// The "/env/" segment is stripped as a keyword before replacing / with _.
func targetFromService(service string) string {
	if strings.HasPrefix(service, "keysync/") {
		trimmed := service[8:]
		// Strip the /env/ keyword: replace "/env/" with "" so it does not
		// appear as a literal part of the target name.
		trimmed = strings.ReplaceAll(trimmed, "/env/", "/")
		return "keysync_" + strings.ReplaceAll(trimmed, "/", "_")
	}
	return "keysync_" + service
}
