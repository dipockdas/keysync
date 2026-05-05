package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/dipockdas/keysync/internal/store"
	"github.com/spf13/cobra"
)

var (
	migrateFile   string
	migrateCloud  string
	migrateDryRun bool
)

// pullCloudVars pulls environment variables from a deployment platform CLI.
func pullCloudVars(platform string) ([]parsedKey, error) {
	switch strings.ToLower(platform) {
	case "vercel":
		return pullVercelVars()
	case "railway":
		return pullRailwayVars()
	case "supabase":
		return pullSupabaseVars()
	default:
		return nil, fmt.Errorf("unsupported cloud platform: %q (supported: vercel, railway, supabase)", platform)
	}
}

// pullVercelVars uses `vercel env pull` to a temp file, then parses it as a .env file.
func pullVercelVars() ([]parsedKey, error) {
	tmpFile, err := os.CreateTemp("", "keysync-vercel-*.env")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	cmd := exec.Command("vercel", "env", "pull", "--yes", tmpPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("vercel env pull: %w\n%s", err, strings.TrimSpace(string(out)))
	}

	return parseEnvFile(tmpPath)
}

// pullRailwayVars uses `railway variables` which outputs JSON key-value pairs.
func pullRailwayVars() ([]parsedKey, error) {
	cmd := exec.Command("railway", "variables")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("railway variables: %w", err)
	}

	var vars map[string]string
	if err := json.Unmarshal(out, &vars); err != nil {
		return nil, fmt.Errorf("parse railway variables (expected JSON): %w", err)
	}

	var secrets []parsedKey
	for k, v := range vars {
		secrets = append(secrets, parsedKey{key: k, value: v})
	}
	return secrets, nil
}

// pullSupabaseVars uses `supabase secrets list` with --json flag, falling back to table parsing.
func pullSupabaseVars() ([]parsedKey, error) {
	// Try JSON output first (newer supabase CLI versions)
	cmd := exec.Command("supabase", "secrets", "list", "--json")
	out, err := cmd.Output()
	if err == nil {
		var entries []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		}
		if json.Unmarshal(out, &entries) == nil {
			var secrets []parsedKey
			for _, e := range entries {
				secrets = append(secrets, parsedKey{key: e.Name, value: e.Value})
			}
			return secrets, nil
		}
	}

	// Fallback: parse table output
	cmd = exec.Command("supabase", "secrets", "list")
	out, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("supabase secrets list: %w", err)
	}

	return parseSupabaseTable(string(out))
}

// parseSupabaseTable parses the table output of `supabase secrets list`.
func parseSupabaseTable(output string) ([]parsedKey, error) {
	lines := strings.Split(output, "\n")
	var secrets []parsedKey
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip header, separator, and empty lines
		if line == "" || strings.Contains(line, "Name") || strings.HasPrefix(line, "─") {
			continue
		}
		parts := strings.SplitN(line, "│", 2)
		if len(parts) < 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if name != "" {
			secrets = append(secrets, parsedKey{key: name, value: value})
		}
	}
	return secrets, nil
}

// migratedKey tracks a key that was migrated from .env to keysync.
type migratedKey struct {
	Key     string
	Scope   string
	Project string
}

