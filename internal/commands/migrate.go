package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/dipockdas/keysync/internal/store"
	"github.com/spf13/cobra"
)

var migrateFile string

// migratedKey tracks a key that was migrated from .env to keysync.
type migratedKey struct {
	Key     string
	Scope   string
	Project string
}

func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate [--file .env] [--project name]",
		Short: "Import secrets from an .env file into keysync",
		Long: `Reads a .env file and interactively imports each secret into the OS secret store.
For each key, you choose the scope (global or project) and confirm storage.

After migration, keysync prints step-by-step instructions for the coding assistant
to replace direct .env usage with keysync commands.

Usage:
  keysync migrate --file .env.local
  keysync migrate --project my-app
  keysync migrate --project my-app --file .env.production`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Determine the file to read
			filePath := migrateFile
			if filePath == "" {
				filePath = ".env"
			}

			// Check the file exists
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				return fmt.Errorf("file not found: %s", filePath)
			}

			// Parse the .env file
			secrets, err := parseEnvFile(filePath)
			if err != nil {
				return fmt.Errorf("parse env file: %w", err)
			}

			if len(secrets) == 0 {
				fmt.Println("No secrets found in", filePath)
				return nil
			}

			fmt.Printf("Found %d secrets in %s\n\n", len(secrets), filePath)

			// Initialize the store (needed since PersistentPreRunE skips migrate)
			initStore()

			// Interactive migration
			scanner := bufio.NewScanner(os.Stdin)
			var migratedKeys []migratedKey

			for _, kv := range secrets {
				prompt := fmt.Sprintf("  %s=%s", kv.key, kv.value)
				if len(prompt) > 80 {
					prompt = prompt[:77] + "..."
				}
				fmt.Println(prompt)

				// Scope prompt
				defaultScope := "g"
				scopeHint := "global"
				if project != "" {
					defaultScope = "p"
					scopeHint = fmt.Sprintf("project/%s", project)
				}
				fmt.Printf("    Scope: [g]lobal / [p]roject (default: %s): ", scopeHint)
				scopeChoice := readLine(scanner, defaultScope)
				if scanner.Err() != nil {
					return fmt.Errorf("read input: %w", scanner.Err())
				}

				scope := store.ScopeGlobal
				proj := ""
				if strings.EqualFold(scopeChoice, "p") || strings.EqualFold(scopeChoice, "project") {
					scope = store.ScopeProject
					proj = project
					if proj == "" {
						fmt.Print("    Project name: ")
						proj = readLine(scanner, "")
						if proj == "" {
							fmt.Println("    Skipping — no project name given.")
							continue
						}
					}
				}

				// Store prompt
				fmt.Printf("    Store in %s? [Y/n]: ", storeLabel())
				storeChoice := readLine(scanner, "y")
				if scanner.Err() != nil {
					return fmt.Errorf("read input: %w", scanner.Err())
				}

				if strings.EqualFold(storeChoice, "n") || strings.EqualFold(storeChoice, "no") {
					fmt.Println("    Skipped.")
					continue
				}

				// Store the secret
				if err := secretSt.Set(cobraCmd.Context(), scope, proj, kv.key, kv.value); err != nil {
					fmt.Fprintf(os.Stderr, "    Error: %v\n", err)
					continue
				}

				fmt.Printf("    Stored as %s/%s\n", scopeLabel(scope, proj), kv.key)
				migratedKeys = append(migratedKeys, migratedKey{
					Key:     kv.key,
					Scope:   string(scope),
					Project: proj,
				})
			}

			// Summary
			fmt.Printf("\n=== Migration Complete ===\n")
			fmt.Printf("%d of %d secrets migrated.\n\n", len(migratedKeys), len(secrets))

			if len(migratedKeys) == 0 {
				return nil
			}

			// Print machine-parseable JSON summary for coding assistants
			fmt.Println("---MIGRATION_RESULT_START---")
			fmt.Printf(`{"project":%q,"migrated":[`, project)
			for i, m := range migratedKeys {
				if i > 0 {
					fmt.Printf(",")
				}
				fmt.Printf(`{"key":%q,"scope":%q,"project":%q}`, m.Key, m.Scope, m.Project)
			}
			fmt.Println(`]}`)
			fmt.Println("---MIGRATION_RESULT_END---")
			fmt.Println()

			// Human-readable instructions for the coding assistant
			printMigrationInstructions(migratedKeys)
			return nil
		},
	}

	cmd.Flags().StringVar(&migrateFile, "file", "", "path to .env file (default: .env)")
	return cmd
}

// parsedKey holds a single parsed key=value from the env file.
type parsedKey struct {
	key   string
	value string
}

// parseEnvFile reads a .env file and returns key=value pairs.
// It handles:
//   - Comments (lines starting with #)
//   - Empty lines
//   - Quoted values (' and ")
//   - Export prefix (export KEY=value)
func parseEnvFile(path string) ([]parsedKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var secrets []parsedKey
	lines := strings.Split(string(raw), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Strip "export " prefix
		line = strings.TrimPrefix(line, "export ")

		// Split on first =
		eq := strings.IndexByte(line, '=')
		if eq < 1 {
			continue
		}

		key := strings.TrimSpace(line[:eq])
		value := strings.TrimSpace(line[eq+1:])

		// Strip surrounding quotes
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		if key != "" {
			secrets = append(secrets, parsedKey{key: key, value: value})
		}
	}
	return secrets, nil
}

