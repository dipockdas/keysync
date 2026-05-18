//go:build darwin

package keysync

import (
	"fmt"
	"os/exec"
	"strings"
)

func init() {
	isNotFound = darwinIsNotFound
	platformGet = darwinGet
	platformList = darwinList
}

// darwinGet retrieves a secret from the macOS Keychain via the security CLI.
func darwinGet(service, account string) (string, error) {
	cmd := exec.Command("security", "find-generic-password",
		"-s", service,
		"-a", account,
		"-w",
	)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("security find-generic-password: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// darwinList lists all keysync secrets by scanning the keychain.
func darwinList(scope, project, environment string) ([]SecretEntry, error) {
	cmd := exec.Command("security", "dump-keychain")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("security dump-keychain: %w", err)
	}

	records := strings.Split(string(out), "\nkeychain:")
	var entries []SecretEntry
	for _, rec := range records {
		rec = strings.TrimSpace(rec)
		if rec == "" || !strings.Contains(rec, `class: "genp"`) {
			continue
		}
		svc := findAttrValue(rec, "svce")
		if !strings.HasPrefix(svc, "keysync/") {
			continue
		}
		entryScope, entryProject, entryEnvironment := parseServiceName(svc)
		acct := findAttrValue(rec, "acct")
		if acct == "" {
			continue
		}
		if (scope == "" || entryScope == scope) &&
			(project == "" || entryProject == project) &&
			(environment == "" || entryEnvironment == environment) {
			entries = append(entries, SecretEntry{
				Scope:       entryScope,
				Project:     entryProject,
				Environment: entryEnvironment,
				Key:         acct,
			})
		}
	}
	return entries, nil
}

func darwinIsNotFound(err error) bool {
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode() == 44 // security returns 44 for item not found
	}
	return false
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