func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate [--file .env] [--cloud vercel|railway|supabase] [--project name]",
		Short: "Import secrets into keysync from an .env file or cloud platform",
		Long: F(`Import secrets into keysync from a local {c}.env{/c} file or directly from
a cloud platform (Vercel, Railway, or Supabase).

For each key found, you interactively choose the scope (global, project, or
project+env) and confirm storage. Use {c}--dry-run{/c} to preview without side effects.

After migration, keysync prints step-by-step instructions for replacing direct
{c}.env{/c} usage with keysync commands (including code examples for Node.js, Go,
Python, and Ruby).

{b}Examples:{/b}
  {c}keysync migrate --file .env.local{/c}
  {c}keysync migrate --file .env --project my-app{/c}
  {c}keysync migrate --cloud vercel --project my-app{/c}
  {c}keysync migrate --cloud railway --project my-app{/c}
  {c}keysync migrate --cloud supabase --project my-app --dry-run{/c}

{b}See also:{/b}
  Migration docs: {u}https://github.com/dipockdas/keysync#migration{/u}
  Tutorial: {u}https://github.com/dipockdas/keysync/blob/main/docs/tutorial-go-project.md{/u}`),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			var secrets []parsedKey
			var sourceLabel string

			// Determine source: --cloud takes precedence over --file
			if migrateCloud != "" {
				sourceLabel = fmt.Sprintf("cloud platform: %s", migrateCloud)
				fmt.Printf("Pulling secrets from %s...\n", sourceLabel)
				var err error
				secrets, err = pullCloudVars(migrateCloud)
				if err != nil {
					return fmt.Errorf("pull from %s: %w", migrateCloud, err)
				}
			} else {
				// Determine the file to read
				filePath := migrateFile
				if filePath == "" {
					filePath = ".env"
				}
				sourceLabel = filePath

				// Check the file exists
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					return fmt.Errorf("file not found: %s. Use --file to specify a path or --cloud to pull from a deployment platform", filePath)
				}

				// Parse the .env file
				var err error
				secrets, err = parseEnvFile(filePath)
				if err != nil {
					return fmt.Errorf("parse env file: %w", err)
				}
			}

			if len(secrets) == 0 {
				fmt.Println("No secrets found in", sourceLabel)
				return nil
			}

			fmt.Printf("Found %d secrets in %s\n", len(secrets), sourceLabel)
			if migrateDryRun {
				fmt.Println("🔎 DRY RUN — no secrets will be stored.")
			}
			fmt.Println()

			// Initialize the store (needed since PersistentPreRunE skips migrate)
			initStore()

			// Interactive migration
			scanner := bufio.NewScanner(os.Stdin)
			var migratedKeys []migratedKey

			for _, kv := range secrets {
				if err := validateKeyName(kv.key); err != nil {
					fmt.Printf("  SKIP: %s (%v)\n", kv.key, err)
					continue
				}
				fmt.Printf("  %s=***\n", kv.key)

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
				env := ""
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
					// Use default env (production) for project-scoped migration
					env = envFlag
				}

				if migrateDryRun {
					fmt.Printf("    [DRY RUN] Would store as %s/%s%s\n", scopeLabel(scope, proj), kv.key, envLabel(env))
				} else {
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
					if err := secretSt.Set(cobraCmd.Context(), scope, proj, env, kv.key, kv.value); err != nil {
						fmt.Fprintf(os.Stderr, "    Error: %v\n", err)
						continue
					}

					fmt.Printf("    Stored as %s/%s%s\n", scopeLabel(scope, proj), kv.key, envLabel(env))
				}

				migratedKeys = append(migratedKeys, migratedKey{
					Key:     kv.key,
					Scope:   string(scope),
					Project: proj,
				})
			}

			// Summary
			summaryHeader := "Migration Complete"
			if migrateDryRun {
				summaryHeader = "DRY RUN Complete"
			}
			fmt.Printf("\n=== %s ===\n", summaryHeader)
			if migrateDryRun {
				fmt.Printf("%d of %d secrets would be migrated.\n", len(migratedKeys), len(secrets))
				fmt.Println("Run without --dry-run to apply.")
			} else {
				fmt.Printf("%d of %d secrets migrated.\n", len(migratedKeys), len(secrets))
			}
			fmt.Println()

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
	cmd.Flags().StringVar(&migrateCloud, "cloud", "", "pull from cloud platform: vercel, railway, or supabase")
	cmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "preview migration without storing secrets")
	return cmd
}

// envLabel returns a human-readable environment suffix.
func envLabel(env string) string {
	if env != "" {
		return fmt.Sprintf(" (env: %s)", env)
	}
	return ""
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

	if migrateDryRun {
		fmt.Println()
		fmt.Println("## NOTE: This was a DRY RUN")
		fmt.Println("No secrets were actually stored. Run the following to apply:")
		fmt.Println("```bash")
		fmt.Println("keysync migrate --file .env")
		fmt.Println("```")
	}

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