// initStore initializes the secret store for commands that skip PersistentPreRunE.
func initStore() {
	if secretSt != nil {
		return
	}
	secretSt = store.NewMemoryStore()
	// Try to open the real store
	if st, err := tryKeychain(nil); err == nil {
		secretSt = st
	} else if fs, err := store.NewFallbackStore(); err == nil {
		secretSt = fs
	}
}

// readLine reads a line from stdin, returning the default if input is empty.
func readLine(scanner *bufio.Scanner, defaultVal string) string {
	if !scanner.Scan() {
		return defaultVal
	}
	text := strings.TrimSpace(scanner.Text())
	if text == "" {
		return defaultVal
	}
	return text
}

// storeLabel returns a human-readable name for the current OS store.
func storeLabel() string {
	// Use type name as a string to avoid platform-specific build constraints
	name := fmt.Sprintf("%T", secretSt)
	switch {
	case strings.Contains(name, "Keychain"):
		return "macOS Keychain"
	case strings.Contains(name, "Fallback"):
		return "encrypted file store"
	case strings.Contains(name, "Memory"):
		return "in-memory store"
	default:
		return "secret store"
	}
}

// printMigrationInstructions outputs step-by-step instructions for replacing
// direct .env usage with keysync commands, formatted for both humans and LLMs.
func printMigrationInstructions(keys []migratedKey) {
	proj := project
	projFlag := ""
	if proj != "" {
		projFlag = fmt.Sprintf(" --project %s", proj)
	}

	fmt.Println("---INSTRUCTIONS_START---")
	fmt.Println("# Migration Instructions for Coding Assistant")
	fmt.Println()
	fmt.Println("The following secrets have been migrated from .env to keysync.")
	fmt.Println("Replace all direct .env access with keysync commands.")

	if proj == "" {
		fmt.Println()
		fmt.Println("## 1. Set a project name (recommended for multi-project setups)")
		fmt.Println("Run: keysync init")
		fmt.Printf("Then edit .keysync.json to add your project.\n")
		fmt.Println("Re-run: keysync migrate --file .env.local --project <name>")
	}

	fmt.Println()
	fmt.Println("## 2. Remove .env loading from your application")
	fmt.Println()
	fmt.Println("### Node.js / TypeScript")
	fmt.Println("```")
	fmt.Println("// REMOVE these lines:")
	fmt.Println("require('dotenv').config()           // CJS")
	fmt.Println("import 'dotenv/config'               // ESM")
	fmt.Println("import { config } from 'dotenv'      // named import")
	fmt.Println("config()")
	fmt.Println()
	fmt.Println("// REPLACE process.env.X with:")
	fmt.Println("const { execSync } = require('child_process');")
	fmt.Println("const DATABASE_URL = execSync('keysync get DATABASE_URL" + projFlag + "', { encoding: 'utf-8' }).trim();")
	fmt.Println("```")
	fmt.Println()
	fmt.Println("Or use the inject approach (recommended for migration):")
	fmt.Println("```bash")
	fmt.Println("keysync inject" + projFlag + " > .env.local")
	fmt.Println("# Then keep your existing dotenv loading — .env.local is gitignored")
	fmt.Println("```")

	fmt.Println()
	fmt.Println("### Go")
	fmt.Println("```")
	fmt.Println("import \"github.com/dipockdas/keysync/client\"")
	fmt.Println()
	fmt.Println("// REPLACE os.Getenv(\"KEY\") with:")
	fmt.Printf("val, err := client.GetSecret(%q, \"KEY\")\n", proj)
	fmt.Println("```")

	fmt.Println()
	fmt.Println("### Python")
	fmt.Println("```python")
	fmt.Println("import subprocess")
	fmt.Println()
	fmt.Println("# REPLACE os.environ[\"KEY\"] with:")
	fmt.Printf("val = subprocess.check_output([\"keysync\", \"get\", \"KEY\"%s]).decode().strip()\n", projFlag)
	fmt.Println("```")

	fmt.Println()
	fmt.Println("### Ruby")
	fmt.Println("```")
	fmt.Println("# REPLACE ENV[\"KEY\"] with:")
	fmt.Printf("val = `keysync get KEY%s`.strip\n", projFlag)
	fmt.Println("```")

	fmt.Println()
	fmt.Println("## 3. Add .env to .gitignore (if not already there)")
	fmt.Println("```")
	fmt.Println(".env")
	fmt.Println(".env.local")
	fmt.Println(".env.*")
	fmt.Println("```")

	fmt.Println()
	fmt.Println("## 4. For CI/CD")
	fmt.Println("Add secrets to GitHub Secrets via:")
	fmt.Println("```bash")
	for _, m := range keys {
		scope := "global"
		if m.Scope == "project" {
			scope = "project/" + m.Project
		}
		fmt.Printf("keysync set %s=<value> --%s\n", m.Key, scope)
	}
	fmt.Println("```")
	fmt.Println()
	fmt.Println("Then the GitHub Action (.github/workflows/sync-secrets.yml) will")
	fmt.Println("automatically push them to deployment platforms on push to main.")
	fmt.Println("---INSTRUCTIONS_END---")
}
