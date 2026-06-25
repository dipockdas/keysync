package store

import "strings"

const (
	keysyncServicePrefix      = "keysync/"
	secretToolGenericSchema   = "org.freedesktop.Secret.Generic"
	secretToolSchemaAttribute = "xdg:schema"
)

// parseSecretToolSearchOutput extracts keysync secrets from secret-tool search output.
// secret-tool writes attribute lines to stderr; callers should pass CombinedOutput.
// Non-keysync entries are ignored.
func parseSecretToolSearchOutput(out string) []SecretEntry {
	var entries []SecretEntry
	lines := strings.Split(out, "\n")
	var currentSvc, currentAcct string

	flush := func() {
		if currentSvc == "" || currentAcct == "" || !strings.HasPrefix(currentSvc, keysyncServicePrefix) {
			currentSvc = ""
			currentAcct = ""
			return
		}
		scope, project, env := parseServiceName(currentSvc)
		entries = append(entries, SecretEntry{
			Scope:       scope,
			Project:     project,
			Environment: env,
			Key:         currentAcct,
		})
		currentSvc = ""
		currentAcct = ""
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			flush()
			continue
		}
		line = strings.TrimPrefix(line, "attribute.")
		if strings.HasPrefix(line, "service") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				currentSvc = strings.TrimSpace(parts[1])
			}
			continue
		}
		if strings.HasPrefix(line, "account") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				currentAcct = strings.TrimSpace(parts[1])
			}
		}
	}
	flush()
	return entries
}
